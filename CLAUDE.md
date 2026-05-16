# CLAUDE.md — 蔚小芯 (WXX)

> 信息学院智慧校园 AI 学工助手。本文为 Claude Code 会话入口，细则按需加载 `docs/`、`specs/`。

## 项目一句话

**蔚小芯**：Flutter 客户端 + Go/Gin 后端 + **Context Engine（结构化 + FTS/BM25 为主）** + Eino 编排 + 第三方大模型 API；`sources` 可追溯，向量与 Agentic RAG 可插拔。

## 技术栈速查

| 层级 | 选型 | 备注 |
|------|------|------|
| 前端 | Flutter / Dio / Provider / Hive | 微信小程序需单独轻量端 |
| 后端 | Go / Gin / JWT / RBAC（六级） | 原生编译，无容器 |
| 存储 | SQLite（含 FTS5） + 内存缓存 | 业务、审计、结构化知识 |
| 编排 | Eino（必选） + 自研 Agent Hub | Temporal 可选 |
| 模型 API | 智谱清言 / DeepSeek / 讯飞星火 | HTTP/SSE，不本地部署 |

## 核心约束（不可默认违背）

- **全项目中文化**：所有文档、代码注释、Git 提交信息、错误提示、日志输出均使用中文；变量/函数名用英文但注释必须中文；`SKILL.md` 等技能文件全中文
- **知识增强必选**：主链路 = 结构化优先 → FTS/BM25 → 上下文拼装 → 模型生成；向量/Agentic RAG 仅按需启用
- **不本地部署大模型**；**不依赖 Coze** 等第三方智能体 SaaS
- **不强制 Docker/容器/集群**；轻量单机部署
- **政策/条件类回答** 必须附带 `sources[]`；低置信走兜底，禁止编造条款与关键数字
- **多智能体管理中心自研**；编排运行时 = Eino（开源）+ 自研封装
- **Flutter UI 规范**：Material Design 3 基线，响应式布局 + 暗黑模式 + 磨砂质感 + 动画微交互，详见 `docs/蔚小芯开发规范.md` §13
- **应用命名规范**：用户可见名称一律为「蔚小芯」（Android `android:label` / Web `<title>` / `manifest.json` 的 `name`/`short_name` / 后端 `/health` 的 `service` 字段）；APK 分发文件名固定为 `蔚小芯.apk`；技术 ID（`wxx_app` / `com.wxx.wxx_app` / 仓库 `wxx`）保持英文。详见 `docs/deployment.md`「应用命名规范」
- **Vercel 部署**：前端项目 `wxx-frontend`（域名 `wxx.pydaydayup.xyz`），后端项目 `wxx-server`（域名 `api.pydaydayup.xyz`）。**绝不可在仓库根目录运行 `vercel deploy`**——根 `.vercel/repo.json` 指向 `wxx-server`，前端产物会污染后端 API。统一使用 `make deploy-web` 一键部署

## Harness 协作纪律

1. **Plan → 人审 → 编码**，禁止大块需求无方案直接改代码
2. **每增量**：测试/Lint + Git 提交 + 更新对应 `docs/`
3. **架构护栏**：分层清晰、Pre-commit、禁止静默引入禁用依赖
4. **长会话污染**时开新会话 + 关联文档恢复记忆

## 文档地图

| 主题 | 路径 | 用途 |
|------|------|------|
| 产品与技术总纲 | `docs/蔚小芯智能体.md` | 架构、Context Engine、API 契约附录、分期、风险、对接字段 |
| 开发主规范 | `docs/蔚小芯开发规范.md` | 约束口径、Harness、知识工程、安全隐私、验收对齐、**Flutter UI 设计规范 (§13)** |
| Context Engine 摘录 | `docs/context-engine.md` | 触发策略速查表、主链路、参数默认值 |
| Harness 工作流 | `docs/harness-workflow.md` | 最小清单、与 Context Engine 衔接 |
| 知识治理与运营 | `docs/knowledge-governance.md` | 审核流程、资源四类、同步导出 |
| 校外系统对接 | `docs/integrations.md` | 学工/一表通/多端入口注意事项 |
| AnswerCard 与导出 | `docs/ui-answer-card.md` | 统一回答结构、导出审计要求 |
| API 契约索引 | `specs/api-contracts-index.md` | 变更登记模板 |
| 知识导出包规范 | `specs/export-package.md` | manifest + ndjson + cursor 增量同步 |
| 资源最小字段 | `specs/resource-schema.md` | 必填语义字段速查 |
| RBAC 矩阵 | `specs/rbac-matrix.md` | 六级基线 + teacher/assistant 扩展 |
| 后端结构与分层 | `server/README.md` | 目录结构、分层规则 |
| 前端初始化指南 | `frontend/README.md` | Flutter 初始化步骤、技术选型、目录建议 |
| 部署指南 | `docs/deployment.md` | 编译、运行、systemd 服务、备份、健康检查、更新流程 |
| 内部 LLM Wiki | `knowledge/README.md` | raw/ / wiki/ 脚手架（非产品知识库） |
| 第二阶段开发计划 | `docs/phase2-plan.md` | 9 专项 / 20 任务 / 3 周 / 角色闭环 |
| 计划模板 | `templates/plan.template.md` | 任务方案模板 |
| 变更日志模板 | `templates/CHANGELOG.template.md` | Keep a Changelog 格式 |

## 知识同步要点

- 与「蔚园智答」对接：`manifest.json` + `resources.ndjson` + 增量 cursor
- 幂等键：`(resourceId, version, status)`
- 安全：HTTPS + Bearer Token + HMAC 包签名 + 哈希校验
- 资源四类：Policy / Process / FAQ / Activity

## RBAC 角色（六级基线）

1. `sys_admin` — 系统管理员
2. `school_admin` — 学校
3. `college_admin` — 二级学院
4. `counselor` — 辅导员/班主任
5. `student_union` — 学生会/班团委
6. `student` — 学生

P1+ 扩展：`teacher`（教师）、`assistant`（教辅）— 枚举已占位。

## 分期（14 天，2026-05-01 ~ 2026-05-14）

- **P0**：鉴权打通 + 问答可用 + 一条真实只读数据 + 入学/离校 MVP + Flutter Web/Android
- **P1**：语音 ASR/TTS、情感预警、多智能体管理端雏形、第二条对接
- **P2**：个性化推荐、Temporal 关键链路、深度集成、趋势分析

## 质量指标（验收）

| 指标 | 目标 |
|------|------|
| 网关 P95（不含模型） | ≤ 300ms |
| 问答整体 P95（含模型） | ≤ 2500ms |
| 核心接口成功率 | ≥ 99.5% |
| 智能问答命中率（抽检） | ≥ 85% |
| 引用覆盖率（政策类） | 100% |
| 引用覆盖率（流程类） | ≥ 95% |
| 兜底率 | ≤ 10% |

## 快速开始

```bash
# 环境准备
cp .env.example .env        # 编辑填入真实密钥

# Go 后端
make migrate                 # 初始化 SQLite
make dev                     # 启动后端（热重载）
make test                    # 单元测试
make lint                    # go vet 静态检查

# Flutter 前端（初始化后）
make flutter-get             # 安装依赖
make flutter-run             # 开发运行
make flutter-build-web       # 构建 Web
make flutter-build-apk       # 构建 APK（自动产出 蔚小芯.apk）
make flutter-test            # 前端测试
make deploy-web              # 构建 Web 并部署到 Vercel wxx-frontend

# 全栈
make all                     # 后端编译 + 前端 Web 构建
make test-all                # 后端测试 + 前端测试
```

## 工程目录结构

```
WXX/
├── CLAUDE.md                # Claude Code 会话入口（本文件）
├── .env.example             # 环境变量模板
├── Makefile                 # 统一构建入口
├── server/                  # Go/Gin 后端
│   ├── cmd/server/          # HTTP 服务入口
│   ├── cmd/migrate/         # SQLite 迁移工具
│   ├── internal/            # 业务代码（分层）
│   │   ├── config/          # 配置加载
│   │   ├── handler/         # HTTP handler（参数校验 + 响应组装）
│   │   ├── middleware/      # JWT / RBAC / 限流 / 审计 / CORS
│   │   ├── model/           # 数据模型
│   │   ├── service/         # 业务逻辑（编排层）
│   │   ├── repository/      # SQLite 数据访问
│   │   ├── agent/           # 多智能体管理 + Eino 编排
│   │   ├── context_engine/  # 结构化查询 + FTS + 拼装
│   │   └── llm/             # 智谱/DeepSeek/讯飞 API 客户端
│   ├── migrations/          # SQLite DDL（001_init.sql ...）
│   └── data/                # 运行时数据（.gitignore）
├── frontend/                # Flutter 客户端
├── docs/                    # 产品与技术文档
├── specs/                   # 规格定义
├── knowledge/               # 内部 LLM Wiki 素材
└── templates/               # 计划/变更日志模板

## 当前状态（2026-05-13）

- P0 核心功能完成；P1 基本完成；**第二阶段计划已制定**（`docs/phase2-plan.md`）
- 待补齐：Eino 多智能体编排 / 安全合规（隐私政策、PII 脱敏、内容过滤）/ 多格式导出 / 评测基线
- 六级角色功能需逐项验证；管理端页面（用户管理、审计日志、质量看板）待开发
- 测试覆盖 62.1%（middleware 80.6% / handler 60.9% / service 82.1% / repository 79.9%）
- Flutter Web 构建通过；后端 `go build -tags fts5 ./...` 零错误
```

## 后端分层规则

- **handler** → 参数校验 + 响应组装，不写业务逻辑
- **service** → 编排 repository + llm + agent，实现业务规则
- **repository** → SQL 操作，不依赖 HTTP 或模型 API
- **禁止** handler 直接调用 repository 或 llm
