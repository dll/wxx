// 蔚小芯 — 流程指南页（入学 / 离校）
var api = require('../../utils/api');
var helpers = require('../../utils/helpers');
var app = getApp();

Page({
  data: {
    tabs: [
      { key: '入学', label: '🎓 入学指南' },
      { key: '离校', label: '🏁 离校指南' }
    ],
    activeTab: '入学',
    resources: [],
    loading: false,
    errorMsg: ''
  },

  onLoad: function () {
    if (!helpers.requireAuth(app)) return;
    this.fetchProcesses();
  },

  onTabChange: function (e) {
    var tab = e.currentTarget.dataset.tab;
    if (tab === this.data.activeTab) return;
    this.setData({ activeTab: tab });
    this.fetchProcesses();
  },

  fetchProcesses: function () {
    var that = this;
    that.setData({ loading: true, errorMsg: '' });

    api.browseKnowledge({ resource_type: 'Process', page_size: 50 }).then(function (res) {
      if (res.code === 0 && res.data) {
        var keyword = that.data.activeTab;
        var filtered = res.data.filter(function (item) {
          return item.title.indexOf(keyword) > -1 ||
            (item.content && item.content.indexOf(keyword) > -1);
        });
        that.setData({ resources: filtered, loading: false });
      } else {
        that.setData({ loading: false, errorMsg: res.message || '加载失败' });
      }
    }).catch(function (err) {
      that.setData({ loading: false, errorMsg: err.message || '网络异常' });
    });
  },

  onItemTap: function (e) {
    var id = e.currentTarget.dataset.id;
    var that = this;

    // 先切换展开/收起
    var resources = this.data.resources.map(function (item) {
      if (item.id === id) {
        item.expanded = !item.expanded;
        if (item.expanded && !item.detailLoaded) {
          // 异步加载详情 — 加载完成后通过 setData 更新
          api.getResource(id).then(function (res) {
            if (res.code === 0 && res.data) {
              var detail = res.data;
              if (typeof detail.tags === 'string') {
                try { detail._tags = JSON.parse(detail.tags); } catch (e) { delete detail._tags; }
              }
              // 重新 map 并 setData，确保视图更新
              var updated = that.data.resources.map(function (r) {
                if (r.id === id) {
                  r.detail = detail;
                  r.detailLoaded = true;
                }
                return r;
              });
              that.setData({ resources: updated });
            }
          }).catch(function () {});
        }
      }
      return item;
    });
    this.setData({ resources: resources });
  },

  onPullDownRefresh: function () {
    this.fetchProcesses();
    wx.stopPullDownRefresh();
  }
});
