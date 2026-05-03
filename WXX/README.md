# WXX — 蔚小芯规范与文档脚手架

本仓库（**wxx**）对应「蔚小芯」智能体（Flutter + Golang 等技术栈见 `docs/蔚小芯智能体.md`）。当前提交内容为 **规范与模板集合**，用于对齐 `docs/蔚小芯开发规范.md` 与总纲；**不包含** Flutter / Go 等业务源码。业务代码可在其它仓库初始化后，将本仓库复制为 submodule 或对照引用。

## 目录结构

| 路径 | 说明 |
|------|------|
| `docs/蔚小芯开发规范.md` | **主规范**：Harness + Context Engine + 约束摘要 |
| `docs/蔚小芯智能体.md` | **总纲**：架构、功能、附录契约；ASCII 示意图与 PDF 导出说明见该文档开篇「PDF 导出说明」 |
| `AGENTS.md` | AI 助手索引（按需加载 `docs/`） |
| `docs/` | 分主题规范摘录与 Harness 工作流 |
| `specs/` | 契约摘要、导出包、RBAC 模板（Markdown） |
| `knowledge/` | 可选：项目组内部 LLM Wiki 式 `raw/` / `wiki/` 脚手架 |
| `templates/` | 计划、变更、评审类 Markdown 模板 |

## 与仓库其它文档的关系

若在本地与 **`Harness驾驭工程规范.md`、`RAGvsLLMwiki.md`** 放在同一父目录（例如课程的「学工」资料夹），可参考其中 Harness 与知识范式对照说明；单独克隆本仓库时无此二文件不影响阅读总纲。详见 `docs/蔚小芯开发规范.md` §2。
