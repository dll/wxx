// 蔚小芯 — AnswerCard 自定义组件
// 用于渲染标准 AnswerCard JSON（结论 / 步骤 / 来源 / 风险 / 追问 / 置信度）
Component({
  properties: {
    // 接收 AnswerCard JSON 字符串或对象
    answerCard: {
      type: Object,
      value: null,
      observer: '_onCardChange'
    }
  },

  data: {
    card: null,
    confidenceLabel: '',
    confidenceClass: '',
    hasSources: false,
    hasFollowUps: false,
    hasSteps: false,
    hasRisks: false
  },

  methods: {
    _onCardChange: function (val) {
      if (!val) {
        this.setData({ card: null });
        return;
      }

      // 解析 confidence 标签
      var conf = val.confidence;
      var confLabel = '';
      var confClass = '';
      if (conf === undefined || conf === null) {
        // 无置信度，不显示
      } else if (typeof conf === 'number') {
        if (conf >= 0.8) {
          confLabel = '高置信度';
          confClass = 'tag-green';
        } else if (conf >= 0.5) {
          confLabel = '中置信度';
          confClass = 'tag-orange';
        } else {
          confLabel = '低置信度';
          confClass = 'tag-red';
        }
      }

      this.setData({
        card: val,
        confidenceLabel: confLabel,
        confidenceClass: confClass,
        hasSources: val.sources && val.sources.length > 0,
        hasFollowUps: val.follow_ups && val.follow_ups.length > 0,
        hasSteps: val.steps && val.steps.length > 0,
        hasRisks: val.risks && val.risks.length > 0
      });
    },

    // 点击追问建议，触发事件向上传递
    _onFollowUpTap: function (e) {
      var question = e.currentTarget.dataset.question;
      this.triggerEvent('followup', { question: question });
    },

    // 点击来源链接
    _onSourceTap: function (e) {
      var link = e.currentTarget.dataset.link;
      if (link) {
        wx.setClipboardData({
          data: link,
          success: function () {
            wx.showToast({ title: '链接已复制', icon: 'success' });
          }
        });
      }
    }
  }
});
