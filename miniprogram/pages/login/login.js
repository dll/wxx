// 蔚小芯 — 登录页（用户名登录）
var api = require('../../utils/api');
var app = getApp();

Page({
  data: {
    username: '',
    loading: false,
    errorMsg: ''
  },

  onLoad: function () {
    // 已登录 → 直接进问答页
    if (app.globalData.isLoggedIn) {
      wx.reLaunch({ url: '/pages/chat/chat' });
    }
  },

  onInput: function (e) {
    this.setData({ username: e.detail.value, errorMsg: '' });
  },

  handleLogin: function () {
    var that = this;
    var username = that.data.username.trim();
    if (!username) {
      that.setData({ errorMsg: '请输入用户名' });
      return;
    }

    that.setData({ loading: true, errorMsg: '' });

    api.usernameLogin(username).then(function (res) {
      if (res.code !== 0 || !res.data) {
        throw new Error(res.message || '登录失败');
      }
      var data = res.data;
      app.setLoginState(data.token, {
        display_name: data.display_name,
        role: data.role
      });
      that.setData({ loading: false });
      wx.reLaunch({ url: '/pages/chat/chat' });
    }).catch(function (err) {
      that.setData({
        loading: false,
        errorMsg: err.message || '登录失败，请检查网络'
      });
    });
  }
});
