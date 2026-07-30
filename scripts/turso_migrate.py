#!/usr/bin/env python3
"""Turso → 本地 SQLite 全量迁移工具（一次性使用）"""
import json, os, sqlite3, urllib.request

TURSO_URL = os.environ['T_URL'].replace('libsql://', 'https://') + '/v2/pipeline'
TURSO_TOKEN = os.environ['T_TOKEN']
DB = '/opt/wxx/data/wxx.db'

def q(sql, limit=50000):
    """从 Turso 查询数据"""
    body = json.dumps({"requests": [
        {"type": "execute", "stmt": {"sql": f"{sql} LIMIT {limit}", "args": []}},
        {"type": "close"}
    ]}).encode()
    req = urllib.request.Request(TURSO_URL, data=body, headers={
        "Authorization": f"Bearer {TURSO_TOKEN}",
        "Content-Type": "application/json"
    })
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            data = json.loads(r.read())
    except Exception as e:
        return None, []
    res = data['results'][0].get('response', {}).get('result', {})
    if not res:
        return None, []
    cols = [c['name'] for c in res.get('cols', [])]
    rows = []
    for row in res.get('rows', []):
        r = {}
        for i, cell in enumerate(row):
            if cell is None or (isinstance(cell, dict) and cell.get('type') == 'null'):
                r[cols[i]] = None
            elif isinstance(cell, dict):
                t, v = cell.get('type'), cell.get('value')
                r[cols[i]] = int(v) if t=='integer' else (float(v) if t=='float' else v)
            else:
                r[cols[i]] = cell
        rows.append(r)
    return cols, rows

def main():
    conn = sqlite3.connect(DB)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA foreign_keys=OFF")

    # 跳过 FTS5 虚拟表（本地 SQLite 已有，靠 trigger 自动填充）
    FTS_SKIP = {'kb_fts','kb_fts_data','kb_fts_idx','kb_fts_content',
                'kb_fts_docsize','kb_fts_config','_migrations'}

    local_tables = [r[0] for r in conn.execute(
        "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name"
    ).fetchall() if r[0] not in FTS_SKIP and not r[0].startswith('sqlite_')]

    print(f"本地 SQLite 共 {len(local_tables)} 张表，开始迁移…\n")
    total = 0

    # 优先迁移关键表
    priority = ['users','user_consents','kb_resources','sessions','messages',
                'model_configs','app_versions','emotion_records','feedback',
                'audit_logs','agent_configs']
    ordered = [t for t in priority if t in local_tables] + \
              [t for t in local_tables if t not in priority]

    for table in ordered:
        existing = conn.execute(f"SELECT COUNT(*) FROM [{table}]").fetchone()[0]
        if existing > 0:
            print(f"  {table:<40} skip  ({existing} 已有)")
            continue
        cols, rows = q(f"SELECT * FROM [{table}]")
        if not rows:
            print(f"  {table:<40} empty (Turso 无数据)")
            continue
        ph = ','.join(['?' for _ in cols])
        cn = ','.join([f'[{c}]' for c in cols])
        inserted = 0
        try:
            for row in rows:
                conn.execute(f"INSERT OR IGNORE INTO [{table}] ({cn}) VALUES ({ph})",
                             [row.get(c) for c in cols])
                inserted += 1
            conn.commit()
            print(f"  {table:<40} ✓ {inserted} 条")
            total += inserted
        except Exception as e:
            conn.rollback()
            print(f"  {table:<40} ✗ {e}")

    conn.execute("PRAGMA foreign_keys=ON")
    conn.close()
    print(f"\n迁移完成：共 {total} 条记录")

    # 重建 FTS5 全文索引
    print("重建 FTS5 索引（知识库搜索用）…")
    conn2 = sqlite3.connect(DB)
    try:
        conn2.execute("INSERT INTO kb_fts(kb_fts) VALUES('rebuild')")
        conn2.commit()
        print("FTS5 索引重建完成 ✓")
    except Exception as e:
        print(f"FTS5 重建跳过: {e}")
    finally:
        conn2.close()

if __name__ == '__main__':
    main()
