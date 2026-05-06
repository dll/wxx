// 蔚小芯 — API 请求封装（wx.request + JWT 注入 + 401 拦截）

var app = getApp();

// 后端 API 基地址（开发环境，生产构建时替换）
const BASE_URL = 'http://localhost:8080/api/v1';

// 模块级 token 缓存，避免每次 request() 都读 app.globalData
var cachedToken = '';

function refreshToken() {
  try {
    cachedToken = app.globalData.token || '';
  } catch (e) {
    cachedToken = '';
  }
}

// app 启动后调用一次，token 变更时调用
function _onTokenChange() {
  refreshToken();
}

function request(options) {
  return new Promise(function (resolve, reject) {
    if (!cachedToken) refreshToken();

    var header = Object.assign({}, options.header || {});
    header['Content-Type'] = 'application/json';
    if (cachedToken) {
      header['Authorization'] = 'Bearer ' + cachedToken;
    }

    wx.request({
      url: BASE_URL + options.url,
      method: options.method || 'GET',
      data: options.data || {},
      header: header,
      success: function (res) {
        if (res.statusCode === 401) {
          cachedToken = '';
          try {
            if (app.clearLoginState) app.clearLoginState();
          } catch (e) { /* ignore */ }
          wx.reLaunch({ url: '/pages/login/login' });
          reject(new Error('登录已过期，请重新登录'));
          return;
        }
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res.data);
        } else {
          var msg = '请求失败';
          if (res.data && res.data.message) msg = res.data.message;
          reject(new Error(msg));
        }
      },
      fail: function (err) {
        reject(new Error('网络请求失败: ' + (err.errMsg || '未知错误')));
      }
    });
  });
}

// ── 认证 API ──

function usernameLogin(username) {
  return request({
    url: '/auth/login',
    method: 'POST',
    data: { username: username }
  });
}

// ── 用户 API ──

function getProfile() {
  return request({ url: '/user/profile' });
}

// ── 问答 API ──

function askQuestion(question, sessionId) {
  return request({
    url: '/chat',
    method: 'POST',
    data: {
      question: question,
      session_id: sessionId || ''
    }
  });
}

function getSessions() {
  return request({ url: '/sessions' });
}

function getSessionMessages(sessionId) {
  return request({ url: '/sessions/' + sessionId + '/messages' });
}

function deleteSession(sessionId) {
  return request({
    url: '/sessions/' + sessionId,
    method: 'DELETE'
  });
}

// ── 知识大厅 API ──

function browseKnowledge(params) {
  var queryParts = [];
  if (params) {
    if (params.resource_type) {
      queryParts.push('resource_type=' + encodeURIComponent(params.resource_type));
    }
    if (params.page !== undefined) {
      queryParts.push('page=' + params.page);
    }
    if (params.page_size) {
      queryParts.push('page_size=' + params.page_size);
    }
  }
  var query = queryParts.length > 0 ? '?' + queryParts.join('&') : '';
  return request({ url: '/knowledge' + query });
}

function getResource(id) {
  return request({ url: '/kb/resources/' + id });
}

// ── 推荐 API ──

function getRecommendations() {
  return request({ url: '/recommendations' });
}

module.exports = {
  BASE_URL,
  _onTokenChange,
  usernameLogin,
  getProfile,
  askQuestion,
  getSessions,
  getSessionMessages,
  deleteSession,
  browseKnowledge,
  getResource,
  getRecommendations
};
