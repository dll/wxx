# knowledge — 项目组内部 LLM Wiki 脚手架（可选）

对照 `RAGvsLLMwiki.md`：**非产品线上知识库**，仅用于团队内部资料编译与复盘。

## 目录约定

| 目录 | 用途 |
|------|------|
| `raw/` | 原始素材（只读存放，可不预先整理） |
| `wiki/` | 由 LLM 协助维护的结构化摘要、索引与交叉引用 |

## 与 Context Engine 的边界

- **上线面向师生的政策/流程/FAQ**：必须经过 Context Engine 治理（元数据、审核、`sources`），路径与同步见 `docs/knowledge-governance.md`、`specs/export-package.md`。  
- **`knowledge/wiki/`**：不得自动等同于对外发布内容；对外发布须走审核与版本流程。

## 维护建议

定期将稳定结论 **反哺** 到总纲或运营条目，避免 wiki 与正式知识漂移。
