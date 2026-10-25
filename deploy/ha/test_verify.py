import datetime as dt
import http.server
import json
import pathlib
import socketserver
import shutil
import ssl
import subprocess
import tempfile
import threading
import unittest
from unittest.mock import patch
import verify
import proxy_agent


class VerificationTests(unittest.TestCase):
    def test_two_live_loopback_ingresses_and_failure(self):
        class Handler(http.server.BaseHTTPRequestHandler):
            healthy = True
            def do_GET(self):
                self.send_response(200 if self.healthy else 503)
                self.end_headers()
                self.wfile.write(b'ready' if self.path == '/ingress-ready' else b'page')
            def log_message(self, *args):
                pass
        servers=[]
        try:
            for _ in range(2):
                server=http.server.ThreadingHTTPServer(('127.0.0.1',0),Handler)
                threading.Thread(target=server.serve_forever,daemon=True).start()
                servers.append(server)
            inv={'domain':'example.test','endpoints':[{'id':str(i),'ip':'127.0.0.1','port':s.server_port,'http':True} for i,s in enumerate(servers)]}
            with self.assertRaises(ValueError): verify.verify(inv)
            good=verify.verify(inv,local_lab=True)
            self.assertTrue(all(e['status']=='pass' for e in good['endpoints']))
            self.assertTrue(all(e['certificate_expires_at'] is None for e in good['endpoints']))
            Handler.healthy=False
            bad=verify.verify(inv,local_lab=True)
            self.assertTrue(all(e['status']=='fail' for e in bad['endpoints']))
        finally:
            for server in servers: server.shutdown(); server.server_close()

    def test_tls_pins_each_ip_but_validates_hostname(self):
        openssl = shutil.which('openssl')
        if not openssl: self.skipTest('openssl required for local TLS fixture')
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            (root/'ssl.cnf').write_text('[req]\ndistinguished_name=dn\nx509_extensions=v3\nprompt=no\n[dn]\nCN=example.test\n[v3]\nsubjectAltName=DNS:example.test\nbasicConstraints=critical,CA:TRUE\n')
            subprocess.run([openssl,'req','-x509','-newkey','rsa:2048','-nodes','-days','1','-config',str(root/'ssl.cnf'),'-keyout',str(root/'key.pem'),'-out',str(root/'cert.pem')],check=True,capture_output=True)
            class Handler(http.server.BaseHTTPRequestHandler):
                def do_GET(self):
                    self.send_response(200); self.end_headers()
                    self.wfile.write(b'ready' if self.path=='/ingress-ready' else b'page')
                def log_message(self,*args): pass
            server = http.server.ThreadingHTTPServer(('127.0.0.1',0),Handler)
            tls=ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
            tls.load_cert_chain(root/'cert.pem',root/'key.pem')
            server.socket=tls.wrap_socket(server.socket,server_side=True)
            threading.Thread(target=server.serve_forever,daemon=True).start()
            try:
                entry={'id':'tls','ip':'127.0.0.1','port':server.server_port}
                context=ssl.create_default_context(cafile=str(root/'cert.pem'))
                self.assertEqual(verify.probe('example.test',entry,context)['status'],'pass')
                self.assertEqual(verify.probe('wrong.test',entry,context)['reason'],'tls_failed')
                self.assertEqual(verify.probe('example.test',entry,ssl.create_default_context())['reason'],'tls_failed')
            finally:
                server.shutdown();server.server_close()

    def test_artifact_manifest_detects_drift_and_blocks_secrets(self):
        with tempfile.TemporaryDirectory() as directory:
            root=pathlib.Path(directory)
            (root/'app').write_text('version-a')
            first=verify.manifest(root,['app']); (root/'a.json').write_text(json.dumps(first))
            (root/'app').write_text('version-b')
            second=verify.manifest(root,['app']); (root/'b.json').write_text(json.dumps(second))
            self.assertFalse(verify.compare_manifests([root/'a.json',root/'b.json']))
            self.assertTrue(verify.compare_manifests([root/'a.json',root/'a.json']))
            for name in ('.env','../outside','config.yaml','tls.key'):
                with self.assertRaises(ValueError): verify.manifest(root,[name])
            verify.write_report(root/'report',{'safe':True})
            self.assertEqual((root/'report').stat().st_mode & 0o777,0o600)

    def test_rejects_host_injection_duplicates_and_false_restore(self):
        inv={'domain':'example.test','endpoints':[{'id':'a','ip':'127.0.0.1'},{'id':'b','ip':'127.0.0.1'}]}
        with self.assertRaises(ValueError): verify.validate(inv)
        inv['domain']='example.test\nInjected: value'
        with self.assertRaises(ValueError): verify.validate(inv)

    def test_proxy_statistics_exclude_frontend_and_backend_totals(self):
        with tempfile.TemporaryDirectory() as directory:
            path=str(pathlib.Path(directory)/'stats.sock')
            class Handler(socketserver.BaseRequestHandler):
                def handle(self):
                    self.request.recv(4096)
                    self.request.sendall(b'# pxname,svname,type,qcur,scur,bin,bout\ngateway,us,2,1,3,20,40\nresponses,hk,2,2,4,10,30\ngateway,BACKEND,1,99,99,999,999\n')
            with socketserver.UnixStreamServer(path,Handler) as server:
                threading.Thread(target=server.handle_request,daemon=True).start()
                result=proxy_agent.traffic(path,'us')
            self.assertEqual(result['queued'],3)
            self.assertEqual(result['active'],7)
            self.assertEqual(result['cross_node_bytes'],40)
            self.assertEqual(result['bytes_out'],70)

    def test_failure_streak_reset_and_stale_probes(self):
        result={'checked_at':verify.now(),'endpoints':[{'id':'a','status':'fail'}]}
        first=verify.track_failures(result,None)
        self.assertEqual(first['endpoints'][0]['consecutive_failures'],1)
        second=verify.track_failures(json.loads(json.dumps(result)),first)
        self.assertEqual(second['endpoints'][0]['consecutive_failures'],2)
        recovered={'checked_at':verify.now(),'endpoints':[{'id':'a','status':'pass'}]}
        self.assertEqual(verify.track_failures(recovered,second)['endpoints'][0]['consecutive_failures'],0)
        second['checked_at']=(dt.datetime.now(dt.timezone.utc)-dt.timedelta(hours=1)).isoformat()
        self.assertEqual(verify.track_failures(result,second)['endpoints'][0]['consecutive_failures'],1)

    def test_cross_storage_requires_distinct_nodes_and_matching_receipts(self):
        with tempfile.TemporaryDirectory() as directory:
            root=pathlib.Path(directory)
            values=[dict(node='a',host='host-a',challenge='challenge',digest='a'*64,read=False),dict(node='b',host='host-b',challenge='challenge',digest='b'*64,read=False),dict(node='a',host='host-a',challenge='challenge',digest='b'*64,read=True),dict(node='b',host='host-b',challenge='challenge',digest='a'*64,read=True)]
            paths=[]
            for i,value in enumerate(values):
                value.update(checked_at=verify.now(),scope='cross_node_probe');path=root/str(i);path.write_text(json.dumps(value));paths.append(path)
            self.assertEqual(verify.combine_storage(paths)['status'],'pass')
            values[-1]['digest']='b'*64;paths[-1].write_text(json.dumps(values[-1]))
            with self.assertRaises(ValueError):verify.combine_storage(paths)


if __name__=='__main__': unittest.main()
