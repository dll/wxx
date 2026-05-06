// 蔚小芯 — 共享工具函数

/**
 * 登录守卫：未登录时跳转登录页，返回 false
 * 用法：在 Page.onLoad 中调用 if (!requireAuth(app)) return;
 */
function requireAuth(app) {
  if (!app.globalData.isLoggedIn) {
    wx.reLaunch({ url: '/pages/login/login' });
    return false;
  }
  return true;
}

/** 数字补零 */
function pad(n) {
  return n < 10 ? '0' + n : '' + n;
}

/** 格式化时间（ISO 字符串 → HH:MM） */
function formatTime(isoStr) {
  if (!isoStr) {
    var d = new Date();
    return pad(d.getHours()) + ':' + pad(d.getMinutes());
  }
  try {
    var d = new Date(isoStr.replace(' ', 'T') + '+08:00');
    if (isNaN(d.getTime())) return isoStr.substring(0, 5);
    return pad(d.getHours()) + ':' + pad(d.getMinutes());
  } catch (e) {
    return isoStr.substring(0, 5);
  }
}

/** 角色映射为中文标签 */
function roleLabel(role) {
  var map = {
    'sys_admin': '系统管理员',
    'school_admin': '学校管理员',
    'college_admin': '学院管理员',
    'counselor': '辅导员',
    'teacher': '教师',
    'assistant': '教辅',
    'student_union': '学生会',
    'student': '学生'
  };
  return map[role] || role || '未知角色';
}

module.exports = {
  requireAuth,
  pad,
  formatTime,
  roleLabel
};
