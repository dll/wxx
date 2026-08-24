import sqlite3, os

db_path = os.path.join('data', 'wxx.db')
if not os.path.exists(db_path):
    print("NO DB:", db_path)
    raise SystemExit(1)

con = sqlite3.connect(db_path)
cur = con.cursor()
# 检查表是否存在
tabs = cur.execute("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'vopc%'").fetchall()
print("vopc tables:", [t[0] for t in tabs])

if ('vopc_projects',) in tabs:
    cols = [r[1] for r in cur.execute("PRAGMA table_info(vopc_projects)").fetchall()]
    print("vopc_projects cols:", cols)
    rows = cur.execute("SELECT id,name,stage,status FROM vopc_projects ORDER BY id").fetchall()
    print("projects (id,name,stage,status):", rows)
    # 每个项目关键字段缺失情况
    for r in rows:
        pid, name = r[0], r[1]
        row = cur.execute("SELECT summary,problem_statement,target_users,expected_outcome,validation_plan,product_form,project_cycle,acceptance_criteria FROM vopc_projects WHERE id=?", (pid,)).fetchone()
        labels = ["summary","problem","target","outcome","validation","product_form","cycle","acceptance"]
        missing = [labels[i] for i,v in enumerate(row) if not v or not (str(v).strip())]
        print(f"  - [{pid}] '{name}' missing fields: {missing}")
else:
    print("vopc_projects 表不存在")
con.close()
