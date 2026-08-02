# RBAC 矩阵 — 蔚小芯权限定义

> 全量角色定义见 `server/internal/auth/capabilities.go`，基线六级见 `docs/蔚小芯智能体.md`（相对 `WXX/`）§6.6。  
> `teacher` / `assistant` 已在能力模型实现中，不再是"建议扩展"。本矩阵以能力模型为基础，按功能模块/端点归纳。

---

## 角色状态更新

> **现状**：`teacher` 与 `assistant` 已在代码中实现完整能力定义与继承关系（`server/internal/auth/capabilities.go`），不再是"建议扩展"角色。  
> **历史决策**（仅供参考）：原 P0 阶段未强制纳入，后因 SSO 职工类型映射需要提前实现。以下保留说明仅作决策记录。

**角色已实现**：`teacher`（继承 `student_union`，拥有备课、出题、学情热力图等能力）、`assistant`（继承 `student_union`，拥有排课检查、毕业审核、考试安排等能力）。`college_admin` 同时继承 counselor / teacher / assistant 三线。

---

## 角色边界（扩展角色与 counselor 区分）

| 角色代码 | 中文 | 典型职能（相对学工助手） |
|----------|------|--------------------------|
| `counselor` | 辅导员 / 班主任 | 思政与日常管理、重点学生关注、心理预警跟进、奖助与请假等流程指导；可访问 **学工敏感个案** 需在策略中单列。 |
| `teacher` | 教师（专任教师等） | **学业与育人协同**：课程相关通知传达、学业/竞赛/毕设类指引、公开政策问答；宜 **默认不包含** 与辅导员同级的个案预警详情库，除非单独授权。 |
| `assistant` | 教辅（教务秘书、实验室管理等） | **事务协办**：证明材料、场地与教务环节衔接、学院行政下发传达；权限宜贴近 **流程与文档** ，而非全员心理画像。 |

具体菜单/API 仍以教务与学工授权批复为准；上表用于矩阵填空时的口径对齐。

---

## 角色列表（固化名称）

### 全部角色（与 server/internal/auth/capabilities.go 一致）

0. 游客（`guest`）— 未登录访问公开信息
1. 系统管理员（`sys_admin`）
2. 学校（`school_admin`）
3. 二级学院（`college_admin`）
4. 辅导员 / 班主任（`counselor`）
5. 教师（`teacher`）
6. 教辅（`assistant`）
7. 学生会 / 班团委（`student_union`）
8. 学生（`student`）

**继承链**：`sys_admin → school_admin → college_admin → {counselor, teacher, assistant} → student_union → student`，另含 `guest` 独立节点。`college_admin` 多父继承 counselor + teacher + assistant。

---

## 权限矩阵 — 基线六级

范围说明：「全部」= 不限范围；「学校级」= 学校层面资源；「本院」= 本二级学院范围；「本人学生」= 辅导员管辖的学生；「禁止」= 无权限。

| 功能模块 / API 端点 | sys_admin | school_admin | college_admin | counselor | student_union | student |
|----------------------|-----------|--------------|---------------|-----------|---------------|---------|
| **认证与用户** | | | | | | |
| `POST /auth/login` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /user/profile` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **对话与会话** | | | | | | |
| `POST /chat` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /sessions` | 全部 | 全部 | 本院 | 本人学生 | 自身 | 本人 |
| `GET /sessions/:id/messages` | 全部 | 全部 | 本院 | 本人学生 | 自身 | 本人 |
| `DELETE /sessions/:id` | 全部 | 全部 | 自身 | 自身 | 自身 | 自身 |
| **知识** | | | | | | |
| `GET /knowledge` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /recommendations` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /kb/resources` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| `POST /kb/resources` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| `PUT /kb/resources/:id` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| `GET /kb/resources/:id` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| `GET /export` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `POST /kb/import` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| `POST /documents/parse` | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| `POST /documents/refine` | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| `POST /kb/batch/refine` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| **校外系统对接** | | | | | | |
| `GET /integration/status` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| `GET /integration/xuegong/*path` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| `GET /integration/ybt/*path` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| **情感预警** | | | | | | |
| `GET /emotion/stats` | 全部 | 全部 | 本院 | 本院摘要 | 禁止 | 禁止 |
| `POST /emotion/analyze` | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| `GET /emotion/alerts` | 全部 | 全部 | 本院 | 本人学生 | ❌ | ❌ |
| `PUT /emotion/alerts/:id` | 全部 | 全部 | 本院 | 本人学生 | ❌ | ❌ |
| `GET /emotion/trends` | 全部 | 全部 | 本院 | 本人学生 | ❌ | ❌ |
| **智能体管理** | | | | | | |
| `GET/POST /agents` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `GET/PUT/DELETE /agents/:id` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **语音** | | | | | | |
| `POST /voice/asr` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `POST /voice/tts` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **校园文化（全员可见）** | | | | | | |
| `GET /culture/anthems` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /culture/radio` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /culture/lectures` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /culture/events` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /culture/volunteer` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **系统（预留）** | | | | | | |
| 用户管理 | 全部 | 学校级 | 本院级 | ❌ | ❌ | ❌ |
| 审计日志查看 | 全部 | 全部 | 本院 | ❌ | ❌ | ❌ |
| 系统配置 | 全部 | ❌ | ❌ | ❌ | ❌ | ❌ |

> ✅ = 已认证即可访问（无额外角色限制）  
> ❌ = 禁止访问（返回 403）  
> 范围说明：「全部」= 不限范围；「学校级」= 学校层面；「本院」= 二级学院范围；「本人学生」= 辅导员管辖学生

---

## 权限矩阵 — teacher / assistant（已实现）

以下为教师与教辅的高层级权限归纳（对应代码中 `teacher.*` 和 `assistant.*` 能力组）：

| 功能模块 | teacher | assistant |
|----------|---------|-----------|
| 智能问答（SSE） | 本院公开 | 本院公开 |
| 语音交互（ASR/TTS） | 本院 | 本院 |
| 会话历史查看 | 自身 | 自身 |
| 知识资源创建 | 学业/课程类 | 事务/流程类 |
| 知识资源审核/发布 | 禁止 | 禁止 |
| FTS 知识检索 | 本院公开 | 本院公开 |
| 流程步骤管理 | 只读 | 协办（编辑材料） |
| 导出（PDF/Word） | 仅本人会话 | 仅本人会话 |
| 审计日志查看 | 禁止 | 禁止 |
| 情感预警查看 | 默认关闭（需授权） | 禁止 |
| 用户管理 | 禁止 | 禁止 |
| 统计看板 | 本人授课摘要 | 禁止 |

**说明**：具体能力定义见 `server/internal/auth/capabilities.go`。本文件作为 **评审与验收对照表**。若修改总纲角色定义请同步更新 `docs/蔚小芯智能体.md` §6.6 及 `roleScope` 说明。
