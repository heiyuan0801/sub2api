#!/usr/bin/env python3
"""Real single-host HA fault lab. Only dedicated loopback ports and fresh data.

Run with Patroni's venv Python. Tools are explicit environment paths; no Docker,
sudo, production configuration, real provider key or external model call.
"""

import argparse
import collections
import concurrent.futures
import json
import os
import pathlib
import secrets
import signal
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request

import yaml

REPO = pathlib.Path(__file__).resolve().parents[2]
ARGS = argparse.ArgumentParser()
ARGS.add_argument("action", choices=["start", "test", "test-witness", "status", "stop"])
ARGS.add_argument("--dir", required=True, type=pathlib.Path)
A = ARGS.parse_args()
ROOT = A.dir.resolve()
TOOLS = {
    key: os.environ.get("SUB2API_HA_" + key, "")
    for key in [
        "APP",
        "RUNNER",
        "GENERATOR",
        "ETCD",
        "PATRONI",
        "PG_BIN",
        "REDIS",
        "HAPROXY",
    ]
}
ENDPOINTS = [f"http://127.0.0.1:{23791 + i}" for i in range(3)]
OP = urllib.request.build_opener(urllib.request.ProxyHandler({}))
BASE = "http://127.0.0.1:28180"
ENV = {
    k: v
    for k, v in os.environ.items()
    if "proxy" not in k.lower()
    and not k.startswith(("CLUSTER_", "DATABASE_", "REDIS_", "SERVER_", "ADMIN_"))
    and k not in ("CONFIG_FILE", "DATA_DIR", "AUTO_SETUP", "SKIP_SETUP")
}
ENV.update(TZ="Asia/Shanghai", SUB2API_HA_LAB_DIR=str(ROOT))


def save(name, data, private=False):
    p = ROOT / name
    p.write_text(json.dumps(data, indent=2, ensure_ascii=False))
    p.chmod(0o600 if private else 0o644)


def load(name):
    return json.loads((ROOT / name).read_text())


def launch(name, args, cwd=None, env=None):
    pids = load("pids.json") if (ROOT / "pids.json").exists() else {}
    if name in pids and alive(pids[name]):
        raise RuntimeError(name + " already alive")
    with (ROOT / (name + ".log")).open("ab") as log:
        p = subprocess.Popen(
            list(map(str, args)),
            cwd=cwd or ROOT,
            env=env or ENV,
            stdin=subprocess.DEVNULL,
            stdout=log,
            stderr=log,
            start_new_session=True,
        )
    pids[name] = p.pid
    save("pids.json", pids, True)
    print("started", name, p.pid, flush=True)


def alive(pid):
    result = subprocess.run(
        ["ps", "-p", str(pid), "-o", "stat="], capture_output=True, text=True
    ).stdout.strip()
    return bool(result) and not result.startswith("Z")


def stop(name, hard=False):
    pid = load("pids.json").get(name)
    if not pid or not alive(pid):
        return
    # Confirm this process was started from the dedicated lab before signalling.
    command = subprocess.check_output(
        ["ps", "-p", str(pid), "-o", "command="], text=True
    )
    if (
        str(ROOT) not in command
        and not any(x and x in command for x in TOOLS.values())
        and not (
            name == "fixture"
            and str(pathlib.Path(__file__).with_name("lab_fixture.py")) in command
        )
    ):
        if not (name.endswith("-redis") and "127.0.0.1:2638" in command):
            raise RuntimeError("unexpected process: " + name)
    os.killpg(pid, signal.SIGKILL if hard else signal.SIGTERM)


def wait(fn, timeout=90):
    end = time.monotonic() + timeout
    last_error = None
    while time.monotonic() < end:
        try:
            value = fn()
            if value:
                return value
        except (OSError, urllib.error.URLError, ValueError, RuntimeError) as exc:
            last_error = str(exc)[:600]
        time.sleep(0.4)
    raise RuntimeError(
        "timeout waiting for "
        + getattr(fn, "__name__", "condition")
        + ": "
        + str(last_error)
    )


def http(path, data=None, token=None, base=BASE, method=None, headers=None, timeout=4):
    h = {"Content-Type": "application/json", **(headers or {})}
    if token:
        h["Authorization"] = "Bearer " + token
    req = urllib.request.Request(
        base + path,
        data=json.dumps(data).encode() if data is not None else None,
        headers=h,
        method=method,
    )
    try:
        r = OP.open(req, timeout=timeout)
    except urllib.error.HTTPError as e:
        r = e
    with r:
        raw = r.read()
        try:
            b = json.loads(raw)
        except ValueError:
            b = raw.decode(errors="replace")
        return r.status, dict(r.headers), b


def data(path, payload=None, token=None, **kw):
    s, h, b = http(path, payload, token, **kw)
    if s != 200:
        raise RuntimeError(str((path, s, b))[:800])
    return b.get("data", b) if isinstance(b, dict) else b


def sql(q, port=15431):
    c = load("credentials.json")
    p = subprocess.run(
        [
            TOOLS["PG_BIN"] + "/psql",
            "-h",
            "127.0.0.1",
            "-p",
            str(port),
            "-U",
            "cluster_ha",
            "-d",
            "sub2api_ha_lab",
            "-X",
            "-qAt",
            "-v",
            "ON_ERROR_STOP=1",
        ],
        input=q,
        env={**ENV, "PGPASSWORD": c["db"]},
        capture_output=True,
        text=True,
    )
    if p.returncode:
        raise RuntimeError(p.stderr[-500:])
    return p.stdout.strip()


def redis_cli(*args, port=26381):
    binary = str(pathlib.Path(TOOLS["REDIS"]).with_name("redis-cli"))
    r = subprocess.run(
        [binary, "-h", "127.0.0.1", "-p", str(port), *args],
        env={**ENV, "REDISCLI_AUTH": load("credentials.json")["redis"]},
        capture_output=True,
        text=True,
    )
    if r.returncode:
        raise RuntimeError(r.stderr[-500:])
    return r.stdout.strip()


def start_host(host):
    n = {"us": 0, "hk": 1, "witness": 2}[host]
    launch(host + "-etcd", [TOOLS["ETCD"], "--config-file", ROOT / host / "etcd.json"])
    if n < 2:
        launch(host + "-patroni", [TOOLS["PATRONI"], ROOT / host / "patroni.json"])
        launch(host + "-redis", [TOOLS["REDIS"], ROOT / host / "redis.conf"])
        launch(
            host + "-proxy",
            [TOOLS["HAPROXY"], "-db", "-f", ROOT / host / "haproxy.cfg"],
        )
    if n < 2:
        launch(host + "-controls", [sys.executable, REPO / "deploy/ha/proxy_agent.py", "--proof", ROOT / host / "proof.json", "--socket", ROOT / host / "admin.sock", "--inventory", ROOT / host / "nodes.json"])
    launch(
        host + "-sentinel",
        [TOOLS["REDIS"], ROOT / host / "sentinel.conf", "--sentinel"],
    )
    if n < 2 and (ROOT / host / "app/config.yaml").exists():
        launch(
            host + "-runner", [TOOLS["RUNNER"], "-config", ROOT / host / "runner.json"]
        )


def kill_host(host):
    # Include independently grouped runner children and Patroni postmasters.
    pids = load("pids.json")
    candidates = []
    lines = subprocess.check_output(
        ["ps", "-axo", "pid=,ppid=,command="], text=True
    ).splitlines()
    parsed = []
    for line in lines:
        parts = line.strip().split(None, 2)
        if len(parts) == 3:
            parsed.append((int(parts[0]), int(parts[1]), parts[2]))
    for name, pid in pids.items():
        if name.startswith(host + "-") and alive(pid):
            candidates.append(pid)
    # Freeze parents before traversing their children to prevent restarts.
    for pid in candidates:
        os.kill(pid, signal.SIGSTOP)
    tree = set(candidates)
    for _ in range(8):
        for pid, ppid, cmd in parsed:
            if ppid in tree:
                tree.add(pid)
    # PostgreSQL can detach from Patroni; data-directory ownership identifies it.
    for pid, ppid, cmd in parsed:
        if str(ROOT / host / "pgdata") in cmd:
            tree.add(pid)
    for pid in sorted(tree, reverse=True):
        try:
            os.kill(pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
    print("power-cut domain", host, "processes", len(tree), flush=True)


def start():
    if ROOT.exists() and any(ROOT.iterdir()):
        raise RuntimeError("lab requires a fresh directory")
    ROOT.mkdir(parents=True, exist_ok=True)
    ROOT.chmod(0o700)
    for key, p in TOOLS.items():
        if not p or not pathlib.Path(p).exists():
            raise RuntimeError("missing tool " + key)
    ports = (
        [28180, 28190, 28199]
        + [23791 + i for i in range(3)]
        + [23801 + i for i in range(3)]
        + [26481 + i for i in range(3)]
    )
    ports += [
        p + i for p in (25431, 28001, 26381, 28181, 15431, 28191) for i in range(2)
    ]
    for port in ports:
        with socket.socket() as s:
            s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            s.bind(("127.0.0.1", port))
    save("tools.json", TOOLS)
    c = {
        "db": secrets.token_hex(24),
        "replication": secrets.token_hex(24),
        "redis": secrets.token_hex(24),
        "jwt": secrets.token_hex(32),
        "totp": secrets.token_hex(32),
        "email": "ha-local@example.test",
        "password": secrets.token_urlsafe(24),
    }
    save("credentials.json", c, True)
    cluster = ",".join(
        f"{h}=http://127.0.0.1:{23801 + i}"
        for i, h in enumerate(("us", "hk", "witness"))
    )
    for i, h in enumerate(("us", "hk", "witness")):
        d = ROOT / h
        d.mkdir()
        (d / "app").mkdir()
        save(
            h + "/etcd.json",
            {
                "name": h,
                "data-dir": str(d / "etcd-data"),
                "listen-client-urls": ENDPOINTS[i],
                "advertise-client-urls": ENDPOINTS[i],
                "listen-peer-urls": f"http://127.0.0.1:{23801 + i}",
                "initial-advertise-peer-urls": f"http://127.0.0.1:{23801 + i}",
                "initial-cluster": cluster,
                "initial-cluster-token": "sub2api-isolated-ha-lab",
                "initial-cluster-state": "new",
                "log-level": "warn",
            },
        )
        (d / "sentinel.conf").write_text(
            f"bind 127.0.0.1\nport {26481 + i}\ndir {d}\nsentinel monitor sub2api-ha 127.0.0.1 26381 2\nsentinel auth-pass sub2api-ha {c['redis']}\nsentinel down-after-milliseconds sub2api-ha 3000\nsentinel failover-timeout sub2api-ha 15000\nsentinel parallel-syncs sub2api-ha 1\n"
        )
        (d / "sentinel.conf").chmod(0o600)
        if i == 2:
            continue
        patroni = {
            "scope": "sub2api-ha-lab",
            "namespace": "/service/",
            "name": h,
            "restapi": {
                "listen": f"127.0.0.1:{28001 + i}",
                "connect_address": f"127.0.0.1:{28001 + i}",
            },
            "etcd3": {
                "hosts": ",".join("127.0.0.1:" + str(23791 + j) for j in range(3))
            },
            "bootstrap": {
                "dcs": {
                    "ttl": 20,
                    "loop_wait": 2,
                    "retry_timeout": 3,
                    "synchronous_mode": True,
                    "synchronous_mode_strict": False,
                    "synchronous_node_count": 1,
                    "check_timeline": True,
                    "postgresql": {
                        "use_pg_rewind": True,
                        "parameters": {
                            "wal_log_hints": "on",
                            "max_connections": 100,
                            "max_wal_senders": 10,
                            "max_replication_slots": 10,
                            "synchronous_commit": "on",
                        },
                    },
                },
                "initdb": [{"encoding": "UTF8"}, "data-checksums"],
                "pg_hba": [
                    "host replication replicator 127.0.0.1/32 scram-sha-256",
                    "host all all 127.0.0.1/32 scram-sha-256",
                ],
            },
            "postgresql": {
                "listen": f"127.0.0.1:{25431 + i}",
                "connect_address": f"127.0.0.1:{25431 + i}",
                "data_dir": str(d / "pgdata"),
                "bin_dir": TOOLS["PG_BIN"],
                "pgpass": str(d / "pgpass"),
                "authentication": {
                    "superuser": {"username": "cluster_ha", "password": c["db"]},
                    "replication": {
                        "username": "replicator",
                        "password": c["replication"],
                    },
                },
                "parameters": {
                    "unix_socket_directories": str(d),
                    "shared_buffers": "64MB",
                },
            },
            "watchdog": {"mode": "off"},
        }
        save(h + "/patroni.json", patroni, True)
        replica = f"replicaof 127.0.0.1 26381\n" if i else ""
        (d / "redis.conf").write_text(
            f'bind 127.0.0.1\nport {26381 + i}\nprotected-mode yes\nrequirepass {c["redis"]}\nmasterauth {c["redis"]}\ndir {d}\nappendonly yes\nappendfsync everysec\nsave ""\n{replica}'
        )
        (d / "redis.conf").chmod(0o600)
        inv = {
            "listen": f"127.0.0.1:{28181 + i}",
            "local_node_id": h,
            "ha": True,
            "nodes": [
                {
                    "id": n,
                    "address": f"127.0.0.1:{28191 + j}",
                    "role": "candidate",
                    "weight": 100,
                }
                for j, n in enumerate(("us", "hk"))
            ],
        }
        save(h + "/nodes.json", inv)
        cfg = subprocess.check_output(
            [TOOLS["GENERATOR"], "-config", str(d / "nodes.json")], text=True
        )
        cfg = cfg.replace(
            "frontend sub2api\n",
            "frontend sub2api\n    http-response set-header X-Local-Test-Node %[srv_name]\n",
        )
        cfg = cfg.replace("global\n", f"global\n    stats socket {d}/admin.sock mode 600 level admin\n", 1)
        cfg += f"\nlisten postgres_writer\n    bind 127.0.0.1:{15431 + i}\n    mode tcp\n    option httpchk GET /primary\n    http-check expect status 200\n    default-server inter 1s fall 2 rise 1 on-marked-down shutdown-sessions\n    server us 127.0.0.1:25431 check port 28001\n    server hk 127.0.0.1:25432 check port 28002\n"
        (d / "haproxy.cfg").write_text(cfg)
        save(
            h + "/runner.json",
            {
                "node_id": h,
                "cluster": "ha-lab",
                "etcd_endpoints": ENDPOINTS,
                "binary": TOOLS["APP"],
                "working_directory": str(d / "app"),
                "proof_file": str(d / "proof.json"),
                "local_lab": True,
            },
        )
    edge = """global
    maxconn 1000
defaults
    mode http
    timeout connect 2s
    timeout client 120s
    timeout server 120s
    retries 0
frontend edge
    bind 127.0.0.1:28180
    default_backend regional_ingress
backend regional_ingress
    option httpchk GET /ingress-ready
    http-check expect status 200
    default-server inter 1s fall 2 rise 1
    server hk 127.0.0.1:28182 check
    server us 127.0.0.1:28181 check
"""
    (ROOT / "edge.cfg").write_text(edge)
    for h in ("us", "hk", "witness"):
        start_host(h)
    wait(
        lambda: any(
            http("/primary", base=f"http://127.0.0.1:{p}")[0] == 200
            for p in (28001, 28002)
        )
    )
    wait(
        lambda: (
            redis_cli("INFO", "replication", port=26382).find("master_link_status:up")
            >= 0
        )
    )
    launch(
        "fixture", [sys.executable, pathlib.Path(__file__).with_name("lab_fixture.py")]
    )
    launch("edge", [TOOLS["HAPROXY"], "-db", "-f", ROOT / "edge.cfg"])
    # Bootstrap once before any HA children exist, then persist complete config.
    e = {
        **ENV,
        "DATA_DIR": str(ROOT / "us/app"),
        "AUTO_SETUP": "true",
        "SERVER_HOST": "127.0.0.1",
        "SERVER_PORT": "28199",
        "DATABASE_HOST": "127.0.0.1",
        "DATABASE_PORT": "15431",
        "DATABASE_USER": "cluster_ha",
        "DATABASE_PASSWORD": c["db"],
        "DATABASE_DBNAME": "sub2api_ha_lab",
        "DATABASE_SSLMODE": "disable",
        "REDIS_HOST": "127.0.0.1",
        "REDIS_PORT": "26381",
        "REDIS_PASSWORD": c["redis"],
        "ADMIN_EMAIL": c["email"],
        "ADMIN_PASSWORD": c["password"],
        "JWT_SECRET": c["jwt"],
        "TOTP_ENCRYPTION_KEY": c["totp"],
        "PRICING_REMOTE_URL": "http://127.0.0.1:28190/pricing.json",
        "PRICING_HASH_URL": "http://127.0.0.1:28190/pricing.sha256",
        "PRICING_FALLBACK_FILE": str(
            REPO
            / "backend/resources/model-pricing/model_prices_and_context_window.json"
        ),
        "LOG_OUTPUT_TO_FILE": "false",
    }
    launch("bootstrap", [TOOLS["APP"]], cwd=ROOT / "us/app", env=e)
    wait(lambda: http("/health", base="http://127.0.0.1:28199")[0] == 200, 120)
    stop("bootstrap")
    wait(lambda: not alive(load("pids.json")["bootstrap"]), 30)
    base = yaml.safe_load((ROOT / "us/app/config.yaml").read_text())
    for i, h in enumerate(("us", "hk")):
        cfg = json.loads(json.dumps(base))
        cfg["server"].update(
            host="127.0.0.1", port=28191 + i, trusted_proxies=["127.0.0.1/32"]
        )
        cfg["database"].update(port=15431 + i, max_open_conns=15, max_idle_conns=3)
        cfg["redis"].update(
            sentinel_master_name="sub2api-ha",
            sentinel_addrs=[f"127.0.0.1:{26481 + j}" for j in range(3)],
            pool_size=16,
            min_idle_conns=2,
        )
        cfg.update(
            totp={"encryption_key": c["totp"]},
            log={"output": {"to_file": False, "to_stdout": True}},
            pricing={
                "remote_url": "http://127.0.0.1:28190/pricing.json",
                "hash_url": "http://127.0.0.1:28190/pricing.sha256",
                "fallback_file": str(
                    REPO
                    / "backend/resources/model-pricing/model_prices_and_context_window.json"
                ),
                "data_dir": str(ROOT / h / "app/pricing"),
            },
            cluster={"drain_seconds": 2, "shutdown_timeout_seconds": 10},
        )
        save(h + "/app/config.yaml", cfg, True)
        launch(h + "-runner", [TOOLS["RUNNER"], "-config", ROOT / h / "runner.json"])
    wait(lambda: http("/ingress-ready")[0] == 200, 120)
    c = load("credentials.json")
    login = data("/api/v1/auth/login", {"email": c["email"], "password": c["password"]})
    uid = login["user"]["id"]
    key = "sk-ha-local-" + secrets.token_hex(24)
    sql(f"""BEGIN;
    UPDATE users SET balance=100,concurrency=20 WHERE id={uid};
    INSERT INTO groups(name,platform,rate_multiplier,subscription_type) VALUES('HA fixture','openai',1,'standard');
    INSERT INTO accounts(name,platform,type,credentials,extra,concurrency,priority) VALUES('LOCAL fixture','openai','apikey','{{"api_key":"local-only","base_url":"http://127.0.0.1:28190","model_mapping":{{"gpt-4.1-mini":"gpt-4.1-mini"}}}}','{{"openai_passthrough":true}}',20,1);
    INSERT INTO account_groups(account_id,group_id) SELECT a.id,g.id FROM accounts a,groups g WHERE a.name='LOCAL fixture' AND g.name='HA fixture';
    INSERT INTO api_keys(user_id,key,name,group_id) SELECT {uid},'{key}','HA test',id FROM groups WHERE name='HA fixture';
    COMMIT;""")
    save(
        "auth.json", {"token": login["access_token"], "key": key, "user_id": uid}, True
    )
    print("HA lab ready at", BASE, flush=True)


def status():
    results = {}
    for i, h in enumerate(("us", "hk")):
        try:
            results[h] = http(
                "/readyz?details=1", base=f"http://127.0.0.1:{28191 + i}"
            )[2]
        except Exception:
            results[h] = {"status": "down"}
    return results


def test_witness():
    auth = load("auth.json")
    wait(lambda: all(v.get("status") == "ready" for v in status().values()))
    kill_host("witness")
    try:
        for _ in range(5):
            nodes = status()
            assert all(v.get("status") == "ready" for v in nodes.values())
            assert sum(v.get("role") == "primary" for v in nodes.values()) == 1
            assert http("/v1/models", token=auth["key"])[0] == 200
            assert http("/api/v1/admin/compliance", token=auth["token"])[0] == 200
            time.sleep(1)
    finally:
        start_host("witness")
    wait(lambda: http("/health", base=ENDPOINTS[2])[2].get("health") == "true")
    report = load("results.json") if (ROOT / "results.json").exists() else {}
    report["witness_whole_host_failure"] = {
        "passed": True,
        "service_remained_available": True,
        "single_application_primary": True,
    }
    save("results.json", report)
    print(
        "witness_whole_host_failure", report["witness_whole_host_failure"], flush=True
    )


def test():
    auth = load("auth.json")
    report = {}

    def record(name, value):
        report[name] = value
        save("results.json", report)
        print(name, value, flush=True)

    def generate(i):
        s, h, b = http(
            "/v1/responses",
            {
                "model": "gpt-4.1-mini",
                "input": "HA local test",
                "max_output_tokens": 16,
            },
            auth["key"],
            headers={"X-Sub2API-Session-ID": f"ha-{i}"},
        )
        if s != 200 or b.get("status") != "completed":
            raise RuntimeError(str((s, b))[:500])
        return next(v for k, v in h.items() if k.lower() == "x-local-test-node")

    def stable_ingress():
        # Allow both layers' rise checks to converge before measured traffic.
        ready_since = None

        def ready():
            nonlocal ready_since
            if http("/ingress-ready")[0] != 200:
                ready_since = None
                return False
            if ready_since is None:
                ready_since = time.monotonic()
            return time.monotonic() - ready_since >= 5

        wait(ready, 60)

    wait(lambda: len([v for v in status().values() if v.get("status") == "ready"]) == 2)
    # Verify synchronous replica has joined before fault tests.
    wait(lambda: "sync" in sql("SELECT sync_state FROM pg_stat_replication"))
    stable_ingress()
    initial_rows = int(sql("SELECT count(*) FROM usage_logs"))
    with concurrent.futures.ThreadPoolExecutor(max_workers=6) as pool:
        counts = collections.Counter(pool.map(generate, range(20)))
    assert len(counts) == 2
    record("normal_generation", dict(counts))
    # Durability check concerns committed rows, separately from in-flight work.
    wait(lambda: int(sql("SELECT count(*) FROM usage_logs")) >= initial_rows + 20, 20)
    before = sql("SELECT count(*) FROM usage_logs")
    record("committed_rows_before_fault", int(before))
    for failed, survivor, dbport in [("us", "hk", 15432), ("hk", "us", 15431)]:
        start_at = time.monotonic()
        kill_host(failed)
        wait(lambda: http("/ingress-ready")[0] == 200, 120)
        wait(
            lambda: any(
                v.get("role") == "primary" and v.get("status") == "ready"
                for v in status().values()
            ),
            120,
        )
        recovered = wait(lambda: generate("recover-" + failed) == survivor, 120)
        elapsed = round(time.monotonic() - start_at, 2)
        assert data("/api/v1/admin/compliance", token=auth["token"])["required"] in (
            True,
            False,
        )
        assert int(sql("SELECT count(*) FROM usage_logs", dbport)) >= int(before)
        record(
            failed + "_whole_host_failure",
            {
                "passed": bool(recovered),
                "recovery_seconds": elapsed,
                "serving_node": survivor,
                "login_session_preserved": True,
                "management_route_restored": True,
            },
        )
        for i in range(4):
            assert generate(f"{failed}-down-{i}") == survivor
        start_host(failed)
        wait(lambda: all(v.get("status") == "ready" for v in status().values()), 120)
        wait(
            lambda: "sync" in sql("SELECT sync_state FROM pg_stat_replication", dbport),
            120,
        )
        # After Redis rejoins, confirm both copies are synchronized before the next cut.
        wait(
            lambda: any(
                "master_link_status:up" in redis_cli("INFO", "replication", port=p)
                for p in (26381, 26382)
            ),
            60,
        )
        stable_ingress()
        before = sql("SELECT count(*) FROM usage_logs", dbport)
    # Lose quorum: both app processes must fence instead of becoming dual primary.
    for h in ("hk", "witness"):
        stop(h + "-etcd", True)
    wait(lambda: all(v.get("status") != "ready" for v in status().values()), 20)
    record("lost_quorum", {"passed": True, "nodes_fenced": True})
    for h in ("hk", "witness"):
        launch(h + "-etcd", [TOOLS["ETCD"], "--config-file", ROOT / h / "etcd.json"])
    wait(lambda: all(v.get("status") == "ready" for v in status().values()), 120)
    wait(lambda: http("/ingress-ready")[0] == 200, 30)
    stable_ingress()
    generate("after-quorum-recovery")
    time.sleep(2)
    sums = json.loads(
        sql(
            "SELECT json_build_object('records',count(*),'distinct_requests',count(DISTINCT request_id),'cost',sum(actual_cost)) FROM usage_logs"
        )
    )
    balance = float(sql("SELECT balance FROM users WHERE id=" + str(auth["user_id"])))
    assert (
        sums["records"] == sums["distinct_requests"]
        and abs(100 - balance - sums["cost"]) < 0.000001
    )
    record("billing", dict(sums, remaining_balance=balance, matched=True))
    record("final_status", status())


if A.action != "start":
    saved = load("tools.json")
    TOOLS.update({k: v for k, v in saved.items() if not TOOLS[k]})
if A.action == "start":
    start()
elif A.action == "test":
    test()
elif A.action == "test-witness":
    test_witness()
elif A.action == "status":
    print(json.dumps(status(), indent=2))
else:
    for h in ("us", "hk", "witness"):
        kill_host(h)
    stop("edge")
    stop("fixture")
    print("lab stopped; data retained")
