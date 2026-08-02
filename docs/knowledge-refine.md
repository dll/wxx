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

## 四、待办（后续增量）

- 管理员知识治理页「批量精修存量低质量资源」：复用 `RefineMetadata` + `KBService.Update`，按资源 ID 批量重生成元数据。
- 解析质量门槛：正文过短/无中文/含乱码时拒绝或强制预览。

## 相关文件

| 层 | 文件 |
|----|------|
| service | `server/internal/service/document_service.go`（RefineMetadata 与 helper） |
| handler | `server/internal/handler/document_handler.go`（RefineDocument） |
| 路由/装配 | `server/pkg/app/app.go`（SetLLMClient + /documents/refine） |
| 迁移 | `server/migrations/049_fts_tags.sql` |
| repository | `server/internal/repository/kb_repo.go`（bm25 权重） |
| 前端 | `frontend/lib/pages/admin/my_submissions_page.dart`、`knowledge_provider.dart`、`api_config.dart` |
