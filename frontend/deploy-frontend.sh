#!/bin/bash
# ═══════════════════════════════════════════════════════════════════
# 蔚小芯前端部署脚本（腾讯云 Lighthouse 集中部署）
# 用法：./deploy-frontend.sh
# 假设：服务器已安装 Caddy，Flutter SDK 已在本机安装
# ═══════════════════════════════════════════════════════════════════

set -euo pipefail

# ── 配置 ──────────────────────────────────────────────────────────
SERVER="root@129.211.223.113"
REMOTE_DIR="/opt/wxx/frontend"
LOCAL_BUILD_DIR="build/web"

echo "=== 蔚小芯前端部署 ==="
echo "目标: $SERVER:$REMOTE_DIR"

# ── 1. 构建 Flutter Web ─────────────────────────────────────────
echo ""
echo "[1/4] 构建 Flutter Web..."

# 读取百度地图 AK (如果 .env 存在)
BAIDU_AK=""
if [ -f ../.env ]; then
    BAIDU_AK=$(grep "^BAIDU_MAP_AK=" ../.env | cut -d= -f2- | tr -d '"'"'"'\r' | tr -d "'")
fi

FLUTTER_ARGS="--release --web-renderer canvaskit"
if [ -n "$BAIDU_AK" ]; then
    FLUTTER_ARGS="$FLUTTER_ARGS --dart-define=BAIDU_MAP_AK=$BAIDU_AK"
fi

flutter build web $FLUTTER_ARGS
echo "✅ 构建完成"

# ── 2. 准备部署文件 ─────────────────────────────────────────────
echo ""
echo "[2/4] 准备部署文件..."

# 确保 _headers 在正确位置（CORS 头）
cp -f web/_headers build/web/_headers 2>/dev/null || true

# 删除不需要的大文件
rm -f build/web/_routes.json
rm -rf build/web/downloads/ 2>/dev/null || true

echo "✅ 准备完成"

# ── 3. 上传到服务器 ─────────────────────────────────────────────
echo ""
echo "[3/4] 上传到 $SERVER..."
rsync -avz --delete \
    --exclude=".git" \
    --exclude="node_modules" \
    --exclude="downloads" \
    build/web/ \
    "$SERVER:$REMOTE_DIR/web/"
echo "✅ 上传完成"

# ── 4. 设置权限并重载 Caddy ────────────────────────────────────
echo ""
echo "[4/4] 重载 Caddy..."
ssh "$SERVER" "chown -R caddy:caddy $REMOTE_DIR && systemctl reload caddy"
echo "✅ Caddy 重载完成"

echo ""
echo "═════════════════════════════════════════════════════════════"
echo "✅ 部署成功！"
echo "访问: https://www.wxx-agent.online"
echo "═════════════════════════════════════════════════════════════"
