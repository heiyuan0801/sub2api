import copy
import datetime as dt
import json
import pathlib
import tempfile
import unittest
from unittest.mock import patch

import backup
import operations
import proxy_agent
import retention


class ControlsTests(unittest.TestCase):
    def test_rejects_stale_and_injected_commands(self):
        snap = {
            "updated_at": dt.datetime.now(dt.timezone.utc).isoformat(),
            "nodes": [
                {
                    "id": "us",
                    "control": {"weight": 100, "max_concurrency": 50, "revision": 2},
                }
            ],
        }
        lines, revisions = proxy_agent.commands(snap, {"us"})
        self.assertEqual(len(lines), 6)
        self.assertEqual(revisions, {"us": 2})
        nano = copy.deepcopy(snap)
        nano["updated_at"] = "2026-10-25T00:00:00.123456789Z"
        self.assertEqual(
            proxy_agent.commands(
                nano, {"us"}, dt.datetime(2026, 10, 25, tzinfo=dt.timezone.utc)
            ),
            (lines, revisions),
        )
        bad = copy.deepcopy(snap)
        bad["nodes"][0]["control"]["weight"] = "1\nshutdown sessions server gateway/us"
        with self.assertRaises(ValueError):
            proxy_agent.commands(bad, {"us"})
        bad = copy.deepcopy(snap)
        bad["updated_at"] = "2020-01-01T00:00:00Z"
        with self.assertRaises(ValueError):
            proxy_agent.commands(bad, {"us"})

    def test_failed_apply_does_not_acknowledge(self):
        with tempfile.TemporaryDirectory() as directory:
            proof = pathlib.Path(directory) / "proof"
            pathlib.Path(str(proof) + ".proxy").write_text(
                json.dumps(
                    {
                        "updated_at": dt.datetime.now(dt.timezone.utc).isoformat(),
                        "nodes": [
                            {
                                "id": "a",
                                "control": {
                                    "weight": 10,
                                    "max_concurrency": 5,
                                    "revision": 1,
                                },
                            }
                        ],
                    }
                )
            )
            with (
                patch.object(
                    proxy_agent, "apply", side_effect=RuntimeError("no socket")
                ),
                self.assertRaises(RuntimeError),
            ):
                proxy_agent.tick(proof, "/missing", {"a"})
            self.assertFalse(pathlib.Path(str(proof) + ".proxy-ack").exists())


class BackupTests(unittest.TestCase):
    def test_archive_never_overwrites_conflicting_wal(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            source = root / "source"
            source.write_bytes(b"original")
            name = "000000010000000000000001"
            backup.archive(source, root, name)
            backup.archive(source, root, name)
            source.write_bytes(b"different")
            with self.assertRaises(ValueError):
                backup.archive(source, root, name)
            self.assertEqual((root / name).read_bytes(), b"original")
            with self.assertRaises(ValueError):
                backup.archive(source, root, "../../outside")

    def test_restore_refuses_existing_directory(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            with self.assertRaises(ValueError):
                backup.prepare_restore(root, root, root, None, None, 29432)


class RetentionTests(unittest.TestCase):
    def test_keeps_two_verified_bases_and_refuses_incomplete_backup(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            for index in range(4):
                base = root / f"base-{index}"
                base.mkdir()
                (base / "backup_manifest").write_text("{}")
                base.with_name(base.name + ".verified.json").write_text(
                    json.dumps(
                        {
                            "manifest_sha256": backup.sha(base / "backup_manifest"),
                            "verified_at": f"2020-01-0{index + 1}T00:00:00+00:00",
                        }
                    )
                )
            with patch.object(backup, "verify") as verify:
                result = retention.plan(root, root)
                self.assertEqual(len(result["backups"]), 2)
                self.assertEqual(len(result["retained_backups"]), 2)
                self.assertEqual(verify.call_count, 2)
            (root / "base-pending").mkdir()
            with self.assertRaises(ValueError):
                retention.plan(root, root)


class FakeAPI:
    def __init__(self):
        self.value = {
            "online": True,
            "instance_id": "old",
            "version": "old",
            "role": "worker",
            "control": {"maintenance": False, "revision": 1},
            "stats": {
                "active_requests": 0,
                "control_revision": 1,
                "database_ok": True,
                "redis_ok": True,
                "lease_ok": True,
            },
        }
        self.changes = []

    def node(self, ident):
        return copy.deepcopy(self.value)

    def control(self, ident, maintenance):
        self.changes.append(maintenance)
        self.value["control"]["maintenance"] = maintenance


class UpgradeTests(unittest.TestCase):
    def test_success_and_failed_boot_rollback(self):
        for fail in (False, True):
            with self.subTest(fail=fail), tempfile.TemporaryDirectory() as directory:
                root = pathlib.Path(directory)
                artifact, installed = root / "new", root / "app"
                artifact.write_bytes(b"new")
                installed.write_bytes(b"old")
                api = FakeAPI()

                def restart(fail=fail, installed=installed, api=api):
                    if fail and installed.read_bytes() == b"new":
                        raise RuntimeError("failed to boot")
                    api.value.update(instance_id="new-instance", version="new")

                if fail:
                    with self.assertRaises(RuntimeError):
                        operations.upgrade(
                            api,
                            "us",
                            artifact,
                            installed,
                            operations.digest(artifact),
                            "new",
                            restart,
                        )
                    self.assertEqual(installed.read_bytes(), b"old")
                    self.assertTrue(api.value["control"]["maintenance"])
                else:
                    operations.upgrade(
                        api,
                        "us",
                        artifact,
                        installed,
                        operations.digest(artifact),
                        "new",
                        restart,
                    )
                    self.assertEqual(installed.read_bytes(), b"new")
                    self.assertEqual(api.changes, [True, False])

    def test_digest_failure_does_not_drain(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            new, old = root / "new", root / "old"
            new.write_text("new")
            old.write_text("old")
            api = FakeAPI()
            with self.assertRaises(ValueError):
                operations.upgrade(api, "us", new, old, "0" * 64, "new", lambda: None)
            self.assertFalse(api.changes)


if __name__ == "__main__":
    unittest.main()
