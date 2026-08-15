# 课表相关修改审核结论

- 日期：2026-08-15
- 审核范围：本次会话所有课表相关代码/脚本改动（前端+后端+导入工具）

---

## 审核结论：通过 ✅（无阻断性问题）

### 1. 导入工具 server/scripts/import_schedules.py（f028e71 / 4f45fe3）
- ✅ **只从 xlsx 读账号/姓名，不编造**（教师工号/姓名、学生学号/姓名/班级）
- ✅ 密码=工号/学号（bcrypt，rounds=10，与 Go bcrypt 兼容）
- ✅ 已存在账号跳过不改密码（幂等）
- ✅ INSERT OR IGNORE 写课表，可重复运行
- ✅ 解析 100% 准确验证：63 教师 + 30 班级有课表全解析；周次(1-16/:odd/:even/2-5,8)与地点精确；班级课表无地点留空不编造
- ✅ 自动发现《班级》学生名单.xlsx 模板（班级取自文件名）
- ⚠️ 注意点（非阻断）：
  - `owner_id='cs'` / `semester_code='2026-2027-1'` 在脚本内硬编码，与当前学院(计算机学院)一致；换学院需改
  - xlsx 密码在脚本运行期内存中，不入库明文

### 2. 后端角色化导入（12b5593 / cb0b974）
- ✅ `ImportMySchedule` 强制 `user_id=当前登录`，杜绝越权改他人课表
- ✅ `BatchScheduleImport(college.schedule.import)` 授予 student_union（向上级联 counselor/teacher/assistant/admin）；**普通 student 不含**，只能导入本人
- ✅ `/admin/schedules/import`（批量）用该能力；`/student/schedule/import`（本人）用 SelfProfileWrite
- ✅ go build ./... 通过

### 3. 前端（my_schedule_import_page / data_import_page / profile / router / api）
- ✅ 学生个人中心"导入我的课表"页（示例+字段说明+节次时段对照）
- ✅ 批量导入页"课表字段说明"帮助面板
- ✅ 前端入口按能力显示；flutter analyze 零 error 零 warning

### 4. 数据可用性（"所有用户可查"）
- ✅ accounts：teacher 195、student 508（9 个有名单班）
- ✅ 课表：教师 361 条 + 学生 ~10510 条（共 10870），semester=2026-2027-1
- ✅ **前端可查**：resolveCurrentCalendar 当前日期不在学期区间时，fallback 取未来最近学期(2026-2027-1)，故导入课表可被首页今日课表/我的课表按 user_id 查到（假期也能查）

### 5. 遗留/待办
- 未导入 37 班（21 级~25 级+2 研班）：21 班有课待填名单（模板已生成 data/roster_templates/，脚本自动发现）、16 班空课表无需导入
- 学生账号默认密码=学号（告知用户，建议首次登录改密；系统有改密能力）

---

**总评**：改动正确、安全、与现有系统一致；权限角色化到位；数据校验通过；无阻断问题。可合入。
