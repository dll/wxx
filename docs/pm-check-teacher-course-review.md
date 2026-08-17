# 需求核对清单：R3 越权边界升级 — 教师授课关系强校验（teacher_courses 申报 + 教辅审核）

- 核对专员：pm-wxx（只读核对，不落任何源码）
- 日期：2026-08-17
- 版本：v1.0
- 相关前置：`docs/pm-check-teacher-grades.md`（方案 A，P0-1/R1）、`server/migrations/092`（created_by）、`093`（updated_by）

---

## 0. 结论摘要

R3 目标成立且可实现，全程沿用成熟先例 `OutcomeReview`（毕业去向：申报→教辅审核→计入）的「状态+审阅人+时间」审核流范式，不造权威关系、不改既有毕业去向审核逻辑。要点：

1. **新增表 `teacher_courses`（迁移 094）**：teacher_id 外键→users、course_id、semester、status 状态机，UNIQUE(teacher_id, course_id, semester)。
2. **复用能力 vs 新增能力**：审核能力建议**新增** `teacher.course.review`（授给 assistant/counselor/college_admin，走线性继承；不给 teacher 以免自审），申报用已有 `TeacherGradeWrite`。双端（capabilities.go ↔ capability_utils.dart）同步最小化。
3. **接线点**：`Phase3Service.ImportTeacherGrades`（phase3_service.go L55）+ `DataImportRepo.UpsertGrade` 写库前插强校验。
4. **过渡语义**：teacher_courses 全新增；强校验自落地起对新录成绩生效，历史已录成绩（现 0 行）保留不回溯拒绝。
5. **共享文件风险**：capabilities.go / routes.go / deps.go 与 D5-3 共享，改动最小化（新增能力常量 + 新增路由块 + deps 加字段，不触碰存量）。
6. **落地拆解**：P0(表+能力+申报/审核端点+角标)、P1(成绩强校验接线)、P2(前端申报+审核页)。

---

## 1. 实证基础（已读文件与结论）

| 文件 | 关键结论 |
|---|---|
| `server/internal/auth/capabilities.go` | **teacher 已有 `TeacherGradeWrite`**；assistant/counselor/teacher 三者平级（parents=student_union），均已有 `OutcomeReview`；college_admin 多父继承三线故自带 OutcomeReview 一切能力。`TeacherGradeWrite` 现仅有 teacher 角色直接持有。 |
| `server/internal/service/secretary_outcome_service.go` | `ReviewOutcome(ctx,id,reviewerID,reviewerName,status,note)` 服务层范式；`CountPending` 角标。 |
| `server/internal/repository/secretary_outcome_repo.go` | `GraduationOutcome` 状态字段 `pending/approved/rejected`；`ReviewOutcome` 仅允许 approved/rejected，落 reviewed_by/reviewed_name/review_note/reviewed_at；`CountPendingOutcomes` 供角标。**这是 teacher_courses 的直接复用范式。** |
| `server/internal/service/phase3_service.go` L55 | `ImportTeacherGrades` 现做：学号/课程非空、0-100 分、passed 一致性、target=student、created_by/updated_by=当前教师。**强校验在此写库前插入。** |
| `server/internal/repository/data_import_repo.go` | `GradeRow`{user_id, course_id, course_name, semester, score, gpa, passed, credits, created_by, updated_by}；`UpsertGrade` 幂等键 UNIQUE(user_id, course_id, semester, grade_type='final')。强校验读 semester/course_id/creator。 |
| `server/migrations/036` | `courses` 表：`teacher` 为纯文本姓名，**无 user_id 外键**（确认结构前提缺失）；`course_id` TEXT UNIQUE 是稳定键。`student_grades` 幂等键 UNIQUE(user_id, course_id, semester, grade_type)。 |
| `server/migrations/037` | `course_schedules`：`user_id` 注释为**学生**（非教师），`teacher` 也是纯文本；`semester_code` 字段。确认 `course_schedules.user_id` 不能当教师权威键。 |
| `server/migrations/092/093` | 仅加列（created_by/updated_by）+ 索引，脚本幂等，方言仅 ADD COLUMN。新表 094 应保持同级保守。 |
| `server/pkg/app/routes.go` L642-666 | teacher 组已有 `/teacher/grades/import`、`/teacher/grades/mine`。审核端点挂在 `assistantGroup`（L692-696，outcome/review 范式）或 teacher 组（UI 归属）。 |
| `server/pkg/app/deps.go` / migration.go | deps 收敛依赖（新增 handler 只需在 deps 加字段 + app.go 装配 + routes 使用）；迁移目录自动嵌入按文件名排序，加 `094_*.sql` 即自动执行。 |
| `frontend/lib/pages/secretary/outcome_manage_page.dart` | **审核 UI 范式**：`_canReview`(Capability.has) + `_buildPendingBar`(待审角标) + 登记表单 + 列表，tab 切换 pending/approved。 |
| `frontend/lib/pages/teacher/teacher_grade_entry_page.dart` | 录入页接线点：JSON 导入 + “我已录入”列表 + 诚实空态。入门控 `teacher.grade.write`。 |
| `frontend/lib/utils/capability_utils.dart` | 前端能力常量单一来源，与 capabilities.go 同步；新增能力需此处同步一条常量。 |

---

## 2. 表设计核对：`teacher_courses`（迁移 094）

### 2.1 建议建表（迁移 `server/migrations/094_teacher_courses_review.sql`）

```sql
-- 094_teacher_courses_review.sql — 教师授课关系申报+教辅审核（R3，2026-08-17）
-- 升级方案A「声明即授权」为「申报→教辅/教务审核确认→确认后才能录入对应课程成绩」强校验。
-- 对称于 graduation_outcome 审核流（pending/approved/rejected + reviewed_at/by/name/note）。
CREATE TABLE IF NOT EXISTS teacher_courses (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    teacher_id    INTEGER NOT NULL,              -- 外键→users.id（教师账号，唯一权威键）
    course_id     TEXT    NOT NULL,              -- 外键语义→courses.course_id（TEXT 稳定键）
    course_name   TEXT    NOT NULL DEFAULT '',   -- 冗余展示名（对齐 student_grades.course_name），非权威
    semester      TEXT    NOT NULL,              -- 对齐 student_grades.semester 口径
    status        TEXT    NOT NULL DEFAULT 'pending',  -- 状态机：pending/approved/rejected
    created_by    INTEGER NOT NULL,              -- 申报人=教师本人 user_id
    reviewed_by   INTEGER NOT NULL DEFAULT 0,    -- 审核人 user_id（0=未审核）
    reviewed_name TEXT    NOT NULL DEFAULT '',   -- 冗余审核人姓名展示
    review_note   TEXT    NOT NULL DEFAULT '',   -- 驳回/通过说明
    reviewed_at   TEXT,                          -- 审核时间 ISO
    created_at    TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at    TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
    UNIQUE(teacher_id, course_id, semester)      -- 幂等键：同教师同课程同学期仅一条
);
CREATE INDEX IF NOT EXISTS idx_teacher_courses_status    ON teacher_courses(status);
CREATE INDEX IF NOT EXISTS idx_teacher_courses_teacher   ON teacher_courses(teacher_id, semester);
```

### 2.2 核对项

- [x] **teacher_id 外键→users**：这是「权威教师映射」基石。注意 `courses.teacher` 是纯文本姓名，`course_schedules.user_id` 是学生——两者都不能作权威键，必须新开 `teacher_id→users.id`，且 `users.role='teacher'` 为权威性前提（申报时校验）。
- [x] **course_id 用 `courses.course_id`（TEXT）**：与 `student_grades.course_id` / `course_schedules.course_id` 同口径，强校验时按 course_id 匹配即通。关系：teacher_courses 是新增的权威「教师↔课程」映射表，独立于 courses（教师姓名）/ course_schedules（学生课表），不替换它们，作为成绩强校验的判据表。
- [x] **semester 口径**：对齐 `student_grades.semester`（如 `2025-2026-2`）。强校验按 semester 精确匹配。
- [x] **uniq 键**：`UNIQUE(teacher_id, course_id, semester)`——同教师同学期同课程仅一条申报，天然幂等/去重。**全小写/全大写冲突**：建议写入前 normalize（trim + 统一 case），避免 `CS101`/`cs101` 双记录。
- [x] **方言兼容**：本表仅 `CREATE TABLE`/`CREATE INDEX`/`datetime('now','localtime')`，SQLite/MySQL 均兼容；不引入 `ON CONFLICT`（保持与 093 同级保守）。若需幂等申报建议用「先查后插事务」而非 UPSERT（MySQL 方言差异）。
- [x] **迁移编号**：当前最新 093 → **建议 094**。migration.go 自动嵌入按文件名排序执行，无需手改清单。
- [x] **与 092/093 兼容**：student_grades 已有 created_by/updated_by；teacher_courses 申报即改「强校验」判据，092/093 审计列保留，不改幂等键。

---

## 3. 状态机核对

参照 `GraduationOutcome`：`pending → approved / rejected`，评审字段齐全。

| 状态 | 含义 | 能否录入成绩 | 触发 |
|---|---|---|---|
| `pending` | 教师已申报，待教辅/教务审核 | 否（拒绝并提示待审核） | `SubmitTeacherCourse` 申报 |
| `approved` | 教辅/教务确认授课关系 | 是（强校验放行） | `ReviewTeacherCourse(status=approved)` |
| `rejected` | 驳回 | 否（提示先重新申报/联系教辅） | `ReviewTeacherCourse(status=rejected)` |

### 核对项
- [x] **幂等/重复申报**：同 (teacher_id, course_id, semester) 已有 `pending` → 拒绝重复申报并提示「待审核中」；已有 `approved` → 返回「已通过」不重复；已有 `rejected` → 允许重新申报（置回 pending）或更新走重审。
- [x] **改课处理**：教师不可自行改已 approved 记录（改课=撤回重新申报，权限收紧）。驳回后可改学期间重新申报。
- [x] **状态取值命名**：复用 `pending/approved/rejected` 与 graduation_outcome 完全一致，前端可复用 tab/徽标逻辑。
- [x] **时间戳**：`reviewed_at` 用 `datetime('now','localtime')`（对齐 graduation_outcome），审核即真实操作留痕。

---

## 4. 权限核对（双端最小化同步）

### 4.1 能力建议

| 动作 | 建议能力 | 持有角色 | 理由 |
|---|---|---|---|
| 教师申报授课关系 | 复用 `TeacherGradeWrite`（已有） | teacher（唯一） | 申报=声明所授课程，与录入成绩同属教师侧门控，无需新增 |
| 教师查询本人申报/待审 | 复用 `TeacherGradeWrite` | teacher | 同申报 |
| 教辅/教务审核 + 待审角标 | **新增 `TeacherCourseReview` = `teacher.course.review`** | assistant / counselor / **college_admin**（继承） | 审核是新增职责，不应让 teacher 自审，也不应白拿 counselor；授给教辅链 |

> **为什么不复用 `OutcomeReview` / 造 `teacher.course.review` vs `assistant.course.review`？**
> 建议命名 `teacher.course.review`（对齐 domain=teacher 业务族，与 `teacher.grade.write` 同域），**不建议 `assistant.course.review`**，因为 counselor 也是教辅审核方（毕业去向为 counselor/teacher/assistant 三者教辅共同审核），用 teacher 域让 teacher 也天然可被理解为核心。最终以 leader 拍板为准，本清单明确并列可选。

- [x] **角色归属（继承树已实证）**：`assistant`、`counselor`、`teacher` 三者平级；`college_admin` 多父继承三线→**自带到审核能力**（无需在 college_admin 重复加，靠继承）。`school_admin`→继承 college_admin，也带。若只授 `teacher.course.review` 给 assistant+counselor（不加给 teacher），则 teacher 角色**只申报不审核**，杜绝自审——推荐此默认。
- [x] **越权红线对应**：审核端点用 `RequireCapability(TeacherCourseReview)`；申报端点用 `RequireCapability(TeacherGradeWrite)`。数据范围：审核人应能看本院教师申报（按 teacher 的 owner_id 归属，可并入；若无法精确对学院则先全校教辅可审，P2 收窄）。
- [x] **双端最小化**：仅新增 **1 个能力常量**（capabilities.go 加 const + 挂到 assistant/counselor 两处 capabilities 列表；前端 capability_utils.dart 加 1 条常量 + 认证拉取自动带上）。其余能力零改动。

---

## 5. 成绩强校验接线核对（P1）

位置：`Phase3Service.ImportTeacherGrades`（phase3_service.go L55），**在 `UpsertGrade` 写库前**逐条校验。

### 5.1 校验逻辑（伪代码）

```
for each grade g:
    # 现有非空/0-100/passed/role=student 校验保留
    ok, state := repo.GetTeacherCourseStatus(g.CreatedBy(即creatorID), g.CourseID, g.Semester)
    if !ok:  → errors「课程 X 学期 Y 尚无授课关系申报，请先在"我的授课"申报，待教辅审核通过后再录入」
    elif state == 'pending' → errors「课程 X 学期 Y 授课申报待审核，确认后方可录入成绩」
    elif state == 'rejected' → errors「课程 X 学期 Y 授课申报被驳回，请联系教辅」
    # state == 'approved' → 放行写入
    UpsertGrade(g)
```

- `g.CreatedBy` 已由现有代码在写库前置为 creatorID（L78），校验用同一值保证「申报者=录入者」。
- **读取边界**：`GetTeacherCourseStatus(teacherID, courseID, semester)` 新增 repository 方法，返回是否存在 + status。

### 5.2 过渡语义（关键）

- [x] **未确认（pending/rejected/无申报）→ 强校验生效，拒绝新录**：从落地起，所有新成绩写库必须已有 approved 记录；这**停用了方案A的「声明即授权」**（不再凭前端声明即写库）。
- [x] **历史已录成绩不受影响**：现有 `student_grades` 0 行；即便未来有历史行，**不回溯拒绝/回滚**。强校验只拦截「新写入」，对已存在的 approved-course 外部历史行不删除。诚实边界：不因缺 approved 而篡改/删除任何既有记录。
- [x] **「声明即授权」停用后的兜底**：可提供「老数据一键申报→教辅批量确认」的迁移助手（P2 可选），把历史上合法声明过的课程批量落为 approved，避免误伤。但**绝不自动 approved**，必须教辅确认。
- [x] **校验失败仅该条记入 Errors，不整批回滚**（对齐现状 ImportResult.Errors 语义），前端逐条红字提示。

---

## 6. 前端核对（P2）

### 6.1 教师侧：授课申报页 + 录入页提示
- 复用 `outcome_manage_page.dart` 审核范式（`_can` 门控 + 表单 + 列表 + 诚实空态）。
- 申报页：教师选课程(course_id/course_name/semester) → POST 申报 → 展示本人申报列表（pending/approved/rejected 徽标）。
- `teacher_grade_entry_page.dart` 接线点：录入前可调「我的 confirmed 课程」接口，**未确认课程禁用/红字提示**「该课程授课关系未获确认，暂不可录入」；导入失败对应 error 文案已由后端返回。
- 诚实空态：0 申报→「暂无授课申报，请先申报并由教辅确认」；0 confirmed→提示需审批。

### 6.2 教辅侧：审核页/待审角标
- 复用 outcome `_buildPendingBar` + `CountPending` 角标逻辑。
- 审核页：列 pending 申报，按教师/学院筛选，`approve/reject + note`。
- 路由归属：监听在 `assistantGroup`（对齐 outcome/review，L692-696）或 teacher 组；最终按 UI 入口设计。

---

## 7. 诚实红线核对

- [x] **不造权威关系**：teacher_courses.approved 判定唯一来源，绝不脚本批量置 approved，更不靠 `courses.teacher` 姓名反推。
- [x] **确认=审核人真实操作**：reviewed_by/reviewed_name/reviewed_at 必须来自审核端点的真实登录用户，不能默认值填充。
- [x] **0 申报/0 确认时诚实展示**：申报页/审核页/录入页均给诚实空态，不伪造成绩或已确认关系。
- [x] **不改既有毕业去向审核逻辑**：OutcomeReview 相关 service/repo/handler/UI 全部零改动，独立新开 teacher-courses 模块。

---

## 8. 与方案A兼容核对

- [x] `teacher_courses` **全新增**，不动 student_grades 表结构（042/093 已加审计列，保留）。
- [x] 强校验**从落地起生效**；历史已录成绩保留、不回溯拒绝。
- [x] 方案A的 created_by/updated_by 审计继续有效；R3 只是在写库前加了「approved 判据」。
- [x] `TeacherGradeWrite` 能力保留（申报复用），不因 R3 移除。

---

## 9. 落地拆解（dev 执行清单）

### P0 — 表 + 能力 + 申报/审核端点（前置，无依赖）
- [ ] 迁移 `094_teacher_courses_review.sql`（§2 建表）。
- [ ] `capabilities.go`：新增 `TeacherCourseReview Capability = "teacher.course.review"`；加到 assistant、counselor 的 capabilities 列表（college_admin 靠继承；不授 teacher）。同步 `capability_utils.dart` 1 条常量。
- [ ] 新 repository `teacher_course_repo.go` + 新 service（可放 secretary_outcome 同目录 `teacher_course_service.go`）：`Submit/CreateTeacherCourse`、`ListTeacherCourses(status, teacherId, college)`、`ReviewTeacherCourse(id, reviewerID, name, status, note)`、`GetTeacherCourseStatus(teacherID, courseID, semester)`、`CountPendingTeacherCourses()`。
- [ ] 新 handler `teacher_course_handler.go`（对齐 secretary_outcome_handler 的 `outcomeRole(c)` 取 opID/opName）。
- [ ] `deps.go` 加 handler 字段 + `app.go` 装配 + `routes.go` 加路由块（§4.1 能力门控）。
- [ ] 测试：迁移可重复执行；申报幂等；审核状态机；权限路由（teacher 不能审、assistant/college_admin 能审）。

### P1 — 成绩强校验接线（依赖 P0 表/仓库就绪）
- [ ] `Phase3Service.ImportTeacherGrades` 写库前调 `GetTeacherCourseStatus`，判 `approved` 才放行（§5.1）。`DataImportRepo` 加该校验 SQL。
- [ ] 测试：无申报→拒；pending→拒；rejected→拒；approved→放行；历史行不回溯；单条错误不整批回滚。

### P2 — 前端（依赖 P0/P1 接口就绪）
- [ ] 教师授课申报页（复用 outcome 表单/列表范式）。
- [ ] `teacher_grade_entry_page.dart`：未确认课程提示 + 录入禁用。
- [ ] 教辅审核页 + 待审角标（复用 OutcomeManagePage.pendingBar / CountPending）。
- [ ] 测试：门控 teacher.grade.write / teacher.course.review 前后端一致；诚实空态文案。

### 优先级与依赖
```
P0 ──▶ P1（强校验接线依赖表+仓库）
 │
 └──▶ P2（前端依赖 P0/P1 接口）
```
**P1 是 R3 的核心价值**（越权边界收紧），可在 P0 的表+仓库 ready 后独立先行；UI(P2) 可后补。建议 P0→P1 连排在同一次发布，避免「只有申报/审核端点但成绩还没接强校验」的半成品区间暴露（此半成品期新成绩仍走旧声明即授权，可接受，属过渡窗口但应尽量缩短）。

---

## 10. 共享文件风险与最小化改动（D5-3 共存）

- `capabilities.go`：D5-3（gov_ticket）与 R3 都在此文件。R3 改动=**新增 1 常量 + 2 行列表追加**（assistant/counselor），不动已有 const/继承图。
- `routes.go`：R3 改动=**新增独立路由块**（teacher 组 + assistant 组各追加子路由），不触碰 gonv_ticket / outcome 存量。
- `deps.go` / `app.go`：仅**追加字段**（handler），不改现有字段。
- 建议：R3 与 D5-3 各自独立提交/独立 PR，冲突面最小；能力常量按字母/域归位即可。

---

## 11. 测试要求汇总（诚实 + 越权 + 幂等）

1. **越权**：teacher 调 `teacher.course.review` → 403；非本 teacher 申报他人课程 → 拒绝；审核人不可审不存在记录。
2. **状态机**：pending→approved→可录；pending→rejected→可重申报；重复申报幂等。
3. **强校验**：四分支（无/pending/rejected/approved）逐条正确；单条错误不整批回滚；历史已录行不动。
4. **诚实**：0 数据时所有端点返回真实空态/not_available 语义，无伪 approved。
5. **方言/迁移**：094 在 SQLite 与 MySQL 均可重复执行（幂等）。
6. **双端能力**：capabilities.go ↔ capability_utils.dart 常量对齐，后端 `CapabilitiesOf('college_admin')` 含 TeacherCourseReview。

---

*本文件为需求核对清单，仅供 dev/leader 落地参考；未对任何源码做修改。*
