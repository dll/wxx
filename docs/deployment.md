# 部署指南 — 蔚小芯

> 蔚小芯采用轻量单机部署，不依赖 Docker/容器/集群。

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
