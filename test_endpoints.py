import json, urllib.request, urllib.error

BASE = 'https://api.pydaydayup.xyz/api/v1'

def login(username, role):
    body = json.dumps({'username': username, 'password': '', 'role': role}).encode()
    req = urllib.request.Request(f'{BASE}/auth/login', data=body, headers={'Content-Type': 'application/json'})
    return json.loads(urllib.request.urlopen(req, timeout=15).read())['data']['token']

def call(token, path):
    req = urllib.request.Request(f'{BASE}{path}', headers={'Authorization': f'Bearer {token}'})
    try:
        urllib.request.urlopen(req, timeout=15)
        return 200
    except urllib.error.HTTPError as e:
        return e.code

ROLES = [
    ('student_cs', 'student'),
    ('counselor_cs', 'counselor'),
    ('teacher1', 'teacher'),
    ('assistant1', 'assistant'),
    ('stunion', 'student_union'),
    ('collegeadmin', 'college_admin'),
    ('sysadmin', 'sys_admin'),
]

# 测试矩阵：路径 → (期望可访问的角色简称)
TESTS = [
    ('/student/daily-briefing', '所有角色（self.briefing.read）'),
    ('/me/daily-briefing', '所有角色（个人入口别名）'),
    ('/counselor/daily-focus', 'counselor 及 college_admin/sys_admin'),
    ('/teacher/daily-overview', 'teacher 及 college_admin/sys_admin'),
    ('/assistant/schedule-check', 'assistant 及 college_admin/sys_admin'),
    ('/college/twin-screen', '仅 college_admin/school_admin/sys_admin'),
    ('/admin/metrics', '仅 college_admin/school_admin/sys_admin'),
    ('/admin/settings', '仅 sys_admin'),
]

# 收集 tokens
tokens = {}
for u, r in ROLES:
    try:
        tokens[r] = login(u, r)
    except Exception as e:
        print(f'登录失败 {u}: {e}')

# 输出矩阵
header = ['endpoint'] + [r[:8] for _, r in ROLES]
print(f"{'endpoint':32s} | " + ' | '.join(f'{c:13s}' for c in header[1:]))
print('-' * 150)
for path, _desc in TESTS:
    row = [path]
    for u, r in ROLES:
        if r in tokens:
            code = call(tokens[r], path)
            row.append('✓' if code == 200 else f'✗{code}')
        else:
            row.append('?')
    print(f"{row[0]:32s} | " + ' | '.join(f'{c:13s}' for c in row[1:]))
