# vOPC PRD v1.0 重构流水线最终汇总

> 汇总方：leader-wxx
>
> 日期：2026-08-22
>
> 项目根目录：`E:\2026-2027\2026-2027-1\MyProjects\wxx`
>
> 流水线：pm-wxx → dev-refactor-wxx → qa-regression-wxx → reviewer-audit-wxx → leader-wxx 汇总

## 1. 最终结论

# **NO-GO**

已完成多轮 vOPC 整改：学院准入、S0 草稿、旧文本直推、基础成果版本门禁、结项/异常状态机、风险治理闭环、R3 专项通道，以及 reviewer 三轮审计发现的全部可落地缺口（platform_operator 提权、版本门禁、SQL 忽略错误、TOCTOU、freeze 未统一拦截、R3 角色不可达）。但 vOPC PRD v1.0 的 P0 仍未完整实现（AI 虚拟员工、私有文件安全、里程碑完整业务门禁、生产同构验收等），整体禁止上线，禁止宣称 P0 验收完成。

当前工作树未 commit、未 push、未部署。

## 2. 流水线交付状态

| 步骤 | 交付物 | 状态 |
|---|---|---|
| PM 需求核对 | `pm-checklist.md` | 完成，33 项逐项核对 |
| 开发重构 | `refactor-notes.md` | 完成首轮及后续整改记录 |
| QA 回归 | `qa-report.md` | 完成，整体 NO-GO |
| Reviewer 复审 | `audit-report.md` | 完成（附录 B、附录 C 多轮只读复审） |
| Leader 最终核验 | 本文档 | 完成，补验复审后增量并据附录 C 收尾 |

说明：最后一轮开发专员总结任务因可用模型额度不足未产生有效回复。因此本文不依赖该回复，而以实际 `git diff`、源码检查和 leader 本地测试为证据。新增安全门禁代码已通过专项测试，但尚未经过独立 reviewer 的第二次完整复审。

## 3. 已确认完成的整改

### 3.1 学院准入与访问边界

- 正式 `/api/v1/vopc` 路由挂载后端 `CollegeAccess`。
- guest、inactive、外院、非 college scope 被拒绝。
- 前端新增 fail-closed 的 vOPC access 判断；桌面/移动入口和 `/vopc` deep link 均接入准入判断。
- 邀请创建时核验学院、账号状态和身份；接受邀请时再次查询数据库复核，身份变化不再放行。
- 私有项目及已有子资源仍按项目关系隔离。

### 3.2 S0 与阶段推进

- S0 草稿支持创建、查看、编辑、保存、补齐、提交。
- S0 提交后禁止直接编辑。
- 正式路由和前端已移除 S1-S9 文本直推通道；旧 URL 回归断言 404。
- 阶段推进必须进入 milestone submission/review 流程。

### 3.3 成果版本基础安全门禁

复审后工作树新增 `server/migrations/100_vopc_artifact_version_gates.sql` 及配套代码：

- 成果版本增加 `status`、`intended_stage`。
- 创建版本要求合法来源类型、非空安全引用、64 位十六进制 SHA-256、S2-S9 阶段绑定。
- 正式里程碑要求 1-20 个版本 ID，拒绝空数组、重复 ID、跨项目、失效、错误阶段和不符合阶段要求成果类型的版本。
- S2-S9 回归改为逐阶段创建独立版本，不再使用同一占位版本贯穿所有阶段。
- 该门禁验证的是系统内结构化元数据和 checksum 格式，**尚不能证明外部仓库 commit 或对象存储对象真实存在**。

### 3.4 权限与事务收紧

- 普通项目邀请角色白名单移除 `platform_operator`，owner/co_owner 不再能通过邀请授予平台治理角色。
- 邀请计数、邀请 CAS、成果更新时间、submission CAS、项目/里程碑推进等关键 SQL 增加 error/RowsAffected 检查。
- 新增邀请后变外院/guest/inactive 的接受失败及 invitation/member/event 原子性测试。
- 新增 platform_operator 越权邀请、重复/失效/跨阶段版本等负向测试。

### 3.5 已有业务闭环

- 任务：创建、负责人校验、验收标准、优先级、截止时间、状态机和审计基础闭环。
- 决策：创建、列表、resolve/cancel、理由和审计基础闭环。
- 邀请：创建、接受/拒绝、成员关系和审计基础闭环。
- 成果：元数据、版本、项目归属和里程碑绑定基础闭环。
- 移动端导航为 5 项，保留 vOPC 一键入口。

## 4. 验证证据

### 4.1 QA 完整回归（2026-08-22）

- `go test ./... -count=1`：PASS
- `go vet ./...`：PASS
- `go build ./...`：PASS
- `flutter test`：PASS，14 tests
- vOPC 定向 Flutter 测试：PASS
- vOPC 定向 `flutter analyze`：PASS
- `flutter build web --release`：PASS
- `git diff --check`：PASS
- 全量 `flutter analyze`：FAIL，268 个既有 info/lint，退出码 1

### 4.2 Leader 对复审后安全门禁增量的补充验证

以下命令于 2026-08-22 本地实际执行并通过：

- `go test ./internal/handler -run "Test.*VOPC|Test.*Milestone|Test.*Invitation" -count=1`
- `go test ./internal/db -run TestToMySQLVOPCMigrations -count=1`
- `go test ./pkg/app -run "Test(KeyRoutesReachable|RouteRegistrationCount|RunMigrationsCreatesSchema|RunMigrationsIdempotent)" -count=1`
- `flutter test test/vopc_access_test.dart test/vopc_provider_test.dart`：8 tests passed
- vOPC 相关文件定向 `flutter analyze`：No issues found
- `git diff --check`：PASS

验证边界：上述补验可证明相关测试、迁移静态转换和当前代码契约成立，不替代真实 MySQL/Turso、对象存储、模型调用、多端真机或独立安全复审。

## 5. 仍阻断上线的 P0

### P0-1 AI 虚拟员工闭环缺失

仍无可验收的 AI task/output/review/context/usage/cost 数据模型和 API/UI；没有真实模型调用、项目级上下文隔离、人工接受/修改/退回/否决、额度、超时重试和成本一致性闭环。

### P0-2 风险治理 — 已闭环（本批），残留边界已记录

已完成：风险创建/列表、R2 双人审批 gate（单人 open、双人 approved、重复 409、未批里程碑 409、批后 201）、R3 独立专项通道（platform_operator ∧ 治理系统角色、双专项审批、任一 reject 封死，未降级 R2）、freeze/unfreeze、appeal/resolve、统一写门禁（risk_frozen 下 CreateArtifact/Version/Risk/Close 均 409）、TOCTOU 修复（SubmitMilestone 与 ReviewMilestone pass 前同源复核）。
残留边界（reviewer 附录 C 指出，非本批引入）：`platform_operator` 项目角色本身仍无 provisioning API，R2/R3 共享该既有边界；正式里程碑的 TOCTOU 回滚测试未断言 submission 回滚为 pending。

### P0-3 私有文件闭环缺失

当前仍主要是 `source_ref` 元数据；没有受控上传、私有对象 key、病毒/类型/大小扫描、鉴权下载、短时签名、归档删除及跨项目下载负向验证。SHA-256 格式门禁不等于文件真实性验证。

### P0-4 结项与异常状态机 — 已闭环（本批）

S9 pass 不再直接 completed，改入 `closeable`，需 CloseProject（结项理由、失败证据、风险处置、成果包/复盘要点、人类结项决策）才落 `completed`；pause/resume/pivot/terminate/archive 合法状态机（CAS+审计+权限 fail-closed），pivot 重置里程碑 S0/draft。仍缺独立 retained retro 报表接口等（非阻断）。

### P0-5 里程碑完整业务门禁仍不足

本轮已显著收紧版本状态、阶段和成果类型，但仍缺 rubric/评分、conditional pass、waiver、甲方 S2/S5/S6 确认、试点/发布审批实体，以及对外部仓库/对象实际存在性的服务端验证。

### P0-6 生产同构与多端验证不足

- 真实 MySQL/Turso 升级、事务、并发、备份恢复尚未实跑。
- 320/360/390dp、WebView/真机和完整 widget/router 生命周期测试不足。
- 全量 Flutter analyze 仍非零；虽然主要是既有 info/lint，但当前全库门禁不能记为 PASS。

## 6. 风险与报告一致性说明

- `audit-report.md` 的最终 NO-GO 结论仍然成立（覆盖最新未提交工作树的严格只读复审，2026-08-22 06:49）。
- 该报告并通过独立复审确认以下旧审计项已正确关闭：`platform_operator` 邀请提权、同一占位版本贯穿 S2-S9、关键 SQL 忽略 error/RowsAffected。
- `qa-report.md` 的“里程碑真实成果版本门禁 PASS”应理解为基础结构化门禁，不应扩张为外部文件/commit 真实性已验证。

## 7. 下一批最小可执行顺序

1. 接入私有文件服务，实现受控上传、校验、鉴权下载和跨项目负向测试（当前最大 P0）。
2. 实现 AI task/output/review/context/usage 最小真实闭环，再接模型调用、额度和重试。
3. 补甲方确认、rubric、conditional pass、waiver 和试点/发布审批实体。
4. 补 `platform_operator` 项目角色的 provisioning API（消除 R2/R3 共享边界）。
5. 补正式里程碑 TOCTOU 回滚断言的 submission=pending；将新增路由纳入 TestRouteRegistrationCount 锚点。
6. 完成真实 MySQL/Turso、备份恢复、多端/WebView、P95 和 Flutter lint baseline 验收。

## 8. 签署

当前可认可为完成：学院准入、外院邀请双时点校验、S0 草稿闭环、S1-S9 旧直推移除、基础正式评审、结构化成果版本门禁、普通邀请角色提权封堵、一批事务错误处理、结项/异常状态机、风险治理闭环、R3 专项通道，以及 reviewer 附录 B/C 指出的 TOCTOU、freeze 统一拦截、R3 角色可达性三类缺口。

当前不可认可为完成：AI 执行与隔离、里程碑完整业务门禁（评分/条件下通过/豁免/甲方结构化证据）、私有文件安全与字段级隔离、生产同构验收、全量 Flutter lint 清零。

**最终判定：NO-GO；继续开发可行，但当前版本不得上线。**
