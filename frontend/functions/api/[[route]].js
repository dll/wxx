// Proxy API requests to Vercel backend
export async function onRequest(context) {
  const { request } = context;
  const url = new URL(request.url);
  const backendPath = url.pathname === '/api/health' ? '/health' : url.pathname;
  const targetUrl = 'https://wxx-server-j1us8ki1c-czldl.vercel.app' + backendPath + url.search;

  if (request.method === 'OPTIONS') {
    return new Response(null, {
      status: 204,
      headers: corsHeaders(),
    });
  }

  const headers = new Headers(request.headers);
  headers.set('Host', 'wxx-server-j1us8ki1c-czldl.vercel.app');

  // 所有非 GET/HEAD 请求都先读取为 arrayBuffer，确保 multipart/form-data 等请求体完整传递
  let requestBody;
  if (request.method !== 'GET' && request.method !== 'HEAD') {
    requestBody = await request.arrayBuffer();
  }

  const isLogin = url.pathname === '/api/v1/auth/login';
  const isFeedbackSubmit = request.method === 'POST' &&
    url.pathname === '/api/v1/feedback';
  // 这些关键接口需要缓冲响应 + 重试
  const shouldBuffer = isLogin ||
    isFeedbackSubmit ||
    url.pathname === '/api/health' ||
    url.pathname === '/api/v1/user/profile' ||
    url.pathname === '/api/v1/user/capabilities';
  // 反馈提交是非幂等写操作，不自动重试，避免上游已写入后重复创建
  const maxAttempts = shouldBuffer && !isFeedbackSubmit ? 2 : 1;
  let lastError;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      const resp = await fetch(targetUrl, {
        method: request.method,
        headers,
        body: requestBody ? requestBody.slice(0) : undefined,
      });

      // 关键接口先完整读取响应，避免上游流中断后浏览器只得到模糊网络错误
      const responseBody = shouldBuffer ? await resp.arrayBuffer() : resp.body;
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
    message: isLogin
      ? '登录服务暂时不可用，请稍后重试'
      : isFeedbackSubmit
        ? '反馈服务暂时不可用，请稍后重试'
        : '后端服务暂时不可用，请稍后重试',
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
