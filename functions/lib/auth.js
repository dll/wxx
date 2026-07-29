// JWT 认证工具（使用 Web Crypto API，兼容 Cloudflare Workers）
// 密码哈希使用 bcryptjs 以与 Go 后端 bcrypt 兼容
import bcrypt from 'bcryptjs';

let _jwtSecret = '';
const JWT_EXPIRES_IN = 7 * 24 * 60 * 60; // 7 天

// 幂等：已注入后忽略重复调用（CF 每请求都调用一次 setJWTSecret）
export function setJWTSecret(s) { if (!_jwtSecret) _jwtSecret = s; }

// 同步即可，不涉及任何 I/O
function getSecret() {
  if (_jwtSecret) return _jwtSecret;
  _jwtSecret = (typeof JWT_SECRET !== 'undefined') ? JWT_SECRET : 'wxx-secret-key-change-in-production';
  return _jwtSecret;
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

// bcrypt 哈希密码（与 Go 后端兼容）
export async function hashPassword(password) {
  const salt = bcrypt.genSaltSync(10);
  return bcrypt.hashSync(password, salt);
}

export async function verifyPassword(password, hash) {
  return bcrypt.compareSync(password, hash);
}
