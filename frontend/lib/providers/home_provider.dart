import 'package:flutter/foundation.dart';

/// 首页仪表盘状态管理
///
/// 告警统计数据由 EmotionProvider 统一提供，避免重复状态。
/// HomeProvider 保留用于未来首页特有状态的扩展点。
class HomeProvider extends ChangeNotifier {
  // 首页特有状态在此扩展
}
