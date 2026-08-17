# 需求核对清单：P1-2 育人归因看板 —— A 路径「快照历史留痕打地基」

> 核对人：pm-wxx（需求核对专员，只读）
> 日期：2026-08-17
> 状态：**已核对**（含代码实证）
> 范围：P1-2 A 路径 —— 给 `student_profile_snapshot` 加**历史留痕**能力（纯新增，不动现有写路径/表语义），并做一个**归因看板占位 + 诚实提示**（数据积累中）。
> 性质：**需求核对清单 / 落地拆解给 dev**，本文件不修改任何源码。
> 关联：`docs/pm-check-twin-aggregate.md`（P1-1，其 P2-10 "Trends" 明确标注"需另立项留痕"，本任务正是该另立项的共因解决）。

---

## 〇、核对基线（代码实证，勿再推翻）

| 事实 | 实证位置 |
|---|---|
| 快照表结构（按 user_id 唯一，无历史版本） | `server/migrations/043_student_profile_snapshot.sql`：`user_id INTEGER NOT NULL UNIQUE REFERENCES users(id)`，每次 Upsert 覆盖同 user_id |
| 快照写入是覆盖 | `twin_repo.go: UpsertSnapshot`：`INSERT ... ON CONFLICT(user_id) DO UPDATE SET ...`（按 user_id 覆盖，无历史版本） |
| 快照生成入口 | `twin_service.go: GetDigitalTwin` → `computeDimensions` → 末尾 `_ = s.repo.UpsertSnapshot(...)`（失败静默不阻断，仅影响看板聚合缓存） |
| 快照只读复用 | `GetSnapshot` / `ListSnapshotsByScope` / `AggregateSnapshotsByScope`（P1-1 已新增去 500 上限的 SQL 聚合） |
| growth_trend 缺口 | `secretary_outcome_repo.go: GetNurtureKPI` L901-903：`nurture.growth_trend` → `notAvailableCard`，desc「系统内无跨周期成长归因指标表…缺纵向对比基准」 |
| 归因看板端点复用面 | `secretary_outcome_handler.go: NurtureKPI` `GET /college/nurture-kpi`（`auth.OutcomeDashboard`），已由前端 `secretary_outcome_dashboard_page.dart` 的「育人成效指标」卡渲染 |
| 能力门控（现） | `capabilities.go`：`OutcomeDashboard="outcome.dashboard"` / `CollabDashboard="college.collab.dashboard"`；`capability_utils.dart` 有 `outcomeDashboard='outcome.dashboard'` 同步 |
| 方言转换层 | `docs/database-migration-mysql.md` + `server/internal/db/dialect.go`：迁移 SQL 运行时 `ToMySQL` 转换，SQLite/MySQL 双兼容；迁移按文件名顺序执行（`cmd/migrate/main.go` Glob + `_migrations` 表跟踪） |
| 当前最新迁移编号 | `server/migrations/` 最大编号 **090**（`090_gov_tickets.sql`）→ 新迁移建议 **091+** |
| 前端 KPI 卡渲染 | `secretary_outcome_dashboard_page.dart`：`real` → 显数值；`not_available` → 显「数据待补充」+「上传材料到知识库」+「生成补料督办工单」两入口 |
| 无后台调度设施 | 全工程无 `Ticker`/`Cron`/定时 goroutine 模式（已检索）→ 周期快照需**新增**调度器或在业务写路径内同写历史 |

---

## 一、缺口实证（三处交叉确认，与代码一致）

1. **结构根因**：`student_profile_snapshot.user_id` 为 `UNIQUE` → **每个学生最多一行**，每次 `UpsertSnapshot` 覆盖旧值。`computed_at` 记录的是「最近一次计算」时间，**没有任何历史版本** → 无纵向基准，无法算「趋势/成长」。
2. **归因缺口**：`GetNurtureKPI` 的 `nurture.growth_trend` 卡直接 `notAvailableCard`，注释/desc 明示「无跨周期成长归因指标表，缺纵向对比基准」。
3. **P1-1 共因未解**：`pm-check-twin-aggregate.md` 的 P2-10 "Trends"（大屏逐维近 N 期趋势）同样卡在「快照表按 user_id 唯一、无历史版本」。**本任务解决"留痕"这个共因后，P1-1 Trends 与 P1-2 growth_trend 可共用同一历史数据底座**，需在对齐时锁定同一口径，避免两处各建一套。

**结论**：A 路径「先打地基（历史留痕）+ 归因占位诚实提示」成立且有明确、可复用价值，不是新造数据。

---

## 二、目标 / 范围（P1-2 A 路径定义为）

**核心目标（打地基）**：给快照加**历史留痕**，让未来能积累纵向数据。**纯新增**，满足：

- ✅ 不动现有写路径语义：现有 `UpsertSnapshot` 的「最近快照」读取、P1-1 的全院聚合、辅导员/学生孪生读取**全部保持行为不变**。
- ✅ 未来能按 `(user_id, 时间)` 取历史版本 → 支撑纵向「趋势/归因」与 P1-1 Trends。
- ✅ 方言兼容：新迁移/新 SQL 同时跑 SQLite（测试/本地）与 MySQL（生产）。
- ✅ 诚实边界：做**归因看板占位 + 「数据积累中，需 N 周」提示**；不造数据、不硬编、不做「空壳归因硬显示」。

**不做**（清晰边界）：
- ✋ 不做真实归因计算（本次只有地基 + 占位；真实归因在数据积累足够后由后续迭代做——见 §三 归因口径预留，但**不实现**）。
- ✋ 不引入本地大模型；不碰情感/党建等业务表写入。
- ✋ 不跨 `owner_scope/owner_id` 读全表（沿用 P1-1 `collegeOwnerID` 越权红线）。
- ✋ 不改变 `student_profile_snapshot` 主表结构与既有写入（除非 §三 权衡选 b 明确提出且不影响现有功能——**本核对默认不改，推荐 a**）。

---

## 三、留痕机制方案权衡（核心决策）

| 维度 | **方案 a：独立历史表 `snapshot_history`**（推荐） | 方案 b：改 043 把 UNIQUE 换成 `(user_id, computed_at)` 复合并保留全量历史 | 方案 c：独立 `growth_baseline` 表只存关键周期基准 |
|---|---|---|---|
| 不动现有写路径 | ✅ 最强。主表 `student_profile_snapshot` & `UpsertSnapshot` **一字不改**；新增一个 repository 方法在写主表时**同写历史**（`WriteSnapshotHistory`），或在每日/每次写主表回调里 append | ❌ 需改 043 UNIQUE（破坏原表结构语义）且要迁移既有行；现有读路径（`WHERE user_id=?`）在扩成 `(user_id, computed_at)` 后**会读到多行**，`GetSnapshot` 的 `QueryRow` 从"唯一一行"变"多行"→必须加 `ORDER BY computed_at DESC LIMIT 1`，改动面大、回归风险高，**违背"不动现有写路径/表语义"目标** | ❌ 表只存"基准"，但"基准"定义（取哪期）需先定归因口径，且历史上每个学生的数据仍要 append 才能形成基准——本质上藏了历史数据只是裁剪存储，**无法应对需回溯的纵向分析** |
| 方言兼容 | ✅ 纯新增 `CREATE TABLE`，走 `ToMySQL` 即可；历史表列同主表（REAL/TEXT/VARCHAR），无 SQLite 专属语法 | ⚠️ 需处理 `UNIQUE(user_id, computed_at)` 复合约束方言（SQLite `UNIQUE` / MySQL 同，尚可），但 DDL 变更 + 数据迁移风险大 | ✅ 同 a |
| 查询成本 | ✅（存全量则膨胀）历史表按 `(user_id, computed_at)` 索引，纵向对比按 user 取近 N 行，成本可控；历史膨胀可后续清理（见下） | ✅ 主表即历史，少一张表；但主表膨胀会拖慢所有现有读（辅导员看板/全院聚合每次都扫全历史）→ 需按 user 取 recent 带索引 | ✅ 最省存储，但归因口径未定前不知要存哪些基准 |
| 历史膨胀 | ⚠️ 全量 append 无上限会持续增长（每个学生每次重算多 1 行）。缓解：**只 append 必要周期**（见 §四：若只在每日/每周首算时 append，增长=学生数×周期数，可控）；可后续加清理/抽样策略 | ❌ 膨胀最重（含 AI 解读大文本每次全存），且毒化所有现有读路径 | ✅ 最省，但灵活性差 |
| 与 P1-1 趋势复用 | ✅ 同一历史底座，P1-1 Trends 与 P1-2 growth_trend 都从 `snapshot_history` 按 `(user_id,computed_at)` 取纵向切片 | ✅ 复用主表，但主表膨胀伤 P1-1 全院聚合读 | ⚠️ 需另起一行按"基准期"取，与 P1-1 各自实现趋势，易口径分裂 |
| **推荐** | **✅ 采用** | ✋ 否决（违反目标红线） | ⚠️ 暂不采用（归因口径未定，灵活性差） |

**结论：推荐方案 a —— 新增独立历史表 `snapshot_history`，与主表并存，纯追加、不改主表与既有写路径。**

方案 a 的关键设计约束（供 dev 定稿）：

- **冗余最小化**：历史表**不存** AI 解读/差距/建议三长大文本列（它们每月会变很多且看板不用，避免膨胀），只存 `user_id / owner_scope / owner_id / college / major / class_name / 五维 score ×5 / computed_at`。纵向归因/趋势只需五维分数，不需要解释文本。
- **主键**：`id` 自增 + `UNIQUE(user_id, computed_at)`（同一天同一学生的多次重算只保留一次，天然去抖），保证「每个学生每一天 ≤ 1 条历史」。这同时缓解膨胀并满足纵向采样。
- **索引**：`(user_id, computed_at)` 复合索引（纵向取历史主路径）、`(owner_scope, owner_id, computed_at)`（书记按院聚合历史趋势）。
- **写出时机**（见 §四 决策，推荐与业务写同写历史，不引入调度器）。

> 新迁移：**091_student_profile_snapshot_history.sql**（当前最新 090，续号）。

---

## 四、数据来源（快照历史从哪来）—— 写出时机权衡

写历史有三种时机，权衡如下：

| 时机 | 说明 | 优点 | 缺点 | 结论 |
|---|---|---|---|---|
| **A. 业务写主表时同写历史**（推荐） | 在 `GetDigitalTwin` 已经 `UpsertSnapshot` 的地方，**追加**一次 `UpsertSnapshotHistory`（写主表成功后 append 到 `snapshot_history`） | 改动最小、无需调度器；快照本来就在这里实时重算；天然有"每次重算 = 一次历史采样" | 若同一学生当日被多次访问会多次 upsert 历史（用 `UNIQUE(user_id,computed_at)` 去重后仅留当日一条）；无后台调度需"有人访问才采样"，冷学生可能长期无历史 | 主路径，改动点仅 twin_service.go 末尾 1 处 + twin_repo 加 1 方法 |
| B. 后台调度周期快照 | 新增一个定时任务（每日/每周）遍历学院学生重算并写历史 | 采样频率稳定、不受访问驱动 | **工程无任何现成调度器**，需新增 ticker + 遍历所有学生（重算成本高、与 GetDigitalTwin 重复算）；需处理启动时机/多实例竞态 | 不建议本次引入；可作为后续增强，地基表结构不受影响 |
| C. 纯独立：不接快照写入，单独每日从业务表聚合 | 绕开 `GetDigitalTwin`，另起一条聚合写历史 | 与现快照解耦 | 重复实现 `computeDimensions` 逻辑，口径易漂移；等于再造一套快照 | 否决，口径分裂风险 |

**结论：推荐 A —— 在 `GetDigitalTwin` 写主快照处**同写历史**。地基为 B 预留扩展（历史表结构不依赖调度器）。**

- 去抖：`UNIQUE(user_id, computed_at)` + upsert 语义 → 同一学生当日多次重算只留 1 条历史，膨胀≈学生数×天数。
- 诚实说明：**在历史积累满 §六 的 N 周前，growth_trend 恒为 not_available**（占位），前端显示「数据积累中」。这不是 bug，是"打地基"阶段的预期行为。

---

## 五、归因口径（growth_trend 要对比什么）—— 预留，不实现

> 本任务**只做地基**，归因口径是**给 dev/leader 预留的设计约束**，促使历史表字段能支撑它；真实归因计算在数据积累后由后续迭代做。

1. **纵向（主口径，必须）**：同一学生 **N 周前 vs 现在** 的五维差值（Δacademic/Δability/Δideological/Δemotional/Δsocial）。从 `snapshot_history` 按 `(user_id, computed_at)` 取"现在"与"N 周前"两条，逐维 `delta = now - past`。
   - 负分维度（如情感）口径对齐 `computeDimensions` 0-100 越高越好。
2. **横向（可选，不阻塞）**：同学/同班均值对比（复用 `AggregateSnapshotsByScope` 的能力，把"其他同学均值"作参照）。**P1-2 不做**，仅留接口位。
3. **N 的配置**：建议 `growth_trend_window_weeks` 默认 4 周（或 config/settings 可配），前端提示「需累计 N 周」。
4. **诚实边界（硬纪律）**：growth_trend 只表达**"趋势 / 相关性"**（delta 升降、方向、幅度），**绝不表达"因果"**（例如"参加 X 活动导致分数上升"是因果判断，超出数据可支持范围）。前端必须用「变化/趋势」措辞，不用「因为/导致」。
5. **样本量与聚合**：院级归因展示时，每个维度涨幅=「有 N 周历史的学生」delta 均值，必须标注 `sample_count`（有多少学生有足够历史），0 样本 → not_available。

---

## 六、接口形态（归因看板端点）权衡

| 方案 | 说明 | 改动面 | 诚实展示 | 评价 |
|---|---|---|---|---|
| **A（推荐）扩展 NurtureKPI**：`nurture.growth_trend` 从 `notAvailableCard` 变为「动态判断」——有足够历史 → 返回趋势 payload；无 → 仍 not_available + 补充 source_desc「数据积累中，需满 N 周」 | 复用现 `/college/nurture-kpi` 端点、现前端 KPI 卡、现 `OutcomeDashboard` 能力，**无新路由/无新能力** | 后端改 GetNurtureKPI 的 growth_trend 卡生成逻辑（1 处）+ 前端 KPI 卡渲染分支（1 处，对 `real` 之外的趋势 payload 显示可视化或占位） | ✅ 趋势数据直接在一个"卡"内：涨跌箭头 + 样本量，或仍占位 | **改动最小、贴合"打地基占位"目标**；趋势 payload 用 `data_source: "trend"` 新值，与现有 real/not_available 并列，前端加一个 `trend` 渲染分支即可 |
| B. 独立端点 `/college/growth-trend` | 新增 handler+service+repo+路由，前端新页/新区块 | 大：需新能力 `college.growth.dashboard`，`capabilities.go`↔`capability_utils.dart` 双端新增 | ✅ 独立干净 | 仅在"既有 nurture-kpi 要冻结"时才选；本次是增强该卡，且 A 已覆盖"占位"需求 → 不选 |

**结论：推荐方案 A（扩展 `NurtureKPI`），不新增端点、不新增能力。**

- growth_trend 卡新形态（供 dev 定稿）：有历史 → `{ key, label:「学生成长度对比(纵向)」, value: {Δ academic…Δ social}, sample_count, data_source:"trend", trend_desc }`；无历史 → 同现状 `not_available`，但 `source_desc` 改为「数据积累中：需累计满 N 周历史快照后生成成长归因（当前系统已开始对数字孪生快照做历史留痕）」。
- `data_source:"trend"` 是**新增枚举值**，需在 `secretary_outcome_kpi_ironrule_test.go` 的铁律枚举里登记，保证"trend 卡 value 不可为空/缺样本则回落 not_available"的诚实铁律。
- **注意**：现前端 `_buildKPICards` 对 `!= 'real'` 一律走「数据待补充」分支。新增 `trend` 分支后，**未满 N 周时后端直接返回 `not_available`（而非 trend）**，前端**不需要特判**「如何知道在积累中」——由 source_desc 文案承担提示，前端保持简单。

---

## 七、能力门控

- **结论：无需新增能力。** 方案 A 复用现有 `GET /college/nurture-kpi`（`auth.OutcomeDashboard="outcome.dashboard"`），后端 `capabilities.go` 与前端 `capability_utils.dart` 已同步，**不改两端**。
- 仅当日后改走方案 B 独立端点，才需新增 `college.growth.dashboard` 并在 `capabilities.go` ↔ `capability_utils.dart` 双端同步——本次不做。
- 越权红线沿用：`GetNurtureKPI(ownerID)` 已按 owner 范围过滤；若 growth_trend 从 snapshot_history 取历史，**必须同样带 `owner_id` 过滤**（`snapshot_history` 保留 `owner_scope/owner_id` 列即为此），绝不能跨院读全表。

---

## 八、前端（书记归因看板 —— 现状与需加）

**现状（实证）**：`secretary_outcome_dashboard_page.dart` 的「育人成效指标」卡已渲染 `nurture.growth_trend`（现为 `not_available` → 显示「数据待补充」+「上传材料到知识库」+「生成补料督办工单」）。**注意：growth_trend 是系统内置纵向，靠"上传材料"补不了——所需的是"时间积累"，不是"上传补料"**，因此其 UI 心智与一般 not_available 卡不同（见下）。

**需改（P1-2 最小集）**：
- [ ] `_buildKPICards` 增加 `data_source == 'trend'` 渲染分支：
  - [ ] 展示五维 Δ（纵向成长）：**五根横向条**（每维 label + 当前分 + Δ箭头：↑涨幅绿 / ↓跌幅红 / →持平灰），标注 `sample_count`（如「基于 128 名有 N 周历史学生」）。
  - [ ] 在卡下方始终显示诚实提示：「数据积累中：需连续记录满 N 周后生成成长归因」。**当前阶段（数据积累不足）由后端返回 `not_available`，前端此分支尚未被触发**，但仍**预先写好**，后续有数据即自然显现。
- [ ] 对 `growth_trend` 在 `not_available` 时的**差异化文案**：现通用分支会给"上传材料到知识库/生成补料工单"两入口，但对 growth_trend 这**会误导**（它是时间积累不是补料）。建议：对 `key=='nurture.growth_trend'` 的 not_available 卡，**隐藏"上传材料/生成工单"两入口**，只显「数据积累中，需满 N 周」提示（可复用 data_src_badge 风格）。这是本次前端**最需要的诚实修正**。
- [ ] 样本诚实：`sample_count==0` → 整体回落到 not_available 占位，不硬画 0 Δ 条。

**增量渲染纪律**：保留现有全部 KPI 卡与两入口逻辑不动，只对一个卡新增 trend 分支 + 对 growth_trend 一个 key 特判 not_available 文案。避免回归其它 real/not_available 卡。

---

## 九、禁止触碰（红线清单）

1. **不造数据 / 不硬编**：任何维度无 N 周历史 → `not_available`，**绝不**返回编造的 Δ 或"因为…所以…"因果表述。`unknownScore`/reference 硬编模式不得用于 growth_trend。
2. **不改现有 UpsertSnapshot 写入语义**：主表 `student_profile_snapshot` 结构、`UpsertSnapshot` 的 `ON CONFLICT(user_id)` 行为**一字不改**。历史留痕是**追加**一个新 repo 方法 + 在写主表后调用，主表行为完全不变。若 dev 认为必须改主表（方案 b）——**本文档否决，需 leader 重新拍板**。
3. **不动既有读路径**：`GetSnapshot`/`ListSnapshotsByScope`/`AggregateSnapshotsByScope`（P1-1）原样；历史表只新增查询，不被这些现有方法与主表 ANY 改动。
4. **不引入本地大模型**：归因解读（若后续做）沿用现有 llmClient 通道或规则，不外接模型。
5. **不跨院越权**：历史查询必须 `owner_scope='college'` + `owner_id=<uc.OwnerID>`（沿用 P1-1 `collegeOwnerID`）。
6. **不做因果**：growth_trend 只报趋势/相关性，不报因果；前端文案与后端 desc 统一措辞。
7. **不做空壳归因硬显示**：数据不足 → 占位 + 诚实积累提示，绝不显示空的 prettified 图表冒充结果。

---

## 十、落地拆解（给 dev 的具体改动项，优先级 + 依赖）

> 新迁移编号：**091_student_profile_snapshot_history.sql**（当前最新 090，续号；确认无并行迁移 090 后即可定 091，若期间有新增则顺延到最新+1）。

### P0（地基：留痕机制 + 迁移，本次核心，先做）
| # | 改动 | 落点 | 依赖 |
|---|---|---|---|
| 1 | 新迁移 `091_student_profile_snapshot_history.sql`：建历史表（仅五维+归属+computed_at，无 AI 长文本）+ `UNIQUE(user_id,computed_at)` + 两索引（`(user_id,computed_at)` 纵向、`(owner_scope,owner_id,computed_at)` 院级聚合）。SQLite/MySQL 双兼容（REAL/TEXT/VARCHAR + `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX`；`computed_at` 走 `ToMySQL` 自动转） | `server/migrations/091_*.sql` | 无 |
| 2 | `twin_repo.go` 新增 `UpsertSnapshotHistory(s *TwinSnapshot)`：写历史表，`INSERT ... ON CONFLICT(user_id,computed_at) DO UPDATE`（方言 `ON DUPLICATE KEY UPDATE`，走 `dbutil.AdaptForDriver`） | `twin_repo.go` | #1 |
| 3 | `twin_service.go getDigitalTwin`：在现有 `UpsertSnapshot` 之后**追加** `UpsertSnapshotHistory`（同一份五维+归属+computed_at）；失败同主快照静默不阻断 | `twin_service.go` | #2 |
| 4 | 单测：历史 upsert 去抖（同 time 只留 1 条）、方言跑通、写在主表后不改变主表行、跨院 ownerID 过滤 | `twin_repo_test.go` / service test | #1-#3 |
| 5 | （可选加分）`computed_at` 取当日：若 `GetDigitalTwin` 多次调用但 computed_at 精确到秒会引起同日多条，建议历史 `computed_at` 归一化到「当天日期」以便按天去重（或保留秒级 + UNIQUE 部分先去抖，见 #2 权衡） | twin_service / repo | #2 |

### P1（归因计算 + 接口，价值、数据依赖）
| # | 改动 | 落点 | 依赖 |
|---|---|---|---|
| 6 | `secretary_outcome_repo.go` 新增 `getGrowthTrend(ownerID, windowWeeks)`：从 `snapshot_history` 按 `(user_id,computed_at)` 纵向差 N 周，返回院级五维 Δ + `sample_count` + `window_weeks`；无 N 周历史学生 → 返回 `has_data=false` | `secretary_outcome_repo.go` | #2 |
| 7 | `GetNurtureKPI` 的 `growth_trend` 分支改造：调 `getGrowthTrend`；`has_data` + `sample_count>0` → 返回 `data_source:"trend"` payload；否则**仍 `not_available`** 但 `source_desc` 改写积累文案 | `secretary_outcome_repo.go` | #6 |
| 8 | 铁律单测登记 `data_source:"trend"` 枚举 + 缺样本回落 not_available 的铁律 | `secretary_outcome_kpi_ironrule_test.go` | #7 |
| 9 | 配置：`growth_trend_window_weeks`（默认 4）读取（settings/config） | config/service | #7 |
| 10 | 前端 `trend` 渲染分支：五维横向条 + Δ箭头 + `sample_count` + 「数据积累中，需满 N 周」提示 | `secretary_outcome_dashboard_page.dart` | #7，#12 |
| 11 | 前端 growth_trend 的 not_available 差异化：隐藏「上传材料/生成工单」两入口，仅显积累提示 | `secretary_outcome_dashboard_page.dart` | #7 |

### P2（可选，数据依赖足够后自然升级）
| # | 改动 | 依赖 |
|---|---|---|
| 12 | 新增后台周期快照调度（每日/每周遍历学院学生写历史）——方案 B：需要时再引入调度器，地基表不受影响 | 方案 B 立项 |
| 13 | 横向（同学/同班均值）对比展示：复用 `AggregateSnapshotsByScope` 作参照基线 | 需有足够样本 |
| 14 | P1-1 Trends 与 P1-2 growth_trend 对齐：共用 `snapshot_history` 底座，统一窗口口径，避免两套趋势实现 | P0 完成 |
| 15 | 归因解读 LLM 注入（可选，仍不编造）：把纵向 Δ 注入解释通道 | 有数据后 |

### 测试覆盖要求
- 后端：无历史 → `growth_trend` 仍 not_available 且文案对；注入 >N 周历史 → 返回 `data_source:trend` 且五维 Δ 正确、sample_count 正确；同日去抖（1 学生 1 天 ≤ 1 条）；跨院只读本院；方言 SQLite/MySQL 双跑。
- 前端：`flutter analyze` 0 error；现有 real/not_available/工单逻辑不回归；trend 分支在无数据时不被触发；growth_trend 的 not_available 卡**不显示上传/工单入口**（诚实修正）。

### 依赖关系图
```
P0-1(迁移091) → P0-2(twin_repo 历史upsert) → P0-3(twin_service 同写历史) → P0-4(单测)
                                                    ↘ P0-5(computed_at 归一)
P1-6(getGrowthTrend) ← P0-2
P1-7(GetNurtureKPI growth_trend 分支) ← P1-6
P1-8(铁律测试) ← P1-7 ｜ P1-9(窗口配置) ← P1-7
P1-10/11(前端) ← P1-7, P2-12 预留
P2-14(P1-1 Trends 对齐) ← P0 完成
```

---

## 十一、待 dev/leader 确认的口径（不阻塞打地基，但需拍板）

1. **历史采样粒度**：写历史时机默认选方案 A（业务写同写历史）。同一学生当日多次访问时，`UNIQUE(user_id,computed_at)` 是否按「天」去抖（推荐，一天一条）还是要保留每次快照（膨胀更重）？建议按天。
2. **N 周窗口默认值**：`growth_trend_window_weeks=4`（周），是否可配置？建议可配（settings），默认 4。
3. **历史表是否含 AI 长文本**：本文档建议**不含**（只存五维+归属），以控膨胀。若归因解读需历史解释文本，再另行加列——确认是否接受"历史不含解释文本"。
4. **growth_trend 对齐 P1-1 Trends**：两处是否统一用同一个 `snapshot_history` 底座 + 同一个窗口口径（强烈建议，避免口径分裂）——需 leader 与 P1-1 dev 对齐。
5. **前端"上传材料/生成工单"对 growth_trend 的隐藏**：是否接受仅对 `key=='nurture.growth_trend'` 隐藏这两个入口（因它靠时间积累而非补料）——建议接受（诚实修正）。

---

## 结论

- **缺口成立**：`student_profile_snapshot` 按 `user_id` UNIQUE、每次覆盖，`UpsertSnapshot` 无历史版本 → `growth_trend` 只能 not_available；P1-1 Trends 同卡在无历史。**共因 = 缺纵向留痕**。
- **推荐方案 a（独立历史表 + 追加留痕）**：纯新增、不改主表写路径、方言兼容、对未来趋势（P1-1 Trends）与归因共用底座，能一并解 P1-1 的 P2-10 依赖。
- **推荐接口方案 A（扩展 NurtureKPI）**：`growth_trend` 动态判断（有历史 → `data_source:"trend"` / 无 → not_available+积累文案），**不新增端点、不新增能力**。
- **打地基阶段诚实预期**：在积累满 N 周前，growth_trend 恒为 not_available 是**预期行为**，前端以「数据积累中，需满 N 周」提示（并对该卡隐藏误导性的补料/工单入口）。
- **不做**：不造数、不硬编、不改现有 UpsertSnapshot 语义（除非按 b 重拍板）、不引入本地模型、不跨院越权、不做因果表述。
- **主要产出**：本核对清单（此文件），供 dev 依 P0→P1 落地。
