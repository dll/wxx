import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../config/router.dart';
import '../models/models.dart';
import '../services/api_service.dart';
import '../utils/storage.dart';

/// 认证状态管理
class AuthProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String? _error;
  UserProfile? _profile;

  bool get loading => _loading;
  String? get error => _error;
  UserProfile? get profile => _profile;
  bool get isLoggedIn => Storage.isLoggedIn;

  AuthProvider() {
    // 设置 401 回调
    _api.onUnauthorized = _handleUnauthorized;
  }

  /// 登录
  Future<bool> login(String username, String password) async {
    _loading = true;
    _error = null;
    notifyListeners();

    try {
      final resp = await _api.post(ApiConfig.login, data: {
        'username': username,
        'password': password,
      });

      final data = resp.data;
      if (data['code'] != 0) {
        _error = data['message'] ?? '登录失败';
        _loading = false;
        notifyListeners();
        return false;
      }

      // 存储 Token
      final token = data['data']?['token'] ?? '';
      await Storage.setToken(token);

      // 获取用户信息
      await fetchProfile();

      _loading = false;
      notifyListeners();
      return true;
    } catch (e) {
      _error = '网络错误: ${e.toString()}';
      _loading = false;
      notifyListeners();
      return false;
    }
  }

  /// 获取用户资料
  Future<void> fetchProfile() async {
    _loading = true;
    _error = null;
    notifyListeners();

    try {
      final resp = await _api.get(ApiConfig.profile);
      _profile = UserProfile.fromJson(resp.data);
      if (_profile != null) {
        await Storage.setUserInfo(
          username: _profile!.username,
          role: _profile!.role,
          displayName: _profile!.displayName,
        );
      }
      _loading = false;
      notifyListeners();
    } catch (e) {
      _error = '加载用户信息失败';
      _loading = false;
      // 静默失败，使用缓存的用户信息
      final name = Storage.displayName;
      if (name != null) {
        _profile = UserProfile(
          id: 0,
          username: Storage.username ?? '',
          role: Storage.role ?? 'student',
          displayName: name,
        );
      }
      notifyListeners();
    }
  }

  /// 退出登录
  Future<void> logout() async {
    await Storage.clearAll();
    _profile = null;
    _error = null;
    notifyListeners();
  }

  void _handleUnauthorized() {
    logout();
    // 通知 GoRouter 重新评估鉴权状态，自动跳转登录页
    authRefreshNotifier.refresh();
  }
}
