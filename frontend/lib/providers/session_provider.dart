import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 会话列表状态管理
class SessionProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  List<Session> _sessions = [];
  bool _loading = false;
  String? _error;

  List<Session> get sessions => List.unmodifiable(_sessions);
  bool get loading => _loading;
  String? get error => _error;

  /// 加载会话列表
  Future<void> fetchSessions() async {
    _loading = true;
    _error = null;
    notifyListeners();

    try {
      final resp = await _api.get(ApiConfig.sessions);
      final data = resp.data;
      final list = data['data'] as List? ?? [];

      _sessions = list.map((s) => Session.fromJson(s)).toList();
      _loading = false;
      notifyListeners();
    } catch (e) {
      _error = '加载会话列表失败';
      _loading = false;
      notifyListeners();
    }
  }

  /// 删除会话
  Future<void> deleteSession(String id) async {
    try {
      await _api.delete(ApiConfig.sessionDelete(id));
      _sessions.removeWhere((s) => s.id == id);
      notifyListeners();
    } catch (e) {
      _error = '删除会话失败';
      notifyListeners();
    }
  }
}
