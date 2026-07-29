# 修复 Turso 整数参数编码 Bug

> 首次接入 Turso（libsql）HTTP API 时几乎必踩的坑。本文记录现象、根因、定位过程、修复方案与避坑清单。

## 一、故障现象

部署在 Cloudflare Pages Functions 上的后端，出现一组很有规律的故障：

| 接口 | 结果 |
|------|------|
| `GET /api/health` | 正常，返回 `{"status":"ok","database":"connected"}` |
| `POST /api/v1/auth/login` | 正常，能签发 token |
| `GET /api/v1/user/profile` | **500** |
| `GET /api/v1/kb/resources` | **500** |
| `GET /api/v1/notifications/unread-count` | **500** |
| 其余所有需要鉴权的接口 | **500** |

500 的响应体里带着上游原始错误：

```json
{
  "code": 500,
  "message": "服务器内部错误: Turso API error 400: {\"error\":\"JSON parse error: invalid type: integer `12`, expected a borrowed string at line 1 column 121\"}"
}
```

关键规律：**只要 SQL 绑定参数里出现整数就失败**。

- `/api/health` 执行的是 `SELECT 1 as ok`，无绑定参数 → 通过
- 登录按 `WHERE username = ?` 查询，参数是字符串 → 通过
- 其余接口都要先执行 `SELECT * FROM users WHERE id = ?`，参数是数字 `12`（当前登录用户 ID）→ 全部失败

也就是说，**故障点不在各个业务接口，而在它们共同依赖的 `getCurrentUser()`**。

## 二、根本原因

Turso 通过 HTTP 访问时使用 libsql 的 **Hrana over HTTP** 协议（端点 `POST /v2/pipeline`）。
SQL 绑定参数用「带类型标签的值」（tagged value）表示：

```json
{ "type": "integer", "value": ... }
```

协议规定：**`integer` 的 `value` 必须是 JSON 字符串，而不是 JSON 数字。**

原因是精度。SQLite 的 INTEGER 是 64 位有符号整数（i64），而 JSON 的 number 在 JS / 多数 JSON 实现里按 IEEE-754 双精度处理，能精确表示的整数上限只有 2^53−1。如果用 JSON number 承载 i64，超过 2^53 的值会**静默丢精度**——这在主键、雪花 ID、时间戳微秒等场景是致命的。协议因此强制用字符串承载整数，把解析责任交给服务端。

服务端（Rust + serde）把该字段声明为借用字符串 `&str`，所以传数字时反序列化直接失败，报出：

```
invalid type: integer `12`, expected a borrowed string
```

各类型的正确写法：

| SQLite 类型 | `type` | `value` 的 JSON 形态 |
|-------------|--------|----------------------|
| INTEGER | `integer` | **字符串**，如 `"12"` |
| REAL | `float` | 数字，如 `3.14` |
| TEXT | `text` | 字符串 |
| BLOB | `blob` | base64 字符串 |
| NULL | `null` | `null` |

注意 `integer` 和 `float` 的规则是**相反**的，这一点很反直觉，也是容易写错的地方。

## 三、定位过程

1. **先找共性**，不要逐个接口排查。健康检查和登录能过、其余全挂，说明问题在公共依赖上，即 `getCurrentUser()` 里的按 ID 查询。
2. **绕开应用层，直接打 Turso HTTP API 做对照实验**。这是最关键的一步——把变量压缩到只剩「整数参数怎么编码」。
3. **对照结果定位到客户端代码**的参数序列化逻辑。

## 四、最小复现

把 `libsql://` 换成 `https://`，直接请求 `/v2/pipeline`：

```bash
H="https://<你的库>.turso.io"
TOK="<你的 token>"

# ① 整数传 JSON 数字 —— 失败
curl -s -X POST "$H/v2/pipeline" \
  -H "Authorization: Bearer $TOK" \
  -H "Content-Type: application/json" \
  -d '{"requests":[{"type":"execute","stmt":{"sql":"SELECT username FROM users WHERE id = ?","args":[{"type":"integer","value":12}]}},{"type":"close"}]}'
# → {"error":"JSON parse error: invalid type: integer `12`, expected a borrowed string ..."}

# ② 整数传 JSON 字符串 —— 成功
curl -s -X POST "$H/v2/pipeline" \
  -H "Authorization: Bearer $TOK" \
  -H "Content-Type: application/json" \
  -d '{"requests":[{"type":"execute","stmt":{"sql":"SELECT username FROM users WHERE id = ?","args":[{"type":"integer","value":"12"}]}},{"type":"close"}]}'
# → {"results":[{"type":"ok","response":{...,"rows":[[{"type":"text","value":"counselor1"}]],...}}]}
```

两条命令只差一对引号，结论就此确定。

## 五、修复

影响文件（两份副本，**必须同步**，否则不同部署路径行为不一致）：

- `functions/lib/turso.js`
- `frontend/functions/lib/turso.js`

修复前：

```js
args: args.map(a => {
  const t = typeof a;
  if (a === null) return { type: 'null', value: null };
  // ↓ value: a 是 JSON 数字，协议要求字符串
  if (t === 'number') return { type: Number.isInteger(a) ? 'integer' : 'float', value: a };
  // ↓ 同样错误，1 / 0 是数字
  if (t === 'boolean') return { type: 'integer', value: a ? 1 : 0 };
  return { type: 'text', value: String(a) };
}),
```

修复后（抽成独立函数，便于复用和测试）：

```js
// Hrana v2 协议要求 integer 的 value 必须是 JSON 字符串（避免 i64 精度丢失），
// 传原始数字会被服务端拒绝：expected a borrowed string。
function encodeArg(a) {
  if (a === null || a === undefined) return { type: 'null', value: null };
  const t = typeof a;
  if (t === 'bigint') return { type: 'integer', value: a.toString() };
  if (t === 'number') {
    return Number.isInteger(a)
      ? { type: 'integer', value: String(a) }   // 整数 → 字符串
      : { type: 'float', value: a };            // 浮点 → 数字
  }
  if (t === 'boolean') return { type: 'integer', value: a ? '1' : '0' };
  return { type: 'text', value: String(a) };
}
```

要点：

- 整数走 `String(a)`，浮点保持数字，两者规则相反
- 布尔转 `'1'` / `'0'` 字符串，而不是数字
- 顺带支持 `bigint`，这是超过 2^53 的整数在 JS 里的正确载体
- `undefined` 并入 `null` 处理，避免序列化成 `{"type":"text","value":"undefined"}` 这种脏数据

## 六、验证

```bash
# 健康检查
curl -s https://wxx-agent.pages.dev/api/health

# 登录拿 token
TOKEN=$(curl -s -X POST https://wxx-agent.pages.dev/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"counselor1","password":"admin123"}' \
  | grep -oE '"token":"[^"]*"' | cut -d'"' -f4)

# 修复前 500，修复后应返回用户信息
curl -s -H "Authorization: Bearer $TOKEN" \
  https://wxx-agent.pages.dev/api/v1/user/profile
```

## 七、为什么这个坑特别容易踩

1. **官方 SDK 已经处理了**。用 `@libsql/client` 不会遇到；只有在 Cloudflare Workers / Pages Functions 这类环境里手写 HTTP 客户端时才会碰上。
2. **错误被应用层包装**。CF Functions 里 catch 之后统一返回 500，如果没把上游原文透出来，只能看到「服务器内部错误」，完全看不出是参数编码问题。这次能快速定位，靠的就是 `message` 里带了 Turso 原始报文。
3. **登录能成功，极易误判**。第一反应往往是「密码错了」或「JWT 有问题」，实际上登录之所以能过，只是因为它恰好只用字符串参数。

## 八、首次接入 Turso 的避坑清单

1. **写入侧**：`integer` 的 `value` 用字符串，`float` 用数字，布尔转 `'1'`/`'0'`。
2. **读取侧同样要注意**：返回的行数据里，整数列也是**字符串**。例如 `users.id` 读出来是 `"12"` 而不是 `12`。如果要把它放进 JWT 交给 Go 后端（`user_id` 声明为 `int64`），必须先 `parseInt(id, 10)`，否则 Go 侧反序列化会因类型不匹配而失败。
3. `last_insert_rowid` 等元数据同样遵循「整数用字符串承载」的约定，不要直接当数字用。
4. **不要把 DB token 硬编码进仓库**。应放进部署平台的加密环境变量；一旦提交进 git 历史，就必须在 Turso 控制台轮换。
5. **迁移必须幂等**。无服务器环境冷启动可能重复执行迁移，统一用 `CREATE TABLE IF NOT EXISTS` / `INSERT OR IGNORE`，并处理 `ALTER TABLE ADD COLUMN` 的重复列错误。

## 九、参考

- libsql 仓库 `docs/` 目录下的 Hrana over HTTP 协议规范，其中「Values」一节明确定义了各类型 `value` 的 JSON 形态。
