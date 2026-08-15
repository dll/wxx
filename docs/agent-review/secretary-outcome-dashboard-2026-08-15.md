# 书记教育成果绩效大屏（Secretary Education Outcome Dashboard）
> 2026-08-15 · 设计定稿 · 待确认口径后编码（暂不上线）

## 目的
对齐书记角色需求：**把「学生绩效 / 教育成果」真实数据全部汇集到书记**——学科竞赛获奖、入党比率、考研率、就业率等，**对齐角色需求、整合所有用户**，一屏看全校/全院育人成效。

## 〇、全生命周期贯通（核心设计原则）

书记要的是「**从入学到离校到就业**」的完整教育成果。系统已有**全生命周期数据骨架**，以学生 `user_id` 为主键串联：

| 生命周期阶段 | 数据表 | 身份/记录 |
|---|---|---|
| **入学**（身份底座） | `users`(074 扩展) | education_level 学历 / study_duration 学制 / expected_graduation_date 预期毕业 / study_mode / campus / political_status 政治面貌 / gender / ethnicity |
| **在校·成长** | `student_profile_snapshot` | 五维快照(学业/能力/思想/情感/社交) + college/major/class + 按 owner_scope 聚合（注释明示"学院大屏"用途） |
| **在校·学业** | student_grades / courses / course_schedules | score/gpa/passed/credits/course_schedules.teacher |
| **在校·能力** | competitions / competition_registrations | award_level / advisor_name / college |
| **在校·思想** | party_progress / party_study_records | 五阶段 / 入党介绍人 / 学习记录 |
| **在校·心理身体** | emotion_logs / health_* / mood_diary | 情绪风险 / 体测 / 日常 |
| **在校·社会活动** | clubs / checkins / student_points / talk_records / facility_records | 活动 / 打卡 / 谈心 / 后勤 |
| **离校**（办理痕迹） | process_records (flow_type=graduation) | 离校流程完成状态 |
| **毕业·论文** | graduation_progress / thesis_topics | 里程碑 / 选题 / 导师 |
| **就业/考研去向** | **❌ 目前无真实落库表**（详见下） | 仅有知识库描述"毕业去向登记"，无表 |

## 一、指标数据源核实（关键：哪些是真实数据，哪些空缺）

| 书记要的成果 | 系统表 | 能否真实统计 | 说明 |
|---|---|---|---|
| **学科竞赛获奖** | competition_registrations | ✅ **可真实** | `status=awarded` + `award_level`(一/二/三/优秀) + `award_date` + `advisor_name`(指导教师) + college/major |
| **入党比率** | party_progress | ✅ **可真实** | `status`(applicant/activist/development/probation/member) + 各阶段日期 + college；比率=党员数/毕业生数 |
| **学业/绩点** | student_grades | ✅ 可真实 | score/gpa/passed/credits_earned |
| **毕业/论文/离校** | graduation_progress + thesis_topics + process_records | ✅ 可真实 | 里程碑完成 + 选题 + 导师 + 离校办理状态 |
| **就业率** | — | ❌ **无真实去向数据** | 有 job_postings(岗位)/user_resumes(简历)/info_sessions(宣讲会)，但**缺"毕业去向/就业状态"记录表** |
| **考研率** | — | ❌ **无真实数据** | 无"考研报名/录取"表；graduation_progress 只到毕业里程碑 |
| **谈心/帮扶** | talk_records | ✅ 可真实 | 谈心人次、覆盖学生数 |
| **后勤/活动** | facility_records / clubs / checkins | ✅ 可真实 | 服务量、活动、参与 |

## 二、诚实边界（不瞎编红线）
- ✅ **竞赛获奖 / 入党比率 / 学业 / 毕业 / 离校 / 谈心 / 活动**：全部基于真实表聚合，可如实展示。
- ✅ **全生命周期贯通**：入学(users录取字段)→在校(五维/学业/竞赛/党建/活动)→离校(process_records)均真实，可直接汇流。
- ❌ **就业率 / 考研率**：当前无真实去向表（仅知识库提到"毕业去向登记"，未落库）。**两个选择**：
  a) 看板留位显示「数据待接入」——诚实空；
  b) **新建 `graduation_outcome`（毕业去向）真实表**（存每个毕业生的入学届别/学院/去向类型：就业/升学读研/出国/灵活就业/未就业 + 单位/升学院校 + 登记人），提供**导入入口**或登记入口，接真实数据后再展示。
  **未接真实数据前，绝不显示编造的就业率/考研率数字。**

## 三、看板设计（书记主视图）
### 顶层结构
```
书记教育成果大屏
├─ ① 教育成果总览（全校 + 按学院下钻）
│     ├ 竞赛获奖: 总数 / 国家级 / 省级 / 校级 / 获奖等级分布 / 指导教师榜
│     ├ 入党: 党员数 / 入党比率 / 各阶段漏斗(申请→积极→发展→预备→正式) / 介绍人榜
│     ├ 学业: 平均绩点 / 通过率 / 学业预警
│     └ 毕业: 毕业进度 / 论文选题 / 导师工作量
├─ ② 育人工作协同
│     ├ 谈心帮扶 / 后勤服务 / 活动组织(按教师/教辅 × 学院)
│     └ 教师/教辅育人负载榜
├─ ③ 学生成长视图(学生为主角)
│     └ 五维孪生聚合(学业/能力/思想/情感/社交 按学院)
└─ ④ 就业考研(诚实标注)
      ├ 就业宣讲 / 岗位 / 简历覆盖(真实可显示)
      └ 就业率 / 考研率(未接入 → 显示"数据待接入", 或接毕业去向表)
```

### 角色对齐与权限
- `school_admin`（学校书记）：看全校，跨学院对比。
- `college_admin`（学院书记）：看本院。
- 数据来自教师/教辅/学生**所有用户在系统内的真实行为**（audit_logs 为调用证据链）。

## 四、实施路线（待确认后编码）
1. **后端**：新增 `SecretaryOutcomeService`，以 user_id 串入学→在校→离校真实数据聚合（成绩/竞赛/入党/毕业/谈心/后勤 + users 录取字段 + process_records 离校）；新增 school/college admin 路由。
2. **（重点）就业/考研去向往来**：若确认，新增 `graduation_outcome` 表 + 导入/登记入口，接真实毕业去向后再点亮就业率/考研率。
3. **诚实占位**：未接真实去向前，就业率/考研率返回 `data_source:"not_available"`，前端显示「数据待接入」。
4. **前端**：书记大屏一页（Flutter）按生命周期分块，复用 data_src_badge（真实/参考/待接入）。

## 五、待确认口径
1. **就业/考研去向**：我新建 `graduation_outcome` 表 + 导入/登记入口（能录真实毕业去向，点亮就业率/考研率）——确认做吗？数据是管理员/辅导员导入还是学生自报？
2. 竞赛"获奖"口径：只算 `status=awarded` 且填了 `award_level` 的？要不要出指导教师榜按 `advisor_name` 归集？
3. "入党比率"分母：用毕业生总数（本届规模）还是累计申请学生？
4. 生命周期时间轴默认从"入学"（users.expected_graduation_date / study_duration）起算本届，跨学院对比口径 OK？
