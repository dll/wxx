# AGENTS.md — 蔚小芯（协作索引）

> 本文件保持 **简短**。细则请按需打开 `docs/` 与 `specs/` 对应文档。

## 项目一句话

计算机科学与工程学院（网络空间安全学院）**蔚小芯**：Flutter 客户端 + Go/Gin 后端 + **Context Engine（结构化 + FTS/BM25 为主）** + Eino 编排 + 第三方大模型 API；**sources 可追溯**，向量与 Agentic RAG 可插拔。

## 必读顺序

1. `docs/蔚小芯开发规范.md`（主规范）
2. `docs/蔚小芯智能体.md`（总纲与附录契约）
3. `docs/context-engine.md`（触发策略与引用摘要）
4. `specs/export-package.md`（与「蔚园智答」同步约定摘要）

## 协作纪律（Harness）

- **Plan → 人审 → 编码**；每增量 **文档 + Git 提交**。
- **勿**在单文件堆砌超长规则；新增约束落到 `docs/` 或 `specs/` 并在此索引。
- **勿**引入本地大模型、Coze、强制 Docker 集群（除非项目书面变更）。

## 技术栈速查

- 前端：Flutter / Dio / Provider / Hive  
- 后端：Go / Gin / JWT / RBAC / SQLite（含 FTS）  
- 编排：Eino；可选 Temporal  
- 模型 API：智谱、DeepSeek、讯飞（语音）

## 文档地图

| 主题 | 路径 |
|------|------|
| **新手入门** | `docs/蔚小芯开发技术手册.md`（从零开始的完整开发指引） |
| Harness 工作流（对齐本项目） | `docs/harness-workflow.md` |
| Context Engine 摘录 | `docs/context-engine.md` |
| 知识治理与运营 | `docs/knowledge-governance.md` |
| 接口与导出契约索引 | `specs/export-package.md`、`specs/resource-schema.md` |
| RBAC 矩阵模板 | `specs/rbac-matrix.md` |
| AnswerCard / 导出审计 | `docs/ui-answer-card.md` |
| 校外系统对接注意 | `docs/integrations.md` |
| 总纲全文（产品与技术） | `docs/蔚小芯智能体.md`（含 PDF 与 ASCII 示意图排版说明） |
| 部署指南 | `docs/deployment.md` |
| **前端部署（Cloudflare Pages）** | `docs/蔚小芯前端重新部署.md`（迁移记录 + 自动部署） |
| **学生用户导入与账号管理** | `docs/user-import.md`（权限、Excel 模板、初始密码与验收） |
| **前端全量构建脚本** | `scripts/build-all.ps1`（一键构建 Web + APK，用法：`pwsh scripts/build-all.ps1` 或 `make all-frontend`） |
| **微信小程序（WebView 壳）** | `frontend/miniprogram/`（AppID: wx811d1225e67b8f38，加载 Cloudflare Pages 前端） |

## 内部知识（可选）

项目组自用资料可使用 `knowledge/raw/` 与 `knowledge/wiki/`（LLM Wiki 范式），**不替代**上线 Context Engine 治理流程。说明见 `knowledge/README.md`。
