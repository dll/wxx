// 蔚小芯 — 个人中心页
var helpers = require('../../utils/helpers');
var app = getApp();

Page({
  data: {
    userInfo: null,
    roleLabel: ''
  },

  onShow: function () {
    if (!helpers.requireAuth(app)) return;

    var userInfo = app.globalData.userInfo;
    this.setData({
      userInfo: userInfo,
      roleLabel: helpers.roleLabel(userInfo.role)
    });
  },

  handleLogout: function () {
    wx.showModal({
      title: '确认退出',
      content: '退出后需要重新登录',
      success: function (res) {
        if (res.confirm) {
          app.clearLoginState();
          wx.reLaunch({ url: '/pages/login/login' });
        }
      }
    });
  },

  handleCopyId: function () {
    if (this.data.userInfo && this.data.userInfo.user_id) {
      wx.setClipboardData({
        data: this.data.userInfo.user_id,
        success: function () {
          wx.showToast({ title: '已复制', icon: 'success' });
        }
      });
    }
  }
});
