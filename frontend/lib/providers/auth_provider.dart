import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:dio/dio.dart';
import '../config/api_config.dart';
import '../config/router.dart';
import '../main.dart';
import '../models/models.dart';
import '../services/api_service.dart';
import '../utils/api_error.dart';
import '../utils/storage.dart';

/// 认证状态管理
class AuthProvider extends ChangeNotifier {
  final ApiService _api = ApiService();
  final ThemeNotifier? _themeNotifier;

  bool _loading = false;
  String? _error;
  UserProfile? _profile;
  int? _voiceEnabled;
  bool _mustChangePassword = false;

  bool get loading => _loading;
  String? get error => _error;
  UserProfile? get profile => _profile;
  bool get isLoggedIn => Storage.isLoggedIn;
  int? get voiceEnabled => _voiceEnabled;

  /// 首次登录是否需强制改密
  bool get mustChangePassword => _mustChangePassword;

  /// 完成改密后清除标记
  void clearMustChangePassword() {
    _mustChangePassword = false;
    notifyListeners();
  }

  // ── 用户自备 AI Key（额度耗尽解锁）──
  bool _aiKeyBound = false;
  String _aiKeyProvider = '';
  bool get aiKeyBound => _aiKeyBound;
  String get aiKeyProvider => _aiKeyProvider;

  /// 获取 AI Key 绑定状态
  Future<void> fetchAIKeyInfo() async {
    try {
      final resp = await _api.get(ApiConfig.aiKey);
      final data = resp.data?['data'];
      _aiKeyBound = data?['bound'] == true;
      _aiKeyProvider = data?['provider'] ?? '';
      notifyListeners();
    } catch (_) {}
  }

  /// 保存 AI Key
  Future<bool> saveAIKey(String provider, String apiKey) async {
    try {
      final resp = await _api.put(ApiConfig.aiKey,
          data: {'provider': provider, 'api_key': apiKey});
      if (resp.data?['code'] == 0) {
        _aiKeyBound = true;
        _aiKeyProvider = provider;
        notifyListeners();
        return true;
      }
      _error = resp.data?['message'] ?? '保存失败';
      return false;
    } catch (e) {
      _error = '网络错误';
      return false;
    }
  }

  /// 清除 AI Key
  Future<bool> clearAIKey() async {
    try {
      final resp = await _api.delete(ApiConfig.aiKey);
      if (resp.data?['code'] == 0) {
        _aiKeyBound = false;
        _aiKeyProvider = '';
        notifyListeners();
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  AuthProvider([this._themeNotifier]) {
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

      // 首次登录需强制改密标记
      _mustChangePassword = data['data']?['must_change_password'] == true;

      // 获取用户信息
      await fetchProfile();

      _loading = false;
      notifyListeners();
      return true;
    } catch (e) {
      debugPrint('登录请求失败: $e');
      _error = _loginErrorMessage(e);
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
          ownerScope: _profile!.ownerScope,
          ownerId: _profile!.ownerId,
          status: _profile!.status,
        );
        // 同步入学年份到年级主题（登录/刷新资料时自动切换主题）
        final year = _profile!.enrollmentYear;
        if (year.isNotEmpty) {
          final n = int.tryParse(year);
          if (n != null && n > 0) {
            _themeNotifier?.setEnrollmentYear(n);
          }
        }
      }
      // 拉取能力清单（用于菜单/按钮可见性）
      await _refreshCapabilities();
      authRefreshNotifier.refresh();
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
    _mustChangePassword = false;
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
      if (resp.data['code'] == 0) {
        _mustChangePassword = false;
        return true;
      }
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
    // 令牌吊销/过期时清空全部敏感内存态，防止跨账号泄露（Q-08 / S-01 联动）
    triggerSessionReset();
    // 通知 GoRouter 重新评估鉴权状态，自动跳转登录页
    authRefreshNotifier.refresh();
  }

  String _loginErrorMessage(Object error) {
    // 登录领域特化文案优先：先透传后端业务 message，再按状态码给出可操作提示
    if (error is DioException) {
      final responseMessage = _responseMessage(error.response?.data);
      if (responseMessage != null && responseMessage.isNotEmpty) {
        return responseMessage;
      }
      switch (error.response?.statusCode) {
        case 400:
          return '请输入完整的账号和密码';
        case 401:
          return '账号或密码错误';
        case 403:
          return '账号尚未启用或已被停用';
      }
    }
    // 其余网络层场景（超时/断网/限流/5xx）统一走全局错误映射，避免重复维护 switch
    return friendlyApiError(error, fallback: '登录失败，请稍后重试');
  }

  String? _responseMessage(dynamic data) {
    if (data is Map) {
      return data['message']?.toString();
    }
    if (data is String && data.isNotEmpty) {
      try {
        final decoded = jsonDecode(data);
        if (decoded is Map) {
          return decoded['message']?.toString();
        }
      } catch (_) {
        // 非 JSON 错误页不直接展示，避免把网关 HTML 暴露给用户。
      }
    }
    return null;
  }
}
