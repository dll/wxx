# WXX — 蔚小芯全栈应用

本仓库是「蔚小芯」完整工程，包含 Flutter 客户端、Go/Gin 后端、Context Engine、Eino 编排、数据库迁移、微信小程序壳和部署脚本。产品与技术约束以 `docs/蔚小芯智能体.md`、`docs/蔚小芯开发规范.md` 为准。

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
| `frontend/` | Flutter 客户端（Web、Android、iOS 等） |
| `server/` | Go/Gin 后端、服务层、仓储层、智能体与迁移 |
| `miniprogram/` | 微信小程序 WebView 壳 |
| `scripts/` | 构建、部署、数据维护脚本 |
| `specs/` | API、导出、RBAC 等契约 |

## 常用验证

```text
go test ./server/...
cd frontend && flutter test
cd frontend && flutter analyze --no-pub --no-fatal-infos --no-fatal-warnings
```

完整构建使用 `pwsh scripts/build-all.ps1`；涉及行为变更时，先更新对应文档，再提交可验证的增量。

## 与仓库其它文档的关系

若在本地与 **`Harness驾驭工程规范.md`、`RAGvsLLMwiki.md`** 放在同一父目录（例如课程的「学工」资料夹），可参考其中 Harness 与知识范式对照说明；单独克隆本仓库时无此二文件不影响阅读总纲。详见 `docs/蔚小芯开发规范.md` §2。
