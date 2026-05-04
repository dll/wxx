---
name: wxx-debug
description: 蔚小芯项目特定问题的调试指南。当调试 SQLite 查询、FTS5 检索、大模型 API 调用（智谱/DeepSeek/讯飞）、JWT 认证、Context Engine 管道、知识同步或 Gin 中间件的错误时触发。也在出现"调试"、"报错"、"debug"、"error"、"500"、"查不到"、"返回空"、"超时"等短语时触发，或在排查功能异常时触发。使用本技能引导系统化诊断，避免盲目猜测。
---

# 蔚小芯 调试指南

本技能为蔚小芯架构中的常见故障模式提供系统化调试流程。系统有多个集成点（SQLite、FTS5、3 个大模型 API、外部同步）— 每个都有特征性的失败模式。

## 调试决策树

```
观察到的错误
    |
    +-- HTTP 401/403 -----------> JWT / RBAC 问题（第 1 节）
    |
    +-- HTTP 500 ----------------> 查 audit_logs 中的 trace_id（第 2 节）
    |
    +-- 回答为空或错误 -----------> Context Engine 管道（第 3 节）
    |
    +-- 大模型超时/报错 ----------> 外部 API 问题（第 4 节）
    |
    +-- FTS 查不到结果 -----------> FTS5 索引问题（第 5 节）
    |
    +-- 同步失败 ----------------> 知识同步问题（第 6 节）
    |
    +-- 情感预警未触发 -----------> 情感管道问题（第 7 节）
```

## 第 1 节：JWT / RBAC 问题

**症状**：401 未授权 或 403 禁止访问

诊断步骤：
1. 解码 JWT 令牌（不验证签名）检查声明内容：
   ```bash
   # 提取载荷（base64 解码）
   echo "<令牌>" | cut -d. -f2 | base64 -d 2>/dev/null
   ```
2. 检查 `exp` — 令牌是否已过期？
3. 检查 `role` 声明 — 是否匹配该接口要求的角色？
4. 检查中间件链 — 该路由组是否应用了 `middleware.Auth()`？
5. 检查 `owner_scope` — 用户是否在访问其权限范围内的资源？

常见修复：
- 令牌过期：客户端需要刷新
- 角色不匹配：核对 `specs/rbac-matrix.md` 中的 RBAC 矩阵
- 缺少中间件：在路由组中添加 `middleware.Auth()`

## 第 2 节：HTTP 500 诊断

**症状**：内部服务器错误

1. 从错误响应中获取 `trace_id`
2. 查询审计日志：
   ```sql
   -- 按 trace_id 查找相关审计记录
   SELECT * FROM audit_logs WHERE trace_id = '<trace_id>' ORDER BY created_at;
   ```
3. 检查 `result_code` 和 `detail` 字段了解错误上下文
4. 沿调用链追踪：handler → service → repository/llm

常见原因：
- SQLite 数据库锁定（并发写入未启用 WAL 模式）
- 大模型 API 返回了非预期格式
- 上下文拼装时空指针（缺少必填字段）

## 第 3 节：Context Engine 管道

**症状**：回答错误、信息缺失或不相关

诊断 — 逐阶段检查：

1. **意图分类**：提问是否被分类到正确的 `resource_type`？
   ```sql
   -- 检查各类型已发布资源的数量
   SELECT resource_type, COUNT(*) FROM kb_resources WHERE status='published' GROUP BY resource_type;
   ```

2. **结构化查询**：匹配数据是否存在？
   ```sql
   -- 按类型和范围查找已发布资源
   SELECT * FROM kb_resources WHERE resource_type = '<类型>' AND status = 'published' AND owner_scope = '<范围>';
   ```

3. **FTS 检索**：搜索词是否能匹配？
   ```sql
   -- 直接测试全文检索
   SELECT resource_id, title, bm25(kb_fts) as score FROM kb_fts WHERE kb_fts MATCH '<查询词>' ORDER BY score LIMIT 10;
   ```

4. **范围过滤**：资源对该角色是否可见？
   - 检查 `role_scope` JSON 数组是否包含用户角色
   - 检查 `owner_scope` 是否匹配用户范围
   - 检查 `effective_at`/`expired_at` 日期

5. **上下文拼装**：拼装的上下文是否太长/被截断？
   - 检查总 token 数与模型限制的关系
   - 确认高相关性结果没有被裁剪

6. **LLM 提示词**：系统提示词是否正确指示"仅基于上下文回答"？

## 第 4 节：外部 API 问题

**症状**：超时、格式异常或 API 报错

各大模型提供商的排查要点：

**智谱清言**：
- 检查 `ZHIPU_API_KEY` 是否已设置且有效
- API 端点：确认 URL 没有变更
- 频率限制：检查配额是否用尽

**DeepSeek**：
- 检查 `DEEPSEEK_API_KEY`
- 响应格式：确认 JSON 解析能处理流式/非流式两种模式

**讯飞星火**（语音）：
- 检查 `XFYUN_APP_ID`、`XFYUN_API_KEY`、`XFYUN_API_SECRET`
- WebSocket 连接：确认握手参数正确
- 音频格式：确保编码正确（PCM 16位，16kHz）

通用 API 调试：
```go
// 添加临时调试日志
log.Printf("[调试] 大模型请求: model=%s, tokens=%d, trace_id=%s", model, tokenCount, traceID)
log.Printf("[调试] 大模型响应: status=%d, latency=%dms", resp.StatusCode, elapsed)
```

## 第 5 节：FTS5 索引问题

**症状**：搜索无结果或结果不正确

1. 确认 FTS 索引存在且数据完整：
   ```sql
   -- 对比 FTS 索引条目数和已发布资源数，两者应相等
   SELECT COUNT(*) FROM kb_fts;
   SELECT COUNT(*) FROM kb_resources WHERE status = 'published';
   ```

2. 如果数量不一致，触发器可能损坏 — 重建索引：
   ```sql
   INSERT INTO kb_fts(kb_fts) VALUES('rebuild');
   ```

3. 直接测试 FTS 匹配：
   ```sql
   SELECT * FROM kb_fts WHERE kb_fts MATCH '奖学金';
   ```

4. 检查分词器 — `unicode61` 支持中文但可能对复合词分词不当

5. 确认同步触发器存在：
   ```sql
   -- 应返回：kb_fts_insert、kb_fts_update、kb_fts_delete
   SELECT name FROM sqlite_master WHERE type='trigger' AND name LIKE 'kb_fts%';
   ```

## 第 6 节：知识同步问题

**症状**：与蔚园智答的同步失败

1. 检查 `sync_cursors` 表获取上次成功同步位置：
   ```sql
   SELECT * FROM sync_cursors WHERE target = 'weiyuan_zhida';
   ```

2. 验证 HMAC 签名校验：
   - `SYNC_HMAC_SECRET` 是否已设置且与发送方密钥匹配？
   - 同步包时间戳是否在可接受窗口内？

3. 检查 manifest.json 结构：
   - `resourceId` + `version` + `status` 必须组成唯一键
   - SHA256 哈希必须与内容匹配

4. 检查幂等性：重复同步相同包应无副作用

## 第 7 节：情感管道

**症状**：高风险情感未触发通知

1. 查询该会话的情感日志：
   ```sql
   SELECT * FROM emotion_logs WHERE session_id = '<会话ID>' ORDER BY created_at DESC;
   ```

2. 确认分数阈值：什么条件触发 `risk_level = 'high'`？
3. 检查 `notified` 标志 — 是否已发送过通知？
4. 确认辅导员角色有权查看情感数据

## 通用技巧

- 始终先找 `trace_id` — 它将一次请求的所有审计记录串联起来
- 开发时用 `sqlite3 data/wxx.sqlite` 直接检查数据库
- 检查 `server/data/` 下的实际数据库文件
- 启用 WAL 模式避免"数据库被锁定"错误：
  ```sql
  PRAGMA journal_mode=WAL;
  ```
