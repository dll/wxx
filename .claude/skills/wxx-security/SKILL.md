---
name: wxx-security
description: 蔚小芯项目的安全审查与执行。当代码涉及认证（JWT）、授权（RBAC）、用户输入处理、数据库查询、API 密钥管理、敏感数据（学号、手机号、身份证号）、审计日志或外部 API 对接时触发。也在出现"安全"、"权限"、"脱敏"、"注入"、"鉴权"、"RBAC"等短语时触发，或在审查处理用户数据的代码时触发。如发现潜在安全问题应主动介入。
---

# 蔚小芯 安全审查

本技能执行蔚小芯学工智能体的安全标准。系统处理敏感学生数据（学业记录、个人信息、情感评估）并集成携带凭证的外部 API — 安全是底线。

## 必须检查项

### 1. RBAC 权限执行

每个接口必须声明所需角色。角色层级（高级可访问低级）：

```
sys_admin > school_admin > college_admin > counselor > student_union > student
```

扩展角色 `teacher` 和 `assistant` 有独立的权限范围。

在代码中验证：
- 中间件在 handler 执行前检查角色
- 知识查询在命中 FTS/数据库之前，先按 `owner_scope` + `role_scope` + `status=published` 过滤
- 角色从 JWT 声明中读取，绝不从请求体中获取
- 完整 RBAC 矩阵维护在 `specs/rbac-matrix.md`

### 2. SQL 注入防护

所有 SQLite 查询必须使用参数化语句。绝不将用户输入拼接进 SQL：

```go
// 正确写法
db.Query("SELECT * FROM kb_resources WHERE resource_id = ?", resourceID)

// 禁止 — SQL 注入漏洞
db.Query("SELECT * FROM kb_resources WHERE resource_id = '" + resourceID + "'")
```

FTS5 查询尤其危险 — 用户搜索词注入到 FTS `MATCH` 语法中可能导致意外行为。必须清洗 FTS 输入：
- 过滤特殊 FTS 运算符（`AND`、`OR`、`NOT`、`NEAR`、`*`、`"`）
- 转义双引号
- 限制查询长度

### 3. 敏感数据保护

以下字段绝不能作为可检索明文存入 `kb_resources.content` 或 FTS 索引：
- 学号
- 手机号
- 身份证号
- 家庭住址
- 家庭经济信息

按角色展示规则：
- `student`：仅可见本人数据，脱敏展示（如 `138****5678`）
- `counselor`：可见本人学生数据，部分脱敏
- `college_admin+`：可见完整数据，附带审计记录

每次访问敏感数据必须生成 `audit_logs` 记录。

### 4. API 密钥与密钥管理

所有密钥通过环境变量经由 `internal/config/` 加载：
- `ZHIPU_API_KEY`、`DEEPSEEK_API_KEY`、`XFYUN_*` — 大模型 API 凭证
- `JWT_SECRET` — JWT 签名密钥
- `SYNC_HMAC_SECRET` — 知识同步 HMAC-SHA256 密钥
- `SSO_*` — 校园统一认证凭证

安全规则：
- 绝不在源码中硬编码密钥
- 绝不在日志中输出 API 密钥或 JWT 令牌（即使是 debug 级别）
- 绝不在 API 响应中返回密钥
- `.env` 已在 `.gitignore` 中 — 每次提交前确认
- 使用 `.env.example` 作为模板（不含真实值）

### 5. JWT 安全

```
令牌生命周期：
  登录 → 签发 JWT（短期有效，如 2 小时）→ 刷新 → 登出时吊销
```

验证项：
- 令牌使用 HS256 或 RS256 签名，绝不使用无签名令牌
- 过期时间（`exp`）始终设置并检查
- 角色声明不能被客户端篡改
- 刷新令牌存储在服务端（sessions 表）
- 登出时使 session 记录失效

### 6. 审计追踪

以下操作必须生成 `audit_logs` 条目：
- 登录 / 登出
- 知识资源增删改查（创建、更新、发布、退役）
- 敏感数据访问
- 导出操作（PDF、Word、Markdown）
- 角色变更
- 认证失败尝试
- 情感风险升级（`risk_level = 'high'`）

每条审计记录包含：`user_id`、`username`、`role`、`action`、`resource`、`detail`、`trace_id`、`ip`、`duration_ms`、`result_code`。

### 7. 外部 API 安全

调用智谱/DeepSeek/讯飞 API 时：
- 仅使用 HTTPS
- 设置请求超时（默认：大模型 30 秒，其他 10 秒）
- 实现指数退避重试（最多 3 次）
- 绝不向外部大模型 API 发送学生个人信息 — 发送前须脱敏
- 每次外部调用记录 `trace_id` 用于调试

与蔚园智答的知识同步：
- HMAC-SHA256 包签名验证
- Bearer Token 认证
- SHA256 哈希内容校验
- 拒绝时间戳过期的同步包

### 8. 情感数据安全

`emotion_logs` 包含风险评估 — 需提升处理级别：
- 仅 `counselor+` 角色可查看情感数据
- `risk_level = 'high'` 触发通知但数据留在系统内
- 情感评分绝不包含在导出包中
- 批量查询情感数据需要 `college_admin+` 权限

## 需要阻止的反模式

| 模式 | 风险 | 修复方式 |
|------|------|----------|
| `fmt.Sprintf("WHERE id = %s", input)` | SQL 注入 | 使用 `?` 占位符 |
| `log.Printf("token: %s", jwt)` | 令牌泄露 | 绝不记录令牌 |
| `r.Header.Get("X-Role")` | 角色伪造 | 仅从 JWT 读取角色 |
| 在 JSON 响应中返回 `id_card` | 个人信息泄露 | 按角色脱敏或省略 |
| `http.Get(url)` 无超时 | 挂起/资源耗尽 | 使用 `http.Client{Timeout}` |
| 数据导出缺少 `audit_logs` | 合规缺口 | 始终审计导出操作 |
