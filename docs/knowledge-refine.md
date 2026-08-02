# 文档元数据 AI 精修与知识库增强

> 目的：解决「导入文件解析得到的标题/摘要/关键词不准确、无人编辑」的问题，让解析结果可直接使用。
> 主链路不变：**结构化优先 → FTS/BM25 → 上下文拼装 → 模型生成**。本特性只提升入库数据质量。

## 一、LLM 元数据精修（P0）

### 背景

`document_service.go` 原有解析全部为规则实现：
- 摘要 = 前 200 字符（常截到文头噪声）；
- 关键词 = n-gram 词频（无分词、无领域词表，产出碎片与公文泛词）；
- 标题 = 前几行启发式扫描。

由于 `filterByRelevance` 要求 title+summary 命中查询二元组、`SearchStructured` 用 tags 做 LIKE 匹配，
低质量元数据会直接拉低检索召回，而非仅影响展示。

### 方案

**后端** `POST /api/v1/documents/refine`
- `DocumentService.RefineMetadata(ctx, title, summary, keywords, content)`
- 一次 LLM 调用（temperature 0.3，超时 30s，正文截断至 6000 字）返回严格 JSON：
  `{"title","summary","keywords"}`
- 三层兜底保证接口始终可用：
  1. 未注入 LLM 客户端 → 直接返回传入值（`fallback=true`）；
  2. 调用超时/失败/响应非 JSON → 回退规则；
  3. 字段缺失/校验不通过 → 用原值补齐后再整体校验，仍不过则回退。
- **只回填编辑表单，不自动写库**：写库仍走 `KBService`（人工确认后提交）。

**前端**（`my_submissions_page.dart` 创建/编辑资源弹窗）
- 「AI 一键精修标题/摘要/关键词」按钮：读取当前正文 → 调用精修接口 → 回填标题/摘要/标签（关键词）。
- 精修失败提示保留当前内容，可手动编辑后提交。

### 权限

与 `/documents/parse` 一致：`counselor.kb.write` 或 `union.kb.submit`。

## 二、知识库增强：关键词参与 BM25（C 档部分）

- **迁移** `049_fts_tags.sql`：重建 `kb_fts` 加入 `tags` 列，改写三个同步触发器携带 tags，并 `rebuild` 回填存量。
  - 采用 DROP + CREATE 而非 `ALTER TABLE ADD COLUMN`（FTS5 虚拟表不支持 ALTER）；
  - Turso 由 `runMigrations(isTurso)` 自动跳过（与既有 FTS5 跳过逻辑一致）。
- **权重**：`bm25(kb_fts, resource_id=0.0, title=10.0, summary=3.0, content=1.0, tags=4.0)`
  （tags 权重高于正文，因为精修后的关键词是比原始正文更可靠的检索信号）。
- 效果：上传时 keywords 已并入 tags，现在关键词将同时参与结构化 LIKE 匹配与全文 BM25。

## 三、知识 Wiki（远期增强路径）

`knowledge/wiki/` 目前为空占位。建议后续：
1. 把「高频问题 → 标准答案 + 引用源」沉淀为 wiki 条目，作为 Context Engine 的高置信锚点；
2. 对解析质量差的资源，引导管理员沉淀到 wiki（人工把关），wiki 条目即结构化高质量数据；
3. 尚未实现，需另行方案评审。

> 更新：`knowledge/wiki/` 运营流程已落地，见 `knowledge/wiki/README.md`（条目生命周期、模板与示例条目）。

## 四、规则精修增强（P1，已完成）

在 LLM 精修不可用/超时/不合法时，规则兜底的质量直接影响上游导入。本次增强：

- **摘要**（`extractDocSummary`）：跳过「附件/目录/关于印发/章节标记」等文头噪声，取首个实质段落（≥40 字）在段界内展开至 200 字；无合适段落回退原逻辑。
- **标题**（`extractDocTitle`）：新增印发/公布/调整/修订/公告/通知等前缀；优先提取《…》书名号内的正题。
- **关键词**（`extractDocKeywordsWithTitle`）：新增 ~46 词校园领域词表加权 + 标题主题词加权，弥补 n-gram 无分词的缺陷；保持 O(n log n)。

## 五、批量精修（管理端，P1 已完成）

- **接口**：`POST /api/v1/kb/batch/refine`（`counselor.kb.write`，单批 ≤20 条）
  `{ids: [...]}` → `{total, success, failed, results[{resource_id, ok, refined, fallback, message}]}`
- **服务**：`KBService.BatchRefine` 复用 `DocumentService.RefineMetadata`（经 `MetadataRefiner` 接口注入），逐条：取正文 → LLM 精修 → 有效且非回退时写库（走 `Update`，FTS 触发器自动同步 tags）。
- **前端**：知识治理页批量操作栏新增「AI 精修」按钮（canWrite），确认后展示成功/失败统计与失败原因。
- **兜底**：未注入精修器 / LLM 失败 / 校验不过 → 单条标记失败或 `fallback=true`，保留原值不写库。

## 六、解析质量门槛（P1 已完成）

- **评估**：`assessDocQuality`（service 层）基于解析正文判定三档质量问题：
  - **过短**：有效字数 < 20（空内容恒判过短）；
  - **无中文**：中文字符数为 0（图片/扫描件/纯外文文档典型特征）；
  - **乱码**：控制字符（除 `\n\t\r`）占比 > 5%，或异常字符（替换符 U+FFFD、非 CJK/ASCII/标点/空白）占比 > 30%。
- **信号**：`/documents/parse` 响应新增 `quality` 字段（`ok/short/no_chinese/garbled/reasons/word_count/chinese_runes/control_ratio/suspicious_runes`），解析流程**不阻断**，仅如实上报。
- **自动入库拒绝**：`/kb/upload` 对文本文档解析后 `quality.ok=false` 时返回 **422** 并附 `reasons`，拒绝污染知识库；传 `force=1` 可显式覆盖（确为英文文档等边缘场景）。
- **强制预览**：前端创建/编辑弹窗导入文件解析后，若 `quality.ok=false` 弹**质量警告对话框**列出原因，用户确认「我已知晓，继续编辑」才回填表单；解析结果卡片同时展示质量警示条。
- **测试**：`document_service_test.go` 质量评估 6 组；`upload_handler_test.go` 拒绝/无中文/force 覆盖/正常入库/未认证 5 组。

## 七、解析可直接使用：上传自动精修 + 编码修复（P2 已完成）

解决「解析结果仍需大量手动编辑、质量差、中文乱码」的问题：

- **`/kb/upload` 入库前自动 LLM 精修**：质量门槛通过后，先调用 `RefineMetadata` 生成标题/摘要/关键词再入库；未配置模型 / 调用失败 / 输出不合法时静默回退启发式结果，不影响上传成功。自动入库内容即高质量元数据。
- **`/documents/parse` 支持 `refine=true`**：解析同时返回 LLM 精修后的元数据，表单回填即高质量；前端 `parseDocument` 已默认带 `?refine=1`，无需再手动点「AI 精修」。
- **中文乱码修复**：新增 `util.DecodeToUTF8`（`golang.org/x/text` GB18030），`.txt/.md/.csv` 非 UTF-8（GBK/GB18030）自动转码，质量门槛从「检出乱码」升级为「修复乱码」。
- **大小限制统一 100MB**：`my_submissions_page.dart` 上传前检查 10MB→100MB、`/kb/upload` 上报 `max_size_mb` 50→100，与后端 `DocumentService(100)` 对齐。

## 相关文件

| 层 | 文件 |
|----|------|
| service | `server/internal/service/document_service.go`（RefineMetadata 与规则精修 helper） |
| service | `server/internal/service/kb_service.go`（BatchRefine / refineOne / MetadataRefiner） |
| handler | `server/internal/handler/document_handler.go`（RefineDocument）、`kb_handler.go`（BatchRefine）、`upload_handler.go`（质量门槛） |
| 路由/装配 | `server/pkg/app/app.go`（SetLLMClient + SetRefiner + 路由注册） |
| 迁移 | `server/migrations/049_fts_tags.sql` |
| repository | `server/internal/repository/kb_repo.go`（bm25 权重） |
| model | `server/internal/model/dto.go`（KBRefineResult / KBRefineResponse） |
| 前端 | `frontend/lib/pages/admin/my_submissions_page.dart`（质量警告对话框 + 警示条）、`knowledge_provider.dart`、`api_config.dart` |
