/**
 * 蔚小芯 — storage.js 烟雾测试
 * 运行: node __tests__/storage.test.js
 */

// Mock wx.storage API（内存实现）
var store = {};
global.wx = {
  getStorageSync: function (key) {
    return store[key] !== undefined ? store[key] : '';
  },
  setStorageSync: function (key, value) {
    store[key] = value;
  },
  removeStorageSync: function (key) {
    delete store[key];
  },
  // storage.js 不用的方法也占位
  reLaunch: function () {},
};

var storage = require('../utils/storage');

var passed = 0;
var failed = 0;

function assert(condition, msg) {
  if (condition) {
    passed++;
  } else {
    failed++;
    console.error('  FAIL: ' + msg);
  }
}

function assertEqual(actual, expected, msg) {
  if (actual === expected) {
    passed++;
  } else {
    failed++;
    console.error('  FAIL: ' + msg + ' (期望: ' + JSON.stringify(expected) + ', 实际: ' + JSON.stringify(actual) + ')');
  }
}

// ── Token ──
console.log('\ngetToken() / setToken() / clearToken()');

// 初始为空
assertEqual(storage.getToken(), null, '初始 token 为 null');

// 设置后读取
storage.setToken('test-jwt-token-123');
assertEqual(storage.getToken(), 'test-jwt-token-123', 'setToken 后 getToken 返回正确值');

// 清除后为空
storage.clearToken();
assertEqual(storage.getToken(), null, 'clearToken 后 getToken 返回 null');

// ── 用户信息 ──
console.log('getUserInfo() / setUserInfo() / clearUserInfo()');

// 初始为空
assertEqual(storage.getUserInfo(), null, '初始 userInfo 为 null');

// 设置后读取
var userInfo = { display_name: '张三', role: 'student' };
storage.setUserInfo(userInfo);
var got = storage.getUserInfo();
assertEqual(got.display_name, '张三', 'setUserInfo 后 get 返回正确 display_name');
assertEqual(got.role, 'student', 'setUserInfo 后 get 返回正确 role');

// 清除后为空
storage.clearUserInfo();
assertEqual(storage.getUserInfo(), null, 'clearUserInfo 后 get 返回 null');

// ── 隔离性 ──
console.log('隔离性');
storage.setToken('token-A');
storage.setUserInfo({ name: 'UserA' });
// Token 和 UserInfo 互不干扰
assertEqual(storage.getToken(), 'token-A', 'token 独立存储');
assertEqual(storage.getUserInfo().name, 'UserA', 'userInfo 独立存储');

storage.clearToken();
assertEqual(storage.getToken(), null, 'clearToken 不影响 userInfo');
assertEqual(storage.getUserInfo().name, 'UserA', 'clearToken 后 userInfo 仍存在');

// 清理
storage.clearUserInfo();

// ── 结果 ──
console.log('\n=== storage.js: ' + passed + '/' + (passed + failed) + ' 通过 ===');
if (failed > 0) {
  process.exit(1);
}
