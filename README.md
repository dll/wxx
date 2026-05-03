# WXX — 蔚小芯规范与文档脚手架

本目录为 **规范与模板集合**，用于对齐 `docs/蔚小芯开发规范.md` 与 `docs/蔚小芯智能体.md` 总纲。**不包含** Flutter / Go 等业务源码；实际代码仓库可在别处初始化后，将本目录复制或 submodule 引入。

## 目录结构

| 路径 | 说明 |
|------|------|
| `docs/蔚小芯开发规范.md` | **主规范**：Harness + Context Engine + 约束摘要 |
| `docs/蔚小芯智能体.md` | **总纲**：架构、功能、附录契约、ASCII 示意图（全文较长） |
| `docs/ASCII代码块方案说明.md` | PDF 导出与 ASCII 代码块排版约定（若与总纲第一章说明重复，以实际维护的一份为准） |
| `AGENTS.md` | AI 助手索引（按需加载 `docs/`） |
| `docs/` | 分主题规范摘录与 Harness 工作流 |
| `specs/` | 契约摘要、导出包、RBAC 模板（Markdown） |
| `knowledge/` | 可选：项目组内部 LLM Wiki 式 `raw/` / `wiki/` 脚手架 |
| `templates/` | 计划、变更、评审类 Markdown 模板 |

## 与仓库其它文档的关系

与 `WXX/` **同级**（均在 `学工/` 目录下）的还有：`Harness驾驭工程规范.md`、`RAGvsLLMwiki.md`，供 Harness 与知识范式对照；总纲正文已集中在 **`docs/蔚小芯智能体.md`**。
