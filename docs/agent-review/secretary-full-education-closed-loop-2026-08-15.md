# 书记 × 蔚小芯：全人教育育人闭环蓝图（2026-08-15，升级版）

> 核心定调：**核心是教育**——包括党建，以及思想、身体、心理等高校生活的方方面面。
> 学生数字孪生五维画像（学业/能力/思想/情感/社交）即"教育方方面面"的总纲。党建是"思想"维度的子集。
> 闭环：书记以蔚小芯为抓手 → 教师教辅执行线下线上活动 → 学生为主角成长 → 全部数据汇流书记看板。

## 一、"教育方方面面" 系统数据底座（已核实齐全）

### 学生五维成长画像（孪生核心，`twin_service.computeDimensions`）
| 维度 | 数据来源（真实表） | 含义 |
|---|---|---|
| **学业** academic | student_grades/courses | GPA、通过率、学分 |
| **能力** ability | competitions/student_plans | 竞赛参与/获奖、规划完成率 |
| **思想** ideological | party_stages/party_progress/party_study_records | **党建阶段**、理论学习记录 |
| **情感**（心理）emotional | emotion_logs | 心情评分、高风险预警 |
| **社交** social | clubs/club_activities | 社团、活动参与 |

### 高校生活方方面面（迁移清单已覆盖，表全在）
- **党建**：party_stages/progress/study_records（发展对象、介绍人、支部）
- **思想/心理**：emotion_logs、psych_scales、mood_diary、counselors/counseling_appointments、crisis_hotlines
- **身体**：health_basic_info/checkups/records/daily_records/activities（体测、日常、活动签报）
- **学业**：courses/student_grades/exam_schedules/mistake_notebook/learning_resources + course_schedules（排课）
- **就业/毕业**：career_policies/job_postings/info_sessions/user_resumes/graduation_progress/thesis_topics
- **社会实践/劳动**：student_checkins、campus_checkin_steps、student_points、volunteer
- **谈心帮扶**：talk_records、counseling_appointments
- **后勤服务**：facility_records
- **社团/活动**：clubs/club_activities

## 二、现状缺口（已核实）

1. **学生五维画像已建，书记看不到全校聚合**：孪生五维 only 单学生视角；`_snapshots` 用于看板缓存，但**书记/学院书记无全校/全院五维聚合接口**。
2. **各维度数据分散（学业/思想/身体/心理/社交各自成表）**，**没有"一屏看全人教育"的书记总览**。
3. **协同育人缺归因**：教师/教辅的育人动作（谈心/后勤/排课/活动）与**学生五维成长未挂钩**，答不出"哪些育人努力带来了学生变化"。
4. **党建子集缺书记侧**（详前文）：`SelfParty` 自报，书记读不到。
5. **情绪风险已全校聚合（risk=urgent+high）** 但仅用于全景，未做成"心理关怀专项看板"回流书记。

## 三、闭环设计（规划 → 待实现）

```
书记: 全校/全院育人总览看板（部署/审批/督办）
   │  ▲ 汇流
   ├── 蔚小芯(抓手): 聚合五维孪生 + 育人动作 + 情绪风险
   │     ├─ 全人教育五维总览(学业/能力/思想/身体/心理/社交 按学院)
   │     ├─ 协同育人归因(谈心/后勤/活动 × 学生成长 变化)
   │     └─ 思想/心理关怀(党建漏斗 + 情感风险 下钻)
   ├── 教师/教辅(执行): 登记党课/活动/谈心/帮扶(带记录人)
   └── 学生(主角): 参与活动 + 五维成长画像 累积
```

### 要新增（3+1，均基于已有真实表，不造假）
1. **全人教育总览（书记主看板）**：五维 + 身体/心理 全校/全院聚合，一屏看"德智体美劳"。
2. **协同育人归因**：教师/教辅真实育人动作 × 学生五维趋势（只做趋势/相关性，不做因果断言）。
3. **党建引领子看板**：入党漏斗 + 党课人次/时长 + 积极分子活动 + 介绍人/支部，接到 school/college admin。
4. **登记动作**：教师/教辅侧"组织活动/党课/谈心"登记入口，带记录人，回流书记。

## 四、诚实边界（不瞎编红线）
- 全部基于已有真实表聚合；学生自报数据标 `data_source`（real / self-reported），不把自报当组织确认。
- 归因只描述趋势/相关性，不做因果断言（样本不足宁可不给结论）。
- 无数据维度如实显示"数据积累中"（孪生已有此模式，书记聚合沿用）。

## 五、待确认口径
1. **书记主看板范围**：`school_admin`（全校） + `college_admin`（本院）两级，对吗？
2. **"教育方方面面"维度分组**：按现有孪生五维（学业/能力/思想/情感/社交）是否够？还是要拆"身体/心理"为独立大屏块？
3. **协同育人归因的育人动作来源**：谈心(talk_records)+后勤(facility)+党建(party_study)+活动(activity) 四类够不够？
