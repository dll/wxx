# API 契约索引

> 蔚小芯后端 API 端点完整登记。所有端点均前缀 `/api/v1`（健康检查除外）。

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
| 2026-05-06 | 新增 `GET /recommendations` 个性化推荐引擎 | 向前兼容 |
| 2026-05-05 | 新增 `GET /export` 知识导出（替代占位符 501） | 向前兼容 |
| 2026-05-05 | 新增 `DELETE /sessions/:id` | 向前兼容 |
| 2026-05-04 | 新增情感预警全套：`/emotion/stats`、`/emotion/analyze`、`/emotion/alerts`、`/emotion/alerts/:id` | 向前兼容 |
| 2026-05-04 | 新增智能体管理 CRUD：`/agents`（5 端点） | 向前兼容 |
| 2026-05-04 | 新增语音：`/voice/asr`、`/voice/tts` | 向前兼容 |
| 2026-05-03 | 新增 `/knowledge` 知识大厅浏览 | 向前兼容 |
| 2026-05-05 | 新增 `POST /kb/import`（NDJSON/JSON 双格式）、校外系统对接 3 端点、`GET /emotion/trends` 趋势分析 | 向前兼容 |
| 2026-05-02 | 基线：`/auth/login`、`/user/profile`、`/chat`、`/sessions`、`/kb/resources` | — |
