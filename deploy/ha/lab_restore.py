#!/usr/bin/env python3
"""Real local PostgreSQL backup + time-point restore drill, new directories only."""

import argparse
import json
import os
import pathlib
import shlex
import subprocess
import sys
import time

import backup


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--dir", type=pathlib.Path, required=True)
    p.add_argument("--pg-bin", type=pathlib.Path, required=True)
    a = p.parse_args()
    root = a.dir.resolve()
    root.mkdir(mode=0o700)
    pg = a.pg_bin.resolve()
    source, restored, wal, base = (
        root / n for n in ("source", "restored", "wal", "base")
    )
    wal.mkdir(mode=0o700)
    env = {
        **os.environ,
        "PGHOST": "127.0.0.1",
        "PGPORT": "29431",
        "PGUSER": "restore_test",
        "PGDATABASE": "postgres",
        "PGPASSWORD": "",
    }

    def run(args, **kwargs):
        return subprocess.run(
            list(map(str, args)),
            check=True,
            capture_output=True,
            text=True,
            env=env,
            **kwargs,
        )

    def sql(query, port=29431):
        return run(
            [pg / "psql", "-XqAt", "-v", "ON_ERROR_STOP=1", "-p", port, "-c", query]
        ).stdout.strip()

    def start(path, port):
        # Explicit loopback test identity; no production user/configuration.
        return run(
            [pg / "pg_ctl", "-D", path, "-l", root / f"{port}.log", "-w", "start"]
        )

    run(
        [
            pg / "initdb",
            "-D",
            source,
            "-U",
            "restore_test",
            "-A",
            "trust",
            "--no-locale",
        ]
    )
    archive_cmd = shlex.join(
        [
            sys.executable,
            str(pathlib.Path(backup.__file__).resolve()),
            "archive",
            "--source",
            "%p",
            "--wal-dir",
            str(wal),
            "--name",
            "%f",
        ]
    )
    config = f"listen_addresses='127.0.0.1'\nport=29431\nunix_socket_directories='{source}'\nwal_level=replica\narchive_mode=on\narchive_timeout=60\narchive_command='{archive_cmd.replace(chr(39), chr(39) * 2)}'\n"
    (source / "postgresql.conf").write_text(config)
    started = []
    try:
        start(source, 29431)
        started.append(source)
        sql(
            "CREATE TABLE ha_ledger(id integer primary key,amount numeric(20,10)); INSERT INTO ha_ledger VALUES(1,1.2500000000)"
        )
        old_env = {
            k: os.environ.get(k)
            for k in ("PGHOST", "PGPORT", "PGUSER", "PGDATABASE", "PGPASSWORD")
        }
        os.environ.update({k: env[k] for k in old_env})
        try:
            backup.basebackup(pg, base)
        finally:
            for k, value in old_env.items():
                if value is None:
                    os.environ.pop(k, None)
                else:
                    os.environ[k] = value
        sql("INSERT INTO ha_ledger VALUES(2,2.5000000000)")
        target = sql("SELECT clock_timestamp()")
        time.sleep(0.1)
        sql("INSERT INTO ha_ledger VALUES(3,100.0000000000)")
        closed = sql("SELECT pg_walfile_name(pg_current_wal_lsn())")
        sql("SELECT pg_switch_wal()")
        deadline = time.monotonic() + 60
        while not (wal / closed).exists():
            if time.monotonic() > deadline:
                raise TimeoutError("WAL did not reach archive")
            time.sleep(0.2)
        backup.prepare_restore(pg, base, restored, wal, target, 29432)
        # This is a new synthetic test cluster only; trust permits test inspection.
        (restored / "pg_hba.conf").write_text(
            "host all restore_test 127.0.0.1/32 trust\n"
        )
        start(restored, 29432)
        started.append(restored)
        deadline = time.monotonic() + 30
        while sql("SELECT pg_is_wal_replay_paused()", 29432) != "t":
            if time.monotonic() > deadline:
                raise TimeoutError("PITR target not reached")
            time.sleep(0.2)
        rows = sql(
            "SELECT string_agg(id::text,',' ORDER BY id),sum(amount)::text FROM ha_ledger",
            29432,
        )
        assert rows == "1,2|3.7500000000", rows
        report = {
            "physical_backup_verified": True,
            "wal_archive_delivered": True,
            "pitr_target_reached": True,
            "restored_ids": [1, 2],
            "later_transaction_excluded": True,
            "exact_amount": "3.7500000000",
            "isolated_ports": [29431, 29432],
            "production_data_modified": False,
        }
        (root / "results.json").write_text(json.dumps(report, indent=2))
        print(json.dumps(report, indent=2))
    finally:
        for path in reversed(started):
            subprocess.run(
                [str(pg / "pg_ctl"), "-D", str(path), "-m", "fast", "-w", "stop"],
                env=env,
                capture_output=True,
                check=False,
            )


if __name__ == "__main__":
    main()
