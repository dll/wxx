import json, urllib.request

BASE = 'https://api.pydaydayup.xyz/api/v1'
USERS = [
    ('student_cs', 'student'),
    ('counselor_cs', 'counselor'),
    ('teacher1', 'teacher'),
    ('assistant1', 'assistant'),
    ('stunion', 'student_union'),
    ('collegeadmin', 'college_admin'),
    ('schooladmin', 'school_admin'),
    ('sysadmin', 'sys_admin'),
]
for u, r in USERS:
    body = json.dumps({'username': u, 'password': '', 'role': r}).encode()
    req = urllib.request.Request(
        f'{BASE}/auth/login', data=body,
        headers={'Content-Type': 'application/json'},
    )
    try:
        d = json.loads(urllib.request.urlopen(req, timeout=15).read())
        token = d['data']['token']
    except Exception as e:
        print(f'{u} login fail: {e}')
        continue
    req2 = urllib.request.Request(
        f'{BASE}/user/capabilities',
        headers={'Authorization': f'Bearer {token}'},
    )
    try:
        d2 = json.loads(urllib.request.urlopen(req2, timeout=15).read())
        caps = d2['data']['capabilities']
        print(f'{u:15s} role={d2["data"]["role"]:14s} caps={len(caps):3d}')
    except Exception as e:
        print(f'{u} caps fail: {e}')
