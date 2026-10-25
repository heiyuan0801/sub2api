#!/usr/bin/env python3
"""Bounded loopback ingress delay/reset exercise using the HA lab's synthetic provider.

Delays apply to TCP reads in both directions, not kernel packet loss or database
replication. Never replays generation POSTs and never edits system networking.
"""
import argparse
import collections
import concurrent.futures
import contextlib
import decimal
import importlib.util
import json
import pathlib
import random
import socket
import socketserver
import sys
import threading
import time
import uuid

class DelayProxy(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True
    request_queue_size = 128
    def __init__(self, target, delay_ms=0, jitter_ms=0, reset_every=0):
        if target[0] != '127.0.0.1':
            raise ValueError('local loopback target required')
        self.target, self.delay, self.jitter, self.reset_every = target, delay_ms, jitter_ms, reset_every
        self.count = 0
        self.lock = threading.Lock()
        super().__init__(('127.0.0.1', 0), DelayHandler)
    def next_reset(self):
        with self.lock:
            self.count += 1
            return self.reset_every > 0 and self.count % self.reset_every == 0
    def handle_error(self, *args):
        pass

class DelayHandler(socketserver.BaseRequestHandler):
    def handle(self):
        if self.server.next_reset():
            return  # Disconnect before forwarding: no uncertain upstream replay.
        with socket.create_connection(self.server.target, timeout=30) as upstream:
            self.request.settimeout(30)
            def copy(source, destination):
                try:
                    while data := source.recv(65536):
                        delay = max(0, self.server.delay + random.uniform(-self.server.jitter, self.server.jitter)) / 1000
                        if delay: time.sleep(delay)
                        destination.sendall(data)
                except (OSError, TimeoutError):
                    pass
                finally:
                    with contextlib.suppress(OSError): destination.shutdown(socket.SHUT_WR)
            thread = threading.Thread(target=copy,args=(self.request,upstream),daemon=True)
            thread.start(); copy(upstream,self.request); thread.join(timeout=31)

@contextlib.contextmanager
def proxy(target, **kwargs):
    server = DelayProxy(target, **kwargs)
    thread = threading.Thread(target=server.serve_forever, daemon=True); thread.start()
    try: yield server
    finally: server.shutdown(); server.server_close(); thread.join(timeout=5)

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--dir',type=pathlib.Path,required=True)
    parser.add_argument('--output',type=pathlib.Path,required=True)
    parser.add_argument('--soak-seconds',type=int,default=120)
    args=parser.parse_args()
    if not 0<=args.soak_seconds<=1800:parser.error('soak must be 0-1800 seconds')
    root=args.dir.resolve()
    if not(root/'auth.json').is_file() or not(root/'tools.json').is_file():parser.error('initialized isolated HA lab required')
    saved=sys.argv;sys.argv=['lab.py','status','--dir',str(root)]
    spec=importlib.util.spec_from_file_location('network_lab',pathlib.Path(__file__).with_name('lab.py'))
    lab=importlib.util.module_from_spec(spec);spec.loader.exec_module(lab);sys.argv=saved
    auth=lab.load('auth.json');uid=int(auth['user_id'])
    report={'scope':'loopback_ingress_delay_and_pre_forward_disconnect','cases':[]}
    def ledger():
        return json.loads(lab.sql(f"SELECT json_build_object('balance',(SELECT balance::text FROM users WHERE id={uid}),'cost',(SELECT COALESCE(sum(actual_cost),0)::text FROM usage_logs WHERE user_id={uid}),'usage',(SELECT count(*) FROM usage_logs WHERE user_id={uid}),'evidence',(SELECT count(*) FROM cluster_billing_evidence WHERE user_id={uid}))"))
    def generate(i,bases):
        start=time.monotonic()
        try:
            status,headers,body=lab.http('/v1/responses',{'model':'gpt-4.1-mini','input':'local network acceptance','stream':i%2==0,'max_output_tokens':16},auth['key'],base=bases[i%2],headers={'X-Sub2API-Session-ID':uuid.uuid4().hex},timeout=30)
            return {'ok':status==200 and 'LOCAL_CLUSTER_OK' in str(body),'status':status,'ms':(time.monotonic()-start)*1000,'node':next((v for k,v in headers.items() if k.lower()=='x-local-test-node'),None)}
        except (OSError,TimeoutError):
            return {'ok':False,'status':0,'ms':(time.monotonic()-start)*1000}
    for concurrency,delay,jitter,reset in [(8,0,0,0),(24,80,20,0),(48,150,40,0),(24,80,20,10)]:
        before=ledger()
        with proxy(('127.0.0.1',28181),delay_ms=delay,jitter_ms=jitter,reset_every=reset) as a,proxy(('127.0.0.1',28182),delay_ms=delay,jitter_ms=jitter,reset_every=reset) as b:
            bases=[f'http://127.0.0.1:{p.server_address[1]}' for p in (a,b)]
            with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as pool:
                results=list(pool.map(lambda i:generate(i,bases),range(96)))
        completed=sum(r['ok'] for r in results)
        lab.wait(lambda:ledger()['usage']==before['usage']+completed,30)
        after=ledger();cost=decimal.Decimal(after['cost'])-decimal.Decimal(before['cost'])
        assert decimal.Decimal(before['balance'])-decimal.Decimal(after['balance'])==cost
        assert after['evidence']-before['evidence']==completed
        counts=collections.Counter(r['status'] for r in results)
        expected_disconnects=0 if reset==0 else 8
        assert counts[0]==expected_disconnects and completed+counts[429]+expected_disconnects==96,(completed,counts)
        assert completed>0
        case={'concurrency':concurrency,'per_read_delay_ms':delay,'jitter_ms':jitter,'reset_every':reset,'requests':96,'completed':completed,'errors':96-completed,'limited':counts[429],'injected_disconnects':counts[0],'p95_ms':sorted(r['ms'] for r in results)[int(len(results)*.95)-1],'nodes':dict(collections.Counter(r.get('node') for r in results if r['ok'])),'billing_matches':True,'cost':str(cost)}
        report['cases'].append(case);print(json.dumps(case),flush=True)
    start=time.monotonic();count=0;before=ledger();bases=['http://127.0.0.1:28181','http://127.0.0.1:28182']
    while time.monotonic()-start<args.soak_seconds:
        result=generate(count,bases);assert result['ok'],result;count+=1
        time.sleep(.25)  # Bounded sustained local fixture load, no paid provider.
    lab.wait(lambda:ledger()['usage']==before['usage']+count,30);after=ledger()
    assert decimal.Decimal(before['balance'])-decimal.Decimal(after['balance'])==decimal.Decimal(after['cost'])-decimal.Decimal(before['cost'])
    assert after['evidence']-before['evidence']==count
    report['soak']={'seconds':round(time.monotonic()-start,1),'completed':count,'billing_matches':True}
    args.output.write_text(json.dumps(report,indent=2));print(json.dumps(report['soak']),flush=True)
if __name__=='__main__':main()
