// 蔚小芯后端 API - Cloudflare Pages Functions 版本
// 角色：JWT 边缘验证 + 反向代理到 Go 后端（完整业务逻辑）
// 登录/注册等公开路由直接透传，Go 后端负责签发 JWT

import { verifyToken, setJWTSecret } from '../lib/auth.js';

export async function onRequest(context) {
  const { request } = context;
  const url = new URL(request.url);

  if (context.env.JWT_SECRET) setJWTSecret(context.env.JWT_SECRET);

  // CORS 预检直接放行
  if (request.method === 'OPTIONS') {
    return new Response(null, { status: 204, headers: CORS_HEADERS });
  }

  const path = url.pathname;

  try {
    if (path === '/' || path === '/api') {
      return jsonResponse({ service: '蔚小芯', version: '1.0.0', docs: '/api/health' });
    }

    // 所有需要透传的路由（login 已迁移至 Go 后端，不再走 Turso 边缘认证）
    if (isPublicPath(path)) {
      return await proxyToGoBackend(request, url, context);
    }

    const user = await getCurrentUser(request);
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
  '/api/v1/auth/login',        // 登录已迁移至 Go 后端，边缘直接透传
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
  for (const [k, v] of Object.entries(CORS_HEADERS)) {
    outHeaders.set(k, v);
  }

  return new Response(response.body, {
    status: response.status,
    headers: outHeaders,
  });
}

async function getCurrentUser(request) {
  const authHeader = request.headers.get('Authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) return null;
  // JWT payload 已包含 user_id/username/role，无需再查数据库
  const payload = await verifyToken(authHeader.substring(7));
  return payload || null;
}

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { ...CORS_HEADERS, 'Content-Type': 'application/json; charset=utf-8' },
  });
}

// CORS 头为部署期间不变的静态常量，提升为模块级常量避免每次响应重复分配对象。
const CORS_HEADERS = Object.freeze({
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'GET,POST,PUT,DELETE,OPTIONS',
  'Access-Control-Allow-Headers': 'Content-Type,Authorization',
});
