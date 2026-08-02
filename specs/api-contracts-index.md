# API 契约索引

> 蔚小芯后端 API 端点完整登记。所有端点均前缀 `/api/v1`（健康检查除外）。

## 目录

- [认证与用户](#认证与用户)
- [对话与会话](#对话与会话)
- [知识](#知识)
- [情感预警](#情感预警)
- [智能体管理](#智能体管理)
- [语音](#语音)
- [系统](#系统)
- [问题预案](#问题预案issue-forecast)
- [毕设选题](#毕设选题graduation-topics)
- [学科竞赛](#学科竞赛competitions)
- [大学规划](#大学规划plan-templates)
- [入党教育](#入党教育party-education)
- [社团生活](#社团生活clubs)
- [通用约定](#通用约定)
- [变更登记](#变更登记)

## 端点总览

### 认证与用户

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `POST` | `/auth/login` | 无 | — | 用户名登录，返回 JWT + 角色 + display_name |
| `GET` | `/user/profile` | JWT | 全部 | 获取当前用户完整资料 |

### 对话与会话

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `POST` | `/chat` | JWT | 全部 | 提交问题，返回 AnswerCard（conclusion + steps + sources + risks + followUps） |
| `GET` | `/sessions` | JWT | 全部 | 当前用户的会话列表 |
| `GET` | `/sessions/:id/messages` | JWT | 全部 | 获取会话历史消息（含 answer_card） |
| `DELETE` | `/sessions/:id` | JWT | 全部 | 删除会话及其消息 |

### 知识

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `GET` | `/knowledge` | JWT | 全部 | 知识大厅浏览，按类型分组返回轻量卡片 `?type=Policy\|Process\|FAQ\|Activity` |
| `GET` | `/recommendations` | JWT | 全部 | 个性化推荐 `?limit=10`，基于用户历史提问+FTS5搜索+冷启动热门兜底 |
| `GET` | `/kb/resources` | JWT | ≥ counselor | 资源分页列表 `?page=&page_size=&resource_type=&status=&owner_scope=` |
| `POST` | `/kb/resources` | JWT | ≥ counselor | 创建知识资源 |
| `GET` | `/kb/resources/:id` | JWT | ≥ counselor | 获取资源完整详情 |
| `PUT` | `/kb/resources/:id` | JWT | ≥ counselor | 更新资源（部分字段合并） |
| `GET` | `/export` | JWT | 全部 | 导出已发布资源 `?resource_type=&since=`，返回 manifest + data |
| `POST` | `/kb/import` | JWT | ≥ counselor | 导入资源（NDJSON 或 JSON 包裹），返回逐条结果统计 |
| `POST` | `/documents/parse` | JWT | ≥ counselor | 文档解析（PDF/DOCX/TXT/MD/CSV/XLSX → 标题/摘要/关键词/正文） |
| `POST` | `/documents/refine` | JWT | ≥ counselor | LLM 精修文档元数据 `{content, title?, summary?, keywords?}` → 精修后标题/摘要/关键词（失败自动回退原值） |
| `GET` | `/integration/status` | JWT | ≥ counselor | 校外系统可用状态 |
| `GET` | `/integration/xuegong/*path` | JWT | ≥ counselor | 代理学工系统查询 |
| `GET` | `/integration/ybt/*path` | JWT | ≥ counselor | 代理一表通查询 |

### 情感预警

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `GET` | `/emotion/stats` | JWT | 全部 | 告警聚合统计（紧急/高/中/低/待处理），角色在 service 层过滤范围 |
| `POST` | `/emotion/analyze` | JWT | ≥ counselor | LLM 分析消息情感，返回风险等级 + 情绪 + 关键词 + 理由 |
| `GET` | `/emotion/alerts` | JWT | ≥ counselor | 告警分页列表 `?risk_level=low\|medium\|high\|urgent&status=pending\|acknowledged\|resolved&page=&page_size=` |
| `PUT` | `/emotion/alerts/:id` | JWT | ≥ counselor | 更新告警状态 `{status: "acknowledged" \| "resolved"}` |
| `GET` | `/emotion/trends` | JWT | ≥ counselor | 每日情感趋势 `?days=7`，返回逐日聚合 + 汇总指标 |

### 智能体管理

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `GET` | `/agents` | JWT | ≥ school_admin | 全部智能体列表（按类型排序） |
| `POST` | `/agents` | JWT | ≥ school_admin | 创建智能体（agent_id 唯一） |
| `GET` | `/agents/:id` | JWT | ≥ school_admin | 获取智能体详情 |
| `PUT` | `/agents/:id` | JWT | ≥ school_admin | 更新智能体（部分字段合并） |
| `DELETE` | `/agents/:id` | JWT | ≥ school_admin | 删除智能体 |

### 语音

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `POST` | `/voice/asr` | JWT | 全部 | 语音识别（multipart/form-data PCM 音频 → JSON 文本） |
| `POST` | `/voice/tts` | JWT | 全部 | 语音合成（JSON 文本 → audio/mpeg 二进制流） |

### 系统

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `GET` | `/health` | 无 | — | 健康检查 `{"status":"running","service":"蔚小芯","version":"0.0.1","db":"ok"}` |

---

### 问题预案（Issue Forecast）

> **可见角色**：`sys_admin`、`college_admin`

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `POST` | `/forecast/analysis` | JWT | sys_admin.forecast | 执行问题分析，汇总多源数据并生成预案 |
| `GET` | `/forecast/issues` | JWT | sys_admin.forecast | 问题预案列表 `?category=&risk_level=&status=&page=&page_size=` |
| `GET` | `/forecast/issues/:id` | JWT | sys_admin.forecast | 获取单个问题预案详情 |
| `PUT` | `/forecast/issues/:id/status` | JWT | sys_admin.forecast | 更新状态 `{status: "pending" \| "processing" \| "resolved" \| "archived"}` |
| `GET` | `/forecast/statistics` | JWT | sys_admin.forecast | 问题统计 `?days=30`，返回风险分布 + 分类分布 + 每日趋势 |

---

### 毕设选题 (Graduation Topics)

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `GET` | `/graduation/advisors` | JWT | 全部 | 导师列表 |
| `GET` | `/graduation/available-topics` | JWT | 全部 | 可选选题列表 |
| `POST` | `/graduation/select-topic` | JWT | student | 选择选题 |
| `GET` | `/graduation/my-selection` | JWT | student | 我的选题 |
| `GET` | `/graduation/milestones` | JWT | 全部 | 里程碑列表 |
| `GET` | `/graduation/stats` | JWT | 全部 | 毕设进度统计 |
| `GET` | `/graduation/selections` | JWT | ≥ counselor | 选题列表（教师/管理员） |
| `PUT` | `/graduation/selections/:id/confirm` | JWT | ≥ counselor | 确认选题 |

### 学科竞赛 (Competitions)

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `GET` | `/competition/list` | JWT | 全部 | 竞赛列表 |
| `GET` | `/competition/:id` | JWT | 全部 | 竞赛详情 |
| `POST` | `/competition/register` | JWT | student | 报名竞赛 |
| `GET` | `/competition/my-registrations` | JWT | student | 我的报名 |
| `POST` | `/competition/submit-work` | JWT | student | 提交作品 |
| `GET` | `/competition/stats` | JWT | 全部 | 竞赛统计 |

### 大学规划 (Plan Templates)

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `GET` | `/plan/templates` | JWT | 全部 | 规划模板列表 |
| `GET` | `/plan/my-plans` | JWT | student | 我的规划 |
| `POST` | `/plan/create` | JWT | student | 创建规划 |
| `PUT` | `/plan/:id/submit` | JWT | student | 提交审核 |
| `PUT` | `/plan/:id/review` | JWT | ≥ counselor | 审核规划 |

### 入党教育 (Party Education)

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `GET` | `/party/stages` | JWT | 全部 | 入党阶段列表 |
| `GET` | `/party/my-progress` | JWT | student | 我的进度 |
| `PUT` | `/party/my-progress` | JWT | student | 更新进度 |
| `GET` | `/party/my-study-records` | JWT | student | 学习记录 |
| `POST` | `/party/study-record` | JWT | student | 新增学习记录 |
| `GET` | `/party/stats` | JWT | 全部 | 入党统计 |

### 社团生活 (Clubs)

| 方法 | 路径 | 认证 | 角色要求 | 说明 |
|------|------|------|----------|------|
| `GET` | `/club/list` | JWT | 全部 | 社团列表 |
| `GET` | `/club/:id` | JWT | 全部 | 社团详情 |
| `POST` | `/club/join` | JWT | student | 加入社团 |
| `GET` | `/club/my-clubs` | JWT | student | 我的社团 |
| `GET` | `/club/activities` | JWT | 全部 | 活动列表 |
| `POST` | `/club/activity/register` | JWT | student | 报名活动 |

---

## 通用约定

| 约定 | 说明 |
|------|------|
| 认证方式 | `Authorization: Bearer <jwt_token>`（JWT 有效期 2 小时） |
| 成功响应 | `{"code": 0, "message": "...", "data": ...}` |
| 错误响应 | `{"code": 4xx/5xx, "message": "...", "trace_id": "..."}` |
| 分页 | `page`（默认 1）、`page_size`（默认 20，最大 100） |
| Content-Type | 请求 `application/json`，语音 `multipart/form-data` |

## 变更登记

| 日期 | 变更摘要 | 兼容策略 |
|------|----------|----------|
| 2026-08-02 | 新增 `POST /documents/refine` LLM 元数据精修；`kb_fts` 索引纳入 tags（049 迁移） | 向前兼容 |
| 2026-06-19 | 新增毕设选题、学科竞赛、大学规划、入党教育、社团生活 5 大模块（35 端点） | 向前兼容 |
| 2026-06-19 | 新增问题预案：`/forecast/analysis`、`/forecast/issues`、`/forecast/issues/:id`、`/forecast/statistics`（5 端点） | 向前兼容 |
| 2026-05-06 | 新增 `GET /recommendations` 个性化推荐引擎 | 向前兼容 |
| 2026-05-05 | 新增 `GET /export` 知识导出（替代占位符 501） | 向前兼容 |
| 2026-05-05 | 新增 `DELETE /sessions/:id` | 向前兼容 |
| 2026-05-04 | 新增情感预警全套：`/emotion/stats`、`/emotion/analyze`、`/emotion/alerts`、`/emotion/alerts/:id` | 向前兼容 |
| 2026-05-04 | 新增智能体管理 CRUD：`/agents`（5 端点） | 向前兼容 |
| 2026-05-04 | 新增语音：`/voice/asr`、`/voice/tts` | 向前兼容 |
| 2026-05-03 | 新增 `/knowledge` 知识大厅浏览 | 向前兼容 |
| 2026-05-05 | 新增 `POST /kb/import`（NDJSON/JSON 双格式）、校外系统对接 3 端点、`GET /emotion/trends` 趋势分析 | 向前兼容 |
| 2026-05-02 | 基线：`/auth/login`、`/user/profile`、`/chat`、`/sessions`、`/kb/resources` | — |
