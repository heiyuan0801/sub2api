#!/usr/bin/env python3
"""Local-host rolling replacement and read-only deployment checks.

An administrator API token is read from SUB2API_ADMIN_TOKEN, never argv or logs.
`upgrade` is a plan unless --apply and --schema-compatible are both supplied.
Run on the node being replaced; the API URL must reach the surviving primary.
"""

import argparse
import datetime as dt
import fcntl
import hashlib
import ipaddress
import json
import os
import pathlib
import re
import shutil
import subprocess
import time
import urllib.parse
import urllib.request


class API:
    def __init__(self, base):
        parsed = urllib.parse.urlsplit(base)
        local = parsed.hostname in ("localhost", "127.0.0.1", "::1")
        if (
            parsed.username
            or parsed.password
            or parsed.query
            or parsed.fragment
            or (parsed.scheme != "https" and not (local and parsed.scheme == "http"))
        ):
            raise ValueError("API URL requires HTTPS, or loopback HTTP")
        self.base = base.rstrip("/") + "/api/v1/admin/cluster"
        self.token = os.environ.get("SUB2API_ADMIN_TOKEN", "")
        if not self.token:
            raise ValueError("SUB2API_ADMIN_TOKEN is required")

        class NoRedirect(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, *args):
                return None

        self.opener = urllib.request.build_opener(
            urllib.request.ProxyHandler({}), NoRedirect()
        )

    def call(self, path="", data=None):
        headers = {
            "Authorization": "Bearer " + self.token,
            "Content-Type": "application/json",
        }
        body = None if data is None else json.dumps(data).encode()
        req = urllib.request.Request(
            self.base + path, body, headers, method="GET" if body is None else "PUT"
        )
        with self.opener.open(req, timeout=10) as response:
            value = json.load(response)
        return value.get("data", value)

    def node(self, ident):
        return next(n for n in self.call()["nodes"] if n["id"] == ident)

    def control(self, ident, maintenance):
        n = self.node(ident)
        c = dict(
            n["control"],
            maintenance=maintenance,
            reason="rolling upgrade orchestration",
        )
        self.call("/nodes/" + urllib.parse.quote(ident, safe="") + "/control", c)


def wait_for(fetch, predicate, timeout):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            value = fetch()
            if predicate(value):
                return value
        except (OSError, ValueError, StopIteration):
            pass
        time.sleep(2)
    raise TimeoutError(
        "node did not reach the required state; maintenance remains enabled"
    )


def digest(path):
    checksum = hashlib.sha256()
    with path.open("rb") as f:
        for block in iter(lambda: f.read(1024 * 1024), b""):
            checksum.update(block)
    return checksum.hexdigest()


def atomic_install(source, target):
    temporary = target.with_name(target.name + ".installing")
    if temporary.exists() or temporary.is_symlink():
        raise ValueError("stale install staging path; inspect before retrying")
    with source.open("rb") as src, temporary.open("xb") as dst:
        shutil.copyfileobj(src, dst)
        dst.flush()
        os.fsync(dst.fileno())
    temporary.chmod(0o755)
    temporary.replace(target)
    fd = os.open(target.parent, os.O_RDONLY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def upgrade(
    api,
    node_id,
    artifact,
    installed,
    expected_sha,
    expected_version,
    restart,
    timeout=180,
):
    """One node only. A failed replacement restores the previous binary and
    leaves the node drained for inspection instead of rejoining silently."""
    if (
        artifact.is_symlink()
        or installed.is_symlink()
        or not artifact.is_file()
        or not installed.is_file()
    ):
        raise ValueError("artifact and installed binary must be regular files")
    if digest(artifact) != expected_sha:
        raise ValueError("artifact SHA-256 mismatch")
    node = api.node(node_id)
    if not node["online"] or node["control"]["maintenance"]:
        raise ValueError("node must be online and serving before an upgrade")
    backup = installed.with_name(
        installed.name
        + ".rollback-"
        + dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%S")
    )
    with installed.open("rb") as src, backup.open("xb") as dst:
        shutil.copyfileobj(src, dst)
        dst.flush()
        os.fsync(dst.fileno())
    backup.chmod(0o755)
    api.control(node_id, True)
    wait_for(
        lambda: api.node(node_id),
        lambda n: (
            n["online"]
            and n["control"]["maintenance"]
            and n["stats"]["control_revision"] >= n["control"]["revision"]
            and n["stats"]["active_requests"] == 0
            and n["role"] == "worker"
        ),
        timeout,
    )
    old_instance = api.node(node_id)["instance_id"]
    try:
        # Verify again after draining; reject an artifact changed mid-operation.
        if digest(artifact) != expected_sha:
            raise ValueError("artifact changed during draining")
        atomic_install(artifact, installed)
        restart()
        wait_for(
            lambda: api.node(node_id),
            lambda n: (
                n["online"]
                and n["instance_id"] != old_instance
                and n["version"] == expected_version
                and all(
                    n["stats"].get(k) for k in ("database_ok", "redis_ok", "lease_ok")
                )
            ),
            timeout,
        )
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError):
        atomic_install(backup, installed)
        restart()
        raise RuntimeError(
            "upgrade failed; old binary restored; node kept in maintenance"
        ) from None
    api.control(node_id, False)
    wait_for(
        lambda: api.node(node_id),
        lambda n: (
            n["online"]
            and not n["control"]["maintenance"]
            and n["stats"]["control_revision"] >= n["control"]["revision"]
        ),
        timeout,
    )
    return {
        "node": node_id,
        "version": expected_version,
        "rollback_binary": str(backup),
        "database_changed_by_tool": False,
    }


def doctor(api, inventory=None, tls=None):
    result = api.call("/preflight")
    if inventory:
        from render import validate

        validate(json.loads(inventory.read_text()))
        result["checks"].append(
            {
                "code": "inventory",
                "status": "pass",
                "detail": "distinct private addresses; physical independence still requires verification",
            }
        )
    if tls:
        for name in ("ca.pem", "node.pem", "client.pem"):
            p = tls / name
            proc = subprocess.run(
                ["openssl", "x509", "-in", str(p), "-noout", "-checkend", "604800"],
                capture_output=True,
                check=False,
            )
            result["checks"].append(
                {
                    "code": "certificate_" + name,
                    "status": "pass" if proc.returncode == 0 else "fail",
                    "detail": "certificate validity for the next seven days",
                }
            )
        for name in ("node.pem", "client.pem"):
            proc = subprocess.run(
                ["openssl", "verify", "-CAfile", str(tls / "ca.pem"), str(tls / name)],
                capture_output=True,
                check=False,
            )
            result["checks"].append(
                {
                    "code": "trust_" + name,
                    "status": "pass" if proc.returncode == 0 else "fail",
                    "detail": "certificate chain verification",
                }
            )
        if inventory:
            for n in json.loads(inventory.read_text())["nodes"]:
                ipaddress.ip_address(n["private_ip"])
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("doctor", "upgrade"))
    parser.add_argument("--api", required=True)
    parser.add_argument("--inventory", type=pathlib.Path)
    parser.add_argument("--tls", type=pathlib.Path)
    parser.add_argument("--node")
    parser.add_argument("--artifact", type=pathlib.Path)
    parser.add_argument("--installed", type=pathlib.Path)
    parser.add_argument("--sha256")
    parser.add_argument("--version")
    parser.add_argument("--service", default="sub2api-ha.service")
    parser.add_argument(
        "--runner-config",
        type=pathlib.Path,
        default=pathlib.Path("/etc/sub2api-ha/runner.json"),
    )
    parser.add_argument("--schema-compatible", action="store_true")
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    api = API(args.api)
    if args.action == "doctor":
        result = doctor(api, args.inventory, args.tls)
        print(json.dumps(result, indent=2))
        if any(check["status"] == "fail" for check in result["checks"]):
            parser.exit(1)
        return
    if not all((args.node, args.artifact, args.installed, args.sha256, args.version)):
        parser.error("upgrade requires node, artifact, installed, sha256 and version")
    if (
        not re.fullmatch(r"[a-fA-F0-9]{64}", args.sha256)
        or args.service != "sub2api-ha.service"
    ):
        parser.error("invalid digest or service")
    if not args.installed.is_absolute() or not args.artifact.is_absolute():
        parser.error("binary paths must be absolute")
    if not args.apply:
        print(
            json.dumps(
                {
                    "plan": [
                        "verify digest",
                        "preserve rollback binary",
                        "drain node and transfer primary",
                        "replace and restart",
                        "verify heartbeat/version",
                        "resume traffic",
                    ],
                    "node": args.node,
                    "sha256_matches": digest(args.artifact) == args.sha256.lower(),
                    "schema_compatibility_required": True,
                },
                indent=2,
            )
        )
        return
    if not args.schema_compatible:
        parser.error(
            "verify migration backward compatibility before passing --schema-compatible"
        )
    runner = json.loads(args.runner_config.read_text())
    if (
        runner.get("node_id") != args.node
        or pathlib.Path(runner.get("binary", "")).resolve() != args.installed.resolve()
    ):
        parser.error(
            "local runner node/binary does not match the requested replacement"
        )
    lock = args.installed.with_name(args.installed.name + ".upgrade.lock")
    with lock.open("a") as f:
        fcntl.flock(f, fcntl.LOCK_EX | fcntl.LOCK_NB)

        def restart():
            subprocess.run(
                ["systemctl", "restart", args.service],
                check=True,
                timeout=90,
                capture_output=True,
            )

        result = upgrade(
            api,
            args.node,
            args.artifact,
            args.installed,
            args.sha256.lower(),
            args.version,
            restart,
        )
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
