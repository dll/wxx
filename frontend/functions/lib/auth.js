// JWT 认证工具（使用 Web Crypto API，兼容 Cloudflare Workers）
// 密码验证已迁移至 Go 后端，bcryptjs 仅按需懒加载（不再被打包进 Workers 导致 503）

let _jwtSecret = '';
const JWT_EXPIRES_IN = 7 * 24 * 60 * 60; // 7 天

// 幂等：已注入后忽略重复调用（CF 每请求都调用一次 setJWTSecret）
export function setJWTSecret(s) { if (!_jwtSecret) _jwtSecret = s; }

// 同步即可，不涉及任何 I/O。
// 未配置 JWT_SECRET 时直接抛出——宁可认证失败也不使用仓库中已公开的弱密钥。
function getSecret() {
  if (_jwtSecret) return _jwtSecret;
  if (typeof JWT_SECRET !== 'undefined' && JWT_SECRET) {
    _jwtSecret = JWT_SECRET;
    return _jwtSecret;
  }
  throw new Error(
    'JWT_SECRET 未配置：请通过 wrangler pages secret put JWT_SECRET 设置加密环境变量'
  );
}

export async function generateToken(payload) {
  const secret = await getSecret();
  const header = { alg: 'HS256', typ: 'JWT' };
  const now = Math.floor(Date.now() / 1000);
  const tokenPayload = { ...payload, iat: now, exp: now + JWT_EXPIRES_IN };

  const headerB64 = base64UrlEncode(JSON.stringify(header));
  const payloadB64 = base64UrlEncode(JSON.stringify(tokenPayload));

  const signature = await hmacSign(`${headerB64}.${payloadB64}`, secret);
  const signatureB64 = base64UrlFromBytes(new Uint8Array(signature));

  return `${headerB64}.${payloadB64}.${signatureB64}`;
}

export async function verifyToken(token) {
  try {
    const secret = await getSecret();
    const parts = token.split('.');
    if (parts.length !== 3) return null;

    const [headerB64, payloadB64, signatureB64] = parts;
    const expectedSignature = await hmacSign(`${headerB64}.${payloadB64}`, secret);
    const expectedB64 = base64UrlFromBytes(new Uint8Array(expectedSignature));

    if (signatureB64 !== expectedB64) return null;

    const payload = JSON.parse(base64UrlDecode(payloadB64));
    const now = Math.floor(Date.now() / 1000);

    if (payload.exp && payload.exp < now) return null;

    return payload;
  } catch (e) {
    console.error('JWT verify error:', e);
    return null;
  }
}

async function hmacSign(data, secret) {
  const encoder = new TextEncoder();
  const keyData = encoder.encode(secret);
  const messageData = encoder.encode(data);

  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    keyData,
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  );

  return await crypto.subtle.sign('HMAC', cryptoKey, messageData);
}

function base64UrlEncode(str) {
  return btoa(unescape(encodeURIComponent(str)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
}

function base64UrlDecode(str) {
  const padded = str + '='.repeat((4 - (str.length % 4)) % 4);
  const base64 = padded.replace(/-/g, '+').replace(/_/g, '/');
  return decodeURIComponent(escape(atob(base64)));
}

function base64UrlFromBytes(bytes) {
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary)
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
}

// 以下 bcrypt 函数仅在边缘需要本地验证密码时使用（当前已迁移至 Go 后端）
let _bcrypt = null;
async function _ensureBcrypt() {
  if (!_bcrypt) {
    _bcrypt = await import('bcryptjs');
  }
  return _bcrypt;
}

// bcrypt 哈希密码（与 Go 后端兼容，仅在必要时通过动态 import 加载 bcryptjs）
export async function hashPassword(password) {
  const bcrypt = await _ensureBcrypt();
  const salt = bcrypt.genSaltSync(10);
  return bcrypt.hashSync(password, salt);
}

export async function verifyPassword(password, hash) {
  const bcrypt = await _ensureBcrypt();
  return bcrypt.compareSync(password, hash);
}
