---
name: wxx-security
description: Security review and enforcement for the WeiXiaoXin (蔚小芯) project. Triggers whenever code touches authentication (JWT), authorization (RBAC), user input handling, database queries, API key management, sensitive data (student IDs, phone numbers, ID cards), audit logging, or external API integration. Also triggers on phrases like "安全", "权限", "脱敏", "注入", "鉴权", "RBAC", or when reviewing code that handles user data. Use proactively - if you see potential security issues in code being written, intervene immediately.
---

# 蔚小芯 Security Review

This skill enforces security standards specific to the 蔚小芯 student affairs AI assistant. The system handles sensitive student data (academic records, personal info, emotional assessments) and integrates with external APIs carrying credentials — security is non-negotiable.

## Mandatory Checks

### 1. RBAC Enforcement

Every endpoint MUST declare its required role. The hierarchy (higher can access lower):

```
sys_admin > school_admin > college_admin > counselor > student_union > student
```

Extension roles `teacher` and `assistant` have separate permission scopes.

Verify in code:
- Middleware checks role before handler executes
- Knowledge queries filter by `owner_scope` + `role_scope` + `status=published` BEFORE hitting FTS/database
- Role is validated from JWT claims, never from request body
- Full RBAC matrix is documented in `specs/rbac-matrix.md`

### 2. SQL Injection Prevention

All SQLite queries MUST use parameterized statements. Never concatenate user input into SQL:

```go
// CORRECT
db.Query("SELECT * FROM kb_resources WHERE resource_id = ?", resourceID)

// FORBIDDEN - SQL injection vector
db.Query("SELECT * FROM kb_resources WHERE resource_id = '" + resourceID + "'")
```

FTS5 queries are especially dangerous — user search terms injected into FTS `MATCH` syntax can cause unexpected behavior. Always sanitize FTS input:
- Strip special FTS operators (`AND`, `OR`, `NOT`, `NEAR`, `*`, `"`)
- Escape double quotes
- Limit query length

### 3. Sensitive Data Protection

These fields must NEVER be stored as searchable plaintext in `kb_resources.content` or FTS index:
- Student ID (学号)
- Phone number (手机号)
- National ID (身份证号)
- Home address (家庭住址)
- Family financial info (家庭经济信息)

Display rules by role:
- `student`: sees only own data, masked (e.g., `138****5678`)
- `counselor`: sees own students' data, partially masked
- `college_admin+`: sees full data with audit trail

Every access to sensitive data MUST create an `audit_logs` entry.

### 4. API Key & Secret Management

All secrets load from environment variables via `internal/config/`:
- `ZHIPU_API_KEY`, `DEEPSEEK_API_KEY`, `XFYUN_*` — LLM API credentials
- `JWT_SECRET` — JWT signing key
- `SYNC_HMAC_SECRET` — knowledge sync HMAC-SHA256 key
- `SSO_*` — campus SSO credentials

Security rules:
- NEVER hardcode secrets in source code
- NEVER log API keys or JWT tokens (even at debug level)
- NEVER return secrets in API responses
- `.env` is in `.gitignore` — verify this before every commit
- Use `.env.example` as the template (no real values)

### 5. JWT Security

```
Token lifecycle:
  Login → Issue JWT (short-lived, e.g., 2h) → Refresh → Revoke on logout
```

Verify:
- Tokens are signed with HS256 or RS256, NOT unsigned
- Expiration (`exp`) is always set and checked
- Role claims cannot be modified client-side
- Refresh tokens are stored server-side (sessions table)
- Logout invalidates the session record

### 6. Audit Trail

These operations MUST generate `audit_logs` entries:
- Login / logout
- Knowledge resource CRUD (create, update, publish, retire)
- Sensitive data access
- Export operations (PDF, Word, Markdown)
- Role changes
- Failed authentication attempts
- Emotion risk escalation (`risk_level = 'high'`)

Each audit entry includes: `user_id`, `username`, `role`, `action`, `resource`, `detail`, `trace_id`, `ip`, `duration_ms`, `result_code`.

### 7. External API Security

When calling 智谱/DeepSeek/讯飞 APIs:
- Use HTTPS only
- Set request timeouts (default: 30s for LLM, 10s for others)
- Implement retry with exponential backoff (max 3 retries)
- Never send student PII to external LLM APIs — strip before sending
- Log `trace_id` for every external call for debugging

Knowledge sync with 蔚园智答:
- HMAC-SHA256 package signature verification
- Bearer token authentication
- SHA256 hash validation on received content
- Reject packages with expired timestamps

### 8. Emotion Data Security

`emotion_logs` contain risk assessments — elevated handling:
- Only `counselor+` roles can view emotion data
- `risk_level = 'high'` triggers notification but data stays in-system
- Emotion scores are never included in export packages
- Batch queries on emotion data require `college_admin+`

## Anti-Patterns to Block

| Pattern | Risk | Fix |
|---------|------|-----|
| `fmt.Sprintf("WHERE id = %s", input)` | SQL injection | Use `?` placeholders |
| `log.Printf("token: %s", jwt)` | Token leakage | Never log tokens |
| `r.Header.Get("X-Role")` | Role spoofing | Read role from JWT only |
| Returning `id_card` in JSON response | PII exposure | Mask or omit by role |
| `http.Get(url)` without timeout | Hang / resource exhaustion | Use `http.Client{Timeout}` |
| Missing `audit_logs` on data export | Compliance gap | Always audit exports |
