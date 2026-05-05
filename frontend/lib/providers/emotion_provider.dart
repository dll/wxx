import 'package:flutter/foundation.dart';
import '../models/models.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';

/// 情感预警状态管理
class EmotionProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  final List<EmotionLog> _alerts = [];
  bool _loading = false;
  String _error = '';
  String _riskFilter = '';
  String _statusFilter = '';
  int _page = 1;
  int _total = 0;
  bool _hasMore = true;

  EmotionStats? _stats;
  bool _statsLoading = false;

  // ── Getters ──

  List<EmotionLog> get alerts => _alerts;
  bool get loading => _loading;
  String get error => _error;
  String get riskFilter => _riskFilter;
  String get statusFilter => _statusFilter;
  int get total => _total;
  bool get hasMore => _hasMore;
  EmotionStats? get stats => _stats;
  bool get statsLoading => _statsLoading;

  // ── 过滤 ──

  void setRiskFilter(String level) {
    _riskFilter = level;
    _page = 1;
    _alerts.clear();
    _hasMore = true;
    notifyListeners();
    loadAlerts();
  }

  void setStatusFilter(String status) {
    _statusFilter = status;
    _page = 1;
    _alerts.clear();
    _hasMore = true;
    notifyListeners();
    loadAlerts();
  }

  // ── 加载告警列表 ──

  Future<void> loadAlerts() async {
    if (_loading || !_hasMore) return;
    _loading = true;
    _error = '';
    notifyListeners();

    try {
      final params = <String, dynamic>{
        'page': _page,
        'page_size': 20,
      };
      if (_riskFilter.isNotEmpty) params['risk_level'] = _riskFilter;
      if (_statusFilter.isNotEmpty) params['status'] = _statusFilter;

      final response = await _api.get(ApiConfig.emotionAlerts, params: params);

      if (response.data['code'] == 0) {
        final List rawList = response.data['data'] ?? [];
        final newAlerts = rawList
            .map((e) => EmotionLog.fromJson(e as Map<String, dynamic>))
            .toList();
        _alerts.addAll(newAlerts);
        _total = response.data['total'] ?? 0;
        _hasMore = _alerts.length < _total;
        _page++;
      } else {
        _error = response.data['message'] ?? '加载失败';
      }
    } catch (e) {
      _error = '网络错误: $e';
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// 下拉刷新
  Future<void> refresh() async {
    _page = 1;
    _alerts.clear();
    _hasMore = true;
    await Future.wait([loadAlerts(), fetchStats()]);
  }

  // ── 加载统计数据 ──

  Future<void> fetchStats() async {
    if (_statsLoading) return;
    _statsLoading = true;
    notifyListeners();

    try {
      final response = await _api.get(ApiConfig.emotionStats);
      if (response.data['code'] == 0 && response.data['data'] != null) {
        _stats = EmotionStats.fromJson(
            response.data['data'] as Map<String, dynamic>);
      }
    } catch (_) {
      // 静默失败
    } finally {
      _statsLoading = false;
      notifyListeners();
    }
  }

  // ── 更新告警状态 ──

  Future<bool> updateAlertStatus(String alertId, String status) async {
    try {
      final response = await _api.put(
        ApiConfig.emotionAlertUpdate(alertId),
        data: EmotionUpdateRequest(status: status).toJson(),
      );

      if (response.data['code'] == 0) {
        // 更新本地缓存
        final idx = _alerts.indexWhere((a) => a.alertId == alertId);
        if (idx >= 0) {
          final updated = EmotionLog.fromJson(
              response.data['data'] as Map<String, dynamic>);
          _alerts[idx] = updated;
          notifyListeners();
        }
        return true;
      }
      _error = response.data['message'] ?? '操作失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  // ── 单条分析（手动触发） ──

  Future<EmotionLog?> analyze(String messageText, String sessionId) async {
    try {
      final response = await _api.post(
        ApiConfig.emotionAnalyze,
        data: EmotionAnalyzeRequest(
          messageText: messageText,
          sessionId: sessionId,
        ).toJson(),
      );

      if (response.data['code'] == 0) {
        return EmotionLog.fromJson(
            response.data['data'] as Map<String, dynamic>);
      }
      _error = response.data['message'] ?? '分析失败';
      notifyListeners();
      return null;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return null;
    }
  }
}
