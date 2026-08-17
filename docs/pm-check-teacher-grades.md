# P0-1 教师端成绩/作业入库通道 — 需求核对清单（只读）

> 核对人：pm-wxx（需求核对专员，只读，未修改任何源码）
> 日期：2026-08-17
> 范围：教师本人为所授班级/课程录入真实成绩 + 前端录入/导入 UI + 诚实空态；作业模块为可选独立小项。
> 原则：**绝不造数**。生产库 student_grades=0（诚实空），需教师上传真成绩。

---

## 0. 结论摘要（TL;DR）

1. **核心缺口确认成立**：`admin.POST("/grades/import")` 需 `CollegeUserRead`，teacher 角色父=student_union，不继承 → **教师当前完全无法调用成绩导入**。这是 P0-1 要打通的核心。
2. **已有可直接复用的范式**：`ImportMySchedule`（data_import_handler.go）就是"角色自主导入 + 后端强制归属"的现成样板，教师成绩录入应严格照此模式新增，**不要**复用在 admin 组（会破坏最小权限 + 范围校验）。
3. **范围判定是最大设计风险（阻塞项）**：**当前系统不存在可靠的 teacher→(course/class/student) 数据关联**。
   - `courses.teacher`、`course_schedules.teacher` 均为纯文本姓名（TEXT），生产无种子、无 FK；
   - 无 teacher_id/teacher_user_id 映射表；
   - `advisors` 表有工号(advisor_id)但未与 users / courses / student_grades 关联。
   - **因此"所授班级"无法从现有数据自动推导**。必须给出落地依据，否则会退化为"全库所有学生都能录"（越权）或"造一堆授课关系"（违反不造数）。
4. **权限最小改动方案**：新增 1 个 capability 常量 `teacher.grade.write`（后端加在教师能力 const 块、前端加在教师能力常量块），teacher roleNode 列表 +1 行。此改动与 D5-1 冲突风险**最小**（详见 §4）。
5. **接口形态**：新增 teacher 自主端点（挂 `/teacher` 组），**不改**现有 admin `/grades/import`。范围校验放 handler/service 内。
6. **作业模块**：不纳入 P0-1，列为 P2 独立小项（当前前后端均无 assignment 实体，工作量独立）。
7. **诚实红线**：生产 0 行时，学生端/教师端相关数据区块均诚实空，不得因补数用非真实数据填满看板。

---

## 1. 【目标/范围】教师端成绩录入的准确增量

**确认为**：教师**本人**、为其**所授班级/课程**的学生，录入/导入**最终成绩（真实数据）**，并复用幂等写入落库到 student_grades，供学生端 /study/grades 与毕业审核读取。

### 1.1 在 P0-1 范围内
- 教师端成绩录入/导入入口（前端页 + 入口卡片）。
- 教师端成绩落库写通（新端点 + 权限 + 范围校验 + 幂等复用 `UpsertGrade`）。
- 诚实空态：生产 student_grades=0 → 学生端成绩页已显示"暂无成绩"（已实证 grades_page.dart L61），教师端录入页空态+"待录入真实成绩"文案。
- 录入后的真实数据即可被现有 `ListMyGrades`/`GetGradeSummary`/`ListGradeSummaries` 读取（无需改学生读取侧）。

### 1.2 不在 P0-1 范围
- 作业模块（assignment 实体前后端均无，独立工作量 → P2）。
- 批量导入的实现形态（JSON 粘贴 vs 表格逐个录入）只影响 UI/端点入参，落库逻辑一致，属 P1 细化。
- 不改动 admin 批量导入 /grades/import（保留 admins 能力）。

---

## 2. 【现状实证】关键事实与引用（已核对，勿重查推翻）

| 项 | 现状 | 文件 |
|---|---|---|
| 成绩表 | `student_grades` 已建，`UNIQUE(user_id,course_id,semester,grade_type)`，grade_type 默认 'final' | migration 036 |
| 幂等写入 | `UpsertGrade` 先 `SELECT COUNT(*) ... grade_type='final'` 判存在，后 INSERT..ON CONFLICT DO UPDATE | data_import_repo.go L33 |
| 批量导入接口 | `ImportGrades`，校验 user_id/course_id 非空、单次≤2000、导入结果 Total/Created/Updated/Errors | data_import_handler.go L23 + phase3_service.go |
| 路由+权限 | `admin.POST("/grades/import", RequireCapability(CollegeUserRead))`，**admin 组** | routes.go L421 |
| 教师无权限 | teacher.roleNode parents=["student_union"]，不继承 college_admin/counselor 的 CollegeUserRead | capabilities.go |
| 学生读取 | `ListMyGrades`(L171)/`GetGradeSummary`(L241) 按 user_id 过滤；GET /study/grades | education_handler.go |
| 前端学生成绩 | grades_page.dart 空态 "暂无成绩"；grade_growth_page.dart 为里程碑模板（非 student_grades 数据驱动） | 前端 |
| 前端导入 | 仅 admin/data_import_page.dart（JSON 粘贴，管理员） | 前端 |
| 作业实体 | 前后端均无 assignment 实体 | — |
| 生产数据 | student_grades=0（诚实空） | leader 实证 |
| 能力下发 | `GET /user/capabilities` 经 `CapabilitiesOf(role)` 继承计算返回；前端 `CapabilityUtils.has()` 读 Storage.capabilities，**无 Dart 端角色→能力硬 map** | auth_handler.go + capability_utils.dart |

### 2.1 已有角色自主导入范式（关键参照）
`ImportMySchedule`（data_import_handler.go）：
- 挂 `secured` 组，路径 `/student/schedule/import`（routes 中对应 student 自主接口）。
- **后端强制归属**：`for _, s := range req.Schedules { s.UserID = userCtx.UserID }`，忽略请求携带的 user_id，杜绝越权。
- 复用同一 `phase3.ImportSchedules` 落库。

> P0-1 教师成绩录入应**严格克隆此模式**：`ImportMyTeacherGrades` 强制 `CourseID ∈ 教师所授集合`（而非强制 user_id，因为成绩要写多个学生）。

---

## 3. 【教师身份与"所授班级/课程"判定 — 数据依据核对（重点）】

### 3.1 结论：现有数据**不足以**可靠判定"所授班级"，必须设计落库依据
已核对的所有候选数据源：
- `course_schedules.teacher`：纯文本教师姓名（TEXT），由学生自己导入课表时填（ImportMySchedule），**无种子、无规范约束、无人保证=教师实名**。不可作为权威依据。
- `courses.teacher`：TEXT 姓名，migration 036 建表但**无 INSERT 种子**（核对了 036，`courses` 仅 CREATE TABLE，无数据行），生产为空。
- `advisors.advisor_id`：工号字段，但仅存在于毕业设计模块（graduation_topics），未关联 users/courses/student_grades。
- 无任何 `teacher_id`/`teacher_user_id`/`course_teacher` 映射表存在。
- `users`：teacher1→王老师（007_seed_users.sql L34），即用户名=teacher1、display_name=王老师，但这两者均未在任何课程/课表表里被引用为关联键。

**风险**：若直接按 `course_schedules.teacher == user.DisplayName` 或 `== user.Username` 匹配，结果是 0 条（生产无课表种子），或靠学生随手填的姓名匹配而误判/漏判，属"伪范围校验"。若不作任何校验，则教师可写任意学生成绩 = 越权，不可接受。

### 3.2 落地依据（二选一，建议 B）
- **方案 A（推荐，最贴合不造数）**：教师录入时**由老师在前端显式声明"本课程ID + 学生学号集合"**，后端仅做**能力门控 + 数据合法性校验**（user_id 存在、course_id 存在、学分为真），**不做 teacher→course 硬关联校验**（因为该关联数据尚不存在，强行校验会卡死真实录入）。「所授班级」的责任与事实由教师自己在前端选定课程/学生，后端记录 `created_by=teacherUserID` 便于审计与后续溯源。
  - 优点：不造任何授课关系数据、不依赖坏课表文本、开发量最小、可 today 落真成绩。
  - 缺点：信任教师自律声明课程（无程序级"只能授自己课"硬约束）。
  - **缓解**：可加 soft 提示 + 审计 `created_by`，后续有权威授课表后升级为强校验。
- **方案 B（更严，但需造/补授课关系数据）**：新增 `teacher_courses(teacher_user_id, course_id, semester)` 表 + 种子/管理入口，确立权威映射，再强校验。这属于**新增数据表 + 数据治理**，工作量更大，且种子本身需真实来源（避免造数），应单独立项。

> **建议 P0-1 走方案 A**，把"权威授课关系建库 + 强校验"列入 P1/P2（依赖校方提供真实授课清单）。这既满足"教师本人录入真实成绩、不过度越权"，又不违反"绝不造数"。

---

## 4. 【权限设计（关键）】最小改动方案

### 4.1 建议：新增 1 个 capability + teacher roleNode 加 1 行
```
TeacherGradeWrite Capability = "teacher.grade.write" // 教师录入所授班级成绩
```
- **常量加在哪**（避开 D5-1）：D5-1 的改动位于 College 能力 const 块末尾（`GovTicketManage/Assignee`）与 teacher roleNode 的 capabilities 列表尾部（`GovTicketAssignee`）。我们：
  - 后端常量：加在「教师能力 const 块」内部（capabilities.go L121-132，即 `TeacherCommunityQA` 之后），**该块 D5-1 未触碰** → 0 冲突。
  - 前端常量：加在「教师能力常量块」内部（capability_utils.dart L71-80，`teacherCommunityQa` 之后），D5-1 改的是 College 块（L101 附近）→ 0 冲突。
  - teacher roleNode：在 capabilities 列表里加一行 `TeacherGradeWrite,`。D5-1 也在同一列表尾部加了 `GovTicketAssignee,` → **同区追加，行型冲突可能**，但两者都是 append 到切片，git 通常可自动合并（不同行）。**最小化建议**：加在列表首部教师能力集中处（`TeacherGrading,` 之后），不与 D5-1 的尾部 append 撞同一上下文块。

### 4.2 不推荐方案及理由
- **复用 `TeacherGrading`（teacher.grading）**：该能力已被"作业批改辅助(AI mock)"占用（teacher_handler.go Grading），语义是 AI 出题/批改，不是成绩落库写权。复用会造成语义混乱。**但不新增也可接受**（若想零新常量），只是职责不清，非首选。
- **复用 `self.study.write`**：那是学生写自己学业数据的能力，授予 teacher 会让 teacher 以为自己能写个人学业，语义错位，排除。
- **改造/复用 admin `/grades/import` + 改路由要求 `RequireAnyCapability(CollegeUserRead, TeacherGradeWrite)`**：
  - 缺点 1：挂 admin 组，teacher 虽可调但进入 admin 命名空间，且无处做 teacher 范围校验；
  - 缺点 2：Range 校验需在 handler 内加 teacher 分支，侵入现有 admin 逻辑；
  - 缺点 3：前端 gateway/权限语义混乱。
  - **不推荐**。独立 teacher 端点更干净。

### 4.3 双端同步
- 后端：`capabilities.go`（常量 + default roleNode teacher 行）+ `routes.go`（teacher 组新路由）。
- 前端：`capability_utils.dart`（teacher.grade.write 常量）+ router Dart route + home_page 工作台入口（`CapabilityUtils.has(Capability.teacherGradeWrite)` 门控）。
- 前端能力清单来自 `GET /user/capabilities` 的继承计算，**新增 roleNode 项后自动下发**，无需额外配置。

---

## 5. 【接口形态】新增 teacher 自主端点（推荐）

### 5.1 推荐：新增 `POST /teacher/grades/import`
- 挂 `secured.Group("/teacher")`，`RequireCapability(TeacherGradeWrite)`。
- handler：`teacher_handler.go`（或 data_import_handler）+ 复用 `phase3.ImportGrades` 落库。
- 入参：`{ semester, class_id?, grades:[{user_id, course_id, course_name?, score, gpa, passed, credits}] }`（可选用 class 分组简化）。
- **强制归属/范围（方案 A）**：handler 内
  1. 校验当前用户 role/能力（中间件已做）；
  2. 逐条校验 user_id 对应的用户存在且为 student 角色（防捞非学生）；
  3. 若要求 course 强校验（走方案 B）则查 teacher_courses；方案 A 只校验 course_id 存在；
  4. 写入 `student_grades` 时记录审计 `created_by=teacherUserID` / 或写入 trace/audit 日志。
- 端点不要求 teacher 具备 CollegeUserRead，仅 `TeacherGradeWrite`。

### 5.2 权衡对比
| 方案 | 优点 | 缺点 |
|---|---|---|
| **A（推荐）教师自主端点** | 最小权限、职责清晰、可扩范围校验、不动 admin | 需新增端点+路由 |
| B 扩展现有 admin /grades/import | 复用既有 handler | teacher 混入 admin 空间、范围校验难做、侵入性强 |
| C 复用 ImportMySchedule 模式改名 | 范式一致 | 成绩要写多个学生，与"强制本人 user_id"冲突，需新写 handler |

> 结论：**方案 A**。路由挂在 `/teacher` 组，与现有 teacher 端点一致（daily-overview/lesson-prep/heatmap…）。

---

## 6. 【作业模块】可选独立小项（P2）

- **当前无 assignment 实体**（前后端均无），做作业=新增表（student_assignments 含 user_id/course_id/assignment_meta/status/score/grade…）+ 教师布置/批改 + 学生提交/查看 + 落库 + UI，**是一个独立完整模块**，不应塞进 P0-1。
- **建议**：P0-1 只做成绩；作业单列 P2（含 DDL migration、handler、前端两扇、权限可复用 teacher.grade.write 或新 teacher.homework.write）。
- 若 leader 想 P0-1 带一点作业雏形，最小可行=加 `assignment_grades` 表 + 教师录入作业成绩（复用同样的 teacher 录入 UI 骨架），但仍建议独立排期。

---

## 7. 【前端】教师端录入页 + 诚实空态

### 7.1 页面
- 新增 `frontend/lib/pages/teacher/teacher_grade_entry_page.dart`（或 teacher_grades_import），风格**复用 data_import_page.dart**（JSON 粘贴示例 + 结果反馈 Total/Created/Updated/Errors）。
- 两种录入方式，推荐**先 JSON 粘贴**（与 admin 一致，改动最小）：
  - **方式 1（P1 首选）**：JSON 粘贴，示例模板含多个学生学号+课程+分数，POST 到 `/teacher/grades/import`。
  - **方式 2（可选增强）**：表格逐个录入（选课程→填学生学号/分数列表→提交），UI 友好但工作量更大。
- 路由：`/teacher/grades-entry`（router.dart）。
- 入口：home_page `_buildRoleWorkbench` 加 `if (CapabilityUtils.has(Capability.teacherGradeWrite)) entries.add(...'/teacher/grades-entry')`。

### 7.2 诚实空态
- **教师端录入页**：首次进入提示「本课程暂无已录入成绩。请在下方录入您所授班级学生的**真实期末成绩**。」
- **数据看板联动**：production student_grades=0 时：
  - 学生端 grades_page.dart 已空态"暂无成绩"（已有，无需改，保留即诚实）。
  - grade_growth_page.dart 是里程碑模板（非 student_grades 驱动），天然诚实，不改。
  - 任何"成绩热力图/学情汇总"若依赖真实成绩且数据为 0，必须**诚实空**（不因补数填满）；若这些看板是 mock（如 teacher_handler.go DailyOverview/Heatmap 返回硬编码），需在 P0-1 明确**标记 data_source=fallback/演示，不得伪装为真实成绩统计**，避免误导。

### 7.3 门控
- 前端：`CapabilityUtils.has(Capability.teacherGradeWrite)` 控制入口卡片 + 路由可达。
- 后端：route 的 `RequireCapability(TeacherGradeWrite)` 双重保险。

---

## 8. 【诚实红线】不得因造数据填满

明确约束（写成可验收的核对项）：
1. 本次**只建"录入通道 + UI + 空态"**，不预置任何假成绩种子。
2. student_grades 保持 0 行，直到老师真实录入；学生端/教师端相关区块保持诚实空。
3. 任何 AI/mock 回退（如 heatmap/daily-overview）不得把 fake 数据写进 student_grades，也不得标成真实统计数据。
4. 不新增"演示学生 + 演示成绩"种子（除非专门的历史数据迁移 + leader 批准）。

---

## 9. 【落地拆解】开发改动项（dev）

| 优先级 | 改动项 | 涉及文件 | 依赖 |
|---|---|---|---|
| **P0** | 新增 capability 常量 `TeacherGradeWrite`（后端教师 const 块） | server/internal/auth/capabilities.go | — |
| **P0** | teacher roleNode 加 `TeacherGradeWrite` 行 | 同上（列表教师能力段） | — |
| **P0** | 前端常量 `teacherGradeWrite`（教师块） | frontend/lib/utils/capability_utils.dart | — |
| **P0** | 新增 `POST /teacher/grades/import`（ReqCapability TeacherGradeWrite）+ handler（强制学生角色校验/审计 created_by，方案A） | routes.go + teacher_handler.go(或 data_import_handler.go) | P0 权限 |
| **P0** | DDL（仅当走方案B或作业）：teacher_courses / assignment 表 | 新 migration | 若选 |
| **P1** | 教师端录入页（复用 data_import 样式）+ 路由 + 入口卡片门控 | 新 frontend page + router.dart + home_page.dart | P0 后端 |
| **P1** | 学生/教师看板诚实空态核对 + mock data_source 标注 | grades_page / 各 teacher 看板 / provider | — |
| **P1** | 单测：UpsertGrade 幂等 + 新端点范围校验/越权拒绝 | data_import_repo_test.go + 新 handler_test | P0 |
| **P2** | 作业模块（独立） | 新表+handler+双端 | 独立 |
| **P2** | 权威授课关系 teacher_courses 强校验（方案B 升级） | 新 migration + 范围校验 | 校方真实清单 |

**优先级与依赖**：P0 是打通通道的最小集（权限+后端端点），全部 P1 依赖 P0；P1 让教师可用 UI 真实录入；P2 为可选扩展，互不阻塞。

---

## 10. 【风险与兼容】

1. **与 D5-1 共享文件冲突（核心风险）**
   - 冲突文件：`server/internal/auth/capabilities.go`、`frontend/lib/utils/capability_utils.dart`（D5-1 已 M）。
   - 最小化策略：新常量只加在**教师能力块**（D5-1 改的是 College 块与 roleNode 尾部 append）；teacher roleNode 新增行加在列表教师能力段（`TeacherGrading` 附近），**避开** D5-1 在尾部加 `GovTicketAssignee` 的同一上下文。
   - **协作建议**：与 D5-1 代理约定——本次不碰 College 常量块、不碰 roleNode 尾部；D5-1 不碰教师能力 const 块。提交前 `git diff` 复核。
2. **范围判定的数据缺环**：见 §3。方案 A 以"教师前端声明 + 审计 created_by"落地，杜绝越权写任意人；方案 B 升级前不得伪装强校验。
3. **测试**：新增单测需建 `student_grades` + `course_schedules` + `users` 测试表（参照 data_import_repo_test.go 的 `setupDataImportTestDB`）。用例：首次新增 created=true、二次更新 created=false、空 user_id/course_id 报错、非学生 user_id 拒绝。
4. **SQLite/MySQL 兼容**：`UpsertGrade` 已用 `dbutil.AdaptForDriver` 适配 ON CONFLICT，新端点直接复用该方法即可；新 DDL（若走方案B/作业）需用同款 Adapt 或在两条 driver 上都验证。`course_schedules`/`student_grades` 均已在测试库有两即通验证。
5. **前端无 role→cap map**：只需新增常量 + 后端 roleNode 下发，无需前端硬编码角色授权，兼容性好。

---

## 11. 待 leader 拍板的关键决策点

1. 范围判定走**方案 A**（教师声明+审计，零新表，最快落真成绩）还是**方案 B**（建 teacher_courses + 强校验，需真实授课清单）？→ **强烈建议 P0-1 走 A**。
2. capability 用**新增 `teacher.grade.write`** 还是接受复用 `teacher.grading`（零新常量）？→ 建议新增，职责清晰。
3. 录入 UI：P1 先做 **JSON 粘贴** 还是表格逐个录入？→ 建议先 JSON（最小改动，与 admin 一致）。
4. 作业模块是否 P2 独立排期（不占 P0-1）→ 建议是。

---

*核对完成。本清单基于对上述后端/前端/迁移/权限/测试文件的实际通读，未修改任何源码。*
