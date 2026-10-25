#!/usr/bin/env python3
"""Additional acceptance checks against an explicitly selected local HA lab.

Uses synthetic accounts only. Control transport is direct fixture SQL; admin
HTTP authorization/compliance is not bypassed or acknowledged. Tests runtime
application/ingress acknowledgement and real rolling process replacement.
"""

import argparse
import collections
import concurrent.futures
import decimal
import importlib.util
import json
import pathlib
import shutil
import socket
import sys
import urllib.error
import urllib.request
import uuid

import operations


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dir", type=pathlib.Path, required=True)
    args = parser.parse_args()
    root = args.dir.resolve()
    if not (root / "auth.json").is_file() or not (root / "tools.json").is_file():
        parser.error("select an initialized isolated HA lab")
    saved = sys.argv
    sys.argv = ["lab.py", "status", "--dir", str(root)]
    spec = importlib.util.spec_from_file_location(
        "local_ha_lab", pathlib.Path(__file__).with_name("lab.py")
    )
    lab = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(lab)
    sys.argv = saved
    auth = lab.load("auth.json")
    report = {}
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

    def record(name, value):
        report[name] = value
        lab.save("ops-results.json", report)
        print(name, value, flush=True)

    def nodes():
        return json.loads(
            lab.sql(
                "SELECT json_agg(json_build_object('id',n.node_id,'instance_id',n.instance_id,'role',n.role,'version',n.version,'online',n.heartbeat_at>NOW()-INTERVAL '20 seconds','stats',n.snapshot,'control',to_jsonb(c))) FROM cluster_nodes n JOIN cluster_node_controls c USING(node_id)"
            )
        )

    class FixtureAPI:
        def node(self, ident):
            return next(n for n in nodes() if n["id"] == ident)

        def control(self, ident, maintenance):
            assert ident in ("us", "hk")
            value = "true" if maintenance else "false"
            lab.sql(
                f"UPDATE cluster_node_controls SET maintenance={value},revision=revision+1,reason='isolated acceptance fixture' WHERE node_id='{ident}'"
            )

    api = FixtureAPI()

    def controls_applied():
        return all(
            n["stats"]["control_revision"] >= n["control"]["revision"]
            and n["stats"]["ingress_revision"] >= n["control"]["revision"]
            for n in nodes()
        )

    def generate(stream=False, base=lab.BASE, slow=False):
        payload = {
            "model": "gpt-4.1-mini",
            "input": "local-drain-stream" if slow else "local ops acceptance",
            "max_output_tokens": 16,
            "stream": stream,
        }
        req = urllib.request.Request(
            base + "/v1/responses",
            json.dumps(payload).encode(),
            {
                "Authorization": "Bearer " + auth["key"],
                "Content-Type": "application/json",
                "X-Sub2API-Session-ID": uuid.uuid4().hex,
            },
        )
        try:
            response = opener.open(req, timeout=30)
        except urllib.error.HTTPError as exc:
            response = exc
        with response:
            body = response.read().decode()
            return response.status, response.headers.get("X-Local-Test-Node"), body

    lab.wait(lambda: len(nodes()) == 2 and controls_applied())
    record("node_controls_acknowledged", True)
    lab.sql("UPDATE accounts SET concurrency=200 WHERE name='LOCAL fixture'")
    initial = decimal.Decimal(
        lab.sql(f"SELECT balance FROM users WHERE id={int(auth['user_id'])}")
    )
    baseline = json.loads(
        lab.sql(
            "SELECT json_build_object('usage',(SELECT count(*) FROM usage_logs),'evidence',(SELECT count(*) FROM cluster_billing_evidence),'dedup',(SELECT count(*) FROM usage_billing_dedup),'journal',(SELECT count(*) FROM cluster_request_journal),'distinct_requests',(SELECT count(DISTINCT request_id) FROM usage_logs),'cost',(SELECT COALESCE(sum(actual_cost),0)::text FROM usage_logs))"
        )
    )
    with concurrent.futures.ThreadPoolExecutor(max_workers=24) as pool:
        results = list(pool.map(lambda i: generate(stream=i % 2 == 0), range(72)))
    assert all(r[0] == 200 and "LOCAL_CLUSTER_OK" in r[2] for r in results), (
        collections.Counter(r[0] for r in results)
    )
    record(
        "concurrency",
        {
            "parallel": 24,
            "completed": len(results),
            "nodes": dict(collections.Counter(r[1] for r in results)),
            "streaming": 36,
        },
    )
    lab.wait(
        lambda: (
            int(lab.sql("SELECT count(*) FROM usage_logs")) == baseline["usage"] + 72
        )
    )
    counts = json.loads(
        lab.sql(
            "SELECT json_build_object('usage', (SELECT count(*) FROM usage_logs),'evidence',(SELECT count(*) FROM cluster_billing_evidence),'dedup',(SELECT count(*) FROM usage_billing_dedup),'journal',(SELECT count(*) FROM cluster_request_journal),'distinct_requests',(SELECT count(DISTINCT request_id) FROM usage_logs),'cost',(SELECT sum(actual_cost)::text FROM usage_logs))"
        )
    )
    for key in ("usage", "evidence", "dedup", "journal", "distinct_requests"):
        counts[key] -= baseline[key]
    counts["cost"] = str(
        decimal.Decimal(counts["cost"]) - decimal.Decimal(baseline["cost"])
    )
    balance = decimal.Decimal(
        lab.sql(f"SELECT balance FROM users WHERE id={int(auth['user_id'])}")
    )
    assert all(
        counts[k] == 72
        for k in ("usage", "evidence", "dedup", "journal", "distinct_requests")
    ), counts
    assert initial - balance == decimal.Decimal(counts["cost"]), counts
    record("billing_exact", counts)
    lab.sql(
        "UPDATE cluster_node_controls SET weight=CASE WHEN node_id='us' THEN 240 ELSE 10 END,revision=revision+1"
    )
    lab.wait(controls_applied)
    with socket.socket(socket.AF_UNIX) as sock:
        sock.connect(str(root / "hk/admin.sock"))
        sock.sendall(b"show stat\n")
        sock.shutdown(socket.SHUT_WR)
        data = b""
        while block := sock.recv(65536):
            data += block
    lines = data.decode().splitlines()
    header = lines[0].lstrip("# ").split(",")
    stats = [
        dict(zip(header, line.split(","), strict=False))
        for line in lines[1:]
        if line.strip()
    ]
    weights = {
        s["svname"]: s["weight"]
        for s in stats
        if s["pxname"] == "gateway" and s["svname"] in ("us", "hk")
    }
    assert weights == {"us": "240", "hk": "10"}, weights
    record("runtime_weights", weights)
    lab.sql(
        "UPDATE cluster_node_controls SET max_concurrency=2,revision=revision+1 WHERE node_id='us'"
    )
    lab.wait(controls_applied)
    with concurrent.futures.ThreadPoolExecutor(max_workers=12) as pool:
        caps = list(
            pool.map(
                lambda _: generate(base="http://127.0.0.1:28191", slow=True), range(12)
            )
        )
    counts = collections.Counter(r[0] for r in caps)
    assert counts[429] > 0 and counts[200] == 2, counts
    record("admission_capacity", dict(counts))
    lab.sql(
        "UPDATE cluster_node_controls SET max_concurrency=1000,weight=100,revision=revision+1"
    )
    lab.wait(controls_applied)
    primary = next(n["id"] for n in nodes() if n["role"] == "primary")
    direct = "http://127.0.0.1:" + ("28191" if primary == "us" else "28192")
    with concurrent.futures.ThreadPoolExecutor(max_workers=1) as pool:
        pending = pool.submit(generate, True, direct, True)
        lab.wait(lambda: api.node(primary)["stats"]["active_requests"] > 0)
        api.control(primary, True)
        result = pending.result(timeout=30)
        assert result[0] == 200 and "LOCAL_CLUSTER_OK" in result[2]
    lab.wait(lambda: api.node(primary)["role"] == "worker", 120)
    lab.wait(lambda: lab.http("/ingress-ready")[0] == 200, 60)
    assert generate()[0] == 200
    api.control(primary, False)
    lab.wait(controls_applied)
    record(
        "primary_drain",
        {
            "old_primary": primary,
            "stream_completed": True,
            "new_primary": next(n["id"] for n in nodes() if n["role"] == "primary"),
        },
    )
    # Give each runner its own binary so replacement cannot change the other node.
    target = primary
    config_path = root / target / "runner.json"
    cfg = json.loads(config_path.read_text())
    installed = root / target / "sub2api-current"
    if pathlib.Path(cfg["binary"]).resolve() != installed.resolve():
        shutil.copy2(cfg["binary"], installed)
    cfg["binary"] = str(installed)
    config_path.write_text(json.dumps(cfg))
    config_path.chmod(0o600)

    def restart():
        lab.stop(target + "-runner")
        lab.wait(lambda: not lab.alive(lab.load("pids.json")[target + "-runner"]))
        lab.launch(target + "-runner", [lab.TOOLS["RUNNER"], "-config", config_path])

    artifact = pathlib.Path(lab.TOOLS["APP"])
    result = operations.upgrade(
        api,
        target,
        artifact,
        installed,
        operations.digest(artifact),
        api.node(target)["version"],
        restart,
    )
    lab.wait(controls_applied)
    record("rolling_restart", result)
    broken = root / target / "deliberately-broken"
    broken.write_text("#!/bin/sh\nexit 1\n")
    broken.chmod(0o755)
    previous_hash = operations.digest(installed)
    try:
        operations.upgrade(
            api,
            target,
            broken,
            installed,
            operations.digest(broken),
            "unbootable",
            restart,
            timeout=20,
        )
        raise AssertionError("broken artifact unexpectedly booted")
    except RuntimeError as exc:
        assert "old binary restored" in str(exc)
    lab.wait(
        lambda: api.node(target)["online"] and api.node(target)["stats"]["lease_ok"], 60
    )
    assert operations.digest(installed) == previous_hash
    assert api.node(target)["control"]["maintenance"]
    api.control(target, False)
    lab.wait(controls_applied)
    record(
        "failed_boot_rollback",
        {"binary_restored": True, "kept_drained_until_verified": True},
    )
    # The actual admin endpoint remains protected; no compliance acknowledgement.
    response = lab.http("/api/v1/admin/cluster", token=auth["token"])
    record(
        "admin_http",
        {
            "status": response[0],
            "control_transport": "isolated fixture SQL; handler authorization tested separately",
        },
    )


if __name__ == "__main__":
    main()
