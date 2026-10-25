#!/usr/bin/env python3
"""Physical backup, immutable WAL archive and isolated PITR preparation.

Connection credentials come only from PG* environment variables / .pgpass.
No action replaces an existing database directory or deletes live WAL.
"""

import argparse
import datetime as dt
import hashlib
import json
import os
import pathlib
import re
import shlex
import shutil
import subprocess
import sys
import tempfile

WAL_NAME = re.compile(
    r"(?:[0-9A-F]{24}|[0-9A-F]{8}\.history|[0-9A-F]{24}\.[0-9A-F]{8}\.backup)\Z"
)


def sha(path):
    checksum = hashlib.sha256()
    with path.open("rb") as f:
        for block in iter(lambda: f.read(1024 * 1024), b""):
            checksum.update(block)
    return checksum.hexdigest()


def archive(source, root, name):
    if (
        not WAL_NAME.fullmatch(name)
        or not root.is_dir()
        or root.is_symlink()
        or source.is_symlink()
        or not source.is_file()
    ):
        raise ValueError("invalid WAL archive input")
    target = root / name
    if target.exists():
        if target.is_symlink() or sha(target) != sha(source):
            raise ValueError("conflicting WAL archive; refusing overwrite")
        return
    fd, path = tempfile.mkstemp(prefix=".wal-", dir=root)
    temporary = pathlib.Path(path)
    try:
        with os.fdopen(fd, "wb") as dst, source.open("rb") as src:
            shutil.copyfileobj(src, dst)
            dst.flush()
            os.fsync(dst.fileno())
        try:
            os.link(temporary, target)
        except FileExistsError:
            if target.is_symlink() or sha(target) != sha(source):
                raise ValueError("conflicting WAL archive") from None
        fd = os.open(root, os.O_RDONLY)
        try:
            os.fsync(fd)
        finally:
            os.close(fd)
    finally:
        temporary.unlink(missing_ok=True)


def fetch(root, name, destination):
    if not WAL_NAME.fullmatch(name):
        raise ValueError("invalid WAL name")
    source = root / name
    if source.is_symlink() or not source.is_file():
        raise FileNotFoundError(name)
    shutil.copyfile(source, destination)


def verify(pg_bin, base):
    subprocess.run(
        [str(pg_bin / "pg_verifybackup"), str(base)],
        check=True,
        timeout=3600,
        capture_output=True,
    )


def basebackup(pg_bin, destination):
    if destination.exists() or destination.is_symlink():
        raise ValueError("backup destination must be new")
    if not os.environ.get("PGHOST") or not os.environ.get("PGUSER"):
        raise ValueError("explicit PGHOST and PGUSER are required")
    subprocess.run(
        [
            str(pg_bin / "pg_basebackup"),
            "-D",
            str(destination),
            "-Fp",
            "-X",
            "stream",
            "-c",
            "fast",
            "--no-password",
            "--manifest-checksums=SHA256",
        ],
        check=True,
        timeout=7200,
        capture_output=True,
    )
    verify(pg_bin, destination)
    marker = destination.with_name(destination.name + ".verified.json")
    marker.write_text(
        json.dumps(
            {
                "manifest_sha256": sha(destination / "backup_manifest"),
                "verified_at": dt.datetime.now(dt.timezone.utc).isoformat(),
            }
        )
    )
    marker.chmod(0o600)
    return {
        "backup": str(destination),
        "verified": True,
        "verified_at": dt.datetime.now(dt.timezone.utc).isoformat(),
    }


def prepare_restore(pg_bin, base, destination, wal_root, target_time, port):
    if (
        destination.exists()
        or destination.is_symlink()
        or destination.resolve().is_relative_to(base.resolve())
    ):
        raise ValueError("restore destination must be new and outside the backup")
    if not 1024 <= port <= 65535 or port in (5432, 15431, 15432, 25431, 25432):
        raise ValueError("use a dedicated unprivileged restore port")
    if base.is_symlink() or not (base / "backup_manifest").is_file():
        raise ValueError("verified base backup is required")
    if any(p.is_symlink() for p in base.rglob("*")):
        raise ValueError("tablespace/symlink backups need explicit isolated mappings")
    if target_time:
        target = dt.datetime.fromisoformat(target_time.replace("Z", "+00:00"))
        if target.tzinfo is None:
            raise ValueError("PITR target must include timezone")
        if wal_root is None or not wal_root.is_dir() or wal_root.is_symlink():
            raise ValueError("PITR requires a readable WAL archive")
    verify(pg_bin, base)
    shutil.copytree(base, destination)
    destination.chmod(0o700)
    for name in (
        "standby.signal",
        "recovery.signal",
        "postmaster.pid",
        "postmaster.opts",
    ):
        (destination / name).unlink(missing_ok=True)
    (destination / "postgresql.auto.conf").write_text("")

    # Override all inherited production paths, replication and archiving.
    def literal(value):
        return "'" + str(value).replace("'", "''") + "'"

    config = f"listen_addresses='127.0.0.1'\nport={port}\nunix_socket_directories={literal(destination)}\nhba_file={literal(destination / 'pg_hba.conf')}\nident_file={literal(destination / 'pg_ident.conf')}\narchive_mode=off\nlogging_collector=off\nshared_preload_libraries=''\n"
    if target_time:
        command = shlex.join(
            [
                sys.executable,
                str(pathlib.Path(__file__).resolve()),
                "fetch",
                "--wal-dir",
                str(wal_root.resolve()),
                "--name",
                "%f",
                "--destination",
                "%p",
            ]
        )
        config += f"restore_command={literal(command)}\nrecovery_target_time={literal(target_time)}\nrecovery_target_action='pause'\nrecovery_target_timeline='latest'\n"
        (destination / "recovery.signal").touch(mode=0o600)
    (destination / "postgresql.conf").write_text(config)
    # Inherit authentication rather than opening the restored database to local clients.
    (destination / "pg_hba.conf").write_text(
        "local all all peer\nhost all all 127.0.0.1/32 scram-sha-256\n"
    )
    for name in ("postgresql.conf", "postgresql.auto.conf", "pg_hba.conf"):
        (destination / name).chmod(0o600)
    return {
        "restore_directory": str(destination),
        "port": port,
        "started": False,
        "target_time": target_time,
        "production_data_modified": False,
    }


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument(
        "action",
        choices=("basebackup", "verify", "prepare-restore", "archive", "fetch", "disk"),
    )
    p.add_argument("--pg-bin", type=pathlib.Path)
    p.add_argument("--base", type=pathlib.Path)
    p.add_argument("--source", type=pathlib.Path)
    p.add_argument("--destination", type=pathlib.Path)
    p.add_argument("--wal-dir", type=pathlib.Path)
    p.add_argument("--name")
    p.add_argument("--target-time")
    p.add_argument("--port", type=int, default=29432)
    a = p.parse_args()
    if a.action == "archive":
        archive(a.source, a.wal_dir, a.name)
    elif a.action == "fetch":
        fetch(a.wal_dir, a.name, a.destination)
    elif a.action == "disk":
        usage = shutil.disk_usage(a.destination)
        print(
            json.dumps(
                {
                    "path": str(a.destination),
                    "free_bytes": usage.free,
                    "used_percent": round(100 * usage.used / usage.total, 2),
                    "warning": usage.free < 10 * 1024**3
                    or usage.used / usage.total > 0.85,
                }
            )
        )
    elif a.action == "basebackup":
        print(json.dumps(basebackup(a.pg_bin, a.destination)))
    elif a.action == "verify":
        verify(a.pg_bin, a.base)
        print('{"verified":true}')
    else:
        print(
            json.dumps(
                prepare_restore(
                    a.pg_bin, a.base, a.destination, a.wal_dir, a.target_time, a.port
                )
            )
        )


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, subprocess.SubprocessError) as exc:
        # Never echo connection parameters, postgres config or command stderr.
        print("Backup operation failed: " + type(exc).__name__, file=sys.stderr)
        sys.exit(1)
