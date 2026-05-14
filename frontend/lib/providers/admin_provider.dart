import 'package:flutter/foundation.dart';
import '../models/models.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';

/// 管理端状态管理
class AdminProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  // ── 质量看板 ──
  AdminMetrics? _metrics;
  bool _metricsLoading = false;

  AdminMetrics? get metrics => _metrics;
  bool get metricsLoading => _metricsLoading;

  // ── 用户管理 ──
  final List<UserProfile> _users = [];
  bool _usersLoading = false;
  int _userPage = 1;
  int _userTotal = 0;
  String _userRoleFilter = '';
  String _userScopeFilter = '';

  List<UserProfile> get users => _users;
  bool get usersLoading => _usersLoading;
  int get userTotal => _userTotal;
  int get userPage => _userPage;
  String get userRoleFilter => _userRoleFilter;
  String get userScopeFilter => _userScopeFilter;

  // ── 审计日志 ──
  final List<AuditLog> _auditLogs = [];
  bool _auditLoading = false;
  int _auditPage = 1;
  int _auditTotal = 0;

  List<AuditLog> get auditLogs => _auditLogs;
  bool get auditLoading => _auditLoading;
  int get auditTotal => _auditTotal;

  // ── 系统配置 ──
  final List<SystemSetting> _settings = [];
  bool _settingsLoading = false;

  List<SystemSetting> get settings => _settings;
  bool get settingsLoading => _settingsLoading;

  String _error = '';
  String get error => _error;

  // ── 质量看板 ──

  Future<void> fetchMetrics() async {
    if (_metricsLoading) return;
    _metricsLoading = true;
    notifyListeners();

    try {
      final response = await _api.get(ApiConfig.adminMetrics);
      if (response.data['code'] == 0 && response.data['data'] != null) {
        _metrics = AdminMetrics.fromJson(
            response.data['data'] as Map<String, dynamic>);
      }
    } catch (e) {
      _error = '获取看板数据失败: $e';
    } finally {
      _metricsLoading = false;
      notifyListeners();
    }
  }

  // ── 用户管理 ──

  Future<void> fetchUsers({bool refresh = false}) async {
    if (_usersLoading) return;
    if (refresh) {
      _userPage = 1;
      _users.clear();
    }
    _usersLoading = true;
    notifyListeners();

    try {
      final params = <String, dynamic>{
        'page': _userPage,
        'page_size': 20,
      };
      if (_userRoleFilter.isNotEmpty) params['role'] = _userRoleFilter;
      if (_userScopeFilter.isNotEmpty) params['owner_scope'] = _userScopeFilter;

      final response = await _api.get(ApiConfig.adminUsers, params: params);
      if (response.data['code'] == 0) {
        final list = (response.data['data'] as List?)
                ?.map((e) => UserProfile.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [];
        _users.addAll(list);
        _userTotal = response.data['total'] ?? 0;
        _userPage++;
      }
    } catch (e) {
      _error = '获取用户列表失败: $e';
    } finally {
      _usersLoading = false;
      notifyListeners();
    }
  }

  void setUserFilter({String? role, String? scope}) {
    _userRoleFilter = role ?? '';
    _userScopeFilter = scope ?? '';
    fetchUsers(refresh: true);
  }

  Future<bool> updateUser(int id, Map<String, dynamic> data) async {
    try {
      final response =
          await _api.put(ApiConfig.adminUserUpdate(id.toString()), data: data);
      if (response.data['code'] == 0) {
        await fetchUsers(refresh: true);
        return true;
      }
      _error = response.data['message'] ?? '更新失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  // ── 审计日志 ──

  Future<void> fetchAuditLogs({bool refresh = false}) async {
    if (_auditLoading) return;
    if (refresh) {
      _auditPage = 1;
      _auditLogs.clear();
    }
    _auditLoading = true;
    notifyListeners();

    try {
      final params = <String, dynamic>{
        'page': _auditPage,
        'page_size': 20,
      };
      final response = await _api.get(ApiConfig.adminAudit, params: params);
      if (response.data['code'] == 0) {
        final list = (response.data['data'] as List?)
                ?.map((e) => AuditLog.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [];
        _auditLogs.addAll(list);
        _auditTotal = response.data['total'] ?? 0;
        _auditPage++;
      }
    } catch (e) {
      _error = '获取审计日志失败: $e';
    } finally {
      _auditLoading = false;
      notifyListeners();
    }
  }

  // ── 系统配置 ──

  Future<void> fetchSettings() async {
    if (_settingsLoading) return;
    _settingsLoading = true;
    notifyListeners();

    try {
      final response = await _api.get(ApiConfig.adminSettings);
      if (response.data['code'] == 0 && response.data['data'] != null) {
        final list = (response.data['data'] as List?)
                ?.map(
                    (e) => SystemSetting.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [];
        _settings.clear();
        _settings.addAll(list);
      }
    } catch (e) {
      _error = '获取系统配置失败: $e';
    } finally {
      _settingsLoading = false;
      notifyListeners();
    }
  }

  Future<bool> updateSettings(Map<String, String> settings) async {
    try {
      final response = await _api.put(ApiConfig.adminSettings,
          data: {'settings': settings});
      if (response.data['code'] == 0) {
        await fetchSettings();
        return true;
      }
      _error = response.data['message'] ?? '更新失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }
}
