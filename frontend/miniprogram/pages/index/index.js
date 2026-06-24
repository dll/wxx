// 首页 — WebView 壳，加载 Flutter Web 构建的前端
Page({
  data: {
    // 前端地址（Cloudflare Pages）
    src: 'https://wxx-agent.pages.dev'
  },

  onLoad() {
    console.log('蔚小芯 WebView 已加载');
  },

  // WebView 加载成功
  onWebViewLoad() {
    console.log('Flutter Web 已加载完成');
  },

  // WebView 加载失败
  onWebViewError(e) {
    console.error('WebView 加载失败:', e.detail);
    wx.showToast({
      title: '页面加载失败，请检查网络',
      icon: 'none',
      duration: 3000
    });
  }
});
