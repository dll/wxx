# AI 简讯模块

> 版本：v2.0 · 2026-08-11
> 首页新增「AI 简讯」卡片，面向全部登录用户；管理端（sys_admin 专属）提供资讯 CRUD、来源设置、自动抓取与多格式导出。
> v2.0 参考 AIHOT 重构用户端为「精选流 / 热度榜 / 我的收藏」三 Tab 资讯门户，见 `docs/ai-briefings-refactor.md`。

## 功能清单

1. **首页 AI 简讯卡片**：登录后首页展示，点击进入资讯门户 `/ai-briefings`。
2. **资讯门户（用户端）**：三 Tab——
   - **精选**：按日期分组的信息流（今天/昨天/日期头），每条含 来源徽标 · 时间 · 热度值 · 分类 · 标题 · 摘要 · 推荐理由 · 阅读原文；支持分类筛选与关键词搜索。
   - **热度榜**：按热度值（heat）降序 Top 50。
   - **我的收藏**：当前用户收藏的资讯列表。
   - 每条支持收藏/取消收藏（★）。
3. **管理端（sys_admin）**：
   - 资讯 CRUD（新增/编辑/上下架/删除），表单含热度值与推荐理由
   - 筛选查找（分类 / 状态 / 关键词）
   - 汇总统计（总数 / 上架 / 下架 / 自动抓取 / 手动录入）
   - 批量操作：多选删除、清空历史
   - 导出：单个 / 多选 / 全部，格式 md、pdf
4. **来源设置**：RSS/Atom 来源 CRUD + 启用开关 + 每日抓取时间（HH:MM）+ 抓取归入分类。
5. **自动获取**：后台每分钟检查，到达来源配置的抓取时刻且当日未抓取时，拉取 RSS/Atom 并入库（RSS 2.0 / Atom 1.0 轻量解析，标题 / 链接 / 摘要 / 发布时间）。

## 主题分类

| key | 说明 |
|-----|------|
| `ai_teaching` | AI 辅助教学 |
| `ai_tool` | AI 工具 |
| `ai_version` | AI 版本 |
| `ai_industry` | AI 行业热点 |

## 接口

### 用户端

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/v1/ai-briefings` | 已认证 | 资讯列表（category/q/limit），返回含 heat/reason/favorited |
| GET | `/api/v1/ai-briefings/hot` | 已认证 | 热度榜 Top 50 |
| GET | `/api/v1/ai-briefings/favorites` | 已认证 | 我的收藏 |
| POST | `/api/v1/ai-briefings/:id/favorite` | 已认证 | 收藏（幂等） |
| DELETE | `/api/v1/ai-briefings/:id/favorite` | 已认证 | 取消收藏（幂等） |

### 管理端

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/v1/admin/ai-briefings` | sys_admin | 管理端分页列表（status/category/q） |
| POST | `/api/v1/admin/ai-briefings` | sys_admin | 新增资讯 |
| PUT | `/api/v1/admin/ai-briefings/:id` | sys_admin | 更新资讯 |
| PUT | `/api/v1/admin/ai-briefings/:id/status` | sys_admin | 上下架 |
| DELETE | `/api/v1/admin/ai-briefings/:id` | sys_admin | 删除 |
| POST | `/api/v1/admin/ai-briefings/batch-delete` | sys_admin | 批量删除 |
| DELETE | `/api/v1/admin/ai-briefings/clear` | sys_admin | 清空历史 |
| GET | `/api/v1/admin/ai-briefings/stats` | sys_admin | 汇总统计 |
| GET | `/api/v1/admin/ai-briefings/export?format=md\|pdf&all=1\|ids=1,2,3` | sys_admin | 导出 |
| POST | `/api/v1/admin/ai-briefings/fetch` | sys_admin | 立即抓取 |
| GET | `/api/v1/admin/ai-briefings/sources` | sys_admin | 来源列表 |
| POST | `/api/v1/admin/ai-briefings/sources` | sys_admin | 新增来源 |
| PUT | `/api/v1/admin/ai-briefings/sources/:id` | sys_admin | 更新来源 |
| DELETE | `/api/v1/admin/ai-briefings/sources/:id` | sys_admin | 删除来源 |

## 数据库

- `ai_briefings`：资讯主体（source/category/topic/summary/content/link/keyword/heat/reason/published_at/fetched_at/status）
- `ai_briefing_sources`：RSS 来源配置（name/url/category/enabled/fetch_enabled/fetch_time/last_fetch_at）
- `ai_briefing_favorites`：用户收藏（user_id/briefing_id，UNIQUE(user_id,briefing_id)）

迁移文件：`server/migrations/066_ai_briefings.sql`、`server/migrations/073_ai_briefings_refactor.sql`

## 能力

- `self.ai_briefing.read`：浏览 AI 简讯（student 基线）
- `system.ai_briefing`：AI 简讯管理（sys_admin 专属）
- 收藏属个人数据，跟随 `self.ai_briefing.read`（student 基线即可使用）
