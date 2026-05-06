// 蔚小芯 — 本地缓存工具（token + 用户信息）

const TOKEN_KEY = 'wxx_token';
const USER_KEY = 'wxx_user';

// ── Token ──
function getToken() {
  return wx.getStorageSync(TOKEN_KEY) || null;
}

function setToken(token) {
  wx.setStorageSync(TOKEN_KEY, token);
}

function clearToken() {
  wx.removeStorageSync(TOKEN_KEY);
}

// ── 用户信息 ──
function getUserInfo() {
  return wx.getStorageSync(USER_KEY) || null;
}

function setUserInfo(info) {
  wx.setStorageSync(USER_KEY, info);
}

function clearUserInfo() {
  wx.removeStorageSync(USER_KEY);
}

module.exports = {
  getToken,
  setToken,
  clearToken,
  getUserInfo,
  setUserInfo,
  clearUserInfo
};
