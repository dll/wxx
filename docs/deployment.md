# 部署指南 — 蔚小芯

> 蔚小芯采用轻量单机部署，不依赖 Docker/容器/集群。
> 后端走 systemd / 自托管二进制；前端 Web 部署到 Vercel；移动端打 APK。

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

> 前端唯一正式入口：`https://wxx-agent.pages.dev`。Vercel 前端旧域名已停用，不再作为发布或验收入口。Vercel 后端 `wxx-server` 仍保留运行，通过 Cloudflare Pages Functions 代理访问。

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

Flutter Web 构建产物为静态文件，可通过以下方式部署：

```bash
# 方式一：由 Go 后端内嵌提供（推荐）
# 在 Go 代码中使用 embed 包嵌入 frontend/build/web/

# 方式二：Nginx 反向代理
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
