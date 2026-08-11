# 任务方案 — AI 简讯参考 AIHOT 重构

> 版本：v1.0 · 2026-08-11 · 待审核

## 背景与目标

现有 AI 简讯（`docs/ai-briefings.md`）仅提供「简单列表 + 管理端 CRUD」，交互与信息呈现单一。
参考站 `https://aihot.virxact.com/`（AI 行业动态聚合）具备：**精选信息流、热度榜、分类 Tab、日期分组、推荐理由、收藏、搜索** 等形态。

目标：以 AIHOT 为蓝本，将 AI 简讯用户端重构为「校园 AI 资讯门户」形态，服务学生/教师；
保留并增强管理端能力。**风格遵循 UI/UX 报告设计 Token**（brandPrimary #1769AA / aiAccent #008F83 / attention #B77900 / surface #FAFBFC），不引入蓝紫渐变。

## 范围

### 做（用户端）
1. **精选流**：按日期分组（今天 / 更早），每条含 来源徽标 · 时间 · 热度值 · 分类标签 · 标题 · 摘要 · 推荐理由（若有）· 详情链接。
2. **热度榜**：Top N 热度排行（`heat` 字段，人工/管理端可编辑，无自动热度算法）。
3. **分类 Tab**：全部 / AI 辅助教学 / AI 工具 / AI 版本 / AI 行业热点（沿用 4 个既有分类 key）。
4. **收藏**：登录用户可收藏/取消收藏单条资讯；收藏 Tab 列出我的收藏。
5. **搜索**：关键词搜索（沿用现有 `q` 参数）。
6. **日期分组**：列表按 `published_at` 天分组，分组头「8月11日 · 周二」样式。

### 做（管理端）
- CRUD 表单新增 热度值（heat）、推荐理由（reason）字段；列表列可展示热度。
- 上架/下架、批量、导出（md/pdf）能力保持，导出内容追加热度与推荐理由。

### 做（后端）
- `ai_briefings` 表新增 `heat INTEGER DEFAULT 0`、`reason TEXT DEFAULT ''` → 迁移 `073_ai_briefings_refactor.sql`。
- 新增 `ai_briefing_favorites` 表（user_id, briefing_id, created_at, UNIQUE(user_id,briefing_id)）。
- 用户端接口扩展：列表（分组+热度+收藏态）、热度榜、收藏 CRUD、收藏列表。
- 管理端 CRUD 增删改 heat/reason。

### 不做
- 不做自动热度算法 / 信源聚合（多源归并）/ AI 日报生成 / 模型榜 / 主题页（超出本次范围，留后续增量）。
- 不动 Vercel 后端与数据库切换；不引入新第三方库。

## 技术要点

| 项 | 说明 |
|----|------|
| 迁移 | `server/migrations/073_ai_briefings_refactor.sql`（加列 + 建收藏表，幂等） |
| 模型 | `model.AIBriefing` 增 Heat/Reason 字段；新增 `model.AIBriefingFavorite` |
| 仓储 | `repo` 新增：ListUserVisible 分组/热度/收藏态联查、HotList、Favorite/Unfavorite/ListFavorites/IsFavorite |
| 服务 | 复用现有分层；用户端列表聚合收藏态（当前用户 ID 传入） |
| Handler | 用户端新增 `/api/v1/ai-briefings/hot`、`/api/v1/ai-briefings/:id/favorite`(POST/DELETE)、`/api/v1/ai-briefings/favorites`(GET) |
| 前端 | `ai_briefings_page.dart` 重构为 精选/热度榜/收藏 三 Tab + 分类 Chip + 日期分组流；Provider 增收藏方法 |
| 首页卡片 | 跳转地址不变 `/ai-briefings`；可复用热度 Top 展示 |

## 接口变更（登记到 `specs/api-contracts-index.md`）

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/v1/ai-briefings` | 已认证 | 列表，支持 `category/q/hot`，返回按日分组 + 收藏态 |
| GET | `/api/v1/ai-briefings/hot` | 已认证 | 热度榜 Top 50（status=1，按 heat DESC, published_at DESC） |
| POST | `/api/v1/ai-briefings/:id/favorite` | 已认证 | 收藏 |
| DELETE | `/api/v1/ai-briefings/:id/favorite` | 已认证 | 取消收藏 |
| GET | `/api/v1/ai-briefings/favorites` | 已认证 | 我的收藏列表 |
| PUT | `/api/v1/admin/ai-briefings/:id` | sys_admin | 表单增加 heat/reason |

> 收藏属个人数据，能力沿用 `self.ai_briefing.read`（student 基线）；管理端能力不变 `system.ai_briefing`。

## 步骤拆分

1. 迁移 073（加列 + 收藏表）+ model 字段
2. repo 方法（列表/热度/收藏）+ service 方法
3. handler 新接口 + 路由注册（`pkg/app/app.go` secured 组）
4. 管理端 heat/reason CRUD 透传
5. 前端 provider 扩展 + model 增字段
6. 前端用户端三 Tab 重构（精选流/热度榜/收藏）+ 首页卡片衔接
7. 管理端表单补 heat/reason 输入
8. 验证：go vet/test + dart analyze/test + 构建 + 代码审查
9. 文档更新（`docs/ai-briefings.md`、`specs/api-contracts-index.md`、`specs/rbac-matrix.md` 核对）

## 验收标准

- 用户端三 Tab 均可用，精选流按日期分组、热度榜排序正确、收藏可增删并在收藏 Tab 可见。
- 列表项含 来源/时间/热度/分类/摘要/推荐理由/链接；点击链接拉起外部浏览器。
- 管理端可录入与编辑 heat/reason；导出含新字段。
- `go test ./...`、`flutter analyze --no-pub`、`flutter test` 通过；`git diff --check` 通过。
- 数据库迁移在既有 SQLite（FTS）上可幂等执行。

## 回滚与检查点

- Git 检查点：迁移、后端接口、前端重构各一次提交（分增量提交）。
- 数据回滚：加列用 `ALTER TABLE ... ADD COLUMN` 幂等；收藏表可 `DROP TABLE IF EXISTS`。
- 前端回滚：`git revert` 对应提交即可，接口向后兼容（heat/reason 字段可缺省）。
