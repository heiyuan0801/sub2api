#!/usr/bin/env python3
"""Apply trusted local cluster controls to a private HAProxy admin socket.

Never changes health/maintenance flags: application readiness owns those.
No network listener, shell command or remote management API is exposed.
"""

import argparse
import csv
import io
import datetime as dt
import json
import os
import pathlib
import re
import socket
import time


def commands(snapshot, allowed, now=None):
    now = now or dt.datetime.now(dt.timezone.utc)
    # Go emits RFC3339Nano; Python 3.10 accepts only microsecond precision.
    timestamp = re.sub(
        r"\.(\d+)",
        lambda match: "." + match.group(1)[:6].ljust(6, "0"),
        snapshot["updated_at"].replace("Z", "+00:00"),
    )
    age = (now - dt.datetime.fromisoformat(timestamp)).total_seconds()
    if not -5 <= age <= 15:
        raise ValueError("control snapshot is stale")
    result, revisions, seen = [], {}, set()
    for node in snapshot["nodes"]:
        ident = node["id"]
        if ident not in allowed:
            continue
        if ident in seen or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_-]{0,63}", ident):
            raise ValueError("invalid node identity")
        seen.add(ident)
        control = node["control"]
        weight, capacity, revision = (
            control[k] for k in ("weight", "max_concurrency", "revision")
        )
        if any(type(v) is not int for v in (weight, capacity, revision)) or not (
            1 <= weight <= 256 and 1 <= capacity <= 20000 and revision >= 1
        ):
            raise ValueError("invalid control bounds")
        for backend in ("primary", "gateway", "responses"):
            result.extend(
                [
                    f"set server {backend}/{ident} weight {weight}",
                    f"set maxconn server {backend}/{ident} {capacity}",
                ]
            )
        revisions[ident] = revision
    if not seen:
        raise ValueError("no configured node controls")
    return result, revisions


def apply(socket_path, lines):
    for line in lines:
        with socket.socket(socket.AF_UNIX) as sock:
            sock.settimeout(2)
            sock.connect(str(socket_path))
            sock.sendall((line + "\n").encode())
            sock.shutdown(socket.SHUT_WR)
            data = b""
            while block := sock.recv(4096):
                data += block
                if len(data) > 65536:
                    raise RuntimeError("unexpected runtime response")
            if data.strip():
                raise RuntimeError("HAProxy rejected a runtime command")


def traffic(socket_path, local_node=None):
    with socket.socket(socket.AF_UNIX) as sock:
        sock.settimeout(2)
        sock.connect(str(socket_path))
        sock.sendall(b"show stat\n")
        sock.shutdown(socket.SHUT_WR)
        data = b""
        while block := sock.recv(8192):
            data += block
            if len(data) > 2*1024*1024:
                raise RuntimeError("runtime statistics exceed limit")
    rows = csv.DictReader(io.StringIO(data.decode().lstrip("# ")))
    result = dict(sampled_at=dt.datetime.now(dt.timezone.utc).isoformat(), queued=0, active=0, bytes_in=0, bytes_out=0, cross_node_bytes=0 if local_node else None)
    seen = 0
    for row in rows:
        if row.get('pxname') not in ('primary', 'gateway', 'responses') or row.get('type') != '2':
            continue
        seen += 1
        for source, target in [('qcur', 'queued'), ('scur', 'active'), ('bin', 'bytes_in'), ('bout', 'bytes_out')]:
            result[target] += max(0, int(row.get(source) or 0))
        if local_node and row['svname'] != local_node:
            result['cross_node_bytes'] += max(0, int(row.get('bin') or 0)) + max(0, int(row.get('bout') or 0))
    if seen == 0:
        raise RuntimeError('no application backend statistics')
    return result


def tick(proof, socket_path, allowed, local_node=None):
    snapshot = json.loads(pathlib.Path(str(proof) + ".proxy").read_text())
    lines, revisions = commands(snapshot, allowed)
    apply(socket_path, lines)
    metrics = traffic(socket_path, local_node)
    target = pathlib.Path(str(proof) + ".proxy-ack")
    temp = target.with_suffix(target.suffix + ".tmp")
    fd = os.open(temp, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, "w") as f:
        json.dump(
            {
                "updated_at": dt.datetime.now(dt.timezone.utc).isoformat(),
                "revisions": revisions,
                "traffic": metrics,
            },
            f,
        )
    temp.replace(target)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--proof", type=pathlib.Path, required=True)
    parser.add_argument("--socket", type=pathlib.Path, required=True)
    parser.add_argument("--inventory", type=pathlib.Path, required=True)
    parser.add_argument("--once", action="store_true")
    args = parser.parse_args()
    inventory = json.loads(args.inventory.read_text())
    allowed = {n["id"] for n in inventory["nodes"]}
    local_node = inventory.get("local_node_id")
    if local_node and local_node not in allowed:
        parser.error("local node must belong to inventory")
    warned = False
    while True:
        try:
            tick(args.proof, args.socket, allowed, local_node)
            warned = False
        except (OSError, ValueError, KeyError, TypeError, RuntimeError):
            if args.once:
                raise
            if not warned:
                print(
                    "Control snapshot or HAProxy unavailable; acknowledgement withheld",
                    flush=True,
                )
            warned = True
        if args.once:
            return
        time.sleep(3)


if __name__ == "__main__":
    main()
