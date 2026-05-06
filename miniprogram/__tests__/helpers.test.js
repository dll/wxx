/**
 * 蔚小芯 — helpers.js 烟雾测试
 * 运行: node __tests__/helpers.test.js
 */

// Mock wx 全局
global.wx = {
  reLaunch: function () {},
};

var helpers = require('../utils/helpers');

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

// ── pad ──
console.log('\npad()');
assertEqual(helpers.pad(5), '05', 'pad(5) => 05');
assertEqual(helpers.pad(0), '00', 'pad(0) => 00');
assertEqual(helpers.pad(10), '10', 'pad(10) => 10');
assertEqual(helpers.pad(9), '09', 'pad(9) => 09');

// ── roleLabel ──
console.log('roleLabel()');
assertEqual(helpers.roleLabel('student'), '学生', 'student => 学生');
assertEqual(helpers.roleLabel('sys_admin'), '系统管理员', 'sys_admin => 系统管理员');
assertEqual(helpers.roleLabel('counselor'), '辅导员', 'counselor => 辅导员');
assertEqual(helpers.roleLabel('teacher'), '教师', 'teacher => 教师');
assertEqual(helpers.roleLabel('unknown_role'), 'unknown_role', '未知角色返回原值');
assertEqual(helpers.roleLabel(null), '未知角色', 'null => 未知角色');
assertEqual(helpers.roleLabel(undefined), '未知角色', 'undefined => 未知角色');
assertEqual(helpers.roleLabel(''), '未知角色', '空字符串 => 未知角色');

// ── formatTime ──
console.log('formatTime()');
// 参数为 null/undefined 时返回当前时间
var now = helpers.formatTime(null);
assert(typeof now === 'string' && now.length === 5, 'formatTime(null) 返回 HH:MM 格式');
assert(now.indexOf(':') === 2, 'formatTime(null) 包含冒号在位置 2');

// 无效日期回退到 substring(0,5)，不抛出异常
var invalid = helpers.formatTime('not-a-date');
assert(typeof invalid === 'string' && invalid.length > 0, 'formatTime(无效日期) 不抛出异常');
assertEqual(invalid, 'not-a', 'formatTime(无效日期) 回退到 substring(0,5)');

// ── requireAuth ──
console.log('requireAuth()');
var mockAppLoggedIn = { globalData: { isLoggedIn: true } };
assertEqual(helpers.requireAuth(mockAppLoggedIn), true, '已登录返回 true');

var mockAppLoggedOut = { globalData: { isLoggedIn: false } };
assertEqual(helpers.requireAuth(mockAppLoggedOut), false, '未登录返回 false');

// ── 结果 ──
console.log('\n=== helpers.js: ' + passed + '/' + (passed + failed) + ' 通过 ===');
if (failed > 0) {
  process.exit(1);
}
