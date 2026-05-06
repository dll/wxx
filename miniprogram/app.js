// 蔚小芯微信小程序 — 全局应用入口
var storage = require('./utils/storage');
var api = require('./utils/api');

App({
  globalData: {
    userInfo: null,
    token: null,
    isLoggedIn: false
  },

  onLaunch: function () {
    var token = storage.getToken();
    var userInfo = storage.getUserInfo();

    if (token && userInfo) {
      this.globalData.token = token;
      this.globalData.userInfo = userInfo;
      this.globalData.isLoggedIn = true;
      api._onTokenChange();
      console.log('已恢复登录状态:', userInfo.display_name);
    } else {
      console.log('未登录，跳转登录页');
    }
  },

  setLoginState: function (token, userInfo) {
    this.globalData.token = token;
    this.globalData.userInfo = userInfo;
    this.globalData.isLoggedIn = true;
    storage.setToken(token);
    storage.setUserInfo(userInfo);
    api._onTokenChange();
  },

  clearLoginState: function () {
    this.globalData.token = null;
    this.globalData.userInfo = null;
    this.globalData.isLoggedIn = false;
    storage.clearToken();
    storage.clearUserInfo();
    api._onTokenChange();
  }
});
