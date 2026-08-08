// 语音指令导航
// 从 ASR 识别文本中匹配导航指令，返回对应的路由路径。
// 匹配逻辑：对识别文本做分词，计算每类指令的关键词命中率，取最高且超过阈值的结果。

class VoiceNavigator {
  VoiceNavigator._();

  /// 关键词命中阈值：至少命中该类 40% 的关键词才触发导航
  static const double _threshold = 0.4;

  /// 指令集：{路由路径: 关键词列表}
  /// 覆盖系统主菜单（首页/对话/知识/办事/我的/帮助）与常用功能页。
  static const Map<String, List<String>> _commands = {
    '/home': ['首页', '主页', '回首页', '开始', '欢迎'],
    '/chat': ['对话', '聊天', '问问题', '问答', '提问', '说话', 'AI', '智能'],
    '/browse': ['知识', '知识大厅', '知识库', '浏览', '资讯'],
    '/enrollment': ['办事', '报到', '入学', '离校', '流程', '指南', '手续'],
    '/profile': ['我的', '个人', '资料', '设置', '个人信息', '个人中心'],
    '/help': ['帮助', '帮助中心', '使用说明', '操作指南', '怎么用'],
    '/sessions': ['历史', '历史记录', '会话', '聊天记录', '查看历史'],
    '/my-feedbacks': ['反馈', '我的反馈', '意见', '投诉'],
    '/notifications': ['通知', '消息', '提醒', '公告'],
    '/bookmarks': ['收藏', '书签', '收藏夹'],
    '/about': ['关于', '关于我们', '版本'],
    '/campus': ['地图', '校园地图', '导航', '报到地图'],
    '/process-manage': ['办事流程', '流程管理', '办事管理', '流程'],
    '/student/daily-briefing': ['日报', '今日', '速览', '每日'],
    '/student/learning-diary': ['日记', '学习日记', '周记'],
    '/student/digital-twin': ['画像', '数字孪生', '孪生', '模型'],
    '/graduation': ['毕业', '毕设', '就业', '毕业设计'],
    '/competition': ['竞赛', '比赛', '学科竞赛'],
    '/student/study': ['学习', '学业', '课程'],
    '/student/mental': ['心理', '心情', '咨询'],
    '/login': ['退出', '注销', '登出', '退出登录'],
  };

  /// 尝试从识别文本中匹配导航指令
  ///
  /// 返回匹配到的路由路径；未匹配到或低于阈值返回 null
  static String? matchCommand(String text) {
    if (text.isEmpty) return null;

    final trimmed = text.trim();

    // 计算每条路由的关键词命中率
    String? bestRoute;
    double bestScore = 0;

    for (final entry in _commands.entries) {
      final route = entry.key;
      final keywords = entry.value;

      int hits = 0;
      for (final keyword in keywords) {
        if (trimmed.contains(keyword)) {
          hits++;
        }
      }

      final score = hits / keywords.length;
      if (score > bestScore) {
        bestScore = score;
        bestRoute = route;
      }
    }

    // 达到阈值才触发导航
    if (bestScore >= _threshold && bestRoute != null) {
      return bestRoute;
    }

    return null;
  }
}
