#!/usr/bin/env bash
# 配置 Vercel 后端环境变量：让 Go 后端连 Turso 云数据库而非 /tmp 本地 SQLite
set -euo pipefail

cd "$(dirname "$0")/.."

VT=$(grep -E '^VERCEL_TOKEN=' .env | cut -d= -f2- | tr -d '"\r')
TURSO_URL=$(grep -E '^TURSO_DB_URL=' .env | cut -d= -f2- | tr -d '"\r')
TURSO_TOKEN=$(grep -E '^TURSO_DB_TOKEN=' .env | cut -d= -f2- | tr -d '"\r')
JWT=$(cat /tmp/wxx_jwt_secret.txt | tr -d '\n\r')

if [ -z "$TURSO_URL" ] || [ -z "$TURSO_TOKEN" ] || [ -z "$JWT" ]; then
  echo "错误：TURSO_DB_URL / TURSO_DB_TOKEN / JWT_SECRET 有缺失"
  exit 1
fi

V="npx --yes vercel@latest"

set_var() {
  local name="$1" value="$2"
  echo "── 设置 $name"
  # 先删除已有值（忽略不存在的报错），再写入
  $V env rm "$name" production --yes --token="$VT" >/dev/null 2>&1 || true
  printf '%s' "$value" | $V env add "$name" production --token="$VT" >/dev/null 2>&1
  echo "   完成"
}

set_var DB_PATH      "$TURSO_URL"
set_var SQLITE_PATH  "$TURSO_URL"
set_var TURSO_DB_URL "$TURSO_URL"
set_var TURSO_DB_TOKEN "$TURSO_TOKEN"
set_var JWT_SECRET   "$JWT"

echo ""
echo "=== 当前生产环境变量 ==="
$V env ls production --token="$VT" 2>&1 | grep -vE "EBADENGINE|npm warn"
