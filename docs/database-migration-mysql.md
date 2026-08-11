# 数据库迁移文档：SQLite → MySQL + Redis

> 版本：v1.0 ｜ 日期：2026-08-12 ｜ 状态：已实施并验证

## 1. 背景与目标

蔚小芯后端原本以 **SQLite（含 FTS5 全文检索）** 作为唯一数据库，单文件存储，无法支撑多实例部署与在线并发。本次迁移目标：

- **数据库**：SQLite → MySQL 8（InnoDB，utf8mb4）
- **缓存**：新增 Redis（可选组件，`REDIS_ADDR` 配置后启用）
- **范围**：仅改造 DB 层（配置加载、连接初始化、迁移执行、repository SQL 方言），业务逻辑与 API 契约不变
- **数据库名**：`wxx`（在腾讯云 Lighthouse 服务器 MySQL 中创建）

## 2. 环境与连接参数

参数来源于服务器 `/opt/eqs/.env`（EQS 工程快捷服务共用同一套 MySQL/Redis 基础设施）：

| 变量 | 值 | 说明 |
|------|-----|------|
| `DB_DRIVER` | `mysql` | 方言开关（缺省自动识别：`libsql://` → Turso，其他 → SQLite） |
| `DB_HOST` / `DB_PORT` | `localhost` / `3306` | MySQL 地址 |
| `DB_USER` / `DB_PASSWORD` | `eqs` / `EQS_DB_Pass_2026!` | 服务器 MySQL 账号 |
| `DB_NAME` | `wxx` | 本次新建的数据库 |
| `REDIS_ADDR` / `REDIS_PASS` | `localhost:6379` / 空 | Redis 缓存（可选） |
| `REDIS_DB` | `0` | Redis 逻辑库 |

服务器环境（已确认）：

- MySQL 8.0.46（`/usr/bin/mysql`，监听 `127.0.0.1:3306`）
- Redis 7.0（`redis-cli ping` → PONG）
- Go 1.26.0（`/usr/local/go/bin/go`，项目 `go.mod` 声明 go 1.25.0）

## 3. 迁移前架构（原状）

```
SQLite 文件（./data/wxx.db，modernc.org/sqlite 纯 Go 驱动）
├── 73 个迁移文件（server/migrations/*.sql，全部为 SQLite 方言）
├── FTS5 虚拟表 kb_fts + 触发器（BM25 全文检索，Context Engine 核心）
├── JSON1 扩展（json_valid / json_array_length / json_each 做角色过滤）
└── SQLite 专有语法（AUTOINCREMENT、datetime('now')、INSERT OR IGNORE、ON CONFLICT、PRAGMA）
```

## 4. 迁移后架构

```
MySQL 8（wxx 库，go-sql-driver/mysql）＋ Redis（go-redis/v9，可选缓存）
├── 迁移执行器：运行时把 SQLite 方言迁移 SQL 转换为 MySQL 方言（server/internal/db/dialect.go）
├── FTS5 → 降级：MySQL 无 FTS5 虚拟表，kb_repo 检索退化为结构化 LIKE 检索（SearchStructured）
├── JSON1 → MySQL JSON 函数（JSON_VALID / JSON_LENGTH / JSON_CONTAINS）
└── 保留 SQLite 方言能力：DB_DRIVER 缺省仍走 SQLite/Turso，测试与本地开发不受影响
```

## 5. 代码实现要点

### 5.1 配置加载（`server/internal/config/config.go`）

新增字段：`DBDriver`、`DBHost`、`DBPort`、`DBUser`、`DBPassword`、`DBName`、`RedisAddr`、`RedisPass`、`RedisDB`。

- `DB_DRIVER=mysql` 时走 MySQL；否则按 `DB_PATH` 前缀自动识别（`libsql://` → Turso，其他 → SQLite）
- 新增 `MySQLDSN()` 构建连接串：`user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true&loc=Local`
- `Validate()` 对 MySQL 模式校验 `DB_HOST` / `DB_NAME` 非空，且不再强制要求 SQLite 路径

### 5.2 方言转换器（新增 `server/internal/db/dialect.go`）

核心是 `ToMySQL(stmt)` 与 `AdaptForDriver(stmt, driver)`，覆盖差异：

| SQLite 语法 | MySQL 转换 |
|-------------|-----------|
| `INTEGER PRIMARY KEY AUTOINCREMENT` | `BIGINT PRIMARY KEY AUTO_INCREMENT` |
| 其余 `INTEGER` 列 | `BIGINT`（保证外键类型一致） |
| `TEXT ... DEFAULT (datetime('now'))` / `datetime('now','localtime')` | `DATETIME ... DEFAULT CURRENT_TIMESTAMP` |
| `datetime('now')` / `datetime('now','localtime')`（表达式） | `CURRENT_TIMESTAMP` |
| `datetime('now', '+N unit')` | `DATE_ADD(NOW(), INTERVAL N UNIT)` |
| 带默认值的短文本 `TEXT` | `VARCHAR(128)`（utf8mb4 下复合唯一键 4 列 < 3072B 上限） |
| 长文本 `TEXT`（content/description 等） | 保留 `TEXT`，去掉 DEFAULT（MySQL TEXT 不允许默认值） |
| 键列/索引列 `TEXT`（UNIQUE/PRIMARY KEY/REFERENCES/普通索引，如 code/source 等） | `VARCHAR(128)` |
| `"双引号"` 标识符（如 `"references"` 列） | `` `反引号` `` |
| `key` / `value` / `rank` 保留字列名（列定义与 INSERT 列清单） | 加反引号 |
| `INSERT OR IGNORE` | `INSERT IGNORE` |
| `CREATE INDEX IF NOT EXISTS` | 去掉 `IF NOT EXISTS`（重复索引按 1061 容错跳过） |
| `PRAGMA ...`（如 foreign_keys） | 跳过 |
| `ON CONFLICT(cols) DO UPDATE SET ... excluded.col`（运行时 DML） | `ON DUPLICATE KEY UPDATE ... VALUES(col)` |
| `a || b` 拼接（运行时） | `CONCAT(a, b)` |

> `_migrations` 记录表按方言分别建表（MySQL：`BIGINT AUTO_INCREMENT` + `DATETIME DEFAULT CURRENT_TIMESTAMP`）。

### 5.3 连接初始化与迁移执行（`server/pkg/app/app.go`）

- `initDB(cfg, dbPath, driver)`：按 driver 选择 `mysql` / `libsql` / `sqlite` 驱动；MySQL/Turso 并发连接池，SQLite 单连接
- `runMigrations(db, driver)` / `execSQL(...)`：MySQL 模式逐条 `ToMySQL` 转换；跳过 FTS5 虚拟表与触发器（与 Turso 同策略）；`ALTER TABLE ADD COLUMN` 重复列、`CREATE INDEX` 重复索引按错误码容错跳过
- `cmd/migrate/main.go`：独立迁移工具同样支持 `DB_DRIVER=mysql`

### 5.4 Repository 层方言适配

- **kb_repo（Context Engine 核心）**：新增 `mysql` 标志（`db.IsMySQL(db)` 探测）；`roleScopeCond()` 按方言生成角色过滤条件（MySQL 用 `JSON_CONTAINS(role_scope, JSON_QUOTE(?))` 替代 `json_each`）；`Search`/`SearchFAQ` 在 MySQL 下退化为 `SearchStructured`（title/tags/category LIKE 结构化检索）
- **INSERT OR IGNORE**：`education_health_handler`、`ai_briefing_repo`、`checkin_repo` 改为 `dbutil.InsertIgnore(db.DriverOf(db))` 前缀，MySQL 输出 `INSERT IGNORE INTO`
- **ON CONFLICT / excluded. / datetime('now','localtime') / ||**：`settings_repo`、`data_import_repo`、`external_app_repo`、`model_config_repo`、`personality_repo`、`portal_credential_repo`、`twin_portrait_repo`、`twin_repo`、`feedback_repair_repo`、`education_health_handler` 均通过 `dbutil.AdaptForDriver(stmt, dbutil.DriverOf(db))` 适配
- **`key`/`value` 保留字列**：`settings_repo` 改为反引号引用（`` `key` ``），双方言兼容

### 5.5 Redis 接入（可选）

- `config` 增加 `RedisAddr` / `RedisPass` / `RedisDB`；`REDIS_ADDR` 非空时初始化 go-redis 客户端，Ping 失败仅告警降级（不阻断启动）
- `/health` 增加 `redis` 依赖状态

### 5.6 健康检查（`server/pkg/app/app.go`）

- `dependencies.database.driver` 报告实际方言（`sqlite` / `mysql` / `turso`）
- FTS5 状态在 MySQL 下显示 `unavailable (mysql)`

## 6. 数据库准备（服务器操作）

```sql
-- root 执行：创建 wxx 库并授权给 eqs 用户
CREATE DATABASE IF NOT EXISTS wxx CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
GRANT ALL PRIVILEGES ON wxx.* TO 'eqs'@'localhost';
FLUSH PRIVILEGES;
```

## 7. 迁移执行

```bash
# 在 server/ 目录下运行（加载 server/.env 或 ../.env 中的 DB_DRIVER=mysql 等配置）
cd server
DB_DRIVER=mysql DB_HOST=localhost DB_PORT=3306 DB_USER=eqs DB_NAME=wxx \
  go run ./cmd/migrate
```

成功输出：`成功执行 70 个迁移文件`（FTS5 相关语句按策略跳过，`_migrations` 完整记录）。

> 服务启动时（`initAppWithConfig` → `runMigrations`）也会自动执行同样的迁移，幂等。

## 8. 验证结果

| 验证项 | 命令 | 结果 |
|--------|------|------|
| 编译 | `go build ./server/...` | ✅ 通过 |
| 静态检查 | `go vet ./server/...` | ✅ 通过 |
| 单元测试（repository/context_engine/handler/service/middleware） | `go test ./server/...` | ✅ 全部通过 |
| 服务器 MySQL 迁移 | `go run ./server/cmd/migrate`（DB_DRIVER=mysql） | ✅ 70 个迁移成功 |
| 启动冒烟（MySQL+Redis） | 服务器运行 wxx-server + `curl /health` | ✅ `status: healthy` |

冒烟 `/health` 输出示例：

```json
{
  "status": "healthy",
  "dependencies": {
    "database": { "status": "ok", "driver": "mysql" },
    "redis":    { "status": "ok" },
    "fts5":     { "status": "unavailable (mysql)" },
    "llm_api":  { "status": "no_api_key" }
  }
}
```

## 9. 已知差异与限制

1. **FTS5 全文检索**：MySQL 无 FTS5 虚拟表/BM25，检索退化为结构化 LIKE 检索（title/tags/category）。精度低于 SQLite FTS5，后续可评估 MySQL FULLTEXT（ngram）作为增强
2. **TEXT 默认值**：MySQL TEXT/BLOB 不允许 DEFAULT，长文本列去掉默认值（应用层 INSERT 均显式赋值，不受影响）
3. **VARCHAR 长度**：短文本统一 128（复合唯一键 4 列在 utf8mb4 下 < 3072B），超长文本用 TEXT
4. **时间时区**：`datetime('now','localtime')` 转换为 `CURRENT_TIMESTAMP`（MySQL 会话时区）
5. **Redis**：当前为基础设施接入（连接 + 健康检查 + 客户端就绪），业务缓存/限流后续按需启用
6. **`settings` 的 key/value 保留字**：代码以反引号引用，双方言兼容

## 10. 回滚方案

- **代码**：`git revert` 迁移提交；方言层（`internal/db`）不影响 SQLite 路径，缺省 `DB_DRIVER` 为空仍走 SQLite
- **数据**：MySQL `wxx` 库与 SQLite 文件互不影响；如启用 MySQL 后出现问题，改回 `DB_DRIVER` 缺省（SQLite）即可恢复原架构
- **服务器**：保留原 `wxx-server` 二进制与 `data/wxx.db`（部署目录 `/opt/wxx` 未改动）

## 11. 相关文件

| 文件 | 说明 |
|------|------|
| `server/internal/db/dialect.go` | 方言判断、`ToMySQL`、`AdaptForDriver`、`InsertIgnore` |
| `server/internal/config/config.go` | MySQL/Redis 配置、`MySQLDSN`、校验 |
| `server/pkg/app/app.go` | `initDB` / `runMigrations` / `execSQL` / 健康检查 / Redis 初始化 |
| `server/pkg/app/sync.go` | Turso 同步适配新签名 |
| `server/cmd/migrate/main.go` | 迁移工具 MySQL 支持 |
| `server/internal/repository/kb_repo.go` | FTS 降级、角色过滤方言化 |
| 各 repository/handler | ON CONFLICT / INSERT OR IGNORE / 保留字列适配 |
