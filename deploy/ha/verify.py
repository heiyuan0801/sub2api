#!/usr/bin/env python3
"""Read-only dual-ingress probes and explicit release-file manifests.

TLS uses the public hostname for SNI/validation while connecting to each IP.
No authentication, generation, payment, redirect, environment proxy or DNS edit.
Reports contain bounded metadata only, never response bodies or credentials.
"""
import argparse
import concurrent.futures
import datetime as dt
import hashlib
import http.client
import ipaddress
import json
import os
import pathlib
import re
import socket
import ssl
import tempfile
import time


def now():
    return dt.datetime.now(dt.timezone.utc).isoformat()


def write_report(path, value):
    path = pathlib.Path(path)
    fd, temp = tempfile.mkstemp(prefix='.verification-', dir=path.parent)
    try:
        with os.fdopen(fd, 'w') as f:
            json.dump(value, f, indent=2)
            f.flush()
            os.fsync(f.fileno())
        os.replace(temp, path)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)


def validate(inv, local_lab=False):
    domain = inv.get('domain', '')
    if not re.fullmatch(r'[A-Za-z0-9](?:[A-Za-z0-9.-]{0,251}[A-Za-z0-9])?', domain):
        raise ValueError('domain must be a hostname without a port or path')
    entries = inv.get('endpoints', [])
    if not 2 <= len(entries) <= 8 or len({e['id'] for e in entries}) != len(entries):
        raise ValueError('select 2-8 distinct ingress identities')
    addresses = set()
    for e in entries:
        if not re.fullmatch(r'[A-Za-z0-9_-]{1,64}', e['id']):
            raise ValueError('invalid ingress identity')
        ip = ipaddress.ip_address(e['ip'])
        port = e.get('port', 443)
        if type(port) is not int or not 1 <= port <= 65535:
            raise ValueError('invalid port')
        if (str(ip), port) in addresses:
            raise ValueError('duplicate ingress address')
        addresses.add((str(ip), port))
        if e.get('http', False) and not (local_lab and ip.is_loopback):
            raise ValueError('plain HTTP is restricted to explicit loopback labs')
    return entries


class PinnedConnection(http.client.HTTPSConnection):
    def __init__(self, domain, entry, context):
        super().__init__(domain, entry.get('port', 443), timeout=5, context=context)
        self.ip = entry['ip']
        self.plain = entry.get('http', False)

    def connect(self):
        sock = socket.create_connection((self.ip, self.port), self.timeout)
        try:
            self.sock = sock if self.plain else self._context.wrap_socket(sock, server_hostname=self.host)
        except Exception:
            sock.close()
            raise


def probe(domain, entry, context):
    result = {'id': entry['id'], 'status': 'fail', 'latency_ms': None, 'certificate_expires_at': None}
    start = time.monotonic()
    try:
        for path in ('/ingress-ready', '/health', '/home'):
            connection = PinnedConnection(domain, entry, context)
            try:
                connection.connect()
                if not entry.get('http', False):
                    expiry = connection.sock.getpeercert().get('notAfter')
                    if expiry:
                        result['certificate_expires_at'] = dt.datetime.fromtimestamp(ssl.cert_time_to_seconds(expiry), dt.timezone.utc).isoformat()
                connection.request('GET', path, headers={'Host': domain, 'Connection': 'close', 'Cache-Control': 'no-cache'})
                response = connection.getresponse()
                body = response.read(4097)
                if response.status != 200 or (path == '/ingress-ready' and body.strip() != b'ready'):
                    result['reason'] = 'unexpected_response'
                    return result
            finally:
                connection.close()
        result['status'] = 'pass'
    except ssl.SSLError:
        result['reason'] = 'tls_failed'
    except (OSError, http.client.HTTPException):
        result['reason'] = 'connection_failed'
    finally:
        result['latency_ms'] = round((time.monotonic()-start)*1000, 2)
    return result


def manifest(root, files):
    root = pathlib.Path(root).resolve(strict=True)
    if not files or len(files) > 5000:
        raise ValueError('provide 1-5000 explicit release files')
    items = {}
    for name in sorted(set(files)):
        relative = pathlib.Path(name)
        if relative.is_absolute() or '..' in relative.parts or any(x.startswith('.') for x in relative.parts):
            raise ValueError('release file paths must be relative and not hidden')
        if relative.suffix.lower() in ('.key', '.pem', '.env', '.yaml', '.yml') or relative.name in ('credentials.json', 'config.json', 'auth.json'):
            raise ValueError('do not include private configuration or credentials in manifests')
        path = root / relative
        if path.is_symlink() or not path.resolve(strict=True).is_relative_to(root) or not path.is_file():
            raise ValueError('release file must remain inside root')
        if path.stat().st_size > 512*1024*1024:
            raise ValueError('release file exceeds 512 MiB')
        digest = hashlib.sha256()
        with path.open('rb') as source:
            while block := source.read(1024*1024):
                digest.update(block)
        items[name] = digest.hexdigest()
    payload = json.dumps(items, sort_keys=True).encode()
    return {'checked_at': now(), 'files': items, 'digest': hashlib.sha256(payload).hexdigest()}


def compare_manifests(paths):
    if len(paths) < 2:
        return None
    values = []
    for path in paths:
        data = json.loads(pathlib.Path(path).read_text())
        sampled = dt.datetime.fromisoformat(data['checked_at'].replace('Z', '+00:00'))
        if not -5 <= (dt.datetime.now(dt.timezone.utc)-sampled).total_seconds() <= 900:
            raise ValueError('recapture release manifests on both nodes within 15 minutes')
        files = data.get('files', {})
        if not files or any(not re.fullmatch(r'[0-9a-f]{64}', v) for v in files.values()):
            raise ValueError('invalid artifact manifest')
        values.append(files)
    return all(value == values[0] for value in values[1:])


def verify(inv, ca_file=None, local_lab=False, manifests=(), restore=None):
    entries = validate(inv, local_lab)
    context = ssl.create_default_context(cafile=ca_file)
    with concurrent.futures.ThreadPoolExecutor(max_workers=len(entries)) as pool:
        probes = list(pool.map(lambda entry: probe(inv['domain'], entry, context), entries))
    result = {'checked_at': now(), 'scope': 'local_lab' if local_lab else 'external_probe', 'endpoints': probes, 'artifacts_match': compare_manifests(manifests)}
    if restore:
        data = json.loads(pathlib.Path(restore).read_text())
        required = ('physical_backup_verified', 'wal_archive_delivered', 'pitr_target_reached', 'later_transaction_excluded')
        if not all(data.get(k) is True for k in required) or not data.get('checked_at') or data.get('scope') != 'local_lab':
            raise ValueError('restore report must contain a completed isolated restore drill')
        result['restore_checked_at'] = data['checked_at']
        result['restore_scope'] = 'local_lab'
    return result


def track_failures(result, previous):
    previous = previous or {}
    try:
        age = (dt.datetime.now(dt.timezone.utc)-dt.datetime.fromisoformat(previous['checked_at'].replace('Z','+00:00'))).total_seconds()
        old = {p['id']: p for p in previous['endpoints']} if 0 <= age <= 300 else {}
    except (KeyError, TypeError, ValueError):
        old = {}
    for p in result['endpoints']:
        count = old.get(p['id'], {}).get('consecutive_failures', 0)
        count = count if type(count) is int and count >= 0 else 0
        p['consecutive_failures'] = min(1000, count+1) if p['status'] == 'fail' else 0
    return result


def storage_report(path):
    data = json.loads(pathlib.Path(path).read_text())
    if data.get('status') not in ('pass','fail') or data.get('scope') not in ('local_lab','cross_node_probe'):
        raise ValueError('invalid shared storage report')
    sampled = dt.datetime.fromisoformat(data['checked_at'].replace('Z','+00:00'))
    if not -5 <= (dt.datetime.now(dt.timezone.utc)-sampled).total_seconds() <= 86400:
        raise ValueError('shared storage report is older than one day')
    return {k:data[k] for k in ('checked_at','status','scope')}

def combine_storage(paths, local_lab=False):
    if len(paths) != 4:
        raise ValueError('supply two write and two cross-read receipts')
    receipts = [json.loads(pathlib.Path(p).read_text()) for p in paths]
    writes = {r['node']: r for r in receipts if r.get('read') is False}
    reads = {r['node']: r for r in receipts if r.get('read') is True}
    if len(writes) != 2 or set(reads) != set(writes) or len({r['challenge'] for r in receipts}) != 1:
        raise ValueError('distinct nodes must cross-read the same challenge')
    if not local_lab and len({r['host'] for r in writes.values()}) != 2:
        raise ValueError('production probes must run on distinct hosts')
    for node, write in writes.items():
        peer = next(r for n,r in writes.items() if n != node)
        read = reads[node]
        if read['digest'] != peer['digest'] or read['host'] != write['host']:
            raise ValueError('cross-read digest or host mismatch')
    for r in receipts:
        age = (dt.datetime.now(dt.timezone.utc)-dt.datetime.fromisoformat(r['checked_at'].replace('Z','+00:00'))).total_seconds()
        if not -5 <= age <= 3600 or (r['scope'] == 'local_lab') != local_lab or not re.fullmatch('[0-9a-f]{64}',r['digest']):
            raise ValueError('stale or invalid storage receipt')
    return {'checked_at':now(),'scope':'local_lab' if local_lab else 'cross_node_probe','status':'pass'}

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='action', required=True)
    check = commands.add_parser('ingress')
    check.add_argument('--inventory', required=True, type=pathlib.Path)
    check.add_argument('--ca-file')
    check.add_argument('--local-lab', action='store_true')
    check.add_argument('--manifest', action='append', default=[])
    check.add_argument('--restore-result')
    check.add_argument('--storage-result')
    files = commands.add_parser('manifest')
    files.add_argument('--root', required=True)
    files.add_argument('--file', action='append', required=True)
    storage = commands.add_parser('storage')
    storage.add_argument('--receipt',action='append',required=True)
    storage.add_argument('--local-lab',action='store_true')
    for command in (check, files, storage):
        command.add_argument('--output', required=True, type=pathlib.Path)
    args = parser.parse_args()
    if args.action == 'manifest':
        result = manifest(args.root, args.file)
    elif args.action == 'storage':
        try:
            result = combine_storage(args.receipt,args.local_lab)
        except (OSError, ValueError, KeyError, TypeError):
            write_report(args.output, {'checked_at':now(),'scope':'local_lab' if args.local_lab else 'cross_node_probe','status':'fail'})
            raise SystemExit('Storage receipts failed verification; failure report saved')
    else:
        result = verify(json.loads(args.inventory.read_text()), args.ca_file, args.local_lab, args.manifest, args.restore_result)
    if args.action == 'ingress':
        try:
            previous = json.loads(args.output.read_text())
        except (OSError, ValueError):
            previous = None
        result = track_failures(result, previous)
        if args.storage_result:
            result['storage'] = storage_report(args.storage_result)
    write_report(args.output, result)
    print('Verification report saved:', args.output)
    if args.action == 'ingress' and (any(e['status'] != 'pass' for e in result['endpoints']) or result['artifacts_match'] is False):
        raise SystemExit(1)


if __name__ == '__main__':
    main()
