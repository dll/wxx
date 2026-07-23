// Proxy API requests to Vercel backend
export async function onRequest(context) {
  const { request } = context;
  const url = new URL(request.url);
  const targetUrl = 'https://wxx-server-j1us8ki1c-czldl.vercel.app' + url.pathname + url.search;

  const headers = new Headers(request.headers);
  headers.set('Host', 'wxx-server-j1us8ki1c-czldl.vercel.app');

  const init = { method: request.method, headers };
  if (request.method !== 'GET' && request.method !== 'HEAD') {
    init.body = await request.arrayBuffer();
  }

  const resp = await fetch(targetUrl, init);

  const out = new Headers(resp.headers);
  out.set('Access-Control-Allow-Origin', '*');
  out.set('Access-Control-Allow-Methods', 'GET,POST,PUT,DELETE,OPTIONS');
  out.set('Access-Control-Allow-Headers', 'Content-Type,Authorization');

  return new Response(resp.body, { status: resp.status, headers: out });
}
