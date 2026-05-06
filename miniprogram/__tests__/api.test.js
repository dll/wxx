/**
 * 蔚小芯 — api.js 烟雾测试
 * 运行: node __tests__/api.test.js
 *
 * 注：api.js 依赖 getApp()（模块级）和 wx.request（运行时），
 * 测试前需 mock 全局对象。部分导出函数（如 request）依赖 Promise，
 * 仅测试不触发 HTTP 请求的路径。
 */

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

// Mock global wx APIs
global.wx = {
  request: function () {},
  reLaunch: function () {},
};

// Mock getApp（api.js 模块加载时调用 getApp().globalData.token）
global.getApp = function () {
  return {
    globalData: { token: 'mocked-jwt-token' },
    clearLoginState: function () {},
    setLoginState: function () {},
  };
};

// 加载 api 模块
var api = require('../utils/api');

// ── 模块导出检查 ──
console.log('\n模块导出');
assert(typeof api.usernameLogin === 'function', 'usernameLogin 已导出');
assert(typeof api.getProfile === 'function', 'getProfile 已导出');
assert(typeof api.askQuestion === 'function', 'askQuestion 已导出');
assert(typeof api.getSessions === 'function', 'getSessions 已导出');
assert(typeof api.getSessionMessages === 'function', 'getSessionMessages 已导出');
assert(typeof api.browseKnowledge === 'function', 'browseKnowledge 已导出');
assert(typeof api.getResource === 'function', 'getResource 已导出');
assert(typeof api.getRecommendations === 'function', 'getRecommendations 已导出');
assert(typeof api.deleteSession === 'function', 'deleteSession 已导出');
assert(typeof api._onTokenChange === 'function', '_onTokenChange 已导出');
assert(typeof api.BASE_URL === 'string', 'BASE_URL 已导出为字符串');
assert(api.BASE_URL.length > 0, 'BASE_URL 不为空');

// ── browseKnowledge URL 拼接 ──
console.log('browseKnowledge URL 拼接');

// 模拟 wx.request 拦截调用参数
var lastRequestOpts = null;
global.wx.request = function (opts) {
  lastRequestOpts = opts;
  opts.success({ statusCode: 200, data: { code: 0, message: 'ok', data: {} } });
};

// 无参数
api.browseKnowledge(null).then(function () {
  assert(lastRequestOpts !== null, 'browseKnowledge 发起了请求');
  assertEqual(lastRequestOpts.url.lastIndexOf('/knowledge'), lastRequestOpts.url.length - '/knowledge'.length, 'URL 以 /knowledge 结尾（无参数时无 ?）');

  // 带 resource_type
  return api.browseKnowledge({ resource_type: 'Policy' });
}).then(function () {
  assert(lastRequestOpts.url.indexOf('resource_type=Policy') !== -1, '包含 resource_type=Policy');

  // 带 page 和 page_size
  return api.browseKnowledge({ resource_type: 'FAQ', page: 1, page_size: 10 });
}).then(function () {
  assert(lastRequestOpts.url.indexOf('resource_type=FAQ') !== -1, '包含 resource_type=FAQ');
  assert(lastRequestOpts.url.indexOf('page=1') !== -1, '包含 page=1');
  assert(lastRequestOpts.url.indexOf('page_size=10') !== -1, '包含 page_size=10');

  // page 为 0（falsy 但 !== undefined）
  return api.browseKnowledge({ page: 0 });
}).then(function () {
  assert(lastRequestOpts.url.indexOf('page=0') !== -1, 'page=0 应包含在 URL 中');

  // ── 结果 ──
  console.log('\n=== api.js: ' + passed + '/' + (passed + failed) + ' 通过 ===');
  if (failed > 0) {
    process.exit(1);
  }
}).catch(function (err) {
  failed++;
  console.error('  FAIL: Promise rejected: ' + err.message);
  process.exit(1);
});
