import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 首页仪表盘状态管理
class HomeProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  EmotionStats? _stats;
  bool _statsLoading = false;
  String? _error;

  EmotionStats? get stats => _stats;
  bool get statsLoading => _statsLoading;
  String? get error => _error;

  /// 是否有未处理的高优先级告警（辅导员及以上可见）
  bool get hasUrgentAlerts => _stats != null && (_stats!.urgent > 0 || _stats!.high > 0);

  /// 加载告警统计（按角色自动过滤范围）
  Future<void> loadStats() async {
    if (_statsLoading) return;
    _statsLoading = true;
    notifyListeners();

    try {
      final resp = await _api.get(ApiConfig.emotionStats);
      if (resp.data['code'] == 0 && resp.data['data'] != null) {
        _stats = EmotionStats.fromJson(resp.data['data']);
      }
    } catch (_) {
      // 静默失败，学生角色可能无权限
    } finally {
      _statsLoading = false;
      notifyListeners();
    }
  }
}
