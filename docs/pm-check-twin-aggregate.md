# 需求核对清单：P1-1 学院书记「五维全院/全院聚合」大屏

> 核对人：pm-wxx（需求核对专员，只读）
> 日期：2026-08-17
> 状态：**已核对**（含代码实证）
> 范围：P1-1 —— 让学院书记能「一屏看全院五维」，补上 full-role 审核缺口 #2（学院书记缺「一屏的全院五维分布」，现只知一个健康度均值）。
> 性质：**需求核对清单 / 落地拆解给 dev**，本文件不修改任何源码。

---

## 〇、核对基线（代码实证，勿再推翻）

| 事实 | 实证位置 |
|---|---|
| 角色链 | `server/internal/auth/capabilities.go`：`sys_admin → school_admin → college_admin → {counselor, teacher, assistant} → student_union → student`；college_admin 多父继承三线 |
| 学院大屏端点 | `collegeGroup.GET("/twin-screen", RequireCapability(CollegeTwinScreen), collegeH.TwinScreen)`（`server/pkg/app/routes.go` L712） |
| 服务层 | `CollegeService.GenerateTwinScreen`（`server/internal/service/college_service.go` L71-140） |
| 底层聚合 | `aggregateCollegeMetrics`（同文件 L44-62）已算五维、但只用于求单值均值 |
| 学生五维底座 | `student_profile_snapshot`（`server/migrations/043_student_profile_snapshot.sql`），五维 REAL 0-100 |
| 快照读写 | `TwinRepo.ListSnapshotsByScope` / `UpsertSnapshot` / `GetSnapshot`（`server/internal/repository/twin_repo.go`） |
| 五维归一化 | `twin_service.go: computeDimensions`（L84-135），五维同为 0-100 口径 |
| 前端页面 | `frontend/lib/pages/college/twin_screen_page.dart`（仅 4 张概览卡 + AI 解读，无五维可视化） |
| 前端能力门控 | `capability_utils.dart` L91 `collegeTwinScreen = 'college.twin.screen'`，与后端 CollegeTwinScreen 已同步 |
| 全校大屏（旁证） | `SchoolAdminService.GenerateSchoolPanorama`（`school_admin_service.go` L84+）：同样**只有 health_score（情感口径）**，无五维聚合，`Trends` 为空；perCollege 仅 name/students/risk |

---

## 一、缺口实证（三份审核文档交叉确认，与代码一致）

1. **`full-role-stickiness-audit-2026-08-15.md` 缺口 #2**：「学院书记缺『一屏的全院五维分布』，现只能看单学生或看板缓存，无『一屏看全人教育』的学院聚合接口」。
2. **代码确实只有 4 个概览单值**：`GenerateTwinScreen` 输出的 `overview` 只有 `total_students / health_score / risk_students / active_rate`；`departments` 与 `trends` 直接置 `[]`/`{}`。
3. **五维已在底层算好但被丢弃**：`aggregateCollegeMetrics` 对每个快照算了 `(A+Ab+I+E+S)/5`（五维均分），再对所有学生取均值得到 `HealthScore`。**即五维分解（academic/ability/ideological/emotional/social 各自全院均值）已触碰但未暴露**。
4. **口径丢失点（关键）**：现 `HealthScore` = 「先每人算五维均分、再平均」；而「一屏看全院五维」需要「**每一维在学院内取均值**」——两者数值不同、信息粒度不同。现有 `HealthScore` 语义应保留（行为不变），**新增的是五维各自聚合，不是替换 health_score**。

**结论**：缺口成立，P1-1 是「把已算未曝的五维补上 + 补全院聚合」，不是新造数据底座。

---

## 二、目标 / 范围（P1-1 定义为）

学院书记（college_admin，经 `college.twin.screen` 能力）进入「数字孪生大屏」后能：

- ✅ **一屏看全院五维评分**：academic / ability / ideological / emotional / social 各一院级均值（0-100），并可查看分布/样本数。
- ✅ **可选按专业 / 班级下钻**（P2 保底，P1 可不做但数据结构要留）：major、class_name 为过滤维度。
- ✅ **可选趋势（Trends）**（P3）：基于 `computed_at` 的历史快照差异做增量趋势（诚实：无历史则空，不造假）。
- ✅ **诚实 DataSrc**：0 样本维 → `not_available`/`fallback`，绝不显示编造的均值。
- ✅ **不动既有口径**：保留 `overview` 四个既有字段的现有计算逻辑，新增字段不改变其值（行为不变红线）。

**不做**（清晰边界）：
- ✋ 不新增本地大模型；AI 解读仍走既有 LLM 通道或规则降级。
- ✋ 不改变 `student_profile_snapshot` 表结构/写入语义（只读聚合）。
- ✋ 不动辅导员 `counselor.twin.board`、学生 `self.twin.read` 的既有读取路径。
- ✋ 本次不做「育人归因」（缺口 #1，独立 P1，别混入）。

---

## 三、数据来源与口径（student_profile_snapshot）

| 项 | 现状（实证） | 核对结论 / 改动建议 |
|---|---|---|
| 字段类型/区间 | `academic/ability/ideological/emotional/social_score` 均为 `REAL NOT NULL DEFAULT 0`，0-100 越高越好（迁移 043 注释）；`computeDimensions` 已统一 0-100 | 与五维聚合口径完全一致，直接用 |
| 过滤方式 | `ListSnapshotsByScope(ownerScope="college", ownerID=<uc.OwnerID>, college, className, limit)`：内部 `WHERE owner_scope=? AND owner_id=?` + 可选 AND college / AND class_name | **可直接复用**；ownerID 来自 `collegeOwnerID(c)`（auth claims，防止跨院越权），缺省回落为传入 college 名 |
| 聚合 SQL | 目前无独立五维聚合 SQL；现靠内存遍历 `ListSnapshotsByScope` 求和 | 建议在 `twin_repo.go` 新增一个**聚合查询**（`AggregateSnapshotsByScope`）：`COUNT(user_id)`、每维 `AVG(...)`，并按需 `GROUP BY major / class_name`。比拉到内存求均值更健壮、可持续分组 |
| ⚠️ **500 上限缺陷（must fix）** | `aggregateCollegeMetrics` 调 `ListSnapshotsByScope(..., 500)`，**硬性 limit=500** | 全院学生 > 500 时会静默漏样本 → 五维均值失真。新聚合接口**必须去上限**（SQL 聚合天然无限制），或显式分页/声明样本覆盖 |
| 样本数 | 无现成字段 | 聚合结果必须带 `sample_count`（√每维均值对应的快照数）作为诚实标注依据 |
| 0 样本 | 目前 `m.HasData=false → DataSource="fallback"` | 五维应按**维度**做 0 样本判断：某维无人写入 → 该维 `score:null` + `data_source:"not_available"`；整体无快照 → `overview` 仍走现有 fallback，语义不变 |
| DataSrc 标注 | 现只有 `real/fallback` | 建议五维对象内**每维带 `data_source`**（real / not_available），沿用 secretary dashboard 的诚实 badge 约定 |
| 快照新鲜度 | 快照是「最近一次计算」（`computed_at`），明细仍以业务表为准 | 大屏聚合的是快照（本就为看板缓存设计，见 043 注释），不需回源重算；趋势需依赖 `computed_at` 历史 |

---

## 四、接口形态（权衡：扩展现有 vs 新建端点）

**结论：扩展现有 `TwinScreenData`，不新建端点。** 理由：

| 方案 | 改动面 | 行为不变 | 评价 |
|---|---|---|---|
| **A（推荐）扩展 TwinScreenData**：在现有 `TwinScreenData` 增加 `FiveDim *CollegeFiveDim` 字段 + 填充现有 `Departments` / `Trends` | 后端结构体小幅 + 聚合函数 + handler 不变；前端同一页面加区块 | `overview` 四字段值与现逻辑完全一致 → 现有前端 4 卡不破坏；数组/空 map 的现有消费方不受影响 | 改动最小、端点/能力/路由复用，贴合「行为不变」红线 |
| B 新建 `/twin-screen/five-dim` 端点 | 新增 handler+service+路由+capability 判断，前端新页面/新 tab | 隔离最干净，但重复造端点、能力需双端同步，成本更高 | 仅在「既有大屏要冻结不改」时才选；本次需求正是增强该大屏，故不选 |

**新增结构（建议，供 dev 定稿）**：

```go
type CollegeFiveDim struct {
    SampleCount int               `json:"sample_count"`          // 参与聚合的快照数（>=1）
    Dimensions  []FiveDimEntry    `json:"dimensions"`            // 5 维
    Distribution map[string]float64 `json:"distribution,omitempty"` // 可选：分档占比（>=80 优良 /60-80/40-60/<40）
}
type FiveDimEntry struct {
    Key         string  `json:"key"`            // academic|ability|ideological|emotional|social
    Name        string  `json:"name"`           // 学业|能力|思想|情感|社交
    Score       *float64 `json:"score"`          // 院级均值；0 样本 → null
    SampleCount int     `json:"sample_count"`   // 该维有数据快照数
    DataSource  string  `json:"data_source"`    // real | not_available
}
```

- `Departments` 填充：`[]map{name: major/class_name, sample_count, academic/.../social, risk?}`（P1 按 major 下钻，P2 可按 class）。
- `Trends` 填充：`{"academic": [...], ...}` 基于 `computed_at` 分桶的最近 N 期均值；无历史 → 返回空 map 且 `data_source` 注明（诚实空，不改现有空 map 行为）。

---

## 五、能力门控（capabilities 双端同步核对）

- **现状**：后端 `auth.CollegeTwinScreen = "college.twin.screen"`（capabilities.go）已存在并已 gate `/twin-screen` 路由；前端 `capability_utils.dart` L91 同名常量，**两端已同步**。
- **结论：无需新增 capability**。P1-1 增强的是同一个大屏、同一段路由、同一能力。若走方案 B（新端点）才需新增 `college.twin.five_dim` 并在两端同步——本次不采用。

---

## 六、前端（学院大屏页 twin_screen_page.dart）

**现状**：`_buildContent` 只渲染 4 张概览卡（学生总数/健康度/风险关注/健康率）+ AI 解读；**无五维可视化、无 departments、无 trends、无空态 badget**。

**需加（P1 最小集）**：
- [ ] 五维区：读取 `five_dim.dimensions`，用**雷达图（5 维主轴）或 5 根柱状条**展示全院均值；`score==null` 维显示「数据积累中」+ `data_source` badge（复用现有 honest badge 样式）。
- [ ] 样本诚实：区块标题显示 `sample_count`（如「全院 234 名有快照学生」）。
- [ ] 空态：`five_dim==null` 或 sample_count==0 时显示「五维画像数据积累中」（沿用孪生既有文案），不硬画 0 分雷达。
- [ ] 分布（可选 P2）：分档占比横向条。
- [ ] 下钻（可选 P2）：major/class 切换筛选，重新请求（若选 A 扩展现端点，可加 `?major=`/`?class=` query 参数）。

**注意**：前端必须是**增量渲染**——保留现有 4 卡与 AI 解读，在其下追加五维区块，避免破坏既有布局与回归测试。

---

## 七、禁止触碰（红线清单）

1. **不造数据**：任何维 0 样本 → `not_available`/fallback + 前端「数据积累中」，**绝不硬编码均值**（现有 `AnalyzeTeacherEfficiency`/`EvaluateCourseQuality` 的 reference 硬编模式**不得**用于五维大屏）。
2. **不改快照语义**：不动 `student_profile_snapshot` 结构/写入；`UpsertSnapshot`/`GetSnapshot` 原样；只新增只读聚合。
3. **不引入本地大模型**：AI 解读沿用现有 llmClient 通道；无 LLM 时走规则降级（现 `GenerateTwinScreen` 已如此），不外接新模型。
4. **行为不变**：`overview` 四字段值、现有 `real/fallback` 语义、现有 `departments/trends` 空值 → 全部保持；新增字段只是「追加」，不改既有 JSON 键。
5. **范围锁定**：聚合必须 `owner_scope='college'` + `owner_id=<uc.OwnerID>`，**绝不能跨院读全表**（沿用 `collegeOwnerID`）。

---

## 八、落地拆解（给 dev 的具体改动项，优先级 + 依赖）

### P0（正确性，先做）
| # | 改动 | 落点 | 依赖 |
|---|---|---|---|
| 1 | **修复 500 上限**：新增无 limit 的全院快照聚合查询（SQL `AVG`+`COUNT`+可选 `GROUP BY major/class`），替换/补充 `ListSnapshotsByScope(...,500)` 的内存求和 | `twin_repo.go` 新方法 `AggregateSnapshotsByScope`；`college_service.go` 五维聚合改调用它 | 无 |
| 2 | 聚合服务：`college_service.go` 新增 `aggregateCollegeFiveDim(ownerID, major, class)`，返回五维均值+各维样本数 | `college_service.go` | #1 |
| 3 | 结构体：`TwinScreenData` 增加 `SevenDim *CollegeFiveDim \`json:"five_dim"\``；填充 `Departments`（按 major）、`Trends` | `college_service.go` | #2 |
| 4 | 单测：`college_service_test.go`/`twin_repo` 覆盖 0 样本、>500 学生去限、跨院不越权 | `server/internal/.../*_test.go` | #1-#3 |
| 5 | handler/路由：`TwinScreen` handler 传可选 `major/class` query 参数（行为不变，不带则全院） | `college_handler.go` | #3 |

### P1（价值，其次）
| # | 改动 | 落点 |
|---|---|---|
| 6 | 前端五维可视化：雷达/柱状 + 每维 `data_source` badge + `sample_count` + 0 样本「数据积累中」 | `twin_screen_page.dart` |
| 7 | 前端 honest badge 复用（沿用 secretary 大屏的 data_src_badge 组件） | 前端公共组件 |
| 8 | 按 major 下钻 UI（P2 可降级：先只展示全院，下钻入口留位） | `twin_screen_page.dart` |
| 9 | LLM AIInsight 提示词扩展：把五维均值注入（可选，仍不编造数字） | `GenerateTwinScreen` |

### P2（可选，数据依赖）
| # | 改动 | 依赖 |
|---|---|---|
| 10 | Trends：基于 `computed_at` 历史快照做近 N 期逐维均值趋势 | 需有历史快照留存（当前表按 user_id 唯一，**无历史版本** → 诚实做法：返回空 trends 并标注，勿造假；或在快照表加历史留痕（另立项，超出本次禁止改动表的范围）） |

### 测试覆盖要求
- 后端：0 快照 → `five_dim` null 或各维 not_available；全院 >500 学生均值完整；跨院 ownerID 只读本院。
- 前端：`flutter analyze` 0 error；4 卡不回归；空态文案可见；雷达不画 0 分。

### 依赖关系图
```
P0-1(twin_repo 聚合) → P0-2(service 聚合) → P0-3(结构体+填充) → P0-4(单测)
   ↘ P0-5(handler 下钻参数) →
P1-6(前端可视化) ← P0 全部
P1-9(LLM) ← P0-3
P2-10(Trends) ← 需另立项留痕，不阻塞 P1
```

---

## 九、与已有大屏的横向一致性（提醒 dev）

- `SchoolAdmin GenerateSchoolPanorama` **同样缺五维聚合**（只有 emotion 口径 health_score）。P1-1 只做「学院级」；校级五维（学校书记「全校一屏」）是后续项，但本次**建议在 `twin_repo` 的聚合方法上做成 scope 通用**（`owner_scope` 可传 `school/college`），为校级复用预留，而不在 college_service 里写死。
- secretary 系列（`EducationOutcomeDashboard`、`CollabDashboard`）的诚实标注模式（`data_source: real/not_available` + 前端 badge）是**本项目五维大屏可直接对齐的范式**，避免风格漂移。

---

## 十、待 dev/leader 确认的口径（不阻塞开发，但需拍板）

1. 五维「学院均值」口径：**每维在全部有快照学生上 `AVG`**（推荐），与现有 `health_score`（每人五维均分再平均）并存、不相干。
2. 分档分布阈值：≥80 优良 / 60-79 一般 / 40-59 偏弱 / <40 薄弱（或沿用 `scoreLevel` 现有档位），建议统一用 `twin_service.go` 的 `scoreLevel` 逻辑避免口径分裂。
3. major/class 下钻是否进 P1 还是 P2：若 P1 只做全院，结构体仍保留 `Departments` 填充为空 or 按 major 填充？建议 P1 直接按 major 填，成本低。
4. Trends 如实为空（无历史快照留痕）时前端如何提示，避免被误判为「没做」。

---

## 结论

- **缺口成立且可低成本落地**：五维数据底座（快照+归一化）齐全，`aggregateCollegeMetrics` 已算五维但只吐单值；P1-1 = 把五维分解暴露给大屏 + 全院正确聚合 + 前端可视化，**不新建数据、不改表、不新增能力、不引入本地模型**。
- **最大非功能性隐患**：现有拉快照内存求和的 500 上限会造成全院数据失真，必须在新增聚合时修复（P0）。
- **推荐方案 A（扩展现有 /twin-screen 端点）**，改动面最小且贴合行为不变红线。
- **主要产出**：本核对清单（此文件），供 dev 依 P0→P1 落地。
