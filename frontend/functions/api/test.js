// 简单测试函数
export async function onRequest(context) {
  return new Response(JSON.stringify({
    status: 'ok',
    message: 'Cloudflare Functions 工作正常',
    time: new Date().toISOString(),
  }), {
    headers: {
      'Content-Type': 'application/json',
      'Access-Control-Allow-Origin': '*',
    },
  });
}
