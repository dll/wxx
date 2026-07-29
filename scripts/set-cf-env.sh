#!/usr/bin/env bash
# 配置 Cloudflare Pages 环境变量：后端地址 + 与 Go 后端一致的 JWT 密钥 + Turso 凭据
set -euo pipefail

cd "$(dirname "$0")/.."

export CLOUDFLARE_API_TOKEN=$(grep -E '^CLOUDFLARE_API_TOKEN=' .env | cut -d= -f2- | tr -d '"\r')
export CLOUDFLARE_ACCOUNT_ID=$(grep -E '^CLOUDFLARE_ACCOUNT_ID=' .env | cut -d= -f2- | tr -d '"\r')

TURSO_URL=$(grep -E '^TURSO_DB_URL=' .env | cut -d= -f2- | tr -d '"\r')
TURSO_TOKEN=$(grep -E '^TURSO_DB_TOKEN=' .env | cut -d= -f2- | tr -d '"\r')
JWT=$(grep -E '^JWT_SECRET=' .env | cut -d= -f2- | tr -d '"\r')
BACKEND="${1:-https://wxx-server-czldl.vercel.app}"

if [ -z "$JWT" ]; then
  echo "错误：.env 中缺少 JWT_SECRET，请先配置后再运行"
  exit 1
fi

W="npx --yes wrangler@4"

put_secret() {
  local name="$1" value="$2"
  echo "── 设置 $name"
  printf '%s' "$value" | $W pages secret put "$name" --project-name wxx-agent >/dev/null 2>&1
  echo "   完成"
}

put_secret GO_BACKEND_URL "$BACKEND"
put_secret JWT_SECRET     "$JWT"
put_secret TURSO_DB_URL   "$TURSO_URL"
put_secret TURSO_DB_TOKEN "$TURSO_TOKEN"

echo ""
echo "=== 当前 CF Pages 生产环境密钥 ==="
$W pages secret list --project-name wxx-agent 2>&1 | tail -10
