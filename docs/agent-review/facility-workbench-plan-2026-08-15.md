# 后勤服务台落地设计（2026-08-15）

> 目标：将「实验室开门关门、教室保洁卫生、热水/宿舍、环卫学习环境、图书馆借阅」等**后勤/服务保障岗**工作落地为真实、强关联、可进绩效画像的功能。
>
> 状态：**方案（待用户审）** → 审后编码。

## 一、背景与问题

用户指出：教辅角色当前只覆盖「教学辅助」（排课/毕业/考试/通知/教学日历/学生信息等 12 路由），**完全缺失后勤保障类岗位**（实验室、保洁、热水、宿舍查岗、环卫、图书馆借阅）的功能，且与「学生、蔚小芯」也无关联。

现状核实（已搜代码确认）：
- 后端 **无** lab / library / facility / clean / dorm / hotwater 相关 handler 或 service。
- 前端 **无** 这些页面；「宿舍/图书馆/门禁」仅出现在新生报到地图的**地点标签**，非运营功能。
- 绩效画像 `AggregateStaffMetrics` 只聚合 talk_records + audit_logs（谈心/排课/考试/通知/材料 + 蔚小芯调用），无后勤维度。

## 二、落地设计（最终，2026-08-15 用户确认合并方案）

### 2.1 角色：不新增 facility 角色，并入驻教辅 `assistant`

用户确认**合并方案**：不收立独立 `facility` 角色，把后勤能力直接挂到 `assistant`（教辅）角色能力面板。
- 新增 3 个能力常量：`FacilityRecordWrite/FacilityRecordRead/FacilityDashboard`（`facility.record.write/read/dashboard`）。
- 加入 `assistant` 角色 `capabilities` 列表。
- 好处：一个数辅账号能力做教务也能做后勤；绩效画像自然合并到「服务学生+蔚小芯」；不用新增角色配套基建。

### 2.2 新增数据表：`facility_records`（迁移 086）

```sql
CREATE TABLE facility_records (
  id            INTEGER PRIMARY KEY,
  role          TEXT NOT NULL,            -- 岗位类型: lab/clean/dorm/hotwater/envir/library
  title         TEXT NOT NULL,            -- 事项简述，如「实验楼A 301 开门」
  location      TEXT NOT NULL DEFAULT '', -- 地点: 实验楼A/3号宿舍楼/图书馆2楼...
  detail        TEXT NOT NULL DEFAULT '', -- 详情/数量/备注
  operator_id   INTEGER NOT NULL,         -- 登记人(后勤岗用户)
  operator_name TEXT NOT NULL DEFAULT '',
  student_id    INTEGER NOT NULL DEFAULT 0, -- 关联学生(0=无)。如借阅人/查岗对象
  student_name  TEXT NOT NULL DEFAULT '',
  occurred_at   TEXT NOT NULL,            -- 服务发生时间(ISO)
  created_at    TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  data_source   TEXT NOT NULL DEFAULT 'real' -- 全部为真实登记，非参考/编造
);
CREATE INDEX idx_facility_operator ON facility_records(operator_id, occurred_at);
CREATE INDEX idx_facility_role ON facility_records(role, occurred_at);
```

> 岗位类型枚举 `role`：`lab`(实验室开门/关门) / `clean`(教室保洁卫生) / `hotwater`(热水供应) / `dorm`(宿舍晚归查岗) / `envir`(校园环卫学习环境) / `library`(图书馆借阅管理)。

### 2.3 后端路由（`facility_handler.go` + `facility_service.go` + `facility_repo.go`）

路由挂载在现有 `assistantGroup`（教辅）下，前缀 `/assistant/facility`：

| 路由 | 方法 | 能力 | 说明 |
|------|------|------|------|
| `/assistant/facility/roles` | GET | facility.record.read | 岗位类型下拉 |
| `/assistant/facility/record` | POST | facility.record.write | 登记一条后勤服务记录（真实数据） |
| `/assistant/facility/records` | GET  | facility.record.read | 查询服务记录（可按 岗位/时间/操作人/学生 过滤） |
| `/assistant/facility/dashboard` | GET  | facility.dashboard | 后勤台看板：各岗位汇总 + 总服务量 + 关联学生数（诚实 0 填充） |

- **不瞎编**：所有记录由操作人**手动登记**（真实调用产生数据），`data_source: real`；无数据时看板**诚实空**，不伪造示例记录。
- **关联学生**：登记时可填 `student_id/student_name`（如「为张同学办理借阅」「3号宿舍 305 查岗正常」），从而把后勤服务挂到具体学生。
- **关联蔚小芯**：记录经 `audit_logs` 中间件自动落库，绩效画像把后勤作为新维度接入。

### 2.4 后包画像接入（`twin_repo.go` / `twin_service.go`）

- `StaffTwinMetrics` 增 `FacilityCount int`（该用户后勤服务记录数）。
- `AggregateStaffMetrics` 增查询：`SELECT COUNT(*) FROM facility_records WHERE operator_id = ?`（operator_id 即登记人）。
- `computeStaffDimensions` 增维度 `mk("facility", "后勤服务", m.FacilityCount, "完成后勤服务 %d 次(实验/保洁/热水/查岗/环卫/借阅)")`。
- `isStaffRole` 已含 assistant，后勤并入后自动命中，无需改动。

### 2.5 前端（教辅角色可见的后勤服务台）

新增 1 页 + 入口：`assistant_role/facility_workbench_page.dart`（登记/今日看板/服务记录 3 个 Tab），路由 `/assistant/facility-workbench`（已在 `/assistant` 鉴权组内）。

## 三、不做什么（边界）

- 不碰教学辅助既有 12 路由。
- 不做「预约/审批流」等重流程（首期只做登记 + 记录 + 看板，够用且不瞎编）。
- 不引入新部署形态；沿用现有 Go + Flutter + MySQL CI/CD 链路。
- 数据库 `.db`/备份不入 git；仅提交迁移 SQL 与代码。

## 四、交付物（审后编码顺序）

1. 迁移 `086_facility_records.sql`
2. 后端：能力常量 + `facility_service.go` + `facility_handler.go` + app.go 注册路由 + 绩效画像接入
3. 前端：`facility_workbench_page.dart` + profile/router/api 接线
4. 文档：本设计定稿 + 归档 `docs/agent-review/facility-workbench-2026-08-15.md`
5. 验证：`go build -tags fts5` + `flutter analyze` + 运行时联调（登录 facility 账号登记一条→看板可见→绩效画像后勤维度有数）
6. Git 提交 + push；部署按用户指令

## 五、风险与说明

- **账号**：需管理员在角色管理中把后勤员工设为 `role=facility`（新角色新增后权限即时生效）；现有演示账号不含 facility，如需演示可加。
- **绩效三方绑定**：facility 进绩效画像后体现「服务学生 + 蔚小芯能力」，与辅导员/教师共用同一画像体系，口径一致。
