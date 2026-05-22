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
  int? _voiceEnabled;

  bool get loading => _loading;
  String? get error => _error;
  UserProfile? get profile => _profile;
  bool get isLoggedIn => Storage.isLoggedIn;
  int? get voiceEnabled => _voiceEnabled;

  AuthProvider() {
    // 设置 401 回调
    _api.onUnauthorized = _handleUnauthorized;
  }

  /// 登录
  Future<bool> login(String username, String password, [String role = 'student']) async {
    _loading = true;
    _error = null;
    notifyListeners();

    try {
      final resp = await _api.post(ApiConfig.login, data: {
        'username': username,
        'password': password,
        'role': role,
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
      debugPrint('登录请求失败: $e');
      _error = '网络错误，请检查网络连接或后端服务是否启动';
      _loading = false;
      notifyListeners();
      return false;
    }
  }

  /// 获取用户资料
  Future<void> fetchProfile() async {
    if (!Storage.isLoggedIn) {
      // 未登录时不调用受保护接口，避免 401
      return;
    }
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
      // 拉取能力清单（用于菜单/按钮可见性）
      await _refreshCapabilities();
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

  /// 拉取并缓存当前用户的能力清单（含继承）
  Future<void> _refreshCapabilities() async {
    try {
      final resp = await _api.get(ApiConfig.capabilities);
      if (resp.data['code'] == 0) {
        final list = resp.data['data']?['capabilities'] as List? ?? [];
        await Storage.setCapabilities(list.map((e) => e.toString()).toList());
      }
    } catch (_) {
      // 静默失败：能力清单加载失败不影响登录主流程，前端按"无能力"降级
    }
  }

  /// 退出登录
  Future<void> logout() async {
    await Storage.clearAll();
    _profile = null;
    _error = null;
    notifyListeners();
  }

  /// 用户自助修改密码
  Future<bool> changePassword(String oldPassword, String newPassword) async {
    try {
      final resp = await _api.put(ApiConfig.changePassword, data: {
        'old_password': oldPassword,
        'new_password': newPassword,
      });
      if (resp.data['code'] == 0) return true;
      _error = resp.data['message'] ?? '修改密码失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误';
      notifyListeners();
      return false;
    }
  }

  /// 获取语音开关配置（优先使用缓存）
  Future<int> getVoiceConfig() async {
    if (_voiceEnabled != null) return _voiceEnabled!;
    if (!Storage.isLoggedIn) return 0;
    try {
      final resp = await _api.get(ApiConfig.voiceConfig);
      if (resp.data['code'] == 0) {
        _voiceEnabled = resp.data['data']?['voice_enabled'] ?? 0;
        return _voiceEnabled!;
      }
      return 0;
    } catch (e) {
      return 0;
    }
  }

  /// 更新语音开关
  Future<bool> updateVoiceConfig(int enabled) async {
    try {
      final resp = await _api.put(ApiConfig.voiceConfig, data: {
        'voice_enabled': enabled,
      });
      if (resp.data['code'] == 0) {
        _voiceEnabled = enabled;
        notifyListeners();
        return true;
      }
      return false;
    } catch (e) {
      return false;
    }
  }

  void _handleUnauthorized() {
    logout();
    // 通知 GoRouter 重新评估鉴权状态，自动跳转登录页
    authRefreshNotifier.refresh();
  }
}
