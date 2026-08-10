import 'package:flutter/foundation.dart';

import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 第三方应用中心 Provider — 拉取当前用户可见的应用列表
class AppCenterProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String _error = '';
  List<ExternalAppItem> _apps = const [];

  bool get loading => _loading;
  String get error => _error;
  List<ExternalAppItem> get apps => _apps;

  /// 按 manifest.category 分组，保持 category 展示顺序
  Map<String, List<ExternalAppItem>> get grouped {
    final map = <String, List<ExternalAppItem>>{};
    for (final app in _apps) {
      map.putIfAbsent(app.category, () => []).add(app);
    }
    return map;
  }

  Future<void> fetchApps() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.externalApps);
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map ? (res.data as Map)['data'] : res.data;
        if (data is List) {
          _apps = data
              .whereType<Map>()
              .map((e) => ExternalAppItem.fromJson(Map<String, dynamic>.from(e)))
              .toList();
        }
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }
}