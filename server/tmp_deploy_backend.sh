#!/bin/bash
# wxx 后端热部署：DB 备份 -> 备份旧二进制 -> 替换 -> 重启 -> 健康检查 -> 失败回滚
set -uo pipefail
NEW=/tmp/wxx-server-new
LIVE=/opt/wxx/wxx-server
TS=$(date +%Y%m%d-%H%M%S)
BACKUP=/opt/wxx/wxx-server.rollback-$TS

echo "TARGET_NEW=$NEW"
echo "TS=$TS"

# 1. 新二进制校验
if [ ! -s "$NEW" ]; then echo "FATAL: new binary missing"; exit 1; fi
chmod +x "$NEW"

# 2. DB 备份
DB_PW=$(grep '^DB_PASSWORD=' /etc/wxx/env | cut -d= -f2-)
DB_NAME=$(grep '^DB_NAME=' /etc/wxx/env | cut -d= -f2-)
DB_USER=$(grep '^DB_USER=' /etc/wxx/env | cut -d= -f2-)
: "${DB_USER:=wxx}"
mkdir -p /opt/wxx/backup
mysqldump --no-tablespaces -h127.0.0.1 -u"$DB_USER" -p"$DB_PW" "$DB_NAME" > "/opt/wxx/backup/wxx-pre-$TS.sql" 2>/tmp/wxx-dumperr
if [ -s "/opt/wxx/backup/wxx-pre-$TS.sql" ]; then
  echo "DB_BACKUP=/opt/wxx/backup/wxx-pre-$TS.sql"
else
  echo "DB_BACKUP_FAIL"; cat /tmp/wxx-dumperr; exit 1
fi

# 3. 备份旧二进制
cp -a "$LIVE" "$BACKUP"
echo "OLD_BACKUP=$BACKUP"

# 4. 热替换 + 重启
systemctl stop wxx
cp "$NEW" "$LIVE"
chmod +x "$LIVE"
systemctl start wxx

# 5. 健康检查（30s 内）
ok=0
for i in $(seq 1 30); do
  if systemctl is-active --quiet wxx && curl -fsS -m 4 http://localhost:8080/health >/tmp/wxx-health.json 2>/dev/null; then
    echo "SERVICE=active after ${i}s"
    head -c 400 /tmp/wxx-health.json; echo
    ok=1
    break
  fi
  sleep 1
done
if [ "$ok" != "1" ]; then
  echo "SERVICE_FAILED_ROLLBACK"
  journalctl -u wxx -n 60 --no-pager || true
  systemctl stop wxx || true
  cp "$BACKUP" "$LIVE"; chmod +x "$LIVE"
  systemctl start wxx
  exit 1
fi
rm -f /tmp/wxx-health.json
echo "DEPLOY_OK"
