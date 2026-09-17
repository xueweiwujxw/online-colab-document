#!/usr/bin/env python3
"""Run only against the disposable Compose test project (creates users/documents)."""
import base64
import http.cookiejar
import io
import json
import os
import secrets
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request
import zipfile

BASE = os.environ.get('TEST_API_URL', 'http://127.0.0.1:8080')
COMPOSE = ['docker', 'compose', '-f', 'deploy/docker-compose.yml']

class Client:
    def __init__(self):
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def call(self, method, path, data=None, content_type='application/json', expected=200, raw=False):
        if isinstance(data, dict):
            data = json.dumps(data).encode()
        req = urllib.request.Request(BASE + path, data=data, method=method, headers={'Content-Type': content_type})
        try:
            response = self.opener.open(req, timeout=20)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            body = response.read()
            # Do not print URLs, response bodies, or cookies: share/WOPI tokens are sensitive.
            assert response.status == expected, f'{method}: expected {expected}, got {response.status}'
            return body if raw or not body else json.loads(body)

    def register(self):
        password = secrets.token_urlsafe(24)
        email = secrets.token_hex(8) + '@rustfs.test'
        user = self.call('POST', '/api/auth/local/register', {'email': email, 'password': password, 'displayName': 'RustFS test'}, expected=201)
        self.call('POST', '/api/auth/local/login', {'email': email, 'password': password})
        return user

    def upload(self, ext, data):
        boundary = secrets.token_hex(16)
        mime = {'md': 'text/markdown', 'docx': 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', 'xlsx': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'}[ext]
        body = (f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="test.{ext}"\r\nContent-Type: {mime}\r\n\r\n'.encode() + data + f'\r\n--{boundary}--\r\n'.encode())
        return self.call('POST', '/api/documents/upload', body, f'multipart/form-data; boundary={boundary}', expected=201)

def ready():
    for _ in range(90):
        try:
            status = Client().call('GET', '/readyz')
            if status['status'] == 'ok':
                return
        except (OSError, AssertionError):
            pass
        time.sleep(1)
    raise AssertionError('backend did not become ready')

def office_bytes(ext, marker):
    output = io.BytesIO()
    with zipfile.ZipFile(output, 'w') as archive:
        archive.writestr('[Content_Types].xml', '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>')
        archive.writestr('word/document.xml' if ext == 'docx' else 'xl/workbook.xml', f'<test>{marker}</test>')
    return output.getvalue()

ready()
owner, viewer, anonymous = Client(), Client(), Client()
owner_user, viewer_user = owner.register(), viewer.register()
persisted = []
for ext in ('md', 'docx', 'xlsx'):
    original = b'# first\n' if ext == 'md' else office_bytes(ext, 'first')
    changed = b'# second\n' if ext == 'md' else office_bytes(ext, 'second')
    doc = owner.upload(ext, original)
    path = '/api/documents/' + doc['id']
    first_version = owner.call('GET', path + '/versions')['items'][0]['id']
    assert owner.call('GET', path + '/download', raw=True) == original
    anonymous.call('GET', path + '/download', expected=401)
    viewer.call('GET', path + '/download', expected=403)
    owner.call('POST', path + '/permissions', {'subjectType': 'user', 'subjectId': viewer_user['id'], 'permission': 'viewer'}, expected=201)
    assert viewer.call('GET', path + '/download', raw=True) == original
    if ext == 'md':
        owner.call('PUT', path + '/markdown', {'content': changed.decode()})
        viewer.call('PUT', path + '/markdown', {'content': 'forbidden'}, expected=403)
    else:
        owner.call('PUT', path + '/office/content', changed, 'application/octet-stream')
        viewer.call('PUT', path + '/office/content', changed, 'application/octet-stream', expected=403)
        session = owner.call('GET', path + '/office/session')
        token = urllib.parse.parse_qs(urllib.parse.urlsplit(session['editorUrl']).query)['access_token'][0]
        wopi = '/wopi/files/' + doc['id'] + '/contents?access_token=' + urllib.parse.quote(token)
        assert anonymous.call('GET', wopi, raw=True) == changed
    assert owner.call('GET', path + '/download', raw=True) == changed
    assert owner.call('GET', path + '/versions/' + first_version + '/download', raw=True) == original
    owner.call('POST', path + '/versions/' + first_version + '/restore', expected=201)
    assert owner.call('GET', path + '/download', raw=True) == original
    share = owner.call('POST', path + '/share-links', {'permission': 'viewer'}, expected=201)
    assert anonymous.call('GET', '/api/share/' + share['token'] + '/download', raw=True) == original
    persisted.append((path, original))
    print(f'PASS {ext}: upload/download, permissions, save, historical download, restore, share')

png = base64.b64decode('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/l9sAAAAASUVORK5CYII=')
owner.call('PUT', '/api/auth/avatar', png, 'image/png')
assert owner.call('GET', '/api/users/' + owner_user['id'] + '/avatar', raw=True) == png
owner.call('GET', '/api/admin/storage', expected=403)
# Promote only this randomly created account inside the disposable database.
subprocess.run(COMPOSE + ['exec', '-T', 'postgres', 'psql', '-U', 'docs', '-d', 'docs', '-c', "UPDATE users SET is_admin=true WHERE id='" + owner_user['id'] + "'"], check=True, stdout=subprocess.DEVNULL)
summary = owner.call('GET', '/api/admin/storage')
assert summary['usage']['objectCount'] >= 10
obj = summary['items'][0]
query = '?key=' + urllib.parse.quote(obj['key'], safe='')
assert len(owner.call('GET', '/api/admin/storage/object' + query, raw=True)) == obj['sizeBytes']
print('PASS avatar, admin authorization, usage/list/download')
subprocess.run(COMPOSE + ['restart', 'rustfs', 'backend'], check=True)
ready()
for path, original in persisted:
    assert owner.call('GET', path + '/download', raw=True) == original
    assert len(owner.call('GET', path + '/versions')['items']) == 3
assert owner.call('GET', '/api/users/' + owner_user['id'] + '/avatar', raw=True) == png
print('PASS persistence after RustFS/backend restart')
owner.call('DELETE', '/api/admin/storage/object' + query)
assert all(item['key'] != obj['key'] for item in owner.call('GET', '/api/admin/storage')['items'])
for path, _ in persisted:
    owner.call('DELETE', path)
    # Permission lookup excludes soft-deleted documents, so access is denied.
    owner.call('GET', path + '/download', expected=403)
    assert all(item['id'] != path.rsplit('/', 1)[1] for item in owner.call('GET', '/api/documents')['items'])
print('PASS admin object deletion and document soft deletion')
