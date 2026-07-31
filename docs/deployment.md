# 部署指南 — 蔚小芯

> 蔚小芯当前正式部署架构（2026-07-31 起）：
> - **前端**：Cloudflare Pages（项目 `wxx-agent`），正式入口 `https://www.wxx-agent.online`（CF Pages 自定义域名），备用入口 `https://wxx-agent.pages.dev`
> - **后端**：腾讯云 Lighthouse（`129.211.223.113`，Ubuntu 22.04），Go 二进制 systemd 常驻，Caddy 反向代理 + 自动 HTTPS（`https://www.wxx-agent.online`）
> - **数据库**：服务器本地 SQLite（`/opt/wxx/data/wxx.db`，含 FTS5）
> - **代理链路**：用户 → CF Pages（前端）→ CF Functions（JWT 鉴权）→ `http://129.211.223.113:8080`（Go 后端）→ 本地 SQLite
>
> 历史方案（Vercel Serverless + Turso）已于 2026-07-31 停用，仅作归档参考，见文末「历史方案」。

## 正式访问地址

| 地址 | 用途 | 状态 |
|------|------|------|
| `https://www.wxx-agent.online` | 用户正式入口（CF Pages 自定义域名） | 需在 CF Pages 绑定自定义域名后生效 |
| `https://wxx-agent.pages.dev` | 备用入口（CF Pages 原生域名） | 始终有效 |
| `https://wxx-agent.pages.dev/downloads/` | Android APK 下载 | 始终有效 |
| `http://129.211.223.113:8080` | 后端 API（CF Functions 内部代理用，用户不直接访问） | 内部 |

> 登录：用户名 + 密码（由管理员分配）。角色与权限见 `docs/蔚小芯角色功能.md`。

## 部署模式对比

| 特性 | 腾讯云 Lighthouse + SQLite（**当前正式**） | Vercel + Turso（历史/已停用） |
|------|-------------------------------------------|------------------------------|
| 前端托管 | Cloudflare Pages（国内加速） | Cloudflare Pages |
| 后端运行 | systemd 常驻（无冷启动） | Vercel Serverless（有冷启动） |
| 数据持久化 | 本地 SQLite `/opt/wxx/data/wxx.db` | Turso 云端 |
| SQLite 延迟 | ~58µs（本地文件） | ~50ms（网络往返） |
| FTS5 全文检索 | ✅ `ok` | ⚠️ Turso 上 `unavailable` |
| LLM 长请求 | ✅ 无超时限制 | ⚠️ Serverless 超时截断 |
| 运维成本 | 中（需维护服务器，约 ¥150–220/月） | 低（免运维，按量计费） |
| 适用场景 | **当前正式方案** | 已停用 |

## 腾讯云 Lighthouse + SQLite 部署（当前正式方案）

详细的服务器迁移与部署过程见 `docs/蔚小芯-后端迁移常驻服务器方案.md` 与 `docs/蔚小芯Fable5审核和开发计划与实现v4.md` §1。核心步骤概览：

```bash
# 1. 服务器初始化（Ubuntu 22.04）
apt-get update && apt-get install -y git curl build-essential
# 安装 Go 1.22+
wget -q https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 2. 上传源码并编译（GOPROXY 走国内代理避免超时）
export GOPROXY=https://goproxy.cn,direct
cd /opt/wxx && go build -tags fts5 -o /opt/wxx/wxx-server ./server/cmd/server

# 3. 环境变量写入 /etc/wxx/env（APP_MODE、APP_PORT=8080、SQLITE_PATH、JWT_SECRET≥32位、各 LLM 密钥、CORS_ALLOWED_ORIGINS）
# 4. systemd 服务 /etc/systemd/system/wxx.service（EnvironmentFile=/etc/wxx/env，Restart=always）
systemctl enable --now wxx

# 5. Caddy 自动 HTTPS（/etc/caddy/Caddyfile）
#    www.wxx-agent.online {
#        reverse_proxy localhost:8080
#    }
systemctl enable --now caddy

# 6. Lighthouse 防火墙放通 TCP 22 / 80 / 443 / 8080
# 7. 健康检查
curl -s http://localhost:8080/health   # status=healthy, fts5=ok, sqlite=ok
```

关键前置条件：

| 条件 | 说明 |
|------|------|
| 服务器 | 腾讯云 Lighthouse Ubuntu 22.04，≥2C2G，公网 IP |
| 域名 | `www.wxx-agent.online`，A 记录指向服务器 IP（或作 CF Pages 自定义域名） |
| 防火墙 | TCP 22（SSH）、80（Let's Encrypt）、443（HTTPS）、8080（CF 代理） |
| ICP 备案 | 大陆直连 `www.wxx-agent.online:443` 需备案；经 CF 边缘访问不需要 |
| GOPROXY | `https://goproxy.cn,direct`（大陆构建避免 GitHub/golang.org 超时） |

## 备份（重要）

SQLite 是唯一数据存储，须定期备份。写入 cron：

```bash
mkdir -p /opt/wxx/backup
# 每日 03:00 备份
echo '0 3 * * * root sqlite3 /opt/wxx/data/wxx.db ".backup /opt/wxx/backup/wxx-$(date +\%F).db" && find /opt/wxx/backup -name "wxx-*.db" -mtime +14 -delete' > /etc/cron.d/wxx-backup
```

## 历史方案（Vercel + Turso，已停用，仅归档）

> 以下内容为 2026-07-31 前的部署方案，已被腾讯云 Lighthouse 方案取代，保留仅供归档参考。

### 1. 创建 Turso 云数据库

```bash
# 安装 Turso CLI
curl -sSfL https://get.tur.so/install.sh | bash

# 登录
turso auth login

# 创建数据库
turso db create wxx-agent

# 获取连接 URL
turso db show wxx-agent --url
# 输出: libsql://wxx-agent-<your-org>.turso.io

# 创建 Auth Token
turso db tokens create wxx-agent
# 输出: eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...
```

### 2. 初始化数据库 Schema

```bash
# 设置环境变量
export DB_PATH="libsql://wxx-agent-<your-org>.turso.io?authToken=<your-token>"

# 执行数据库迁移
cd server && go run cmd/migrate/main.go
```

### 3. 配置 Vercel 环境变量

在 Vercel 项目设置 → Environment Variables 中添加：

| 变量名 | 值 | 说明 |
|--------|-----|------|
| `DB_PATH` | `libsql://wxx-agent-xxx.turso.io?authToken=xxx` | Turso 数据库连接串 |
| `JWT_SECRET` | `<你的密钥>` | JWT 签名密钥（至少32字符） |
| `ZHIPU_API_KEY` | `<你的密钥>` | 智谱 API Key |
| `DEEPSEEK_API_KEY` | `<你的密钥>` | DeepSeek API Key |
| `APP_MODE` | `release` | 生产模式 |
| `CORS_ALLOWED_ORIGINS` | `https://wxx-agent.pages.dev` | 允许的前端域名 |

### 4. 部署后端到 Vercel

```bash
# 安装 Vercel CLI
npm i -g vercel

# 在项目根目录部署
vercel --prod
```

### 5. 验证部署

```bash
# 检查健康状态
curl -s https://wxx-server.vercel.app/api/v1/health
# 应返回 status=ok, sqlite=ok, fts5=ok

# 检查知识库数据
curl -s https://wxx-server.vercel.app/api/v1/kb/stats
# 应返回 44 条种子数据
```

### 技术实现说明

后端通过 DSN 协议自动选择数据库驱动：
- `libsql://` 开头 → 使用 Turso 云数据库（`libsql-client-go` 驱动）
- 其他 → 使用本地 SQLite 文件（`modernc.org/sqlite` 驱动）

代码位置：`server/pkg/app/app.go` → `initDB()` 函数

---

## 应用命名规范（强制）

> 所有面向用户的入口名称统一为「蔚小芯」，包括：

| 平面 | 字段 | 值 |
|------|------|-----|
| Android | `android:label` (`AndroidManifest.xml`) | 蔚小芯 |
| Android | APK 输出（Gradle `outputFileName`） | `蔚小芯-release.apk` / `蔚小芯-debug.apk` |
| Android | 最终分发文件（Web 下载） | `蔚小芯-v版本号.apk` |
| Web | `<title>` (`web/index.html`) | 蔚小芯 |
| Web | `meta description` | 蔚小芯 — 滁州学院计算机学院智慧学工 AI 助手 |
| Web | `apple-mobile-web-app-title` | 蔚小芯 |
| Web | `manifest.json` `name` / `short_name` | 蔚小芯 |
| Web | `manifest.json` `description` | 蔚小芯 — 滁州学院计算机学院智慧学工 AI 助手 |
| 后端 | `/health` 返回 `service` 字段 | 蔚小芯 |

技术 ID（不暴露给用户）保留英文：
- Flutter 包名：`wxx_app`（`pubspec.yaml`）
- Android applicationId：`com.wxx.wxx_app`
- 仓库名：`wxx`
- 后端二进制：`wxx-server`

## 构建与产物规范（强制）

### Flutter Web

```bash
cd frontend && flutter build web --release
```

产物：`frontend/build/web/`。验证标题是否为「蔚小芯」：

```bash
grep -c "蔚小芯" frontend/build/web/index.html  # 期望 ≥ 3
grep -c "蔚小芯" frontend/build/web/manifest.json  # 期望 ≥ 3
```

### Android APK

```bash
make flutter-build-apk        # 路径含 ASCII，零中文
make flutter-build-apk-safe   # 路径含中文（推荐，复制到 ASCII 临时目录）
```

执行后产物：

| 路径 | 用途 |
|------|------|
| `frontend/build/app/outputs/apk/release/蔚小芯-release.apk` | Gradle 直出（中文名由 `app/build.gradle.kts` 的 `outputFileName` 控制） |
| `frontend/build/app/outputs/flutter-apk/app-release.apk` | Flutter SDK 内部固定名（保留备份） |
| `frontend/build/app/outputs/flutter-apk/weixiaoxin-release.apk` | ASCII 备份文件，便于脚本处理 |
| `frontend/build/web/downloads/蔚小芯-v版本号.apk` | **Web 对外分发文件**，首页二维码指向此路径 |
| `frontend/build/web/downloads/release.json` | Web 发布清单 |

首发版本：`蔚小芯-v0.0.1.apk`，版本号 `0.0.1+1`，发布日期 `2026-07-20`。

发布构建命令：

```bash
make deploy-release
```

该 target 会自动递增 `frontend/pubspec.yaml` 的 patch 版本和 build number，例如 `0.0.1+1` -> `0.0.2+2`，同步更新 `frontend/lib/config/release_config.dart`，构建 Web + APK，将 APK 注入 `build/web/downloads/`，再部署网站。

如果只是重建当前版本（首发或排查构建问题），使用：

```bash
pwsh -ExecutionPolicy Bypass -NoProfile -File scripts/build-all.ps1 -NoVersionBump
```

`-NoVersionBump` 不用于常规发布；常规发布必须使用 `make deploy-release` 自动递增版本。

## Cloudflare Pages 前端部署（强制流程）

> 前端正式入口：`https://www.wxx-agent.online`（CF Pages 自定义域名），备用入口 `https://wxx-agent.pages.dev`。后端由 Cloudflare Pages Functions 鉴权后代理至腾讯云 Lighthouse（`129.211.223.113:8080`）。Vercel 前端旧域名与 Vercel 后端均已停用。

### 标准部署命令（推荐）

```bash
make deploy-web
```

该 target 会：
1. 重新构建 `frontend/build/web/`
2. 同步 `frontend/functions/` 到部署目录
3. 通过 `wrangler pages deploy` 部署到 Cloudflare Pages `wxx-agent` 项目

Web + APK 联合发布请使用：

```bash
make deploy-release
```

### 绑定自定义域名

生产环境已绑定 `www.wxx-agent.online` 作为 CF Pages 自定义域名（由 Cloudflare 边缘提供，全球可达，无需 ICP 备案即可访问前端）。后续可再绑定学校官方子域名（如 `wxx-agent.chzu.edu.cn`）。

> 注意：`www.wxx-agent.online` 作为**前端入口**时应指向 Cloudflare（CNAME → `wxx-agent.pages.dev`）；而后端 API 经 Caddy 使用同名 HTTPS 时指向服务器 IP。二者不能共用同一 DNS 记录——若前端用此域名，后端另用子域名（如 `api.wxx-agent.online`）或直连 IP。

```bash
# 1. 打开 Cloudflare Dashboard → Workers & Pages → wxx-agent
# 2. 进入 Custom domains 选项卡
# 3. 点击 Set up custom domain
# 4. 输入域名（例如 wxx-agent.chzu.edu.cn），点击 Continue
# 5. Cloudflare 自动检测 DNS 记录：
#    - 如果域名在 Cloudflare 托管 → 自动添加 CNAME 到 pages.dev
#    - 如果域名不在 Cloudflare 托管 → 显示 DNS 目标 (CNAME)，需在 DNS 提供商手动添加
# 6. 等待 SSL 证书自动下发（Let's Encrypt，约 1–5 分钟）
# 7. 验证域名解析
dig wxx-agent.chzu.edu.cn CNAME +short  # 应返回 wxx-agent.pages.dev
curl -I https://wxx-agent.chzu.edu.cn   # 应返回 200
```

**注意**：
- 自定义域名绑定后，`*.pages.dev` 域名仍有效，两者均可访问。
- 如果之前设置过 `CORS_ALLOWED_ORIGINS` 环境变量，需追加自定义域名。
- 如需自定义域名为唯一入口，可在 Cloudflare 添加 Page Rule 将 `*.pages.dev` 301 重定向到自定义域名。
- 自定义域名变更后需同步更新 `frontend/lib/config/api_config.dart` 中的 `baseUrl`（非 Web 端），以及 `frontend/functions/api/[[route]].js` 中的 `targetUrl`（如果走代理模式）。

### 仅推送已构建产物

```bash
make deploy-web-prebuilt
```

### 手动部署（与 Makefile 等价）

```bash
cd frontend
flutter build web --release
rm -rf deploy && mkdir -p deploy
cp -rf build/web/* deploy/
cp -rf functions deploy/
npx --yes wrangler pages deploy deploy --project-name wxx-agent --branch main
```

### 部署后验证

```bash
curl -s https://wxx-agent.pages.dev/ | grep -o "蔚小芯"   # 必须有输出
curl -I https://wxx-agent.pages.dev/downloads/蔚小芯-v0.0.3.apk
curl -s https://wxx-agent.pages.dev/downloads/release.json
curl -s https://wxx-agent.pages.dev/api/health            # 必须 200，service=蔚小芯
```

### 误部署回滚

如果不小心把前端产物推到了 `wxx-server`：

```bash
# 1. 列出 wxx-server 历史
npx vercel ls wxx-server

# 2. 找一个最近 ● Ready 的稳定部署 url（30s 左右构建时长的就是 Go 后端正常部署）
# 3. 把生产域名 alias 切回去
npx vercel alias set <stable-deployment-url> api.pydaydayup.xyz

# 4. 验证
curl -s https://api.pydaydayup.xyz/health
```

## 环境要求

| 项目 | 最低要求 | 推荐 |
|------|---------|------|
| 操作系统 | Linux（Ubuntu 22.04+）/ Windows Server | Ubuntu 24.04 LTS |
| Go | 1.23+ | 1.23 |
| GCC | 需要（CGO 编译 go-sqlite3） | gcc 12+ |
| Flutter | 3.22+（仅构建前端时需要） | 3.24+ |
| 内存 | 512MB | 2GB |
| 磁盘 | 1GB（含 SQLite 数据库） | 10GB |

## 编译

```bash
# 克隆仓库
git clone git@github.com:dll/wxx.git
cd wxx

# 复制环境配置
cp .env.example .env
# 编辑 .env 填入真实密钥

# 编译后端
make build
# 输出：./bin/wxx-server

# 编译前端（可选，用于内嵌静态资源）
make flutter-build-web
# 输出：./frontend/build/web/
```

## 数据库初始化

```bash
# 执行 SQLite 迁移
make migrate
# 数据库文件默认位置：./data/wxx.db（由 SQLITE_PATH 环境变量控制）
```

## 运行

### 开发环境

```bash
make dev
# 访问 http://localhost:8080
```

### 生产环境（systemd 服务）

创建服务文件 `/etc/systemd/system/wxx.service`：

```ini
[Unit]
Description=蔚小芯学工智能体后端
After=network.target

[Service]
Type=simple
User=wxx
Group=wxx
WorkingDirectory=/opt/wxx
ExecStart=/opt/wxx/bin/wxx-server
EnvironmentFile=/opt/wxx/.env
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

```bash
# 部署步骤
sudo mkdir -p /opt/wxx/{bin,data}
sudo cp bin/wxx-server /opt/wxx/bin/
sudo cp .env /opt/wxx/
sudo cp -r server/migrations /opt/wxx/

# 创建服务用户
sudo useradd -r -s /sbin/nologin wxx
sudo chown -R wxx:wxx /opt/wxx

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable wxx
sudo systemctl start wxx

# 查看日志
sudo journalctl -u wxx -f
```

### 前端部署

Flutter Web 构建产物为静态文件，当前正式方案通过 Cloudflare Pages 部署：

```bash
# 方式一：Cloudflare Pages（当前正式方案）
# 见上方 §Cloudflare Pages 前端部署
cd frontend && flutter build web --release
npx wrangler pages deploy build/web --project-name wxx-agent --branch main

# 方式二（备选）：由 Go 后端内嵌提供
# 在 Go 代码中使用 embed 包嵌入 frontend/build/web/

# 方式三（备选）：Nginx 反向代理
# 将 frontend/build/web/ 复制到 Nginx 静态目录
```

## 数据备份

```bash
# SQLite 在线备份（不中断服务）
sqlite3 /opt/wxx/data/wxx.db ".backup '/opt/wxx/backup/wxx-$(date +%Y%m%d).db'"

# 建议：每日定时备份
# crontab -e
# 0 3 * * * sqlite3 /opt/wxx/data/wxx.db ".backup '/opt/wxx/backup/wxx-$(date +\%Y\%m\%d).db'"
```

## 日志管理

- 应用日志通过 systemd journal 管理
- 审计日志存储在 SQLite `audit_logs` 表中
- 建议保留审计日志至少 180 天

## 健康检查

```bash
# 检查服务状态
sudo systemctl status wxx

# 检查接口可用性
curl -s http://localhost:8080/api/v1/health

# 检查数据库完整性
sqlite3 /opt/wxx/data/wxx.db "PRAGMA integrity_check;"

# 检查 FTS 索引同步
sqlite3 /opt/wxx/data/wxx.db "SELECT COUNT(*) FROM kb_fts; SELECT COUNT(*) FROM kb_resources WHERE status='published';"
```

## 更新流程

```bash
# 1. 拉取最新代码
cd /path/to/wxx && git pull

# 2. 编译
make build

# 3. 备份数据库
sqlite3 /opt/wxx/data/wxx.db ".backup '/opt/wxx/backup/wxx-pre-update.db'"

# 4. 替换二进制
sudo cp bin/wxx-server /opt/wxx/bin/

# 5. 执行迁移（如有新 migration 文件）
cd /opt/wxx && ./bin/wxx-server migrate

# 6. 重启服务
sudo systemctl restart wxx

# 7. 验证
curl -s http://localhost:8080/api/v1/health
```
