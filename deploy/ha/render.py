#!/usr/bin/env python3
"""Render a reviewable HA installation bundle; never deploy or start services.

Secrets are read from HA_DB_PASSWORD, HA_REPLICATION_PASSWORD,
HA_REDIS_PASSWORD and HA_SENTINEL_PASSWORD. Output must be outside the repo.
"""

import argparse
import ipaddress
import json
import os
import pathlib
import re
import subprocess


def validate(inv):
    valid = lambda s: (
        isinstance(s, str) and re.fullmatch(r"[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}", s)
    )
    if not valid(inv.get("cluster")):
        raise ValueError("invalid cluster name")
    nodes = inv.get("nodes", [])
    if (
        len(nodes) != 3
        or sum(n.get("kind") == "application" for n in nodes) != 2
        or sum(n.get("kind") == "witness" for n in nodes) != 1
    ):
        raise ValueError(
            "exactly two application hosts and one independent witness are required"
        )
    if len({n["id"] for n in nodes}) != 3 or len({n["private_ip"] for n in nodes}) != 3:
        raise ValueError("duplicate node or IP")
    for n in nodes:
        ip = ipaddress.ip_address(n["private_ip"])
        if not valid(n["id"]) or ip.version != 4 or not ip.is_private or ip.is_loopback:
            raise ValueError("use distinct private IPv4 tunnel addresses")
    if not pathlib.Path(inv.get("postgres_bin_dir", "")).is_absolute():
        raise ValueError("postgres_bin_dir must be absolute")
    return nodes


def render(inv, out, generator, secrets):
    nodes = validate(inv)
    apps = [n for n in nodes if n["kind"] == "application"]
    cluster = inv["cluster"]
    for v in secrets.values():
        if not re.fullmatch(r"[A-Za-z0-9_-]{32,}", v):
            raise ValueError(
                "use 32+ character generated alphanumeric secrets (underscore/hyphen allowed)"
            )
    repo = pathlib.Path(__file__).resolve().parents[2]
    out = out.resolve()
    if out.is_relative_to(repo) or out.exists():
        raise ValueError("output must be a new directory outside the repository")
    out.mkdir(parents=True, mode=0o700)

    def write(node, name, value):
        p = out / node / name
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(
            json.dumps(value, indent=2) if not isinstance(value, str) else value
        )
        p.chmod(0o600)

    endpoints = ["https://" + n["private_ip"] + ":2379" for n in nodes]
    peers = ",".join(n["id"] + "=https://" + n["private_ip"] + ":2380" for n in nodes)
    for n in nodes:
        ident = n["id"]
        ip = n["private_ip"]
        tls = "/etc/sub2api-ha/tls"
        security = {
            "cert-file": tls + "/node.pem",
            "key-file": tls + "/node-key.pem",
            "trusted-ca-file": tls + "/ca.pem",
            "client-cert-auth": True,
        }
        etcd = {
            "name": ident,
            "data-dir": "/var/lib/sub2api-ha/etcd",
            "listen-client-urls": "https://" + ip + ":2379",
            "advertise-client-urls": "https://" + ip + ":2379",
            "listen-peer-urls": "https://" + ip + ":2380",
            "initial-advertise-peer-urls": "https://" + ip + ":2380",
            "initial-cluster": peers,
            "initial-cluster-state": "new",
            "initial-cluster-token": cluster,
            "client-transport-security": security,
            "peer-transport-security": security,
            "heartbeat-interval": 500,
            "election-timeout": 5000,
            "log-level": "warn",
        }
        write(ident, "etcd.yaml", etcd)
        sentinel = f"bind {ip}\nport 26379\nprotected-mode yes\ndir /var/lib/sub2api-ha/sentinel\nrequirepass {secrets['sentinel']}\nsentinel sentinel-pass {secrets['sentinel']}\nsentinel monitor {cluster} {apps[0]['private_ip']} 6379 2\nsentinel auth-pass {cluster} {secrets['redis']}\nsentinel down-after-milliseconds {cluster} 10000\nsentinel failover-timeout {cluster} 60000\nsentinel parallel-syncs {cluster} 1\n"
        write(ident, "sentinel.conf", sentinel)
        if n["kind"] == "witness":
            continue
        pg = {
            "scope": cluster,
            "name": ident,
            "namespace": "/service/",
            "restapi": {"listen": ip + ":8008", "connect_address": ip + ":8008"},
            "etcd3": {
                "hosts": ",".join(x["private_ip"] + ":2379" for x in nodes),
                "protocol": "https",
                "cacert": tls + "/ca.pem",
                "cert": tls + "/client.pem",
                "key": tls + "/client-key.pem",
            },
            "watchdog": {
                "mode": "required",
                "device": "/dev/watchdog",
                "safety_margin": -1,
            },
            "bootstrap": {
                "dcs": {
                    "ttl": 40,
                    "loop_wait": 5,
                    "retry_timeout": 5,
                    "synchronous_mode": True,
                    "synchronous_mode_strict": False,
                    "synchronous_node_count": 1,
                    "check_timeline": True,
                    "postgresql": {
                        "use_pg_rewind": True,
                        "parameters": {
                            "wal_log_hints": "on",
                            "synchronous_commit": "on",
                            "max_connections": 300,
                            "max_wal_senders": 10,
                            "max_replication_slots": 10,
                        },
                    },
                },
                "initdb": [{"encoding": "UTF8"}, "data-checksums"],
                "pg_hba": [
                    f"host replication replicator {a['private_ip']}/32 scram-sha-256"
                    for a in apps
                ]
                + [f"host all all {a['private_ip']}/32 scram-sha-256" for a in apps]
                + ["host all all 127.0.0.1/32 scram-sha-256"],
            },
            "postgresql": {
                "listen": ip + ":5432",
                "connect_address": ip + ":5432",
                "data_dir": "/var/lib/sub2api-ha/postgres",
                "bin_dir": inv["postgres_bin_dir"],
                "pgpass": "/var/lib/sub2api-ha/pgpass",
                "authentication": {
                    "superuser": {"username": "postgres", "password": secrets["db"]},
                    "replication": {
                        "username": "replicator",
                        "password": secrets["replication"],
                    },
                },
            },
        }
        write(ident, "patroni.yaml", pg)
        follower = "" if n == apps[0] else f"replicaof {apps[0]['private_ip']} 6379\n"
        write(
            ident,
            "redis.conf",
            f"bind {ip}\nport 6379\nprotected-mode yes\nrequirepass {secrets['redis']}\nmasterauth {secrets['redis']}\ndir /var/lib/sub2api-ha/redis\nappendonly yes\nappendfsync everysec\n{follower}",
        )
        inventory = {
            "listen": "127.0.0.1:8088",
            "ha": True,
            "nodes": [
                {
                    "id": a["id"],
                    "address": a["private_ip"] + ":8080",
                    "role": "candidate",
                    "weight": 100,
                    "max_connections": 1000,
                }
                for a in apps
            ],
        }
        write(ident, "nodes.json", inventory)
        hap = subprocess.check_output(
            [str(generator), "-config", str(out / ident / "nodes.json")], text=True
        )
        hap = hap.replace("global\n", "global\n    stats socket /run/sub2api-proxy/admin.sock mode 660 level admin\n", 1)
        hap += "\nlisten postgres_writer\n    bind 127.0.0.1:15432\n    mode tcp\n    option httpchk GET /primary\n    http-check expect status 200\n    timeout client 60s\n    timeout server 60s\n    default-server inter 2s fall 2 rise 1 on-marked-down shutdown-sessions\n"
        for a in apps:
            hap += f"    server {a['id']} {a['private_ip']}:5432 check port 8008\n"
        write(ident, "haproxy.cfg", hap)
        write(
            ident,
            "runner.json",
            {
                "node_id": ident,
                "cluster": cluster,
                "etcd_endpoints": endpoints,
                "ca_file": tls + "/ca.pem",
                "cert_file": tls + "/client.pem",
                "key_file": tls + "/client-key.pem",
                "binary": "/usr/local/bin/sub2api",
                "working_directory": "/var/lib/sub2api",
                "proof_file": "/run/sub2api-ha/proof.json",
            },
        )
        # This is an overlay for the existing installation, not a replacement.
        write(
            ident,
            "app-overlay.json",
            {
                "server": {
                    "host": ip,
                    "port": 8080,
                    "trusted_proxies": [a["private_ip"] + "/32" for a in apps]
                    + ["127.0.0.1/32"],
                },
                "database": {
                    "host": "127.0.0.1",
                    "port": 15432,
                    "max_open_conns": 80,
                    "max_idle_conns": 20,
                },
                "redis": {
                    "sentinel_master_name": cluster,
                    "sentinel_addrs": [x["private_ip"] + ":26379" for x in nodes],
                    "sentinel_password": secrets["sentinel"],
                    "password": secrets["redis"],
                    "pool_size": 256,
                    "min_idle_conns": 16,
                },
                "security": {"trust_forwarded_ip_for_api_key_acl": False},
                "cluster": {"enabled": True, "billing_failure_mode": "closed"},
            },
        )
    print("Rendered private review bundle:", out)


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--inventory", required=True, type=pathlib.Path)
    ap.add_argument("--output", required=True, type=pathlib.Path)
    ap.add_argument("--generator", required=True, type=pathlib.Path)
    a = ap.parse_args()
    sec = {
        key: os.environ.get("HA_" + env + "_PASSWORD", "")
        for key, env in [
            ("db", "DB"),
            ("replication", "REPLICATION"),
            ("redis", "REDIS"),
            ("sentinel", "SENTINEL"),
        ]
    }
    render(json.loads(a.inventory.read_text()), a.output, a.generator, sec)
