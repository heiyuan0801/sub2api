#!/usr/bin/env python3
"""Plan retention of verified physical backups and their WAL archive.

Only operates on tool-created base-* directories with matching manifest hashes.
Keeps at least two verified bases. A partial/unmarked backup blocks cleanup.
The WAL archive must be ROOT/wal; live PostgreSQL data is never accepted.
"""

import argparse
import datetime as dt
import json
import pathlib
import re
import shutil
import subprocess

import backup


def plan(root, pg_bin, days=14, keep=2, now=None):
    if days < 1 or keep < 2 or root.is_symlink() or (root / "PG_VERSION").exists():
        raise ValueError("invalid retention policy or backup root")
    now = now or dt.datetime.now(dt.timezone.utc)
    bases = []
    for directory in root.glob("base-*"):
        if directory.name.endswith(".verified.json"):
            continue
        if directory.is_symlink() or not directory.is_dir():
            raise ValueError("unexpected backup path")
        marker = directory.with_name(directory.name + ".verified.json")
        if not marker.is_file() or marker.is_symlink():
            raise ValueError("unverified or in-progress backup blocks retention")
        meta = json.loads(marker.read_text())
        if meta["manifest_sha256"] != backup.sha(directory / "backup_manifest"):
            raise ValueError("backup manifest changed")
        if any(p.is_symlink() for p in directory.rglob("*")):
            raise ValueError("symlink backup requires explicit tablespace handling")
        timestamp = dt.datetime.fromisoformat(meta["verified_at"])
        bases.append((timestamp, directory))
    bases.sort(reverse=True)
    cutoff = now - dt.timedelta(days=days)
    remove = [p for i, (stamp, p) in enumerate(bases) if i >= keep and stamp < cutoff]
    retained = [p for _, p in bases if p not in remove]
    archive = root / "wal"
    wal_remove = []
    if len(retained) >= 2 and archive.is_dir() and not archive.is_symlink():
        manifests = [json.loads((p / "backup_manifest").read_text()) for p in retained]
        if len({m["System-Identifier"] for m in manifests}) != 1:
            raise ValueError("mixed database clusters in backup root")
        ranges = [r for m in manifests for r in m["WAL-Ranges"]]
        if not ranges:
            raise ValueError("backup WAL boundary unavailable")

        def lsn(text):
            return (int(text.split("/")[0], 16) << 32) + int(text.split("/")[1], 16)

        oldest = min(lsn(r["Start-LSN"]) for r in ranges)
        info = subprocess.run(
            [str(pg_bin / "pg_controldata"), str(retained[0])],
            env={"LC_ALL": "C"},
            capture_output=True,
            text=True,
            check=True,
        ).stdout
        size = int(
            next(
                line.split(":", 1)[1].strip()
                for line in info.splitlines()
                if line.startswith("Bytes per WAL segment:")
            )
        )
        if size < 1024 * 1024 or size > 1024**3 or size & (size - 1):
            raise ValueError("unexpected WAL segment size")
        for file in archive.iterdir():
            if file.is_symlink():
                raise ValueError("symlink in WAL archive")
            if file.is_file() and re.fullmatch(r"[0-9A-F]{24}", file.name):
                end = (int(file.name[8:16], 16) << 32) + (
                    int(file.name[16:], 16) + 1
                ) * size
                if end < oldest:
                    wal_remove.append(file)
    for directory in retained:
        backup.verify(pg_bin, directory)
    return {"backups": remove, "wal_segments": wal_remove, "retained_backups": retained}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--root", type=pathlib.Path, required=True)
    p.add_argument("--pg-bin", type=pathlib.Path, required=True)
    p.add_argument("--days", type=int, default=14)
    p.add_argument("--keep", type=int, default=2)
    p.add_argument("--apply", action="store_true")
    a = p.parse_args()
    if a.root.is_symlink():
        p.error("backup root cannot be a symlink")
    result = plan(a.root.resolve(), a.pg_bin, a.days, a.keep)
    print(json.dumps({k: [str(p) for p in v] for k, v in result.items()}, indent=2))
    if a.apply:
        for directory in result["backups"]:
            shutil.rmtree(directory)
            directory.with_name(directory.name + ".verified.json").unlink()
        for file in result["wal_segments"]:
            file.unlink()


if __name__ == "__main__":
    main()
