# 部署指南 — 蔚小芯

> 蔚小芯采用轻量单机部署，不依赖 Docker/容器/集群。
> 后端走 systemd / 自托管二进制；前端 Web 部署到 Vercel；移动端打 APK。

## 应用命名规范（强制）

> 所有面向用户的入口名称统一为「蔚小芯」，包括：

| 平面 | 字段 | 值 |
|------|------|-----|
| Android | `android:label` (`AndroidManifest.xml`) | 蔚小芯 |
| Android | APK 输出（Gradle `outputFileName`） | `蔚小芯-release.apk` / `蔚小芯-debug.apk` |
| Android | 最终分发文件（`build/app/outputs/flutter-apk/`） | `蔚小芯.apk` |
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
cd frontend && flutter build web --release --web-renderer html
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
| `frontend/build/app/outputs/flutter-apk/蔚小芯.apk` | **对外分发文件**，由 Makefile 自动复制 |

> Gradle 改名后 Flutter SDK 仍会生成 `app-release.apk`（来自其内部硬编码逻辑）。Makefile 在构建结束后会自动把中文 APK 复制到 `flutter-apk/蔚小芯.apk`，分发请使用此文件。

## Vercel 前端部署（强制流程）

> **注意**：前端已迁移至 Cloudflare Pages（`wxx-agent.pages.dev`），详见 `docs/蔚小芯前端重新部署.md`。Vercel 后端 `wxx-server` 仍保留运行，通过 Cloudflare Functions 代理访问。以下为 Vercel 历史部署记录。

> 前端 Vercel 项目：`wxx-frontend`（绑定域名 `https://wxx.pydaydayup.xyz`）
> 后端 Vercel 项目：`wxx-server`（绑定域名 `https://api.pydaydayup.xyz`，`api/index.go` 入口）

**两个项目必须各自独立部署，绝不可在仓库根执行 `vercel deploy build/web`** —— 仓库根 `.vercel/repo.json` 指向 `wxx-server`，会把前端产物错传到后端项目，导致 API 服务中断。

### 标准部署命令（推荐）

```bash
make deploy-web
```

该 target 会：
1. 重新构建 `frontend/build/web/`
2. 同步到 `frontend/.vercel/output/static/`
3. 通过 `vercel deploy --prebuilt --prod --cwd frontend` 部署到 `wxx-frontend` 项目

### 仅推送已构建产物

```bash
make deploy-web-prebuilt
```

### 手动部署（与 Makefile 等价）

```bash
cd frontend
flutter build web --release --web-renderer html
cp -rf build/web/* .vercel/output/static/
npx --yes vercel deploy --prebuilt --prod
```

### 部署后验证

```bash
curl -s https://wxx.pydaydayup.xyz/ | grep -o "蔚小芯"   # 必须有输出
curl -s https://api.pydaydayup.xyz/health                # 必须 200，service=蔚小芯
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
