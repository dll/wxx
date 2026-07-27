// JWT 认证工具（使用 Web Crypto API，兼容 Cloudflare Workers）
import { stringToHex, hexToString } from './utils.js';

const JWT_SECRET = 'wxx-secret-key-change-in-production';
const JWT_EXPIRES_IN = 7 * 24 * 60 * 60; // 7 天

export async function generateToken(payload) {
  const header = { alg: 'HS256', typ: 'JWT' };
  const now = Math.floor(Date.now() / 1000);
  const tokenPayload = {
    ...payload,
    iat: now,
    exp: now + JWT_EXPIRES_IN,
  };

  const headerB64 = base64UrlEncode(JSON.stringify(header));
  const payloadB64 = base64UrlEncode(JSON.stringify(tokenPayload));

  const signature = await hmacSign(`${headerB64}.${payloadB64}`, JWT_SECRET);
  const signatureB64 = base64UrlEncode(String.fromCharCode(...new Uint8Array(signature)));

  return `${headerB64}.${payloadB64}.${signatureB64}`;
}

export async function verifyToken(token) {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;

    const [headerB64, payloadB64, signatureB64] = parts;

    const expectedSignature = await hmacSign(`${headerB64}.${payloadB64}`, JWT_SECRET);
    const expectedB64 = base64UrlEncode(String.fromCharCode(...new Uint8Array(expectedSignature)));

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

// 简单的密码哈希（实际项目应使用 bcrypt/argon2）
export async function hashPassword(password) {
  const encoder = new TextEncoder();
  const data = encoder.encode(password + JWT_SECRET);
  const hash = await crypto.subtle.digest('SHA-256', data);
  return Array.from(new Uint8Array(hash))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

export async function verifyPassword(password, hash) {
  const hashed = await hashPassword(password);
  return hashed === hash;
}
