

## vOPC 决策中心闭环（2026-08-21）
- 新增兼容迁移 `server/migrations/098_vopc_decisions.sql`：为 097 决策表补齐 `status`、`decided_at` 及索引，不回退 097。
- 新增决策 API：`GET/POST /api/v1/vopc/projects/:id/decisions`、`PUT .../decisions/:decisionId`，状态仅允许 pending→resolved/cancelled；项目可见性、项目角色、学院归属、Capability、冻结/终止/归档/完成边界均在服务端校验。
- 创建与处理均使用事务，并将 `decision.created` / `decision.status_changed` 审计事件与业务写入原子提交；事件失败会整体回滚，禁止伪成功。
- Provider 增加决策列表、创建、处理、服务端错误透传；详情加载真实拉取决策数据。后续应补充详情页决策卡片/表单 UI 与 Provider 定向测试。
- 最终门禁已完成：`go test ./internal/handler ./pkg/app -count=1`、`go vet ./internal/handler ./pkg/app`、Flutter 定向 analyze、Flutter 全量 12 项测试及 `git diff --check` 均通过。真实浏览器/设备截图验证及由视觉验证触发的修复按用户 2026-08-21 15:52 指示豁免。

## vOPC v1.0 剩余闭环（2026-08-21）
- 保留 097/098 语义，新增 `099_vopc_collaboration_delivery.sql`。
- 成员：学院 active 用户邀请、接受/拒绝、项目角色与状态审计。
- 成果：仅保存成果/版本元数据、安全引用、校验和与说明，不上传或执行任意文件。
- 正式里程碑：主理人提交下一阶段证据与项目内版本；指定评审或平台运营通过/退回，通过才推进合法状态机。
- Flutter 补齐决策卡片/创建/resolve/cancel、成员邀请、成果版本、里程碑提交与评审 UI/Provider。
- 所有写接口事务+审计；跨项目资源 404，准入/能力不足 403，冲突 409，字段/学院/证据校验 422。AI 只提供建议，不代 OPC 执行重大事项。
