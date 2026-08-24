#!/bin/bash
# 前端 Web 部署：备份 -> 解压替换 -> 校验 -> 重载 caddy
set -euo pipefail
TS=$(date +%Y%m%d-%H%M%S)
WEB=/opt/wxx/frontend/web
STAGE=/opt/wxx/frontend/web.new-$TS
BACKUP=/opt/wxx/frontend/web.bak-$TS
TAR=/tmp/wxx-web-deploy.tar.gz

echo "TS=$TS"
test -s "$TAR" || { echo "TAR_MISSING"; exit 1; }

rm -rf "$STAGE"
mkdir -p "$STAGE"
tar -xzf "$TAR" -C "$STAGE"
test -f "$STAGE/main.dart.js" || { echo "NO_MAIN_DART"; exit 1; }
echo "main.dart.js size: $(stat -c '%s' "$STAGE/main.dart.js")"
grep -a -q 'vOPC' "$STAGE/main.dart.js" && echo "HAS_VOPC_REF" || echo "WARN_NO_VOPC_REF"

# 保留 downloads 目录（如有）
[ -d "$WEB/downloads" ] && cp -a "$WEB/downloads" "$STAGE/downloads" 2>/dev/null || true
chown -R caddy:caddy "$STAGE"

# 备份 + 原子替换
cp -a "$WEB" "$BACKUP" 2>/dev/null || true
mv "$WEB" "$WEB.old-$TS"
mv "$STAGE" "$WEB"
rm -rf "$WEB.old-$TS"
echo "DEPLOYED_MAIN=$(stat -c '%y %s' "$WEB/main.dart.js")"
echo "BACKUP=$BACKUP"
systemctl reload caddy 2>/dev/null || systemctl restart caddy || true
echo "CADDY_RELOADED"
rm -f "$TAR"
