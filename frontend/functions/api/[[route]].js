// Proxy API requests to Vercel backend
export async function onRequest(context) {
  const { request } = context;
  const url = new URL(request.url);
  const targetUrl = 'https://wxx-server-j1us8ki1c-czldl.vercel.app' + url.pathname + url.search;

  if (request.method === 'OPTIONS') {
    return new Response(null, {
      status: 204,
      headers: corsHeaders(),
    });
  }

  const headers = new Headers(request.headers);
  headers.set('Host', 'wxx-server-j1us8ki1c-czldl.vercel.app');

  const init = { method: request.method, headers };
  let requestBody;
  if (request.method !== 'GET' && request.method !== 'HEAD') {
    requestBody = await request.arrayBuffer();
  }

  const isLogin = url.pathname === '/api/v1/auth/login';
  const maxAttempts = isLogin ? 2 : 1;
  let lastError;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      const resp = await fetch(targetUrl, {
        ...init,
        body: requestBody ? requestBody.slice(0) : undefined,
      });

      // 登录响应先完整读取，避免上游流中断后浏览器只得到模糊网络错误。
      const responseBody = isLogin ? await resp.arrayBuffer() : resp.body;
      const out = new Headers(resp.headers);
      applyCorsHeaders(out);

      return new Response(responseBody, {
        status: resp.status,
        headers: out,
      });
    } catch (error) {
      lastError = error;
      if (attempt < maxAttempts) {
        await new Promise((resolve) => setTimeout(resolve, 200));
      }
    }
  }

  console.error('代理后端请求失败', lastError);
  return new Response(JSON.stringify({
    code: 502,
    message: isLogin ? '登录服务暂时不可用，请稍后重试' : '后端服务暂时不可用，请稍后重试',
  }), {
    status: 502,
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

function applyCorsHeaders(headers) {
  for (const [key, value] of Object.entries(corsHeaders())) {
    headers.set(key, value);
  }
}
