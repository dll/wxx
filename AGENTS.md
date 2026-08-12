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
| **文档解析与 AI 精修** | `docs/knowledge-refine.md`（LLM 元数据精修 + FTS tags 增强） |
| 接口与导出契约索引 | `specs/export-package.md`、`specs/resource-schema.md` |
| RBAC 矩阵模板 | `specs/rbac-matrix.md` |
| AnswerCard / 导出审计 | `docs/ui-answer-card.md` |
| **反馈管理与 AI 在线修复** | `docs/ui-feedback.md`（模块定位 + GLM-4.6V 截图解析 + AI 诊断接口） |
| **反馈闭环完整文档** | `docs/蔚小芯问题反馈与在线修复.md`（提交→入库→结构化复制→AI修复→验证→解决） |
| **学生操作手册** | `docs/蔚小芯智能体学生操作手册.md`（系统简介 + 新生操作流程 + AI额度/API Key 说明） |
| 办事流程管理与提醒 | `docs/办事流程管理.md`（动态流程、CRUD、审核、导出与提醒） |
| 校外系统对接注意 | `docs/integrations.md` |
| 总纲全文（产品与技术） | `docs/蔚小芯智能体.md`（含 PDF 与 ASCII 示意图排版说明） |
| 部署指南 | `docs/deployment.md` |
| **前端部署（Cloudflare Pages）** | `docs/蔚小芯前端重新部署.md`（迁移记录 + 自动部署） |
| **学生用户导入与账号管理** | `docs/user-import.md`（权限、Excel 模板、初始密码与验收） |
| **报到节点坐标管理** | `docs/蔚小芯报到节点坐标管理.md`（管理员拖拽校正 + CRUD 工作流） |
| **前端全量构建脚本** | `scripts/build-all.ps1`（一键构建 Web + APK，用法：`pwsh scripts/build-all.ps1` 或 `make all-frontend`） |
| **AI 简讯模块** | `docs/ai-briefings.md`（首页资讯 + 管理 CRUD + RSS/Atom 自动抓取 + md/pdf 导出） |
| **微信小程序（WebView 壳）** | `frontend/miniprogram/`（AppID: wx811d1225e67b8f38，加载 Cloudflare Pages 前端） |
| **数据库迁移（SQLite → MySQL + Redis）** | `docs/database-migration-mysql.md`（方言转换层、迁移执行、验证与回滚） |

## 内部知识（可选）

项目组自用资料可使用 `knowledge/raw/` 与 `knowledge/wiki/`（LLM Wiki 范式），**不替代**上线 Context Engine 治理流程。说明见 `knowledge/README.md`；wiki 运营流程与条目模板见 `knowledge/wiki/README.md` 与 `knowledge/wiki/_template.md`。

## Code Exploration Policy

Always use jCodemunch-MCP tools for code navigation. Never fall back to Read, Grep, Glob, or Bash for code exploration.
**Exception:** Use `Read` when you need to edit a file — the agent harness requires a `Read` before `Edit`/`Write` will succeed. Use jCodemunch tools to *find and understand* code, then `Read` only the specific file you're about to modify.

**Start any session:**
1. `resolve_repo { "path": "." }` — confirm the project is indexed. If not: `index_folder { "path": "." }`
2. `suggest_queries` — when the repo is unfamiliar

**Finding code:**
- symbol by name → `search_symbols` (add `kind=`, `language=`, `file_pattern=`, `decorator=` to narrow)
- decorator-aware queries → `search_symbols(decorator="X")` to find symbols with a specific decorator (e.g. `@property`, `@route`); combine with set-difference to find symbols *lacking* a decorator (e.g. "which endpoints lack CSRF protection?")
- string, comment, config value → `search_text` (supports regex, `context_lines`)
- database columns (dbt/SQLMesh) → `search_columns`

**Reading code:**
- before opening any file → `get_file_outline` first
- one or more symbols → `get_symbol_source` (single ID → flat object; array → batch)
- symbol + its imports → `get_context_bundle`
- specific line range only → `get_file_content` (last resort)

**Repo structure:**
- `get_repo_outline` → dirs, languages, symbol counts
- `get_file_tree` → file layout, filter with `path_prefix`

**Relationships & impact:**
- what imports this file → `find_importers`
- where is this name used → `find_references`
- is this identifier used anywhere → `check_references`
- file dependency graph → `get_dependency_graph`
- what breaks if I change X → `get_blast_radius`
- what symbols actually changed since last commit → `get_changed_symbols`
- find unreachable/dead code → `find_dead_code`
- class hierarchy → `get_class_hierarchy`

## Session-Aware Routing

**Opening move for any task:**
1. `plan_turn { "repo": "...", "query": "your task description", "model": "<your-model-id>" }` — get confidence + recommended files; the `model` parameter narrows the exposed tool list to match your capabilities at zero extra requests.
2. Obey the confidence level:
   - `high` → go directly to recommended symbols, max 2 supplementary reads
   - `medium` → explore recommended files, max 5 supplementary reads
   - `low` → the feature likely doesn't exist. Report the gap to the user. Do NOT search further hoping to find it.

**Interpreting search results:**
- If `search_symbols` returns `negative_evidence` with `verdict: "no_implementation_found"`:
  - Do NOT re-search with different terms hoping to find it
  - Do NOT assume a related file (e.g. auth middleware) implements the missing feature (e.g. CSRF)
  - DO report: "No existing implementation found for X. This would need to be created."
  - DO check `related_existing` files — they show what's nearby, not what exists
- If `verdict: "low_confidence_matches"`: examine the matches critically before assuming they implement the feature

**After editing files:**
- If PostToolUse hooks are installed (Claude Code only), edited files are auto-reindexed
- Otherwise, call `register_edit` with edited file paths to invalidate caches and keep the index fresh
- For bulk edits (5+ files), always use `register_edit` with all paths to batch-invalidate

**Token efficiency:**
- If `_meta` contains `budget_warning`: stop exploring and work with what you have
- If `auto_compacted: true` appears: results were automatically compressed due to turn budget
- Use `get_session_context` to check what you've already read — avoid re-reading the same files

## Model-Driven Tool Tiering

Your jcodemunch-mcp server narrows the exposed tool list based on the model you are running as. To avoid wasting requests on primitives when a composite would do, always include `model="<your-model-id>"` in your opening `plan_turn` call.

Replace `<your-model-id>` with your active model:
- Claude Opus variants → `claude-opus-4-7` (or any `claude-opus-*`)
- Claude Sonnet variants → `claude-sonnet-4-6`
- Claude Haiku variants → `claude-haiku-4-5`
- GPT-4o / GPT-5 / o1 / Llama → use the model id as printed by your runner

The `model=` parameter rides on the existing `plan_turn` call — it does **not** add a separate tool invocation. If `plan_turn` is not appropriate for a non-code task, call `announce_model(model="...")` once instead.

