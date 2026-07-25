// 下载路由：动态读取同目录静态 release.json 作为版本单一事实源，
// 不再硬编码版本号，避免发布新版本后此函数与静态清单/实际 APK 断链（Q-02）。
//
// 发布新版本只需：1) 更新 web/downloads/release.json；2) 放入对应 蔚小芯-vX.Y.Z.apk。
// 本函数无需改动。

// 允许下载的 APK 文件名白名单规则：蔚小芯-vX.Y.Z.apk 或固定分发名 蔚小芯.apk
function isAllowedApk(name) {
  return /^蔚小芯-v\d+\.\d+\.\d+\.apk$/.test(name) || name === '蔚小芯.apk';
}

export async function onRequest(context) {
  const { request, params } = context;
  const file = Array.isArray(params.file) ? params.file.join('/') : params.file;

  if (request.method === 'OPTIONS') {
    return new Response(null, { status: 204, headers: corsHeaders() });
  }

  // release.json 直接透传静态资产（单一事实源），仅补充 CORS 头
  if (file === 'release.json') {
    const assetResponse = await context.env.ASSETS.fetch(request);
    if (!assetResponse.ok) {
      return jsonResponse({ error: '版本信息暂不可用，请稍后重试' }, 502);
    }
    const headers = new Headers(assetResponse.headers);
    applyCorsHeaders(headers);
    headers.set('Content-Type', 'application/json; charset=utf-8');
    headers.set('Cache-Control', 'public, max-age=60');
    return new Response(assetResponse.body, { status: 200, headers });
  }

  // 解码后校验是否为允许的 APK 文件名
  let decoded = file;
  try {
    decoded = decodeURIComponent(file);
  } catch (_) {
    // file 非合法编码，保持原值，交由白名单判定
  }
  if (!isAllowedApk(decoded)) {
    return new Response('Not Found', { status: 404, headers: corsHeaders() });
  }

  // 交给静态资产服务，由其决定该 APK 是否真实存在
  const assetResponse = await context.env.ASSETS.fetch(request);
  if (!assetResponse.ok || (request.method !== 'HEAD' && !assetResponse.body)) {
    return new Response('APK 暂时不可下载，请稍后重试', {
      status: 502,
      headers: corsHeaders(),
    });
  }

  const headers = new Headers(assetResponse.headers);
  applyCorsHeaders(headers);
  headers.set('Content-Type', 'application/vnd.android.package-archive');
  headers.set('Content-Disposition', `attachment; filename*=UTF-8''${encodeURIComponent(decoded)}`);
  headers.set('Cache-Control', 'public, max-age=300');
  return new Response(request.method === 'HEAD' ? null : assetResponse.body, { status: 200, headers });
}

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
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
