#!/usr/bin/env python3
"""Exercise both real loopback HA ingresses; synthetic provider and lab users only."""
import argparse
import collections
import concurrent.futures
import decimal
import importlib.util
import json
import pathlib
import sys
import uuid

import verify


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--dir',required=True,type=pathlib.Path)
    parser.add_argument('--restore-result',type=pathlib.Path)
    args=parser.parse_args()
    root=args.dir.resolve()
    if not (root/'auth.json').is_file() or not (root/'tools.json').is_file():
        parser.error('select an initialized isolated HA lab')
    saved=sys.argv
    sys.argv=['lab.py','status','--dir',str(root)]
    spec=importlib.util.spec_from_file_location('dual_ingress_lab',pathlib.Path(__file__).with_name('lab.py'))
    lab=importlib.util.module_from_spec(spec);spec.loader.exec_module(lab);sys.argv=saved
    auth=lab.load('auth.json')
    origins=['http://127.0.0.1:28181','http://127.0.0.1:28182']
    report={}
    for origin in origins:
        lab.wait(lambda:lab.http('/ingress-ready',base=origin)[0]==200,45)
    identities=[lab.data('/api/v1/auth/me',token=auth['token'],base=origin) for origin in origins]
    assert identities[0]==identities[1], 'same login must have the same identity at both ingresses'
    report['login_shared']=True

    def generate(index,session=None):
        status,headers,body=lab.http('/v1/responses',{'model':'gpt-4.1-mini','input':'dual ingress local acceptance','stream':True,'max_output_tokens':16},auth['key'],base=origins[index%2],headers={'X-Sub2API-Session-ID':session or uuid.uuid4().hex},timeout=30)
        assert status==200 and 'LOCAL_CLUSTER_OK' in str(body), status
        return next(v for k,v in headers.items() if k.lower()=='x-local-test-node')

    # Two requests of one conversation, alternating independent ingress processes.
    for _ in range(8):
        session=uuid.uuid4().hex
        assert generate(0,session)==generate(1,session)
    report['cross_ingress_sessions']=8
    lab.wait(lambda:all(v.get('active_requests')==0 for v in lab.status().values()),30)
    initial=decimal.Decimal(lab.sql(f"SELECT balance::text FROM users WHERE id={int(auth['user_id'])}"))
    def ledger():
        return json.loads(lab.sql("SELECT json_build_object('usage',(SELECT count(*) FROM usage_logs),'dedup',(SELECT count(*) FROM usage_billing_dedup),'evidence',(SELECT count(*) FROM cluster_billing_evidence),'cost',(SELECT COALESCE(sum(actual_cost),0)::text FROM usage_logs))"))
    before=ledger()
    with concurrent.futures.ThreadPoolExecutor(max_workers=24) as pool:
        result=list(pool.map(generate,range(72)))
    lab.wait(lambda:ledger()['usage']==before['usage']+72,30)
    after=ledger()
    balance=decimal.Decimal(lab.sql(f"SELECT balance::text FROM users WHERE id={int(auth['user_id'])}"))
    delta=decimal.Decimal(after['cost'])-decimal.Decimal(before['cost'])
    assert initial-balance==delta
    assert all(after[key]-before[key]==72 for key in ('usage','dedup','evidence'))
    assert set(result)=={'us','hk'}
    report['concurrency']={'parallel':24,'completed':72,'nodes':dict(collections.Counter(result)),'origins':{'us':36,'hk':36},'billing_cost':str(delta),'balance_matches':True}

    def measurements():
        snapshot=lab.data('/api/v1/admin/cluster',token=auth['token'])
        for node in snapshot['nodes']:
            p=node['stats'].get('performance',{})
            traffic=node['stats'].get('ingress_traffic',{})
            if not (p.get('first_token_samples',0)>0 and p.get('cache_read_percent') is not None and traffic.get('cross_node_bytes',0)>0):
                return False
        return snapshot
    snapshot=lab.wait(measurements,30)
    report['performance']={n['id']:{'performance':n['stats']['performance'],'ingress_traffic':n['stats']['ingress_traffic']} for n in snapshot['nodes']}
    inv={'domain':'localhost','endpoints':[{'id':'us','ip':'127.0.0.1','port':28181,'http':True},{'id':'hk','ip':'127.0.0.1','port':28182,'http':True}]}
    manifests=[]
    for host in ('us','hk'):
        path=root/host/'release-manifest.json'
        verify.write_report(path,verify.manifest(root/host,['sub2api-current']))
        manifests.append(path)
    proof=verify.verify(inv,local_lab=True,manifests=manifests,restore=args.restore_result)
    assert all(e['status']=='pass' for e in proof['endpoints']) and proof['artifacts_match']
    for host in ('us','hk'): verify.write_report(root/host/'proof.json.verification',proof)
    report['verification']=proof
    verify.write_report(root/'dual-ingress-results.json',report)
    print(json.dumps(report,indent=2))


if __name__=='__main__': main()
