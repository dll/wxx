---
name: wxx-debug
description: Debugging guide for WeiXiaoXin (蔚小芯) project-specific issues. Triggers when debugging errors in SQLite queries, FTS5 search, LLM API calls (智谱/DeepSeek/讯飞), JWT authentication, Context Engine pipeline, knowledge sync, or Gin middleware. Also triggers on phrases like "调试", "报错", "debug", "error", "500", "查不到", "返回空", "超时", or when investigating why a feature isn't working. Use this skill to guide systematic diagnosis rather than random guessing.
---

# 蔚小芯 Debugging Guide

This skill provides systematic debugging procedures for the common failure modes in 蔚小芯's architecture. The system has several integration points (SQLite, FTS5, 3 LLM APIs, external sync) — each has characteristic failure patterns.

## Debugging Decision Tree

```
Error observed
    |
    +-- HTTP 401/403 -----------> JWT / RBAC issue (Section 1)
    |
    +-- HTTP 500 ----------------> Check trace_id in audit_logs (Section 2)
    |
    +-- Empty/wrong answer ------> Context Engine pipeline (Section 3)
    |
    +-- LLM timeout/error -------> External API issue (Section 4)
    |
    +-- FTS returns nothing -----> FTS5 index issue (Section 5)
    |
    +-- Sync failure ------------> Knowledge sync issue (Section 6)
    |
    +-- Emotion alert missing ---> Emotion pipeline issue (Section 7)
```

## Section 1: JWT / RBAC Issues

**Symptom**: 401 Unauthorized or 403 Forbidden

Diagnosis steps:
1. Decode the JWT token (without verification) to inspect claims:
   ```bash
   # Extract payload (base64)
   echo "<token>" | cut -d. -f2 | base64 -d 2>/dev/null
   ```
2. Check `exp` — is the token expired?
3. Check `role` claim — does it match required role for this endpoint?
4. Check middleware chain — is `middleware.Auth()` applied to this route group?
5. Check `owner_scope` — is the user accessing resources within their scope?

Common fixes:
- Token expired: client needs to refresh
- Wrong role: verify RBAC matrix in `specs/rbac-matrix.md`
- Missing middleware: add `middleware.Auth()` to route group

## Section 2: HTTP 500 Diagnosis

**Symptom**: Internal server error

1. Find the `trace_id` from the error response
2. Query audit_logs:
   ```sql
   SELECT * FROM audit_logs WHERE trace_id = '<trace_id>' ORDER BY created_at;
   ```
3. Check `result_code` and `detail` for the error context
4. Trace through the call stack: handler -> service -> repository/llm

Common causes:
- SQLite database locked (concurrent writes without proper WAL mode)
- LLM API returned unexpected format
- Nil pointer in context assembly (missing required field)

## Section 3: Context Engine Pipeline

**Symptom**: Wrong answer, missing information, or irrelevant response

Diagnosis — check each pipeline stage:

1. **Intent classification**: is the query classified to the right `resource_type`?
   ```sql
   SELECT resource_type, COUNT(*) FROM kb_resources WHERE status='published' GROUP BY resource_type;
   ```

2. **Structured query**: does matching data exist?
   ```sql
   SELECT * FROM kb_resources WHERE resource_type = '<type>' AND status = 'published' AND owner_scope = '<scope>';
   ```

3. **FTS search**: does the search term match?
   ```sql
   SELECT resource_id, title, bm25(kb_fts) as score FROM kb_fts WHERE kb_fts MATCH '<query>' ORDER BY score LIMIT 10;
   ```

4. **Scope filtering**: is the resource visible to this role?
   - Check `role_scope` JSON array includes user's role
   - Check `owner_scope` matches user's scope
   - Check `effective_at`/`expired_at` dates

5. **Context assembly**: is the assembled context too long / truncated?
   - Check total token count vs model limit
   - Verify high-relevance results aren't being cut

6. **LLM prompt**: does the system prompt correctly instruct "answer from context only"?

## Section 4: External API Issues

**Symptom**: Timeout, malformed response, or API error

For each LLM provider:

**智谱清言**:
- Check `ZHIPU_API_KEY` is set and valid
- API endpoint: verify URL hasn't changed
- Rate limits: check if quota exceeded

**DeepSeek**:
- Check `DEEPSEEK_API_KEY`
- Response format: verify JSON parsing handles streaming/non-streaming modes

**讯飞星火** (voice):
- Check `XFYUN_APP_ID`, `XFYUN_API_KEY`, `XFYUN_API_SECRET`
- WebSocket connection: verify handshake parameters
- Audio format: ensure correct encoding (PCM 16-bit, 16kHz)

General API debugging:
```go
// Add temporary debug logging
log.Printf("[DEBUG] LLM request: model=%s, tokens=%d, trace_id=%s", model, tokenCount, traceID)
log.Printf("[DEBUG] LLM response: status=%d, latency=%dms", resp.StatusCode, elapsed)
```

## Section 5: FTS5 Index Issues

**Symptom**: Search returns no results or wrong results

1. Verify FTS index exists and is populated:
   ```sql
   SELECT COUNT(*) FROM kb_fts;
   SELECT COUNT(*) FROM kb_resources WHERE status = 'published';
   -- These should be equal
   ```

2. If counts differ, triggers may be broken — rebuild:
   ```sql
   INSERT INTO kb_fts(kb_fts) VALUES('rebuild');
   ```

3. Test direct FTS match:
   ```sql
   SELECT * FROM kb_fts WHERE kb_fts MATCH '奖学金';
   ```

4. Check tokenizer — `unicode61` handles CJK but may split compound words unexpectedly

5. Verify sync triggers exist:
   ```sql
   SELECT name FROM sqlite_master WHERE type='trigger' AND name LIKE 'kb_fts%';
   -- Should return: kb_fts_insert, kb_fts_update, kb_fts_delete
   ```

## Section 6: Knowledge Sync Issues

**Symptom**: Sync with 蔚园智答 fails

1. Check `sync_cursors` table for last successful sync:
   ```sql
   SELECT * FROM sync_cursors WHERE target = 'weiyuan_zhida';
   ```

2. Verify HMAC signature validation:
   - Is `SYNC_HMAC_SECRET` set and matching the sender's key?
   - Is the package timestamp within acceptable window?

3. Check manifest.json structure:
   - `resourceId` + `version` + `status` must form a unique key
   - SHA256 hash must match content

4. Check for idempotency: re-syncing same package should be a no-op

## Section 7: Emotion Pipeline

**Symptom**: High-risk emotion not triggering notification

1. Check emotion_logs for the session:
   ```sql
   SELECT * FROM emotion_logs WHERE session_id = '<session_id>' ORDER BY created_at DESC;
   ```

2. Verify score threshold: what triggers `risk_level = 'high'`?
3. Check `notified` flag — has notification already been sent?
4. Verify counselor role has access to view emotion data

## General Tips

- Always start by finding the `trace_id` — it connects all audit entries for one request
- Use `sqlite3 data/wxx.sqlite` for direct database inspection during development
- Check `server/data/` for the actual database file
- Enable WAL mode for SQLite to avoid "database is locked" errors:
  ```sql
  PRAGMA journal_mode=WAL;
  ```
