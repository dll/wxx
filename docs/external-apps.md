# 蔚小芯 第三方应用接入协议

> 版本：v0.1（骨架）· 更新日期：2026-05-16
> 目标：让校园已有系统（教务管理、图书馆、微信公众号、视频号、广播台、第三方学习平台等）通过**统一协议**挂入蔚小芯，避免每接一个应用就改一遍核心代码。

## 设计原则

1. **不强制原系统改造** — 通过外链跳转 / WebView 内嵌 / 后端反代三种适配模式覆盖绝大多数场景
2. **能力授权统一** — 第三方应用的可见性继续走蔚小芯 RBAC，按角色配置可见名单
3. **可热插拔** — 新增/启停应用不需要重启服务（manifest 写入 SQLite，运行时加载）
4. **保留审计** — 任何跨应用跳转都打入 `audit_logs`，含 traceId + 用户角色 + 目标应用

## Manifest Schema

```jsonc
{
  "$schema": "https://wxx.example.edu.cn/specs/external-apps.schema.json",
  "id": "lib_seat",                           // 全局唯一，仅小写字母数字下划线
  "name": "图书馆座位预约",                     // 用户可见名称
  "icon": "https://...",                      // 64×64 PNG，或 Material 图标名（如 "menu_book"）
  "category": "study",                        // 分类：study | culture | service | admin | external
  "summary": "查询并预约自习室座位",            // 一句话描述（≤30 字）
  "version": "1.0.0",                         // 语义化版本，升级时强制刷新缓存
  "adapter": {
    "type": "external_link",                  // external_link | webview | reverse_proxy
    "url": "https://lib.example.edu.cn/seat", // 必填
    "auth": {
      "mode": "sso",                          // none | sso | basic | bearer
      "ssoProvider": "cas",                   // CAS / OAuth2 / SAML，由后端代理换 token
      "bearerSource": "user.profile.libToken" // 当 mode=bearer 时，从用户 profile 取字段
    },
    "openIn": "_self"                         // _self（内嵌）| _blank（新窗口）| _native（移动端拉起原生）
  },
  "visibleTo": {
    "roles": ["student", "teacher"],          // 角色白名单，留空即全员
    "capabilities": ["self.chat"],            // 必须持有的能力（AND 关系），通常留空
    "scope": "self"                           // 数据范围：self | college | school | all
  },
  "updatedAt": "2026-05-16T10:00:00+08:00"
}
```

## 三种适配模式

### A. 外链跳转（external_link）

最简单：点击后浏览器/APK 拉起目标 URL。适合：

- 已有完善 SSO 的校外系统（如教务系统）
- 无需把数据带回蔚小芯展示的场景
- 视频号/公众号/B 站直播

**优点**：零开发成本；**缺点**：脱离蔚小芯主框架，体验割裂。

### B. WebView 内嵌（webview）

在蔚小芯内开一个二级页面，用 `flutter_inappwebview` / iframe 加载目标。适合：

- 需要在蔚小芯壳内完成的轻量交互（图书馆座位选座、活动报名）
- 目标系统已自带响应式 UI

**注意**：跨域 cookie 受浏览器限制；移动端可注入 JS Bridge 与 Flutter 通信。

### C. 反向代理（reverse_proxy）

后端 `/api/v1/integration/<id>/*path` 代理到目标系统，前端按蔚小芯风格重新渲染。适合：

- 高度集成的核心系统（学工、一表通已采用）
- 数据要进蔚小芯 Context Engine 做检索增强

**实现位置**：`server/internal/handler/integration_handler.go`，每个新应用扩展一个 `Proxy<App>` 方法。

## 数据库 Schema

```sql
CREATE TABLE external_apps (
    id           TEXT PRIMARY KEY,           -- manifest.id
    manifest     TEXT NOT NULL,              -- 完整 JSON
    enabled      INTEGER NOT NULL DEFAULT 1, -- 启停标志
    created_by   INTEGER NOT NULL,           -- sys_admin 用户 ID
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
);

CREATE INDEX idx_external_apps_enabled ON external_apps(enabled);
```

## API 草案

| 方法 | 路径 | 角色 | 说明 |
|------|------|------|------|
| GET | `/api/v1/apps` | 已认证 | 返回当前用户可见的应用列表（按角色过滤） |
| POST | `/api/v1/admin/apps` | sys_admin | 注册新应用（提交 manifest） |
| PUT | `/api/v1/admin/apps/:id` | sys_admin | 更新 manifest |
| DELETE | `/api/v1/admin/apps/:id` | sys_admin | 删除（软删，置 `enabled=0`） |
| GET | `/api/v1/integration/<id>/*path` | 视应用而定 | 反向代理（type=reverse_proxy 时） |

## 前端入口

- 个人中心增加"应用中心"卡片 → `/apps`
- 应用中心页按 manifest.category 分组，按 visibleTo.roles 过滤
- 卡片点击根据 adapter.openIn 决定跳转方式

## 渐进上线策略

| 阶段 | 时间 | 范围 |
|------|------|------|
| P0（当前） | 2026-05 | 仅文档，capability 占位 |
| P1 | 2026-06 | external_apps 表 + GET `/apps` 列表 + external_link 模式 + 应用中心页 |
| P2 | 2026-07 | webview 模式 + sys_admin 管理界面 + 审计联动 |
| P3 | 2026-08+ | reverse_proxy 模式（按需，复用 integration_handler） |

## 与「校园文化智能体」的关系

校园文化目前是**蔚小芯内置功能**（`/culture/*`），未来的「校歌音频流」「学校广播台直播」「B 站讲座外链」等如果走第三方资源，可在 manifest 中定义为 `category=culture` 应用，与现有内置页面共存：内置页面提供 AI 整合能力（如歌词解读、讲座摘要），外链应用提供原生体验（如音频流播放）。
