const APK_FILE = '蔚小芯-v0.0.4.apk';
const APK_URL = 'https://wxx-agent.pages.dev/downloads/%E8%94%9A%E5%B0%8F%E8%8A%AF-v0.0.4.apk';

export async function onRequest(context) {
  const { request, params } = context;
  const file = Array.isArray(params.file) ? params.file.join('/') : params.file;

  if (request.method === 'OPTIONS') {
    return new Response(null, { status: 204, headers: corsHeaders() });
  }

  if (file === 'release.json') {
    return jsonResponse({
      app: '蔚小芯',
      release_date: '2026-07-20',
      apk_url: APK_URL,
      build_number: 4,
      apk_file: APK_FILE,
      version: '0.0.4',
    });
  }

  if (file !== APK_FILE && file !== encodeURIComponent(APK_FILE)) {
    return new Response('Not Found', { status: 404, headers: corsHeaders() });
  }

  const assetResponse = await context.env.ASSETS.fetch(request);
  if (!assetResponse.ok || !assetResponse.body) {
    return new Response('APK 暂时不可下载，请稍后重试', {
      status: 502,
      headers: corsHeaders(),
    });
  }

  const headers = new Headers(assetResponse.headers);
  applyCorsHeaders(headers);
  headers.set('Content-Type', 'application/vnd.android.package-archive');
  headers.set('Content-Disposition', `attachment; filename*=UTF-8''${encodeURIComponent(APK_FILE)}`);
  headers.set('Cache-Control', 'public, max-age=300');
  return new Response(assetResponse.body, { status: 200, headers });
}

function jsonResponse(data) {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: {
      ...corsHeaders(),
      'Content-Type': 'application/json; charset=utf-8',
      'Cache-Control': 'public, max-age=60',
    },
  });
}

function corsHeaders() {
  return {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET,HEAD,OPTIONS',
    'Access-Control-Allow-Headers': '*',
  };
}

function applyCorsHeaders(headers) {
  for (const [key, value] of Object.entries(corsHeaders())) {
    headers.set(key, value);
  }
}
