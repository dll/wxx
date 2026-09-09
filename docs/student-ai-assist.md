# 学生端统一 AI 助手

学生页面通过认证接口 `POST /api/v1/student/ai-assist` 调用统一 AI 辅助能力。接口只生成建议或可编辑草稿，不直接写入业务数据。

请求体：

```json
{"feature":"study","input":"帮我把数据结构期末复习拆成两周计划"}
```

`feature` 支持 `general`、`study`、`career`、`mental`、`competition`、`campus`、`process`、`profile`、`vopc`。返回 `response`、`data_source`（`ai` 或 `rule`）、`review_required=true`、`sources` 和 `next_action`。模型不可用时走本地规则建议，并明确要求核对正式通知、日期和个人数据。

vOPC G0 表单另提供 `POST /api/v1/vopc/project-draft/assist`：输入一句想法，返回 `fields` 草稿示例。前端仅填充空白字段，已有内容不会被覆盖；学生必须自行审阅后保存或提交。

全局悬浮菜单在学生端各页面均提供 AI 助手入口，并依据当前路由预选学习、职业、心理、办事或 vOPC 领域。采用统一页面骨架的页面还会在 AppBar 显示同领域的 AI 快捷按钮。规则兜底只提供不含实时事实断言的行动模板，统一返回 `review_required=true`、空 `sources` 与 `next_action`，避免把示例误认为课程、活动、成绩、名额或个人阶段。
