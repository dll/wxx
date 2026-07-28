// 蔚小芯后端 API - Cloudflare Pages Functions 版本
// 直接连接 Turso 云数据库，实现数据持久化

import { TursoClient } from '../lib/turso.js';
import { generateToken, verifyToken, hashPassword, verifyPassword } from '../lib/auth.js';
import { hasCapability, getRoleCapabilities } from '../lib/roles.js';
import { generateId, now } from '../lib/utils.js';

let db = null;

function getDb(context) {
  if (db) return db;
  const dbUrl = context.env.TURSO_DB_URL || 'libsql://wxx-agent-czldl.aws-ap-northeast-1.turso.io';
  const dbToken = context.env.TURSO_DB_TOKEN || 'eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3ODUwNjk1NjEsImlkIjoiMDE5ZjllNmYtNmEwMS03MDIyLTg3OGEtNWI4NzgxOTY2MDk3Iiwia2lkIjoiUUI0MFdrbDBrcDJObV80RHR3eWZoMzV2RXQwZXI5ejF1N1VBUTlyTGU5byIsInJpZCI6IjZiOTNlMmFhLTI0MjEtNGMxOS1iNzljLTM1MzlkZGFmZWE3MyJ9.x_OgL5_1__Pd6cc51Cfxy42jDMesuJ48xcHpMJAowjuHehS3OrY3NkaeHwBrhRgZaszr4horjXQSBouqI3hpDg';
  db = new TursoClient(dbUrl, dbToken);
  return db;
}

export async function onRequest(context) {
  const { request } = context;
  const url = new URL(request.url);

  if (request.method === 'OPTIONS') {
    return new Response(null, {
      status: 204,
      headers: corsHeaders(),
    });
  }

  const path = url.pathname;
  const db = getDb(context);

  try {
    // 健康检查
    if (path === '/health' || path === '/api/health') {
      return await handleHealth(db);
    }

    // 根路由
    if (path === '/' || path === '/api') {
      return jsonResponse({
        service: '蔚小芯',
        version: '1.0.0',
        docs: '/health',
      });
    }

    // 公开 API
    if (path === '/api/v1/auth/login' && request.method === 'POST') {
      return await handleLogin(db, request);
    }

    if (path === '/api/v1/knowledge/public' && request.method === 'GET') {
      return await handlePublicKnowledge(db, url);
    }

    if (path === '/api/v1/version/check' && request.method === 'GET') {
      return jsonResponse({ code: 0, data: { latest: false, version: '1.0.0' } });
    }

    if (path === '/api/v1/version/latest' && request.method === 'GET') {
      return jsonResponse({ code: 0, data: { version: '1.0.0', url: '' } });
    }

    // 需要认证的 API
    const user = await getCurrentUser(db, request);
    if (!user) {
      return jsonResponse({ code: 401, message: '未登录或登录已过期' }, 401);
    }

    return await handleAuthenticatedRoutes(db, request, url, path, user, context);
  } catch (error) {
    console.error('API Error:', error);
    return jsonResponse({
      code: 500,
      message: '服务器内部错误: ' + error.message,
    }, 500);
  }
}

async function handleAuthenticatedRoutes(db, request, url, path, user, context) {
  // 用户相关
  if (path === '/api/v1/user/profile' && request.method === 'GET') {
    return jsonResponse({ code: 0, data: user });
  }

  if (path === '/api/v1/user/capabilities' && request.method === 'GET') {
    const capabilities = getRoleCapabilities(user.role);
    return jsonResponse({ code: 0, data: capabilities });
  }

  if (path === '/api/v1/user/consent' && request.method === 'POST') {
    return jsonResponse({ code: 0, message: 'success' });
  }

  if (path === '/api/v1/user/password' && request.method === 'PUT') {
    return await handleChangePassword(db, user, request);
  }

  // 知识库
  if (path.startsWith('/api/v1/kb/')) {
    return await handleKBRoutes(db, request, url, path, user);
  }

  // 审核
  if (path.startsWith('/api/v1/review/')) {
    return await handleReviewRoutes(db, request, url, path, user);
  }

  // 管理端
  if (path.startsWith('/api/v1/admin/')) {
    return await handleAdminRoutes(db, request, url, path, user);
  }

  // 通知
  if (path === '/api/v1/notifications' && request.method === 'GET') {
    return jsonResponse({ code: 0, data: { items: [], total: 0 } });
  }

  if (path === '/api/v1/notifications/unread-count' && request.method === 'GET') {
    return jsonResponse({ code: 0, data: { unread_count: 0 } });
  }

  if (path === '/api/v1/notifications/read-all' && request.method === 'PUT') {
    return jsonResponse({ code: 0, message: 'success' });
  }

  // 会话（简化版）
  if (path === '/api/v1/sessions' && request.method === 'GET') {
    return jsonResponse({ code: 0, data: [] });
  }

  // 智能流程引导
  if (path === '/api/v1/student/process-enhanced' && request.method === 'GET') {
    return await handleProcessEnhanced(db, url);
  }

  // 默认返回
  return jsonResponse({ code: 404, message: '接口不存在: ' + path }, 404);
}

async function handleKBRoutes(db, request, url, path, user) {
  if (path === '/api/v1/kb/stats' && request.method === 'GET') {
    if (!hasCapability(user.role, 'counselor.kb.write')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBStats(db, user);
  }

  if (path === '/api/v1/kb/dict' && request.method === 'GET') {
    if (!hasCapability(user.role, 'counselor.kb.write')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return handleKBDict();
  }

  if (path === '/api/v1/kb/resources/advanced' && request.method === 'GET') {
    if (!hasCapability(user.role, 'counselor.kb.write')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBListAdvanced(db, url, user);
  }

  if (path === '/api/v1/kb/resources' && request.method === 'GET') {
    if (!hasCapability(user.role, 'counselor.kb.write')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBList(db, url, user);
  }

  if (path === '/api/v1/kb/resources' && request.method === 'POST') {
    if (!hasCapability(user.role, 'counselor.kb.write')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBCreate(db, request, user);
  }

  const resourceMatch = path.match(/^\/api\/v1\/kb\/resources\/([^/]+)$/);
  if (resourceMatch) {
    const id = resourceMatch[1];
    if (request.method === 'GET') {
      if (!hasCapability(user.role, 'counselor.kb.write')) {
        return jsonResponse({ code: 403, message: '无权限' }, 403);
      }
      return await handleKBGet(db, id);
    }
    if (request.method === 'PUT') {
      if (!hasCapability(user.role, 'counselor.kb.write')) {
        return jsonResponse({ code: 403, message: '无权限' }, 403);
      }
      return await handleKBUpdate(db, id, request);
    }
  }

  const submitMatch = path.match(/^\/api\/v1\/kb\/resources\/([^/]+)\/submit$/);
  if (submitMatch && request.method === 'POST') {
    if (!hasCapability(user.role, 'union.kb.submit') && !hasCapability(user.role, 'counselor.kb.write')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBSubmit(db, submitMatch[1], user);
  }

  const approveMatch = path.match(/^\/api\/v1\/kb\/resources\/([^/]+)\/approve$/);
  if (approveMatch && request.method === 'POST') {
    if (!hasCapability(user.role, 'counselor.kb.review')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBApprove(db, approveMatch[1], user);
  }

  const rejectMatch = path.match(/^\/api\/v1\/kb\/resources\/([^/]+)\/reject$/);
  if (rejectMatch && request.method === 'POST') {
    if (!hasCapability(user.role, 'counselor.kb.review')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBReject(db, rejectMatch[1], request, user);
  }

  const retireMatch = path.match(/^\/api\/v1\/kb\/resources\/([^/]+)\/retire$/);
  if (retireMatch && request.method === 'POST') {
    if (!hasCapability(user.role, 'counselor.kb.review')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBRetire(db, retireMatch[1], user);
  }

  if (path === '/api/v1/kb/batch/approve' && request.method === 'POST') {
    if (!hasCapability(user.role, 'counselor.kb.review')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBBatchApprove(db, request, user);
  }

  if (path === '/api/v1/kb/batch/reject' && request.method === 'POST') {
    if (!hasCapability(user.role, 'counselor.kb.review')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBBatchReject(db, request, user);
  }

  if (path === '/api/v1/kb/batch/retire' && request.method === 'POST') {
    if (!hasCapability(user.role, 'counselor.kb.review')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBBatchRetire(db, request, user);
  }

  if (path === '/api/v1/kb/batch/delete' && request.method === 'POST') {
    if (!hasCapability(user.role, 'counselor.kb.write')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBBatchDelete(db, request, user);
  }

  if (path === '/api/v1/kb/upload' && request.method === 'POST') {
    if (!hasCapability(user.role, 'union.kb.submit') && !hasCapability(user.role, 'counselor.kb.write')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleKBUpload(db, request, user);
  }

  if (path === '/api/v1/kb/formats' && request.method === 'GET') {
    return jsonResponse({
      formats: ['txt', 'csv', 'pdf', 'docx', 'xlsx', 'png', 'jpg', 'jpeg', 'gif', 'bmp', 'webp', 'mp4', 'avi', 'mov', 'mkv'],
      max_size_mb: 50,
    });
  }

  return jsonResponse({ code: 404, message: '知识库接口不存在' }, 404);
}

async function handleReviewRoutes(db, request, url, path, user) {
  if (path === '/api/v1/review/pending' && request.method === 'GET') {
    if (!hasCapability(user.role, 'counselor.review.pending')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleReviewPending(db, url, user);
  }

  return jsonResponse({ code: 404, message: '审核接口不存在' }, 404);
}

async function handleAdminRoutes(db, request, url, path, user) {
  if (path === '/api/v1/admin/stats/dashboard' && request.method === 'GET') {
    if (!hasCapability(user.role, 'college.metrics.read')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleDashboardStats(db);
  }

  if (path === '/api/v1/admin/users' && request.method === 'GET') {
    if (!hasCapability(user.role, 'college.user.read')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return await handleUserList(db, url);
  }

  if (path === '/api/v1/admin/audit' && request.method === 'GET') {
    if (!hasCapability(user.role, 'college.audit.read')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return jsonResponse({ code: 0, data: { items: [], total: 0 } });
  }

  if (path === '/api/v1/admin/settings' && request.method === 'GET') {
    if (!hasCapability(user.role, 'system.settings.write')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return jsonResponse({ code: 0, data: {} });
  }

  if (path === '/api/v1/admin/settings' && request.method === 'PUT') {
    if (!hasCapability(user.role, 'system.settings.write')) {
      return jsonResponse({ code: 403, message: '无权限' }, 403);
    }
    return jsonResponse({ code: 0, message: 'success' });
  }

  return jsonResponse({ code: 404, message: '管理端接口不存在' }, 404);
}

async function handleHealth(db) {
  try {
    const result = await db.query('SELECT 1 as ok');
    return jsonResponse({
      status: 'ok',
      database: 'connected',
      timestamp: now(),
    });
  } catch (e) {
    return jsonResponse({
      status: 'error',
      database: 'disconnected',
      error: e.message,
    }, 500);
  }
}

async function handleLogin(db, request) {
  const body = await request.json();
  const { username, password } = body;

  if (!username || !password) {
    return jsonResponse({ code: 400, message: '用户名和密码不能为空' }, 400);
  }

  const user = await db.getOne('SELECT * FROM users WHERE username = ?', [username]);
  if (!user) {
    return jsonResponse({ code: 401, message: '用户名或密码错误' }, 401);
  }

  const valid = await verifyPassword(password, user.password_hash);
  if (!valid) {
    return jsonResponse({ code: 401, message: '用户名或密码错误' }, 401);
  }

  if (user.status === 'disabled') {
    return jsonResponse({ code: 403, message: '账号已被禁用' }, 403);
  }

  const token = await generateToken({
    user_id: user.id,
    username: user.username,
    role: user.role,
  });

  return jsonResponse({
    code: 0,
    message: '登录成功',
    data: {
      token,
      user: sanitizeUser(user),
    },
  });
}

async function handleChangePassword(db, user, request) {
  const body = await request.json();
  const { old_password, new_password } = body;

  const userRecord = await db.getOne('SELECT * FROM users WHERE id = ?', [user.id]);
  const valid = await verifyPassword(old_password, userRecord.password_hash);
  if (!valid) {
    return jsonResponse({ code: 400, message: '原密码错误' }, 400);
  }

  const newHash = await hashPassword(new_password);
  await db.execute(
    'UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?',
    [newHash, now(), user.id]
  );

  return jsonResponse({ code: 0, message: '密码修改成功' });
}

async function getCurrentUser(db, request) {
  const authHeader = request.headers.get('Authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return null;
  }

  const token = authHeader.substring(7);
  const payload = await verifyToken(token);
  if (!payload) return null;

  const user = await db.getOne('SELECT * FROM users WHERE id = ?', [payload.user_id]);
  if (!user) return null;

  return sanitizeUser(user);
}

function sanitizeUser(user) {
  return {
    id: user.id,
    username: user.username,
    display_name: user.display_name,
    role: user.role,
    owner_scope: user.owner_scope,
    owner_id: user.owner_id,
    avatar: user.avatar,
    email: user.email,
    phone: user.phone,
    status: user.status,
    created_at: user.created_at,
    updated_at: user.updated_at,
  };
}

async function handlePublicKnowledge(db, url) {
  const page = parseInt(url.searchParams.get('page')) || 1;
  const pageSize = parseInt(url.searchParams.get('page_size')) || 10;

  const countResult = await db.getOne(
    'SELECT COUNT(*) as count FROM kb_resources WHERE status = ? AND visibility = ?',
    ['published', 'public']
  );
  const total = countResult?.count || 0;

  const offset = (page - 1) * pageSize;
  const rows = await db.getAll(
    'SELECT * FROM kb_resources WHERE status = ? AND visibility = ? ORDER BY created_at DESC LIMIT ? OFFSET ?',
    ['published', 'public', pageSize, offset]
  );

  return jsonResponse({
    code: 0,
    data: {
      items: rows,
      total,
      page,
      page_size: pageSize,
    },
  });
}

async function handleKBStats(db, user) {
  const statuses = ['draft', 'pending_review', 'published', 'retired'];
  const stats = {};
  let total = 0;

  for (const status of statuses) {
    const result = await db.getOne(
      'SELECT COUNT(*) as count FROM kb_resources WHERE status = ?',
      [status]
    );
    const count = result?.count || 0;
    stats[status] = count;
    total += count;
  }

  return jsonResponse({
    code: 0,
    data: {
      ...stats,
      total,
    },
  });
}

function handleKBDict() {
  return jsonResponse({
    code: 0,
    data: {
      resource_types: [
        { value: 'Policy', label: '政策文件' },
        { value: 'Process', label: '办事流程' },
        { value: 'FAQ', label: '常见问题' },
        { value: 'Activity', label: '活动通知' },
        { value: 'Document', label: '文档资料' },
        { value: 'News', label: '新闻资讯' },
      ],
      statuses: [
        { value: 'draft', label: '草稿' },
        { value: 'pending_review', label: '待审核' },
        { value: 'published', label: '已发布' },
        { value: 'retired', label: '已下架' },
      ],
      visibilities: [
        { value: 'public', label: '全校公开' },
        { value: 'college', label: '学院内可见' },
        { value: 'private', label: '仅创建者可见' },
      ],
    },
  });
}

async function handleKBList(db, url, user) {
  const page = parseInt(url.searchParams.get('page')) || 1;
  const pageSize = parseInt(url.searchParams.get('page_size')) || 20;
  const status = url.searchParams.get('status');
  const type = url.searchParams.get('resource_type');
  const keyword = url.searchParams.get('keyword');
  const createdBy = url.searchParams.get('created_by');

  let whereClauses = [];
  let params = [];

  if (status) {
    whereClauses.push('status = ?');
    params.push(status);
  }
  if (type) {
    whereClauses.push('resource_type = ?');
    params.push(type);
  }
  if (keyword) {
    whereClauses.push('(title LIKE ? OR summary LIKE ?)');
    params.push('%' + keyword + '%', '%' + keyword + '%');
  }
  if (createdBy) {
    whereClauses.push('created_by = ?');
    params.push(createdBy);
  }

  const whereSql = whereClauses.length > 0 ? 'WHERE ' + whereClauses.join(' AND ') : '';

  const countResult = await db.getOne('SELECT COUNT(*) as count FROM kb_resources ' + whereSql, params);
  const total = countResult?.count || 0;

  const offset = (page - 1) * pageSize;
  const rows = await db.getAll(
    'SELECT * FROM kb_resources ' + whereSql + ' ORDER BY created_at DESC LIMIT ? OFFSET ?',
    [...params, pageSize, offset]
  );

  return jsonResponse({
    code: 0,
    data: {
      items: rows,
      total,
      page,
      page_size: pageSize,
    },
  });
}

async function handleKBListAdvanced(db, url, user) {
  return handleKBList(db, url, user);
}

async function handleKBGet(db, id) {
  const resource = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [id]);
  if (!resource) {
    return jsonResponse({ code: 404, message: '资源不存在' }, 404);
  }
  return jsonResponse({ code: 0, data: resource });
}

async function handleKBCreate(db, request, user) {
  const body = await request.json();
  const resourceId = body.resource_id || generateId();
  const createdAt = now();

  await db.execute(
    'INSERT INTO kb_resources (resource_id, title, summary, content, resource_type, status, visibility, keywords, tags, author, source, version, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)',
    [
      resourceId,
      body.title || '未命名资源',
      body.summary || '',
      body.content || '',
      body.resource_type || 'FAQ',
      body.status || 'draft',
      body.visibility || 'private',
      JSON.stringify(body.keywords || []),
      JSON.stringify(body.tags || []),
      body.author || user.display_name || user.username,
      body.source || '',
      body.version || 'v1.0',
      user.username,
      createdAt,
      createdAt,
    ]
  );

  const resource = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [resourceId]);
  return jsonResponse({ code: 0, data: resource, message: '创建成功' });
}

async function handleKBUpdate(db, id, request) {
  const body = await request.json();

  const existing = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [id]);
  if (!existing) {
    return jsonResponse({ code: 404, message: '资源不存在' }, 404);
  }

  const fields = [];
  const params = [];

  if (body.title !== undefined) { fields.push('title = ?'); params.push(body.title); }
  if (body.summary !== undefined) { fields.push('summary = ?'); params.push(body.summary); }
  if (body.content !== undefined) { fields.push('content = ?'); params.push(body.content); }
  if (body.resource_type !== undefined) { fields.push('resource_type = ?'); params.push(body.resource_type); }
  if (body.status !== undefined) { fields.push('status = ?'); params.push(body.status); }
  if (body.visibility !== undefined) { fields.push('visibility = ?'); params.push(body.visibility); }
  if (body.keywords !== undefined) { fields.push('keywords = ?'); params.push(JSON.stringify(body.keywords)); }
  if (body.tags !== undefined) { fields.push('tags = ?'); params.push(JSON.stringify(body.tags)); }
  if (body.author !== undefined) { fields.push('author = ?'); params.push(body.author); }
  if (body.version !== undefined) { fields.push('version = ?'); params.push(body.version); }
  if (body.remark !== undefined) { fields.push('remark = ?'); params.push(body.remark); }

  fields.push('updated_at = ?');
  params.push(now());

  params.push(id);

  await db.execute('UPDATE kb_resources SET ' + fields.join(', ') + ' WHERE resource_id = ?', params);

  const updated = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [id]);
  return jsonResponse({ code: 0, data: updated, message: '更新成功' });
}

async function handleKBSubmit(db, id, user) {
  const existing = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [id]);
  if (!existing) {
    return jsonResponse({ code: 404, message: '资源不存在' }, 404);
  }

  await db.execute(
    'UPDATE kb_resources SET status = ?, submitted_at = ?, updated_at = ? WHERE resource_id = ?',
    ['pending_review', now(), now(), id]
  );

  const updated = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [id]);
  return jsonResponse({ code: 0, data: updated, message: '已提交审核' });
}

async function handleKBApprove(db, id, user) {
  const existing = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [id]);
  if (!existing) {
    return jsonResponse({ code: 404, message: '资源不存在' }, 404);
  }

  await db.execute(
    'UPDATE kb_resources SET status = ?, reviewed_by = ?, reviewed_at = ?, updated_at = ? WHERE resource_id = ?',
    ['published', user.username, now(), now(), id]
  );

  const updated = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [id]);
  return jsonResponse({ code: 0, data: updated, message: '审核通过' });
}

async function handleKBReject(db, id, request, user) {
  const body = await request.json();
  const existing = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [id]);
  if (!existing) {
    return jsonResponse({ code: 404, message: '资源不存在' }, 404);
  }

  await db.execute(
    'UPDATE kb_resources SET status = ?, review_remark = ?, reviewed_by = ?, reviewed_at = ?, updated_at = ? WHERE resource_id = ?',
    ['draft', body?.remark || '审核未通过', user.username, now(), now(), id]
  );

  const updated = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [id]);
  return jsonResponse({ code: 0, data: updated, message: '已驳回' });
}

async function handleKBRetire(db, id, user) {
  const existing = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [id]);
  if (!existing) {
    return jsonResponse({ code: 404, message: '资源不存在' }, 404);
  }

  await db.execute(
    'UPDATE kb_resources SET status = ?, updated_at = ? WHERE resource_id = ?',
    ['retired', now(), id]
  );

  const updated = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [id]);
  return jsonResponse({ code: 0, data: updated, message: '已下架' });
}

async function handleKBBatchApprove(db, request, user) {
  const body = await request.json();
  const ids = body.ids || [];
  if (ids.length === 0) {
    return jsonResponse({ code: 400, message: '请选择要操作的资源' }, 400);
  }

  const placeholders = ids.map(() => '?').join(',');
  await db.execute(
    "UPDATE kb_resources SET status = 'published', reviewed_by = ?, reviewed_at = ?, updated_at = ? WHERE resource_id IN (" + placeholders + ")",
    [user.username, now(), now(), ...ids]
  );

  return jsonResponse({ code: 0, message: '已通过 ' + ids.length + ' 条' });
}

async function handleKBBatchReject(db, request, user) {
  const body = await request.json();
  const ids = body.ids || [];
  if (ids.length === 0) {
    return jsonResponse({ code: 400, message: '请选择要操作的资源' }, 400);
  }

  const placeholders = ids.map(() => '?').join(',');
  await db.execute(
    "UPDATE kb_resources SET status = 'draft', review_remark = ?, reviewed_by = ?, reviewed_at = ?, updated_at = ? WHERE resource_id IN (" + placeholders + ")",
    [body?.remark || '批量驳回', user.username, now(), now(), ...ids]
  );

  return jsonResponse({ code: 0, message: '已驳回 ' + ids.length + ' 条' });
}

async function handleKBBatchRetire(db, request, user) {
  const body = await request.json();
  const ids = body.ids || [];
  if (ids.length === 0) {
    return jsonResponse({ code: 400, message: '请选择要操作的资源' }, 400);
  }

  const placeholders = ids.map(() => '?').join(',');
  await db.execute(
    "UPDATE kb_resources SET status = 'retired', updated_at = ? WHERE resource_id IN (" + placeholders + ")",
    [now(), ...ids]
  );

  return jsonResponse({ code: 0, message: '已下架 ' + ids.length + ' 条' });
}

async function handleKBBatchDelete(db, request, user) {
  const body = await request.json();
  const ids = body.ids || [];
  if (ids.length === 0) {
    return jsonResponse({ code: 400, message: '请选择要删除的资源' }, 400);
  }

  const placeholders = ids.map(() => '?').join(',');
  await db.execute('DELETE FROM kb_resources WHERE resource_id IN (' + placeholders + ')', ids);

  return jsonResponse({ code: 0, message: '已删除 ' + ids.length + ' 条' });
}

async function handleReviewPending(db, url, user) {
  const page = parseInt(url.searchParams.get('page')) || 1;
  const pageSize = parseInt(url.searchParams.get('page_size')) || 20;

  const countResult = await db.getOne(
    'SELECT COUNT(*) as count FROM kb_resources WHERE status = ?',
    ['pending_review']
  );
  const total = countResult?.count || 0;

  const offset = (page - 1) * pageSize;
  const rows = await db.getAll(
    'SELECT * FROM kb_resources WHERE status = ? ORDER BY submitted_at ASC LIMIT ? OFFSET ?',
    ['pending_review', pageSize, offset]
  );

  return jsonResponse({
    code: 0,
    data: {
      items: rows,
      total,
      page,
      page_size: pageSize,
    },
  });
}

async function handleKBUpload(db, request, user) {
  const formData = await request.formData();
  const file = formData.get('file');
  const resourceType = formData.get('resource_type') || 'FAQ';

  if (!file) {
    return jsonResponse({ code: 400, message: '请选择要上传的文件' }, 400);
  }

  const filename = file.name || 'document';
  const resourceId = generateId();
  const createdAt = now();

  let title = filename.replace(/\.[^/.]+$/, '');
  let summary = '上传文档：' + filename;
  let content = '';

  try {
    const text = await file.text();
    content = text;
    if (text.length > 0) {
      const firstLine = text.split('\n').find(l => l.trim());
      if (firstLine && firstLine.length < 200) {
        title = firstLine.replace(/^#+\s*/, '').trim();
      }
      summary = text.substring(0, 200) + (text.length > 200 ? '...' : '');
    }
  } catch (e) {
    // 非文本文件
  }

  await db.execute(
    'INSERT INTO kb_resources (resource_id, title, summary, content, resource_type, status, visibility, created_by, file_name, file_size, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)',
    [resourceId, title, summary, content, resourceType, 'draft', 'private', user.username, filename, file.size || 0, createdAt, createdAt]
  );

  const resource = await db.getOne('SELECT * FROM kb_resources WHERE resource_id = ?', [resourceId]);
  return jsonResponse({
    code: 0,
    message: '上传成功',
    data: resource,
  });
}

async function handleDashboardStats(db) {
  const userCount = await db.getOne('SELECT COUNT(*) as count FROM users');
  const kbCount = await db.getOne('SELECT COUNT(*) as count FROM kb_resources WHERE status = ?', ['published']);
  const pendingCount = await db.getOne('SELECT COUNT(*) as count FROM kb_resources WHERE status = ?', ['pending_review']);

  return jsonResponse({
    code: 0,
    data: {
      total_users: userCount?.count || 0,
      total_knowledge: kbCount?.count || 0,
      pending_review: pendingCount?.count || 0,
    },
  });
}

async function handleUserList(db, url) {
  const page = parseInt(url.searchParams.get('page')) || 1;
  const pageSize = parseInt(url.searchParams.get('page_size')) || 20;

  const countResult = await db.getOne('SELECT COUNT(*) as count FROM users');
  const total = countResult?.count || 0;

  const offset = (page - 1) * pageSize;
  const rows = await db.getAll(
    'SELECT id, username, display_name, role, owner_scope, owner_id, email, phone, status, created_at FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?',
    [pageSize, offset]
  );

  return jsonResponse({
    code: 0,
    data: {
      items: rows,
      total,
      page,
      page_size: pageSize,
    },
  });
}

async function handleProcessEnhanced(db, url) {
  const type = url.searchParams.get('type') || 'enrollment';
  const typeMap = {
    'enrollment': ['入学', '报名', '报到'],
    'leave': ['请假', '销假'],
    'scholarship': ['奖学金', '助学金'],
    'graduation': ['毕业', '离校'],
  };
  const keywords = typeMap[type] || typeMap['enrollment'];

  const processRows = await db.getAll(
    `SELECT ps.*, kr.title as resource_title, kr.summary as resource_summary
     FROM process_steps ps
     JOIN kb_resources kr ON ps.resource_id = kr.resource_id
     WHERE kr.status = 'published'
       AND (kr.title LIKE ? OR kr.summary LIKE ?)
     ORDER BY ps.step_order`,
    [`%${keywords[0]}%`, `%${keywords[0]}%`]
  );

  return jsonResponse({
    code: 0,
    data: {
      processes: processRows,
      type,
    },
  });
}

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      ...corsHeaders(),
      'Content-Type': 'application/json; charset=utf-8',
    },
  });
}

function corsHeaders() {
  return {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET,POST,PUT,DELETE,OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type,Authorization',
  };
}
