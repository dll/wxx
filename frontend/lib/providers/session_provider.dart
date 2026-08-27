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

      _sessions = list
          .whereType<Map>()
          .map((s) => Session.fromJson(Map<String, dynamic>.from(s)))
          .toList();
      _loading = false;
      notifyListeners();
    } catch (e) {
      _error = '加载会话列表失败';
      _loading = false;
      notifyListeners();
    }
  }

  /// 删除会话（乐观更新）。
  ///
  /// 必须先同步从列表移除再发请求：会话列表页使用 [Dismissible]，
  /// 若列表仍保留已滑出项，重建会触发 "dismissed Dismissible still in tree" 崩溃，
  /// 页面表现为空白/长时间无响应。移除后再调接口，失败时交由调用方刷新恢复。
  ///
  /// 返回 true 表示删除成功（或列表中已不存在该会话），false 表示 API 失败。
  /// 注意：失败时**不**在此同步回插被删项，否则会话列表页的 Dismissible 会把
  /// 已滑出项重新插回树中而崩溃（空白页）；调用方应调用 [fetchSessions] 以服务器为准恢复。
  Future<bool> deleteSession(String id) async {
    final idx = _sessions.indexWhere((s) => s.id == id);
    if (idx == -1) return true; // 已不在列表中，视为成功

    // 先同步移除并刷新，避免 Dismissible 出现「已 dismiss 的项仍在树中」断言崩溃。
    _sessions.removeAt(idx);
    notifyListeners();

    try {
      await _api.delete(ApiConfig.sessionDelete(id));
      return true;
    } catch (e) {
      _error = '删除会话失败';
      notifyListeners();
      return false;
    }
  }

  /// 重命名会话
  Future<bool> renameSession(String id, String title) async {
    try {
      await _api.patch(ApiConfig.sessionRename(id), data: {'title': title});
      // 本地更新（避免刷新整张列表）
      final idx = _sessions.indexWhere((s) => s.id == id);
      if (idx != -1) {
        final old = _sessions[idx];
        _sessions[idx] = Session(
          id: old.id,
          title: title,
          createdAt: old.createdAt,
          updatedAt: old.updatedAt,
        );
        notifyListeners();
      }
      return true;
    } catch (e) {
      _error = '重命名失败';
      notifyListeners();
      return false;
    }
  }
}
