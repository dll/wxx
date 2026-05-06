// 蔚小芯 — 知识大厅页
var api = require('../../utils/api');
var helpers = require('../../utils/helpers');
var app = getApp();

var MAX_ITEMS = 200; // 无限滚动上限，防止内存膨胀

Page({
  data: {
    tabs: [
      { key: '', label: '全部' },
      { key: 'Policy', label: '政策' },
      { key: 'Process', label: '流程' },
      { key: 'FAQ', label: 'FAQ' },
      { key: 'Activity', label: '活动' }
    ],
    activeTab: '',
    resources: [],
    loading: false,
    errorMsg: '',
    page: 1,
    hasMore: true,
    _fetchLock: false  // 防并发重复请求
  },

  onLoad: function () {
    if (!helpers.requireAuth(app)) return;
    this.fetchResources();
  },

  onTabChange: function (e) {
    var tab = e.currentTarget.dataset.tab;
    if (tab === this.data.activeTab) return; // 防抖：同 tab 忽略
    this.setData({ activeTab: tab, resources: [], page: 1, hasMore: true, _fetchLock: false });
    this.fetchResources();
  },

  fetchResources: function () {
    var that = this;
    if (that.data.loading || !that.data.hasMore || that.data._fetchLock) return;

    that.setData({ loading: true, errorMsg: '', _fetchLock: true });

    api.browseKnowledge({
      resource_type: that.data.activeTab || undefined,
      page: that.data.page,
      page_size: 20
    }).then(function (res) {
      if (res.code === 0 && res.data) {
        var combined = that.data.resources.concat(res.data);
        if (combined.length > MAX_ITEMS) {
          combined = combined.slice(-MAX_ITEMS); // 保留最新
        }
        that.setData({
          resources: combined,
          page: that.data.page + 1,
          hasMore: res.data.length >= 20 && combined.length < MAX_ITEMS,
          loading: false,
          _fetchLock: false
        });
      } else {
        that.setData({ loading: false, errorMsg: res.message || '加载失败', _fetchLock: false });
      }
    }).catch(function (err) {
      that.setData({ loading: false, errorMsg: err.message || '网络异常', _fetchLock: false });
    });
  },

  onPullDownRefresh: function () {
    this.setData({ resources: [], page: 1, hasMore: true, _fetchLock: false });
    this.fetchResources();
    wx.stopPullDownRefresh();
  },

  onReachBottom: function () {
    this.fetchResources();
  },

  onItemTap: function (e) {
    var id = e.currentTarget.dataset.id;
    var resources = this.data.resources.map(function (item) {
      if (item.id === id) {
        item.expanded = !item.expanded;
      }
      return item;
    });
    this.setData({ resources: resources });
  }
});
