#!/usr/bin/env python3
"""Turso 查询helper：从 .env 读凭据，执行 SQL 并打印结果。

用法：
    python scripts/turso_query.py "SELECT COUNT(*) FROM users"
    python scripts/turso_query.py -f query.sql
"""
import json
import os
import sys
import urllib.request


def load_env(path=".env"):
    env = {}
    with open(path, encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, v = line.split("=", 1)
            env[k.strip()] = v.strip().strip('"').strip("'")
    return env


def query(sql, args=None):
    env = load_env()
    url = env["TURSO_DB_URL"].replace("libsql://", "https://") + "/v2/pipeline"
    token = env["TURSO_DB_TOKEN"]

    def encode(a):
        if a is None:
            return {"type": "null", "value": None}
        if isinstance(a, bool):
            return {"type": "integer", "value": "1" if a else "0"}
        if isinstance(a, int):
            return {"type": "integer", "value": str(a)}
        if isinstance(a, float):
            return {"type": "float", "value": a}
        return {"type": "text", "value": str(a)}

    body = {
        "requests": [
            {"type": "execute", "stmt": {"sql": sql, "args": [encode(a) for a in (args or [])]}},
            {"type": "close"},
        ]
    }
    req = urllib.request.Request(
        url,
        data=json.dumps(body).encode(),
        headers={"Authorization": "Bearer " + token, "Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=60) as resp:
        data = json.loads(resp.read())

    first = data["results"][0]
    if first.get("type") == "error":
        raise SystemExit("SQL 错误: " + json.dumps(first.get("error"), ensure_ascii=False))
    res = first["response"]["result"]
    cols = [c["name"] for c in res["cols"]]
    rows = [[(c or {}).get("value") for c in row] for row in res["rows"]]
    return cols, rows


def main():
    if len(sys.argv) >= 3 and sys.argv[1] == "-f":
        sql = open(sys.argv[2], encoding="utf-8").read()
    elif len(sys.argv) >= 2:
        sql = sys.argv[1]
    else:
        raise SystemExit(__doc__)

    cols, rows = query(sql)
    print(" | ".join(cols))
    print("-" * 60)
    for r in rows:
        print(" | ".join("" if v is None else str(v) for v in r))
    print("-" * 60)
    print(f"{len(rows)} 行")


if __name__ == "__main__":
    main()
