# 知识资源最小字段（Markdown 摘要）

> 完整 JSON Schema 与 NDJSON 样例见 `docs/蔚小芯智能体.md`（相对 `WXX/`）§6.8.6、§6.8.7.1。

## 必填语义字段

| 字段 | 说明 |
|------|------|
| `resourceId` | 全局唯一 |
| `resourceType` | Policy \| Process \| FAQ \| Activity |
| `ownerScope` | school \| college \| class |
| `roleScope` | 角色数组（六级子集） |
| `version` | 如 `YYYYMMDD-vN` |
| `status` | draft \| pending \| published \| retired |
| `title` / `summary` / `content` | `content` 为可检索正文 |
| `sourceLink` / `sourceVersion` | 溯源 |
| `tags` / `updatedBy` / `updatedAt` | 治理 |

可选扩展：`chunks[]`（若对接方支持向量索引）、`aliases[]`（FAQ 同义问）。
