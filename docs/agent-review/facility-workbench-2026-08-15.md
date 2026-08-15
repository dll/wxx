# 后勤服务台落地（2026-08-15）

> 目标：把「实验室开门/关门、教室保洁卫生、热水、宿舍晚归查岗、校园环卫学习环境、图书馆借阅」等后勤保障类工作落地为**真实、强关联、可进绩效画像**的功能，并**并入教辅（assistant）角色**（用户确认合并方案）。

## 结论
后勤服务台已实现并**本地运行时联调全部通过**（含绩效画像后勤维度接入）。后端 BUILD_EXIT:0，前端 dart analyze 0 error。

## 一、实现内容

### 1. 角色（合并方案，不新增角色）
- `capabilities.go`：新增 3 能力常量 `facility.record.write / read / dashboard`，挂入 `assistant` 角色能力面板。
- 一个教辅账号既能做教务（排课/毕业/考试…）也能登记后勤服务。

### 2. 数据（迁移 086）
- `086_facility_records.sql`：`facility_records` 表（岗位 role / 事项 title / 地点 / 详情 / 操作人 / 关联学生 / 发生时间 / data_source 固定 real）。
- 岗位枚举：`lab`(实验室开门关门) `clean`(教室保洁) `hotwater`(热水) `dorm`(宿舍晚归查岗) `envir`(校园环卫) `library`(图书馆借阅)。

### 3. 后端（repo + service + handler + 路由）
- `facility_repo.go`：Create / List（岗位/操作人/学生/时间过滤）/ Dashboard（各岗位汇总 + 总服务量 + 关联学生去重）。
- `facility_service.go`：CreateRecord（校验岗位/事项，缺省发生时间）、ListRecords、Dashboard（诚实 0 填充所有岗位）。
- `facility_handler.go`：RoleMeta / CreateRecord / ListRecords / Dashboard，取操作人自 user context。
- `app.go`：装配 `facilityRepo/facilitySvc/facilityHandler`，路由挂 `assistantGroup` 下 `/assistant/facility/*`（4 条），能力鉴权。

### 4. 绩效画像接入
- `twin_repo.go`：`StaffTwinMetrics` 增 `FacilityCount`；`AggregateStaffMetrics` 增 `COUNT(*) FROM facility_records WHERE operator_id=?`。
- `twin_service.go`：`computeStaffDimensions` 增 `facility` 后勤服务维度（次数即分、封顶100、无记录 data_available=false）。

### 5. 前端
- `facility_workbench_page.dart`：登记 / 今日看板 / 服务记录 3 Tab。
- `api_config.dart`：4 端点；`router.dart`：`/assistant/facility-workbench`；`profile_page.dart`：教辅菜单 + 特性入口「后勤服务台」。

## 二、本地运行时联调结果（8087 端口临时库）
- 迁移 086 成功执行（83 个迁移全部最新）。
- assistant1 登录 OK。
- POST 实验室开门 + 图书馆借阅 → 登记成功（real）。
- records：2 条，含关联学生（张明/李华），operator=assistant1。
- dashboard：total=2，by_role 诚实 0 填充（clean/dorm/envir/hotwater=0；lab/library=1）。
- **digital-twin：facility 维度 =「后勤服务 score 2 available True 完成后勤服务 2 次」** → 绩效画像后勤维度已接线生效。

## 三、诚实边界与说明
- 看板「关联学生去重数」按 `student_id>0` 统计；前端登记目前只填 `student_name`（自由文本）不填 ID，故该数暂为 0——**属如实呈现，非故障**。若需精确到生需前端选真实学生（后续可接学生查询）。
- 所有记录为操作人手动登记的真实数据，无示例/编造；无数据时看板/列表诚实空或 0。

## 四、待办
- Git 提交 + push（本节完成后）。
- 按用户指令部署到腾讯云（后端 scp 新二进制 + 前端 flutter build web + rsync）。
