// 蔚小芯后端 API - Cloudflare Pages Functions 版本
// 仅做认证（边缘加速） + 反向代理到 Go 后端（完整业务逻辑）

import { TursoClient } from '../lib/turso.js';
import { generateToken, verifyToken, verifyPassword, setJWTSecret } from '../lib/auth.js';
import { now } from '../lib/utils.js';

let db = null;

function getDb(context) {
  if (db) return db;
  db = new TursoClient(context.env.TURSO_DB_URL, context.env.TURSO_DB_TOKEN);
  return db;
}

export async function onRequest(context) {
  const { request } = context;
  const url = new URL(request.url);

  if (context.env.JWT_SECRET) setJWTSecret(context.env.JWT_SECRET);

  if (request.method === 'OPTIONS') {
    return new Response(null, { status: 204, headers: corsHeaders() });
  }

  const path = url.pathname;
  const db = getDb(context);

  try {
    if (path === '/health' || path === '/api/health') {
      return await handleHealth(db);
    }

    if (path === '/' || path === '/api') {
      return jsonResponse({ service: '蔚小芯', version: '1.0.0', docs: '/health' });
    }

    if (path === '/api/v1/auth/login' && request.method === 'POST') {
      return await handleLogin(db, request);
    }

    // 公开路由：与 Go 后端 setupRouter 中未挂 JWTAuth 的路由保持一致，
    // 直接透传，否则会被边缘 401 拦截而永远到不了后端。
    if (isPublicPath(path)) {
      return await proxyToGoBackend(request, url, context);
    }

    const user = await getCurrentUser(db, request);
    if (!user) {
      return jsonResponse({ code: 401, message: '未登录或登录已过期' }, 401);
    }

    return await proxyToGoBackend(request, url, context);
  } catch (error) {
    console.error('API Error:', error);
    return jsonResponse({ code: 500, message: '服务器内部错误: ' + error.message }, 500);
  }
}

const PUBLIC_PATHS = new Set([
  '/api/v1/auth/qr-login',
  '/api/v1/auth/qr-status',
  '/api/v1/auth/qr-scan',
  '/api/v1/auth/send-code',
  '/api/v1/auth/guest-register',
  '/api/v1/version/check',
  '/api/v1/version/latest',
  '/api/v1/knowledge/public',
]);

function isPublicPath(path) {
  return PUBLIC_PATHS.has(path);
}

async function proxyToGoBackend(request, url, context) {
  const backendUrl = context.env.GO_BACKEND_URL;
  if (!backendUrl) {
    return jsonResponse({ code: 500, message: 'GO_BACKEND_URL 未配置' }, 500);
  }

  const targetUrl = backendUrl.replace(/\/+$/, '') + url.pathname + url.search;
  const headers = new Headers(request.headers);
  // Host 必须由目标地址决定，否则后端会收到 Pages 域名
  headers.delete('host');

  const hasBody = request.method !== 'GET' && request.method !== 'HEAD';
  const response = await fetch(targetUrl, {
    method: request.method,
    headers,
    // 用 arrayBuffer 而非 text()，避免语音/文档等二进制上传被破坏
    body: hasBody ? await request.arrayBuffer() : undefined,
  });

  // 透传后端响应头（Content-Disposition 等对导出下载是必需的），再叠加 CORS
  const outHeaders = new Headers(response.headers);
  for (const [k, v] of Object.entries(corsHeaders())) {
    outHeaders.set(k, v);
  }

  return new Response(response.body, {
    status: response.status,
    headers: outHeaders,
  });
}

async function handleHealth(db) {
  try {
    await db.query('SELECT 1 as ok');
    return jsonResponse({ status: 'ok', database: 'connected', timestamp: now() });
  } catch (e) {
    return jsonResponse({ status: 'error', database: 'disconnected', error: e.message }, 500);
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
    user_id: parseInt(user.id, 10),
    username: user.username,
    role: user.role,
  });

  return jsonResponse({
    code: 0,
    message: '登录成功',
    data: { token, user: sanitizeUser(user) },
  });
}

async function getCurrentUser(db, request) {
  const authHeader = request.headers.get('Authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) return null;

  const payload = await verifyToken(authHeader.substring(7));
  if (!payload) return null;

  const user = await db.getOne('SELECT * FROM users WHERE id = ?', [payload.user_id]);
  return user ? sanitizeUser(user) : null;
}

function sanitizeUser(user) {
  return {
    id: user.id, username: user.username, display_name: user.display_name,
    role: user.role, owner_scope: user.owner_scope, owner_id: user.owner_id,
    avatar: user.avatar, email: user.email, phone: user.phone,
    status: user.status, created_at: user.created_at, updated_at: user.updated_at,
  };
}

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { ...corsHeaders(), 'Content-Type': 'application/json; charset=utf-8' },
  });
}

function corsHeaders() {
  return {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET,POST,PUT,DELETE,OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type,Authorization',
  };
}
