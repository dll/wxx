# P2 教师端作业模块（轻量版）— 需求核对清单（只读）

> 核对人：pm-wxx（需求核对专员，只读，未修改任何源码）
> 日期：2026-08-17
> 范围：教师端**作业信息发布** + **成绩统计（只读，复用现有成绩数据）**。蔚小芯侧重**教育不是教学**——作业不做完整教学活动。
> 原则：**绝不造数**；作业归属必须 **approved** 授课关系（杜绝无关课程作业）；成绩统计基于真实 `student_grades`，0 行诚实空。

---

## 0. 结论摘要（TL;DR）

1. **P2 定位与边界已由用户明确定义，予以确认**：作业模块仅「①作业信息发布 ②成绩统计（可导入）」。**不做**学生提交、教师批改、作业内容/文档流转等完整教学活动。蔚小芯是教育辅助平台，不做教学管理。
2. **成绩导入通道已存在且可整体复用，无需新导入逻辑**：
   - `teacher.POST /grades/import`（`TeacherGradeWrite`，R3 强校验：仅 approved 授课课程可录）+ `GET /grades/mine`（`ListMyTeacherGrades`，按 created_by）已就绪。
   - 成绩统计仅需新增**只读**聚合查询，基于真实 `student_grades`。
3. **发布端门控最小化**：**复用现成 `TeacherGradeWrite`**，**不需要新增 capability**。与 `courses/apply`、`courses/mine`（routes.go L667-668 均已门控 `TeacherGradeWrite`）完全一致，两端（capabilities.go + capability_utils.dart）**零改动**。
4. **作业归属由 `teacher_courses` approved 授课关系强约束**：教师只能对「已确认授课（approved）」的课程发布作业，复用 `GetTeacherCourseStatus`，对称 R3 成绩强校验语义。
5. **成绩统计按「课程维度」而非「作业维度」**：作业只是「信息发布」不带成绩语义，统计对象是课程（人数/均分/及格率/分布），基于 `student_grades` 真实数据聚合。若要多作业维度，因作业无 score 关联，只能冗余展示（不推荐，见 §5）。
6. **落地分三档**：P0（表+发布端点+归属校验）/ P1（成绩统计只读查询+展示）/ P2（前端页+入口），优先级与依赖见 §7。
7. **共享文件最小化**：`capabilities.go`/`capability_utils.dart` 不改；`routes.go` 仅 teacher 组新增 3 条路由（发布/列表/统计），不与 D5-3 冲突。

---

## 1. 【表设计】`homework`（迁移 095）

**结论**：表**轻量**，仅存「信息发布 + 归属」，**无作业内容/附件/提交表**。仿 094/092/093 的方言兼容范式（`CREATE TABLE IF NOT EXISTS` + 索引，不引入 `ON CONFLICT`；时间默认 `datetime('now','localtime')`）。

### 建议 DDL（供落地参考，列为核对项而非已实现）
```sql
-- 095_homework.sql — 教师作业信息发布（P2 轻量版，2026-08-17）
-- 蔚小芯侧重教育非教学：作业仅存信息（标题/说明/时间/归属），不做学生提交、批改、内容流转。
-- 归属强约束：course_id 必须对应该教师 approved 授课关系（teacher_courses），对称 R3 成绩强校验。
CREATE TABLE IF NOT EXISTS homework (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    teacher_id    INTEGER NOT NULL,              -- 外键→users.id（发布教师，唯一权威键）
    course_id     TEXT    NOT NULL,              -- 外键语义→courses.course_id（TEXT 稳定键）
    course_name   TEXT    NOT NULL DEFAULT '',   -- 冗余展示名，非权威（对齐 teacher_courses.course_name 口径）
    semester      TEXT    NOT NULL,              -- 非权威展示口径（发布学期，对齐 student_grades/teacher_courses）
    title         TEXT    NOT NULL,              -- 作业标题（信息发布核心字段）
    description   TEXT    NOT NULL DEFAULT '',   -- 作业说明/要求（信息发布用，纯文本）
    publish_at    TEXT,                          -- 发布日期（ISO；空=使用创建时间）
    due_at        TEXT,                          -- 截止日期（ISO；模糊时间，无截止提醒流转）
    status        TEXT    NOT NULL DEFAULT 'active',  -- active/published/archived（轻量状态，非审核流）
    created_at    TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at    TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
    UNIQUE(teacher_id, course_id, semester, title)   -- 幂等键：同教师同课程同学期同标题仅一条
);
CREATE INDEX IF NOT EXISTS idx_homework_teacher ON homework(teacher_id, semester);
CREATE INDEX IF NOT EXISTS idx_homework_course  ON homework(course_id, semester);
```

### 核对要点
| 项 | 判定 | 备注 |
|---|---|---|
| 轻量：无作业内容/附件/提交表 | ✅ 必要 | `description` 为纯文本信息说明；**不建** homework_submissions / homework_attachments / homework_grades |
| 归属键 | ✅ 必要 | `teacher_id`(int,→users) + `course_id`(TEXT,→courses) + `course_name`(冗余) + `semester`，对齐 `teacher_courses` 口径，保证统计能 join |
| 幂等/索引 | ✅ 建议 | `UNIQUE(teacher_id, course_id, semester, title)` + 教师/课程索引 |
| 方言兼容 | ✅ 必须 | `CREATE TABLE IF NOT EXISTS`+`CREATE INDEX IF NOT EXISTS`，无 `ON CONFLICT`，时间默认 localtime（对齐 094/092/093） |
| 与 courses 关系 | ✅ 语义外键 | course_id 规范性复用 `normalizeCourseID`（trim+大写），不建物理 FK（对齐 teacher_courses/CourseExists 风格） |
| 与 teacher_courses 关系 | ✅ 核心约束 | 发布前必须验证该 teacher 对该 (course_id, semester) 已有 **approved** 授课关系（见 §2 门控） |

---

## 2. 【发布】教师发布作业信息端点

**结论**：新增独立 homework 服务（对齐 teacher_course 服务范式），落在 `/teacher` 组；**门控复用 `TeacherGradeWrite`**，**不新增 capability**。

### 2.1 门控评估
| 方案 | 判定 | 理由 |
|---|---|---|
| 复用 `TeacherGradeWrite` | ✅ **选此** | `courses/apply`（routes.go L667）、`courses/mine`（L668）、`grades/mine`（L665）已全部门控 `TeacherGradeWrite`，作业发布与之同属「教师本人操作所授课程」，语义完全一致；两端零改动 |
| 新增 `teacher.homework.write` | ❌ 不推荐 | 需在 capabilities.go + roles 表 + capability_utils.dart 三处同步，与 D5-3/既有共享改动叠加，违背「最小化」，收益（仅是更细语义）不抵成本 |

### 2.2 建议端点（对称 courses 范式，最小改动）
| 方法 | 路径 | 门控 | Handler | 说明 |
|---|---|---|---|---|
| POST | `/teacher/homework` | `TeacherGradeWrite` | 新增 `HomeworkHandler.PublishHomework` | 发布作业信息；**写库前强校验 approved 授课关系** |
| PUT | `/teacher/homework/:id` | `TeacherGradeWrite` | `HomeworkHandler.UpdateHomework` | 编辑（仅本人，且课程归属不变） |
| DELETE 或状态位 | `/teacher/homework/:id` | `TeacherGradeWrite` | `HomeworkHandler.ArchiveHomework` | 下架（置 archived，软删更稳，审计可溯） |
| GET | `/teacher/homework/mine` | `TeacherGradeWrite` | `HomeworkHandler.ListMyHomework` | 我的作业清单（按 teacher_id） |

### 2.3 发布强校验（对称 R3 成绩强校验）
- **归属校验**：写库前须调 `GetTeacherCourseStatus(teacherID, courseID, semester)`（teacher_course_service.go / repo），要求 `exists==true && status==CourseStatusApproved`，否则拒绝并给出诚实提示（无申报→先申报；pending→待审核；rejected→联系教辅）。
- **不造权威关系**：`teacher_id` 强制取当前登录教师（杜绝代发），与 `ImportTeacherGrades` 的 created_by=当前教师铁律一致。
- **入参校验**：`course_id` 复用 `CourseExists`/`normalizeCourseID` 校验（防虚构课程号）；`title` 必填；`semester` 必填。
- **明确不做**：学生提交、教师批改、作业内容/附件流转、学分/分值语义——都不进 homework 表。

---

## 3. 【成绩统计】复用 grades/import + /grades/mine（只读聚合，不做新导入）

**结论**：**成绩导入完全复用现有通道，不写任何新导入逻辑**。P2 只加一个**只读按课程聚合**查询端点 + 前端展示。

### 3.1 复用判定
| 项 | 判定 | 理由 |
|---|---|---|
| 成绩导入（写入） | ✅ 完全复用 `teacher.POST /grades/import` | 已含 R3 强校验（仅 approved 授课课程可录）+ 0-100/passed/student-role/power 校验，无需动 |
| 教师已录清单 | ✅ 复用 `GET /grades/mine` | `ListMyTeacherGrades` 按 created_by 返回本人录入记录，供发布页联动展示「可统计的前提」 |
| 是否按作业维度聚合 | ❌ 不推荐 | 作业仅「信息发布」，不带 score 语义；作业与成绩无一对多可信关联，按作业聚合会产生**无依据的归属**，违「不造数」 |

### 3.2 建议统计端点（只读）
| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/teacher/homework/:courseId/grade-stats`（或 `/teacher/courses/:courseId/grade-stats`） | 该课程成绩统计（人数/均分/及格率/分档分布） |

- **数据源**：`student_grades` 真实数据，`WHERE course_id=? AND semester=? AND grade_type='final'` 聚合。
- **只统计 approved 授课课程**：复用 `ListTeacherCourses(teacherID, "approved")` 作为「可统计课程白名单」；未确认课程统计返回空/拒绝，诚实口径。
- **输出建议**：`total`（人数）、`avg_score`（均分）、`pass_rate`（及格率，passed 计数）、分档分布（优秀/良好/及格/不及格四档，`gradeLevelOf` 已在 repo 存在，可对称复用）。
- **诚实红线**：`student_grades` 0 行时返回 `total=0` + 明确空态，前端显示「暂无成绩记录，需教师先录入真实成绩」，**不得**补造分布。

---

## 4. 【能力/双端】不新增 capability，两端零改动

**结论**：**不新增**。作业发布与成绩录入语义同域，`TeacherGradeWrite` 足够。

- `server/internal/auth/capabilities.go`：**不改**（作业复用 `TeacherGradeWrite`）。
- `frontend/lib/utils/capability_utils.dart`：**不改**（前端复用 `teacherGradeWrite` 判断，`CapabilityUtils.has(...)`）。
- 能力映射完整核对（已实证）：
  - teacher role：`TeacherGradeWrite` ✔（capabilities.go L308）
  - 前端：`teacherGradeWrite` ✔（capability_utils.dart）
  - 路由：`/teacher/homework*` 挂 `/teacher` 组 `RequireCapability(auth.TeacherGradeWrite)` ✔

---

## 5. 【前端】教师作业发布页 + 列表 + 该课程成绩统计卡

**结论**：新增 1 个轻量页 + 入口路由；强对齐 `TeacherGradeEntryPage`/`TeacherCourseApplyPage` 的门控、诚实空态范式。

### 5.1 新增页面与路由
| 项 | 建议 | 范本 |
|---|---|---|
| 页面 | `frontend/lib/pages/teacher/teacher_homework_page.dart` | `teacher_course_apply_page.dart`（表单+列表双层 Card）、`teacher_grade_entry_page.dart`（门控拦截+诚实空态） |
| 路由 | `/teacher/homework` 注册进 `config/router.dart`，对称 `course-apply` | `router.dart` L631 后追加 |
| 入口 | teacher 端 home/工作台对应入口卡，`CapabilityUtils.has(teacherGradeWrite)` 门控隐藏/显示 | — |

### 5.2 页面结构（建议）
1. **发布作业表单**（轻量）：课程（可从本人 approved 授课下拉）、学期、标题、说明、发布日期、截止日期。
   - 课程下拉数据源：`GET /teacher/courses/mine` 过滤 `status=='approved'`（**只给已确认授课课程**，杜绝无关课程作业）；0 条时诚实提示「先申报并经教辅确认」。
2. **我的作业列表**：`GET /teacher/homework/mine`，状态/标题/课程/时间；支持编辑、下架。
3. **该课程成绩统计卡/区块**：点击某个作业（或课程）查看 `grade-stats`——人数/均分/及格率/分档；0 行诚实空（「暂无成绩，先录入真实成绩」）。

### 5.3 诚实空态核对
| 场景 | 前端须显示 | 不得 |
|---|---|---|
| 无 approved 授课课程 | 「暂无已确认授课的课程，请先申报并由教辅确认」 | 填充虚构课程 |
| 有课程、无作业 | 「暂未发布作业」 | 伪造示例作业 |
| 有成绩、0 行 | 统计块「暂无成绩记录」+ total=0 | 补造分布/伪均分 |

---

## 6. 【诚实红线】汇总

1. **不造数据**：作业表仅教师真实发布；成绩统计仅基于真实 `student_grades`；approved 授课关系唯一来源为教辅真实审核（teacher_courses），绝不脚本批量置位。
2. **作业归属必须 approved**：发布/编辑一律经 `GetTeacherCourseStatus` 校验，杜绝无关课程作业可被发布。
3. **0 行诚实空**：`student_grades`=0 时统计 `not_available`/`total=0`，前端诚实文案。
4. **蔚小芯是教育平台不是教学平台**：作业只做信息发布+成绩统计，**不做**学生提交/教师批改/内容流转——即使技术上可行也不实现，守护产品定位边界。

---

## 7. 【落地拆解】dev 改动项

> 每项标注优先级、依赖、测试要求。全部改动不破坏 D5-1/D5-3（见 §8）。

### P0 — 表 + 发布端点 + 归属校验（核心，先落）
| # | 改动 | 文件 | 依赖 | 测试要求 |
|---|---|---|---|---|
| P0.1 | 迁移 095 `homework` 表 | `server/migrations/095_homework.sql` | 无 | 迁移可重复执行（IF NOT EXISTS）；SQLite+MySQL 建表成功 |
| P0.2 | 新 `HomeworkRepo` + `HomeworkService` | `server/internal/repository/homework_repo.go`、`server/internal/service/homework_service.go` | P0.1 | 发布成功/幂等；编辑仅本人；下架软删；`GetTeacherCourseStatus` 非 approved 一律拒绝（pending/rejected/无申报三态用例）；`CourseExists` 虚构课程拒绝 |
| P0.3 | 路由 + 接线 | `server/pkg/app/routes.go`（teacher 组）、`app.go`/`deps.go` | P0.2 | 4 端点带 `TeacherGradeWrite` 门控可调；登录校验生效 |

### P1 — 成绩统计只读查询 + 数据层
| # | 改动 | 文件 | 依赖 | 测试要求 |
|---|---|---|---|---|
| P1.1 | `student_grades` 按课程只读聚合查询 | `homework_repo.go` 或 `data_import_repo.go` 新增只读方法 | P0.2 | 含 student join、`passed` 计数、分档 `gradeLevelOf`；0 行返回 total=0 不报错；仅 approved 课程返回 |
| P1.2 | 统计端点 + service | `homework_service.go` + routes.go | P1.1 | 返回人数/均分/及格率/分布；未确认课程返回空/拒绝 |

### P2 — 前端页面与入口
| # | 改动 | 文件 | 依赖 | 测试要求 |
|---|---|---|---|---|
| P2.1 | 作业发布页+列表+统计卡 | `teacher_homework_page.dart` | P0/P1 | 门控拦截；课程下拉仅 approved；诚实空态；统计 0 行不崩 |
| P2.2 | 路由注册 | `config/router.dart` | P2.1 | `/teacher/homework` 可达 |
| P2.3 | 入口卡 + 门控 | teacher 端 home/工作台 | P2.1 | 无权限隐藏；有权限可进 |

---

## 8. 【共享文件】最小化核对

| 文件 | 是否改动 | 说明 |
|---|---|---|
| `server/internal/auth/capabilities.go` | **不改** | 复用 `TeacherGradeWrite`，与 D5-3 共享但零冲突 |
| `frontend/lib/utils/capability_utils.dart` | **不改** | 前端复用 `teacherGradeWrite` |
| `server/pkg/app/routes.go` | 仅 +3~4 条 | 只加在 `/teacher` 组 L642 之后（P0），**不改** D5-3 的工单/洞察路由（L70x/L72x） |
| `app.go`/`deps.go` | +homework 服务接线 | 对齐 teacherCourse 注入范式，不触碰 D5-3 节点 |
| `server/migrations/` | +095 | 与 094 同级，不改历史迁移 |

**D5-3 兼容**：作业模块为新增独立路由 + 独立表，不触碰工单/洞察现有链表；capabilities 零改动 → 与 D5-3 共享区域（capabilities.go/routes.go teacher 组）无冲突面。

---

## 9. 遗留待确认项（不阻塞，供 leader/用户拍板）
1. **作业统计粒度**：已定为**课程维度**（作业为纯信息，无成绩关联）。若未来产品想按「某次作业」看统计，需作业与 student_grades 建立可信归属（新建关联/作业内评分），届时再评估，当前**不做**。
2. **作业是否对「学生端」可见**：本 P2 严格限定**教师端**；学生端是否展示作业信息列表属独立范围，默认**不在本模块**（蔚小芯不做教学，学生端以成果/成绩为准）。
3. **「作业已发布」是否在教师授课概览（daily-overview）联动**：属锦上添花，非 P2 必须，可后置。
