#!/usr/bin/env bash
# 探测各测试账号可用密码（仅用于部署验收，不打印 token）
BASE="${BASE:-https://wxx-agent.pages.dev}"
USERS="sysadmin schooladmin collegeadmin counselor_cs counselor_math stunion student_cs student_math teacher1 assistant1 student1 counselor1 counselor2 admin"
CANDS="admin123 Admin@123 admin@123 123456 wxx123456"

for u in $USERS; do
  hit=""
  for p in $CANDS; do
    code=$(curl -s -m 30 -o /tmp/li.json -w "%{http_code}" -X POST "$BASE/api/v1/auth/login" \
      -H "Content-Type: application/json" \
      -d "{\"username\":\"$u\",\"password\":\"$p\"}")
    if [ "$code" = "200" ] && grep -q '"token"' /tmp