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

- **知识增强必选**：主链路 = 结构化优先 → FTS/BM25 → 上下文拼装 → 模型生成；向量/Agentic RAG 仅按需启用
- **不本地部署大模型**；**不依赖 Coze** 等第三方智能体 SaaS
- **不强制 Docker/容器/集群**；轻量单机部署
- **政策/条件类回答** 必须附带 `sources[]`；低置信走兜底，禁止编造条款与关键数字
- **多智能体管理中心自研**；编排运行时 = Eino（开源）+ 自研封装

## Harness 协作纪律

1. **Plan → 人审 → 编码**，禁止大块需求无方案直接改代码
2. **每增量**：测试/Lint + Git 提交 + 更新对应 `docs/`
3. **架构护栏**：分层清晰、Pre-commit、禁止静默引入禁用依赖
4. **长会话污染**时开新会话 + 关联文档恢复记忆

## 文档地图

| 主题 | 路径 | 用途 |
|------|------|------|
| 产品与技术总纲 | `docs/蔚小芯智能体.md` | 架构、Context Engine、API 契约附录、分期、风险、对接字段 |
| 开发主规范 | `docs/蔚小芯开发规范.md` | 约束口径、Harness、知识工程、安全隐私、验收对齐 |
| Context Engine 摘录 | `docs/context-engine.md` | 触发策略速查表、主链路、参数默认值 |
| Harness 工作流 | `docs/harness-workflow.md` | 最小清单、与 Context Engine 衔接 |
| 知识治理与运营 | `docs/knowledge-governance.md` | 审核流程、资源四类、同步导出 |
| 校外系统对接 | `docs/integrations.md` | 学工/一表通/多端入口注意事项 |
| AnswerCard 与导出 | `docs/ui-answer-card.md` | 统一回答结构、导出审计要求 |
| API 契约索引 | `specs/api-contracts-index.md` | 变更登记模板 |
| 知识导出包规范 | `specs/export-package.md` | manifest + ndjson + cursor 增量同步 |
| 资源最小字段 | `specs/resource-schema.md` | 必填语义字段速查 |
| RBAC 矩阵 | `specs/rbac-matrix.md` | 六级基线 + teacher/assistant 扩展 |
| 内部 LLM Wiki | `knowledge/README.md` | raw/ / wiki/ 脚手架（非产品知识库） |
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

## 当前仓库状态

本仓库（wxx）目前为 **规范与模板集合**，不包含 Flutter/Go 业务源码。业务代码初始化后可将本仓库作为 submodule 或对照引用。同级目录 `../` 下的 `Harness驾驭工程规范.md`、`RAGvsLLMwiki.md` 为对照参考文档。
