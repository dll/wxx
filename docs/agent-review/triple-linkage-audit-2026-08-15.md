# 教师/教辅 × 学生 × 蔚小芯 三者关联强度审核（2026-08-15）

> 目标：核实「教师、教辅与学生、蔚小芯」三者关联的真实强度，找出断开点与造假风险，并落地优化。

## 结论
- **学生 ↔ 教辅(assistant/counselor) ↔ 蔚小芯**：✅ 已打通，三者真实关联并汇聚到同一绩效画像。
- **教师(teacher) ↔ 学生 ↔ 蔚小芯**：❌ **基本断开**，是三者中最弱一环，且存在一处编造学生假数据风险。
- 优化方向：接入 `course_schedules.teacher` 这一**真实师生关联源**，把教师拉进三方绑定。

---

## 一、现状核实（逐条实证）

### 1. 绩效画像如何汇聚「三方绑定」
`twin_service.go GetStaffTwin → computeStaffDimensions` 输出维度含三方绑定块：
- `bind_student`（服务学生）：=`COUNT(DISTINCT student_id) FROM talk_records WHERE counselor_id=? AND student_id>0` — **真实**（谈心绑定学生主键）
- `bind_wxx`（蔚小芯能力）：=`COUNT(DISTINCT resource) FROM audit_logs WHERE user_id=? AND resource LIKE '%/assistant/%' OR '%/counselor/%'` — **真实**（工具调用去重）
- `bind_teacher`（协作教师）：**硬编码 `Score:0, DataAvailable:false`**，注释明说「暂无独立教师协作表」

> 即：**教辅 → 学生(真实) + 蔚小芯(真实) 双向打通**，但**教师环是空的**。

### 2. 教师侧三个断点
| 断点 | 位置 | 问题 |
|---|---|---|
| **教师查看学生=编造假数据** | `teacher_handler.go StudentTwin` → `teacher_service.go GenerateStudentTwinTeaching` | 返回固定 `focus_students:[{张明 挂科风险},{李华 需辅导},{王芳 可任助教}]` + `total:45, distribution:...`，`DataSource:"reference"`。若被当真实学生=**严重造假**（虚构的学生挂靠真实教学场景） |
| **协作教师维度无真实源** | `twin_service.go L351` | `bind_teacher` 永远空，无真实师生关联表可聚合 |
| **蔚小芯使用漏 teacher** | `twin_repo.go AggregateStaffMetrics` | `WxxUseCount/WxxBindCount` 只统计 `%assistant/%`+`%counselor/%`，**漏掉 `/teacher/*` 路由**——教师用备课本/评阅/答疑等工具不计入 |

### 3. 真正存在的师生关联源（未被利用）
- `course_schedules` 表有 `teacher`（任课老师名）+ `course_id/user_id`：本地实测 **10 条排课、5 位任课老师、绑定 1 学生**。
- `course` 表有 `teacher` 字段（课程任课老师）。
- 这是**真实「学生↔教师」关联**，只是从没接进三方绑定/画像。

---

## 二、优化方案（按「不瞎编 + 真实数据」落地）

1. **修复教师假数据**：`StudentTwin` 不再返回编造固定学生，改为**诚实空/基于真实 course_schedules 的学生清单**（无则 `data_available:false`），并保留 `data_source:"real"` 标注。
2. **教师接入三方绑定**：
   - `StaffTwinMetrics` 增 `TeacherStudentCount`（该教师关联学生数，来源真实 course 数据）
   - `bind_teacher` 维度用真实计数填充（该教师被多少学生选课/授课关联），无则诚实空
3. **蔚小芯使用统计纳入 teacher**：`WxxUseCount/WxxBindCount` 的 resource LIKE 条件扩展 `%teacher/%`，让教师用蔚小芯工具也计入画像。

## 三、诚实边界
- 教师侧改完仍是「有真实关联才显示」，无数据时 `data_available:false` + 前端「数据积累中」，**不编造学生**。
- `course_schedules` 依赖排课导入的真实数据；未导入课程的学生无教师关联属正常，如实呈现。

## 四、风险
- `StudentTwin` 若被前端用于「教师看学生」，改前要先确认前端是否有页面在消费这些编造字段（`total/at_risk/top_students`），避免改后前端空态报错。**已确认：前端无任何页面调用 `/teacher/student-twin`，编造数据为死代码，改空后无影响。**

---

## 五、优化落地结果（已实现 + 运行时验证）

### 改动
1. **教师三方绑定真实化**：`twin_repo.go` `AggregateStaffMetrics` 增 `TeacherStuCount`（`COUNT(DISTINCT cs.user_id) FROM course_schedules WHERE teacher=?`，displayName 匹配任课老师）；`twin_service.go` `bind_teacher` 由硬编码空改为真实计数（`关联 N 名学生（任课/排课，来自课程数据）`）。
2. **蔚小芯统计纳入教师**：`WxxUseCount/WxxBindCount` 的 resource LIKE 条件由 `assistant+counselor` 扩展为 `+teacher`，教师调 `/teacher/*` 工具也计入绩效画像。
3. **移除教师造假数据**：`GenerateStudentTwinTeaching` 不再返回编造的固定学生（张明/李华/王芳），改诚实返回空结构；`StudentTwin` handler 兜底也改 `data_source:empty`。

### 运行时验证（8088 临时库）
- `bind_teacher`：硬编码 `false` → **真实 `score:2, avail:True, 关联 2 名学生（course数据）`** ✓
- 教师调 `/teacher/heatmap` + `/teacher/daily-overview` → **`wxx_use:2`、`bind_wxx:2`（avail True）**，audit_logs 确认 role=teacher 落库 ✓
- `student-twin`：编造学生 → **`data_source:empty, focus_students:[]`** ✓

### 诚实边界
- 教师关联数依赖真实排课导入（`course_schedules.teacher`），未导入课程则 0/空，如实呈现；不编造学生名单。
