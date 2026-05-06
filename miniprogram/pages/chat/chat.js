// 蔚小芯 — 问答对话页
var api = require('../../utils/api');
var helpers = require('../../utils/helpers');
var app = getApp();

Page({
  data: {
    messages: [],
    inputValue: '',
    sending: false,
    sessionId: '',
    scrollToView: '',
    showEmpty: true
  },

  onLoad: function () {
    if (!helpers.requireAuth(app)) return;
    this.loadSessions();
  },

  loadSessions: function () {
    var that = this;
    api.getSessions().then(function (res) {
      if (res.code === 0 && res.data && res.data.length > 0) {
        var latestSession = res.data[0];
        that.setData({ sessionId: latestSession.session_id });
        that.loadMessages(latestSession.session_id);
      }
    }).catch(function () {});
  },

  loadMessages: function (sessionId) {
    var that = this;
    api.getSessionMessages(sessionId).then(function (res) {
      if (res.code === 0 && res.data) {
        var msgs = res.data.map(function (m) {
          return {
            id: m.id,
            role: m.role,
            content: m.content,
            answerCard: m.answer_card || null,
            time: helpers.formatTime(m.created_at)
          };
        });
        that.setData({
          messages: msgs,
          showEmpty: msgs.length === 0,
          scrollToView: msgs.length > 0 ? 'msg-' + (msgs.length - 1) : ''
        });
      }
    }).catch(function () {});
  },

  onInput: function (e) {
    this.setData({ inputValue: e.detail.value });
  },

  sendMessage: function () {
    var that = this;
    var question = that.data.inputValue.trim();
    if (!question || that.data.sending) return;

    var userMsg = {
      id: 'u-' + Date.now(),
      role: 'user',
      content: question,
      answerCard: null,
      time: helpers.formatTime()
    };

    // 一次 setData：用户消息 + 思考中占位
    var messages = that.data.messages.concat([
      userMsg,
      {
        id: 't-' + Date.now(),
        role: 'assistant',
        content: '',
        thinking: true,
        answerCard: null,
        time: ''
      }
    ]);

    that.setData({
      messages: messages,
      inputValue: '',
      sending: true,
      showEmpty: false,
      scrollToView: 'msg-' + (messages.length - 1)
    });

    api.askQuestion(question, that.data.sessionId).then(function (res) {
      var msgs = _replaceThinking(that, res);
      that.setData({
        messages: msgs,
        sending: false,
        scrollToView: 'msg-' + (msgs.length - 1)
      });
    }).catch(function () {
      var msgs = _replaceThinking(that, null);
      that.setData({
        messages: msgs,
        sending: false,
        scrollToView: 'msg-' + (msgs.length - 1)
      });
    });
  },

  onFollowUp: function (e) {
    this.setData({ inputValue: e.detail.question });
    this.sendMessage();
  },

  newChat: function () {
    this.setData({
      messages: [],
      sessionId: '',
      showEmpty: true,
      scrollToView: ''
    });
  }
});

/** 替换思考中占位为真实回答（或错误兜底） */
function _replaceThinking(that, res) {
  var msgs = that.data.messages.slice();
  msgs.pop(); // 移除 thinking 占位

  if (res && res.code === 0 && res.data) {
    if (res.data.session_id) {
      that.setData({ sessionId: res.data.session_id });
    }
    msgs.push({
      id: 'a-' + Date.now(),
      role: 'assistant',
      content: res.data.answer || '',
      answerCard: res.data.answer_card || null,
      time: helpers.formatTime()
    });
  } else {
    msgs.push({
      id: 'a-' + Date.now(),
      role: 'assistant',
      content: '抱歉，暂时无法回答，请稍后再试。',
      answerCard: null,
      time: helpers.formatTime()
    });
  }
  return msgs;
}
