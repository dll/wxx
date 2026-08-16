# OpenClaw 实践重构蔚小芯（WXX）记录与总结

> 时间跨度：2026-08-14 ~ 2026-08-16（核心集中开发窗口）
> 主角：OpenClaw 智能体（本会话）作为开发主力，协同用户交付「蔚小芯数字化学生教育工作智能体」
> 文档性质：完整复盘一次「AI 深度参与生产级项目」的开场——从接手真实课表数据入库，到 CI/CD 部署架构根治，到党建育人闭环两块功能上线，全程记录开始、过程、成果、问题与解法。

---

## 一、开场：项目现状与首个任务

### 1.1 项目背景快照
蔚小芯（WXX）是面向高校学生教育工作的数字化智能体平台：

- **前端**：Flutter 3.22 / Dio / Provider
- **后端**：Go 1.x / Gin / JWT / RBAC（能力门控）
- **数据**：MySQL 8.0.46 + Redis（本阶段刚从 SQLite 迁到 MySQL 生产库）
- **模型侧**：智谱 / DeepSeek / 讯飞
- **生产**：腾讯云 `129.211.223.113`（Ubuntu），systemd `wxx` 服务，Caddy 反代 `wxx-agent.online`
- **部署**：GitHub Actions CI/CD（quality-gate + 前端部署 + 后端部署 + APK 构建）

### 1.2 会话起始指令（用户原话）
> 「教师课表已经入库。真实，不要假的。成绩来自教师导入。教师课程学生关联。重跑」

这奠定了本阶段的第一条铁律，并延伸出整个工作纲领：

- **真实数据，绝不编造**（绩效/学分/成绩只来自真实来源）
- **诚实边界**：`data_source: real / not_available / fallback` + 前端 `DataSrcBadge` 标注来源
- 所有开发围绕「把真实业务数据接进系统、让各角色看到自己范围的真实育人数据」展开

---

## 二、开始工作的推进脉络

本阶段按「数据 → 归属 → 闭环 → 部署 → 审核」五步走：

| 阶段 | 核心问题 | 产出 |
|------|---------|------|
| 1. 数据入库 | 真实课表如何进生产 MySQL | `import_schedules_mysql.py` + 真实课表 10,860 条 |
| 2. 数据归属 | 学生 owner_id 断点，辅导员查不到学生 | 491 学生归属统一为 `cs` 短码 |
| 3. 党建闭环 | 书记看板 → 党课登记 → 协同育人 | party-dashboard / party-register / collab-dashboard |
| 4. 部署根治 | 服务器→GitHub 拉源码链路不稳，反复失败 | runner 编译 + rsync 上传二进制 |
| 5. 发布审核 | 达可发布标准 | 审核报告 v2 全角色×流程矩阵 |

---

## 三、主要成果（含 Commit 对应）

### 3.1 真实课表落库（已完成，commit `6874217` / `f3298a3`）
- 写 **MySQL 版**导入脚本 `server/scripts/import_schedules_mysql.py`（原脚本是 SQLite 版，已废弃）
- 生产新增 **194 个教师账号**（密码=工号）+ **346 节教师课表** + **10,514 节学生课表**
- 课程表 `course_schedules = 10,860`（INSERT IGNORE 去重）
- 用户数：`users=699`、`teacher=195`、`student=494`
- 用真实工号（203197 / 203129）登录验证成功
- 备份 `wxx_pre_sched_import_20260816_000042.sql`
- **诚实补记**：131 份教师课表为空模板（无课）非格式错，63 份有课全导入无遗漏——不误判丢数据

### 3.2 数据归属断点修复（已完成，commit 后验证）
- **根因**：491 个真实学生（全在计算机学院）owner_id 存中文学院名（「学院(网安、大学计算机教学部)」360 +「学院(网安)」131）；而辅导员/学院管理员/学生会/教学助理全用短码 `cs` → 后端按 `owner_id=cs` 精确过滤查不到学生
- **修复**：备份 → `UPDATE users SET owner_id='cs'`（491行） + `UPDATE student_profile_snapshot`（116行）
- **核对**：学生 owner_id 现 `cs=493`（491真实+2test）、`math=1`（仅test）、中文名残留 0
- 后端 scope 过滤全为 owner_id 精确匹配，数据级验证通过

### 3.3 党建育人闭环（本阶段核心功能）
#### 第 1 块：书记党建聚合看板（commit `2cea406`，生产冒烟全绿）
- `GET /api/v1/college/party-dashboard`（复用 outcome.dashboard 权限）
- 聚合：①入党漏斗（party_progress.stage）②党员数（正式/预备）③党课学习（人次/时长/按类型）
- **关键修复**：本院过滤用 `JOIN users.owner_id`（短码 cs），而非旧 EducationOutcomeDashboard 用的 `party_progress.college`（中文学名）——修掉「书记本院永远查不到」的 bug
- 诚实 data_source：0行=not_available / 有行=self_reported（自报非组织确认，符合不瞎编红线）
- 前端 SecretaryProvider.fetchPartyDashboard + 书记大屏 _buildParty + DataSrcBadge

#### 第 2 块：协同育人总览（commit `b4cfabc`，已上线）
- `GET /api/v1/college/collab-dashboard`（capability `college.collab.dashboard`，学院书记本院 / 学校书记全校）
- 聚合：谈心（talk_records）、后勤服务（facility_records）、党建活动（party_study by created_by）、教学排课（course_schedules by teacher）按学院（users.owner_id）+ 角色
- 前端 fetchCollabDashboard + 大屏新块

#### 第 3 块：党课/活动登记（commit `b4cfabc`，已上线）
- **设计决策**：复用 `party_study_records` + 迁移 089 加 `created_by / created_by_role / paid` 列——不改表、登记直进现有 party 聚合 → 书记看板立即可见
- `POST /teacher/party/register` + `GET /teacher/party/records` + `DELETE /teacher/party/records/:id`
- 教师/辅导员/教辅持有 `party.record.write/read`
- 前端 PartyActivityRegisterPage + home 入口

#### 内置账号统一密码（commit `b4cfabc`）
- `fixPasswordHashes` 启动强制重置 14 个种子账号 → 统一 `Wxx@2026`（有效 bcrypt 也覆盖，真实账号不动）
- 生产验证：collegeadmin / schooladmin 登录成功

### 3.4 清理过时旧数据（已完成，约 1.6G）
- 临时 tarball / 旧解压目录 / 15 个部署日志 / 4 个陈旧 json / go-cache-audit(266M) / 19 个旧二进制备份 + server.bak 源码目录
- 保留 2 个最新二进制回滚 + backup/11 份 MySQL dump
- 遗留 SQLite wxx.db(6.5M) 归档到 `backup/legacy-sqlite-wxx.db`

### 3.5 发布审核（commit `12da117`，达可发布标准）
- `docs/发布审核报告v2-2026-08-16.md`：角色 × 流程矩阵
- 结论：全角色×四维审核、双端能力同步、达可发布标准
- 遗留（数据待补，非缺陷）：成绩入库、A类21班名单、党建真实数据、协同育人口径#3

### 3.6 CI/CD 部署架构根治（核心工程成就，commit `2d796c9→91eecb2→8834b91`，全绿验证）
**从根上消除「服务器→GitHub 拉源码」的不稳定链路**（用户痛点：避免测试重试）。

- **旧架构问题**：deploy-server 在服务器本地编译，依赖服务器去 GitHub `git fetch` / codeload 下载源码，国内网络极不稳定，反复 job 失败
- **新架构**：
  1. runner 上 checkout + setup-go(1.25) → `go build -tags fts5` → `./server/dist/wxx-server.new`
  2. 上传：**easingthemes/ssh-deploy@v5.0.3（rsync over SSH）** `server/dist/` → 服务器 `/tmp/wxx-dist`
  3. 服务器 ssh：校验二进制 → 备份 MySQL → 保留回滚二进制 → 停服 → 替换 → 启动 → 健康检查
- **彻底绕开服务器外网依赖**，部署不再因网络抖动重试
- **验证**：Deploy Frontend success（7m20s），二进制 08:29 上生产（64,512,843 字节），服务 active、`database ok`

---

## 四、遇到的问题及解决（重点复盘）

### 问题 1：真实课表导入脚本是 SQLite 版
- **表象**：`import_schedules.py` 面向旧 SQLite 库
- **解决**：重写 MySQL 版 `import_schedules_mysql.py`，走 TCP（`-h127.0.0.1`），INSERT IGNORE 去重；当前库 `DB_DRIVER=mysql`，MySQL 才是真实来源

### 问题 2：数据归属断点（owner_id 中文名 vs 短码）
- **表象**：辅导员按 owner_id 只能查到少数学生，第二课堂/名单看板不诚实
- **解决**：统一 491 学生 owner_id 为 `cs`；**未擅自改归属体系**（影响面大），从根上让 scopes 生效
- **纪律**：改动前备份，改后数据级核对

### 问题 3：`users.real_name` 列不存在（Error 1054）
- **表象**：党课登记列表 `ListPartyRecords` 引用 `u.real_name` 报 MySQL `Error 1054 Unknown column`
- **根因**：users 表人名列实为 `display_name`
- **解决**：commit `89594d3` 改查 `display_name`；冒烟暴露并修复

### 问题 4：CI 静默用旧代码编译（严重）
- **表象**：party-dashboard（2cea406）已提交，但生产端点 404——部署看似 success 实为编译了旧源码
- **根因**：git fetch 3 次失败 → 回退逻辑「用现有源码编译并打 [OK]」，job 显示 success 但代码是旧的
- **解决（commit `be1c301`）**：源码获取三态——A) git fetch 重试3次成功则 reset origin/main；B) 全败回退 HTTPS tarball 到 /tmp 全新检出；C) 都失败则 **exit 1 整个 job 失败**（绝不静默用旧代码）

### 问题 5：tarball 下载截断但 curl 不报错（数据完整性）
- **表象**：codeload 下载中途断连产出 23MB 截断文件，`curl -fsSL` 返回 0（curl 不校验 Content-Length）
- **解决（commit `9cf3633`）**：tarball 下载后 `gzip -t` 校验，截断则删除重试3次；curl 加 `--retry 2`
- **教训**：codeload tarball 下载务必备 gzip -t 校验；curl -f 检测不出中途断连

### 问题 6：scp-action 对 64MB 二进制 tar 空包（本阶段最磨人的坑）
- **表象**：`appleboy/scp-action`（内部 drone-scp）+ 完整日志尾部 `tar: empty archive` + `exit status 1`，连续两轮失败
  - 第 1 次：`source: "wxx-server.new"`（裸顶层文件名）→ 空包
  - 第 2 次：`source: "dist/wxx-server.new"` + `strip_components: 1`（按 README 目录前缀模式）→ 仍空包
- **根因**：drone-scp 在其 Docker 容器里对超大二进制 tar 不可靠
- **解决（commit `8834b91`）**：弃用 scp-action，改用与前端 deploy-web 同款、已稳定运行的 **easingthemes/ssh-deploy@v5.0.3（rsync over SSH）**
- **关键经验**：GitHub Actions 传大二进制优先 rsync（ssh-deploy），不要用 scp-action（drone-scp 的 docker tar 对大文件不可靠）；编译产物须放 workspace 内（`server/dist/`）action 才能找到

### 问题 7：CI quality-gate 早先被 gofmt / flutter analyze 阻断
- **表象**：gofmt 未格式化新文件 + flutter analyze warning（unused 字段）卡红质量门禁
- **解决**：`gofmt -l server/` 干净 + `flutter analyze --no-fatal-infos` 0 error/warning；全绿后才放行部署

---

## 五、关键纪律与决策（沉淀为长期经验）

1. **诚实边界「不瞎编」**：虚构学生/人物以真实身份呈现=最严重违规；绩效/学分只来自真实记录；`data_source` 三态 + `DataSrcBadge`
2. **部署走 CI/CD，不手动 scp 乱试**（用户明确批评过）
3. **CI 绝不静默旧代码**：源码获取三态 + tarball gzip -t 校验，失败就 job 失败
4. **大文件上传用 rsync（ssh-deploy）不用 scp-action**
5. **改生产数据先展示方案、用户确认再动**；大数据量修改（491行）注意安全和备份
6. **能力驱动非硬编码**：前后端能力常量双端同步
7. **复杂 SQL 用文件方式**（write 本地 → scp → `mysql < file`），shell 内嵌中文引号/反引号易被吃
8. **GitHub Actions 编译产物放 workspace 内**（rsync/scp action 才能定位）
9. **MySQL 连 TCP**：`-h127.0.0.1` 显式（localhost socket 认证失败）
10. **gh CLI 走机器级 `GH_TOKEN`**（非当前会话环境变量）

---

## 六、生产验证（最终端到端冒烟全绿）

| 验证项 | 结果 |
|--------|------|
| 内置密码 Wxx@2026 登录 collegeadmin/schooladmin | ✅ OK |
| 协同育人总览 本院(owner_id=cs) students_total=493 | ✅ |
| 协同育人总览 全校 students_total=494 | ✅ |
| 党课登记列表（display_name 修复）真实教师203197 data_source=real | ✅（无 Error1054）|
| 未带 token 访问 401 | ✅ |
| 服务 active + health database ok | ✅ |
| 协同/党建记录数=0 | 诚实显示 not_available（等真实数据，非缺陷）|

---

## 七、总体总结

### 这次实践证明了什么
1. **AI 可承担生产级全栈重构的主力**：从数据库迁移适配、数据修复、后端端点、前端页面、CI/CD 排障到审核文档，一条链路闭环交付，而非零散补丁。
2. **「真实数据 + 诚实边界」是可交付的护城河**：多次主动停下不编造（成绩/A类21班等真实数据），把「哪些是缺陷、哪些是数据待补」讲清楚，反而让审核更快达标。
3. **工程韧性比功能数量更重要**：本阶段真正磨人的不是写功能，而是让「部署稳定不重试」。从服务器本地编译 → 三态源码获取 → gzip 校验 → runner 编译 + rsync 上传，一步步把不稳定链路连根拔掉——这是对用户「避免测试重试」最实在的回应。
4. **排障方法论沉淀有效**：每次失败都「拉完整日志 → 定位根因 → 改结构而非打补丁 → 留文档/记忆」。scp-action 的坑、display_name 的坑都成了可复用的经验。

### 遗留与展望（数据待补，非缺陷）
- **成绩入库**：`admin.grades.import` 入口已有，等教师上传真实成绩（student_grades 保持 0）
- **A 类 21 班课表**：缺学生名单未导，等真实名单
- **党建真实数据**：party_progress=0、party_study_records 只有冒烟登记，看板诚实显示「待党建真实数据」
- **协同育人口径 #3**：是否只统计教师/教辅真实动作，已按真实记录实现，待用户确认口径

### 给后来的自己 / 读者的一句话
> 生产级 AI 开发的分水岭不在「能不能写代码」，而在「能不能在数据真实、部署可靠、边界诚实的前提下，把一段不稳定链路一路修到不需要重试」。本阶段做到了，这是蔚小芯可以开始被信任的起点。
