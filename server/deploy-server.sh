#!/bin/bash
# ═══════════════════════════════════════════════════════════════════
# 蔚小芯后端部署脚本（腾讯云 Lighthouse）
# 用法：在服务器上运行 ./deploy-server.sh
# ═══════════════════════════════════════════════════════════════════

set -euo pipefail

echo "=== 蔚小芯服务器初始化 ==="

# ── 1. 系统依赖 ─────────────────────────────────────────────────
echo ""
echo "[1/7] 安装系统依赖..."
apt-get update -qq
apt-get install -y -qq curl build-essential git
echo "✅ 系统依赖安装完成"

# ── 2. Go 编译环境 ──────────────────────────────────────────────
echo ""
echo "[2/7] 检查 Go 编译环境..."
if ! command -v go &>/dev/null; then
    echo "安装 Go 1.22..."
    wget -q https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    export PATH=$PATH:/usr/local/go/bin
    rm -f go1.22.5.linux-amd64.tar.gz
fi
echo "Go 版本: $(go version)"

# ── 3. Caddy ────────────────────────────────────────────────────
echo ""
echo "[3/7] 安装 Caddy..."
if ! command -v caddy &>/dev/null; then
    apt install -y -qq debian-keyring debian-archive-keyring
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | \
        gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | \
        tee /etc/apt/sources.list.d/caddy-stable.list
    apt update -qq && apt install -y -qq caddy
fi
echo "✅ Caddy: $(caddy version | head -1)"

# ── 4. 目录结构 ─────────────────────────────────────────────────
echo ""
echo "[4/7] 创建目录结构..."
mkdir -p /opt/wxx/frontend/web
mkdir -p /opt/wxx/data/backup
echo "✅ 目录已创建"

# ── 5. 编译 Go 后端 ─────────────────────────────────────────────
echo ""
echo "[5/7] 编译 Go 后端..."
cd /opt/wxx
export GOPROXY=https://goproxy.cn,direct
go build -tags fts5 -o /opt/wxx/wxx-server ./server/cmd/server
echo "✅ 编译完成: $(file /opt/wxx/wxx-server)"

# ── 6. systemd 服务 ─────────────────────────────────────────────
echo ""
echo "[6/7] 配置 systemd 服务..."
cat > /etc/systemd/system/wxx.service << 'WXXUNIT'
[Unit]
Description=蔚小芯 Go 后端
After=network.target

[Service]
Type=simple
EnvironmentFile=/etc/wxx/env
ExecStart=/opt/wxx/wxx-server
Restart=always
RestartSec=5
WorkingDirectory=/opt/wxx

[Install]
WantedBy=multi-user.target
WXXUNIT
systemctl daemon-reload
systemctl enable wxx
echo "✅ systemd 服务已配置"

# ── 7. 环境变量模板 ─────────────────────────────────────────────
echo ""
echo "[7/7] 环境变量模板..."
if [ ! -f /etc/wxx/env ]; then
    cat > /etc/wxx/env << 'WXXENV'
APP_MODE=production
APP_PORT=8080
SQLITE_PATH=/opt/wxx/data/wxx.db
JWT_SECRET=<请填写64位随机字符串>
DEEPSEEK_API_KEY=<请填写>
ZHIPU_API_KEY=<请填写>
CORS_ALLOWED_ORIGINS=https://wxx-agent.online,https://www.wxx-agent.online,https://wxx-agent.pages.dev
WXXENV
    echo "⚠️  请编辑 /etc/wxx/env 填写实际的密钥和配置"
fi
echo "✅ 环境变量模板已创建"

# ── 完成 ────────────────────────────────────────────────────────
echo ""
echo "═════════════════════════════════════════════════════════════"
echo "服务器初始化完成。后续步骤："
echo "  1. 编辑 /etc/wxx/env 填写密钥"
echo "  2. 复制 Caddyfile 到 /etc/caddy/Caddyfile"
echo "  3. 上传前端静态文件到 /opt/wxx/frontend/web/"
echo "  4. systemctl start wxx && systemctl start caddy"
echo "═════════════════════════════════════════════════════════════"
