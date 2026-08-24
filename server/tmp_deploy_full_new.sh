#!/bin/bash
# 合入新源码 -> 编译 fts5 -> 热部署（DB备份+旧二进制备份+替换+重启+健康检查+回滚）
set -uo pipefail
export GOPROXY=https://goproxy.cn,direct

echo "=== 1. 合入新源码 ==="
rm -f /tmp/wxx-src2/kbfetch.exe /tmp/wxx-src2/tmp_*.sh /tmp/wxx-src2/tmp_*.py
cp -a /tmp/wxx-src2/. /opt/wxx/server/
echo "源码已合入: $(grep -c SearchUsers /opt/wxx/server/internal/handler/vopc_handler.go) 处 SearchUsers"

echo "=== 2. 编译 fts5 ==="
cd /opt/wxx
go build -tags fts5 -o /tmp/wxx-server-new2 ./server/cmd/server 2>&1 | head -40
if [ ! -s /tmp/wxx-server-new2 ]; then echo "COMPILE_FAIL"; exit 1; fi
echo "编译成功: $(stat -c '%s' /tmp/wxx-server-new2) bytes"

echo "=== 3. 热部署 ==="
NEW=/tmp/wxx-server-new2
LIVE=/opt/wxx/wxx-server
TS=$(date +%Y%m%d-%H%M%S)
BACKUP=/opt/wxx/wxx-server.rollback-$TS
chmod +x "$NEW"

DB_PW=$(grep '^DB_PASSWORD=' /etc/wxx/env | cut -d= -f2-)
DB_NAME=$(grep '^DB_NAME=' /etc/wxx/env | cut -d= -f2-)
DB_USER=$(grep '^DB_USER=' /etc/wxx/env | cut -d= -f2-)
: "${DB_USER:=wxx}"
mkdir -p /opt/wxx/backup
mysqldump --no-tablespaces -h127.0.0.1 -u"$DB_USER" -p"$DB_PW" "$DB_NAME" > "/opt/wxx/backup/wxx-pre-$TS.sql" 2>/dev/null
if [ ! -s "/opt/wxx/backup/wxx-pre-$TS.sql" ]; then echo "DB_BACKUP_FAIL"; exit 1; fi
echo "DB_BACKUP=/opt/wxx/backup/wxx-pre-$TS.sql"

cp -a "$LIVE" "$BACKUP"
systemctl stop wxx
cp "$NEW" "$LIVE"; chmod +x "$LIVE"
systemctl start wxx

ok=0
for i in $(seq 1 30); do
  if systemctl is-active --quiet wxx && curl -fsS -m 4 http://localhost:8080/health >/tmp/h2.json 2>/dev/null; then
    echo "SERVICE=active after ${i}s: $(head -c 200 /tmp/h2.json)"
    ok=1; break
  fi
  sleep 1
done
if [ "$ok" != "1" ]; then echo "ROLLBACK"; journalctl -u wxx -n 40 | tail -30; systemctl stop wxx||true; cp "$BACKUP" "$LIVE"; chmod +x "$LIVE"; systemctl start wxx; exit 1; fi
echo "DEPLOY_OK"
