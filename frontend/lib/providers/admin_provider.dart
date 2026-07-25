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

  // ── 数据仪表盘 ──
  DashboardStats? _dashboard;
  bool _dashboardLoading = false;

  DashboardStats? get dashboard => _dashboard;
  bool get dashboardLoading => _dashboardLoading;

  // ── 用户管理 ──
  final List<UserProfile> _users = [];
  bool _usersLoading = false;
  int _userPage = 1;
  int _userTotal = 0;
  int _userPageSize = 50;

  // 筛选条件
  String _keyword = '';
  String _userRoleFilter = '';
  String _userScopeFilter = '';
  String _collegeFilter = '';
  String _majorFilter = '';
  String _classFilter = '';
  String _enrollmentYearFilter = '';
  String _statusFilter = '';

  // 字典值（用于下拉筛选）
  List<String> _collegeList = [];
  List<String> _majorList = [];
  List<String> _classList = [];
  List<String> _enrollmentYearList = [];

  // 批量选择
  final Set<int> _selectedUserIds = {};

  List<UserProfile> get users => _users;
  bool get usersLoading => _usersLoading;
  int get userTotal => _userTotal;
  int get userPage => _userPage;
  int get userPageSize => _userPageSize;
  String get keyword => _keyword;
  String get userRoleFilter => _userRoleFilter;
  String get userScopeFilter => _userScopeFilter;
  String get collegeFilter => _collegeFilter;
  String get majorFilter => _majorFilter;
  String get classFilter => _classFilter;
  String get enrollmentYearFilter => _enrollmentYearFilter;
  String get statusFilter => _statusFilter;
  List<String> get collegeList => _collegeList;
  List<String> get majorList => _majorList;
  List<String> get classList => _classList;
  List<String> get enrollmentYearList => _enrollmentYearList;
  Set<int> get selectedUserIds => _selectedUserIds;
  int get selectedCount => _selectedUserIds.length;

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

  /// 退出登录时重置全部内存态，防止跨账号泄露（Q-08）
  void reset() {
    _metrics = null;
    _metricsLoading = false;
    _dashboard = null;
    _dashboardLoading = false;
    _users.clear();
    _usersLoading = false;
    _userPage = 1;
    _userTotal = 0;
    _keyword = '';
    _userRoleFilter = '';
    _userScopeFilter = '';
    _collegeFilter = '';
    _majorFilter = '';
    _classFilter = '';
    _enrollmentYearFilter = '';
    _statusFilter = '';
    _collegeList = [];
    _majorList = [];
    _classList = [];
    _enrollmentYearList = [];
    _selectedUserIds.clear();
    _auditLogs.clear();
    _auditLoading = false;
    _auditPage = 1;
    _auditTotal = 0;
    _settings.clear();
    _settingsLoading = false;
    _error = '';
    notifyListeners();
  }

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

  // ── 数据仪表盘 ──

  Future<void> fetchDashboard() async {
    if (_dashboardLoading) return;
    _dashboardLoading = true;
    _error = '';
    notifyListeners();

    try {
      final response = await _api.get(ApiConfig.adminDashboard);
      if (response.data['code'] == 0 && response.data['data'] != null) {
        _dashboard = DashboardStats.fromJson(
            response.data['data'] as Map<String, dynamic>);
      }
    } catch (e) {
      _error = '获取仪表盘数据失败: $e';
    } finally {
      _dashboardLoading = false;
      notifyListeners();
    }
  }

  // ── 用户管理 ──

  /// 搜索用户（高级查询，分页刷新）
  Future<void> searchUsers({bool refresh = false}) async {
    if (_usersLoading && !refresh) return;
    if (refresh) {
      _userPage = 1;
      _users.clear();
      _selectedUserIds.clear();
    }
    _error = '';
    _usersLoading = true;
    notifyListeners();

    try {
      final params = <String, dynamic>{
        'page': _userPage,
        'page_size': _userPageSize,
      };
      if (_keyword.isNotEmpty) params['keyword'] = _keyword;
      if (_userRoleFilter.isNotEmpty) params['role'] = _userRoleFilter;
      if (_userScopeFilter.isNotEmpty) params['owner_scope'] = _userScopeFilter;
      if (_collegeFilter.isNotEmpty) params['college'] = _collegeFilter;
      if (_majorFilter.isNotEmpty) params['major'] = _majorFilter;
      if (_classFilter.isNotEmpty) params['class_name'] = _classFilter;
      if (_enrollmentYearFilter.isNotEmpty) {
        params['enrollment_year'] = _enrollmentYearFilter;
      }
      if (_statusFilter.isNotEmpty) params['status'] = _statusFilter;

      final response =
          await _api.get(ApiConfig.adminUsersAdvanced, params: params);
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

  /// 旧方法兼容：直接调用 searchUsers
  Future<void> fetchUsers({bool refresh = false}) async {
    await searchUsers(refresh: refresh);
  }

  /// 设置搜索关键词
  void setKeyword(String value) {
    _keyword = value.trim();
    searchUsers(refresh: true);
  }

  /// 设置筛选条件（变更后自动刷新）
  void setUserFilter({
    String? role,
    String? scope,
    String? college,
    String? major,
    String? className,
    String? enrollmentYear,
    String? status,
    int? pageSize,
  }) {
    if (role != null) _userRoleFilter = role;
    if (scope != null) _userScopeFilter = scope;
    if (college != null) _collegeFilter = college;
    if (major != null) _majorFilter = major;
    if (className != null) _classFilter = className;
    if (enrollmentYear != null) _enrollmentYearFilter = enrollmentYear;
    if (status != null) _statusFilter = status;
    if (pageSize != null) _userPageSize = pageSize;
    searchUsers(refresh: true);
  }

  /// 重置所有筛选条件
  void resetFilters() {
    _keyword = '';
    _userRoleFilter = '';
    _userScopeFilter = '';
    _collegeFilter = '';
    _majorFilter = '';
    _classFilter = '';
    _enrollmentYearFilter = '';
    _statusFilter = '';
    searchUsers(refresh: true);
  }

  /// 获取字典值（学院/专业/班级/入学年份）
  Future<List<String>> fetchDictValues(String column) async {
    try {
      final params = <String, dynamic>{'column': column};
      if (_userRoleFilter.isNotEmpty) params['role'] = _userRoleFilter;
      final response =
          await _api.get(ApiConfig.adminUsersDict, params: params);
      if (response.data['code'] == 0 && response.data['data'] != null) {
        final list = (response.data['data'] as List)
            .map((e) => e.toString())
            .toList();
        switch (column) {
          case 'college':
            _collegeList = list;
            break;
          case 'major':
            _majorList = list;
            break;
          case 'class_name':
            _classList = list;
            break;
          case 'enrollment_year':
            _enrollmentYearList = list;
            break;
        }
        notifyListeners();
        return list;
      }
      return [];
    } catch (e) {
      _error = '获取字典值失败: $e';
      return [];
    }
  }

  // ── 批量选择 ──

  void toggleSelect(int userId) {
    if (_selectedUserIds.contains(userId)) {
      _selectedUserIds.remove(userId);
    } else {
      _selectedUserIds.add(userId);
    }
    notifyListeners();
  }

  void selectAllVisible() {
    for (final u in _users) {
      _selectedUserIds.add(u.id);
    }
    notifyListeners();
  }

  void deselectAll() {
    _selectedUserIds.clear();
    notifyListeners();
  }

  // ── 批量操作 ──

  Future<bool> batchUpdateStatus(String status) async {
    if (_selectedUserIds.isEmpty) return false;
    try {
      final response = await _api.post(
        ApiConfig.adminUsersBatchStatus,
        data: {
          'ids': _selectedUserIds.toList(),
          'status': status,
        },
      );
      if (response.data['code'] == 0) {
        _selectedUserIds.clear();
        await searchUsers(refresh: true);
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

  Future<bool> batchResetPassword(String newPassword) async {
    if (_selectedUserIds.isEmpty) return false;
    try {
      final response = await _api.post(
        ApiConfig.adminUsersBatchPassword,
        data: {
          'ids': _selectedUserIds.toList(),
          'new_password': newPassword,
        },
      );
      if (response.data['code'] == 0) {
        _selectedUserIds.clear();
        return true;
      }
      _error = response.data['message'] ?? '重置失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  Future<bool> batchDelete() async {
    if (_selectedUserIds.isEmpty) return false;
    try {
      final response = await _api.post(
        ApiConfig.adminUsersBatchDelete,
        data: {'ids': _selectedUserIds.toList()},
      );
      if (response.data['code'] == 0) {
        _selectedUserIds.clear();
        await searchUsers(refresh: true);
        return true;
      }
      _error = response.data['message'] ?? '删除失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  Future<bool> updateUser(int id, Map<String, dynamic> data) async {
    try {
      final response =
          await _api.put(ApiConfig.adminUserUpdate(id.toString()), data: data);
      if (response.data['code'] == 0) {
        await searchUsers(refresh: true);
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

  /// 删除用户
  Future<bool> deleteUser(int id) async {
    try {
      final response =
          await _api.delete(ApiConfig.adminUserDelete(id.toString()));
      if (response.data['code'] == 0) {
        await searchUsers(refresh: true);
        return true;
      }
      _error = response.data['message'] ?? '删除失败';
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
                ?.map((e) => SystemSetting.fromJson(e as Map<String, dynamic>))
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
      final response =
          await _api.put(ApiConfig.adminSettings, data: {'settings': settings});
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

  /// 管理员重置用户密码（仅 sys_admin）
  Future<bool> resetUserPassword(int userId, String newPassword) async {
    try {
      final resp = await _api.put(ApiConfig.resetPassword(userId), data: {
        'password': newPassword,
      });
      if (resp.data['code'] == 0) return true;
      _error = resp.data['message'] ?? '重置密码失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  // ── 学生导入 ──

  ImportResultData? _importResult;
  bool _importing = false;

  ImportResultData? get importResult => _importResult;
  bool get importing => _importing;

  /// 从 xlsx 批量导入学生
  Future<bool> importStudents(List<int> fileBytes, String filename,
      {String? defaultPassword}) async {
    _importing = true;
    _error = '';
    notifyListeners();

    try {
      final response = await _api.uploadBytes(
        ApiConfig.importStudents,
        bytes: fileBytes,
        filename: filename,
        fieldName: 'file',
        fields: {
          if (defaultPassword != null && defaultPassword.isNotEmpty)
            'default_password': defaultPassword,
        },
      );
      if (response.data['code'] == 0 && response.data['data'] != null) {
        _importResult = ImportResultData.fromJson(
            response.data['data'] as Map<String, dynamic>);
        return true;
      }
      _error = response.data['message'] ?? '导入失败';
      return false;
    } catch (e) {
      _error = '导入失败: ' + e.toString();
      return false;
    } finally {
      _importing = false;
      notifyListeners();
    }
  }

  void clearImportResult() {
    _importResult = null;
    notifyListeners();
  }
}
