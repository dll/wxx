// 蔚小芯 API 代理 Worker — 转发请求到 Vercel 后端
const TARGET = 'https://wxx-server-2apyus7gx-czldl.vercel.app';

export default {
  async fetch(request) {
    const url = new URL(request.url);
    const targetUrl = TARGET + url.pathname + url.search;

    const headers = new Headers(request.headers);
    headers.set('Host', new URL(TARGET).hostname);

    const resp = await fetch(targetUrl, {
      method: request.method,
      headers,
      body: request.method !== 'GET' && request.method !== 'HEAD'
        ? await request.arrayBuffer()
        : undefined,
    });

    const corsHeaders = new Headers(resp.headers);
    corsHeaders.set('Access-Control-Allow-Origin', '*');
    corsHeaders.set('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    corsHeaders.set('Access-Control-Allow-Headers', 'Content-Type, Authorization');

    return new Response(resp.body, {
      status: resp.status,
      headers: corsHeaders,
    });
  },
};
