/// 语音指令导航
///
/// 从 ASR 识别文本中匹配导航指令，返回对应的路由路径。
/// 匹配逻辑：对识别文本做分词，计算每类指令的关键词命中率，取最高且超过阈值的结果。

class VoiceNavigator {
  VoiceNavigator._();

  /// 关键词命中阈值：至少命中该类 40% 的关键词才触发导航
  static const double _threshold = 0.4;

  /// 指令集：{路由路径: 关键词列表}
  static const Map<String, List<String>> _commands = {
    '/chat': ['首页', '主页', '对话', '聊天', '问问题', '说话'],
    '/sessions': ['历史', '历史记录', '会话', '聊天记录', '查看历史'],
    '/enrollment': ['办事', '入学', '离校', '流程', '指南', '手续'],
    '/profile': ['我的', '个人', '资料', '设置', '个人信息'],
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
