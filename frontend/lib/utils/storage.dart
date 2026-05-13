import 'package:shared_preferences/shared_preferences.dart';

/// 本地存储工具，封装 SharedPreferences
/// 用于持久化 JWT Token、用户信息等
class Storage {
  static const String _keyToken = 'jwt_token';
  static const String _keyUsername = 'username';
  static const String _keyRole = 'role';
  static const String _keyDisplayName = 'display_name';
  static const String _keyConsented = 'consented';
  static const String _keyThemeMode = 'theme_mode';

  static late SharedPreferences _prefs;

  /// 初始化（在 main 中调用）
  static Future<void> init() async {
    _prefs = await SharedPreferences.getInstance();
  }

  // ── Token ──
  static String? get token => _prefs.getString(_keyToken);
  static Future<void> setToken(String token) => _prefs.setString(_keyToken, token);
  static Future<void> clearToken() => _prefs.remove(_keyToken);
  static bool get isLoggedIn => token != null && token!.isNotEmpty;

  // ── 用户信息 ──
  static String? get username => _prefs.getString(_keyUsername);
  static String? get role => _prefs.getString(_keyRole);
  static String? get displayName => _prefs.getString(_keyDisplayName);

  static Future<void> setUserInfo({
    required String username,
    required String role,
    required String displayName,
  }) async {
    await _prefs.setString(_keyUsername, username);
    await _prefs.setString(_keyRole, role);
    await _prefs.setString(_keyDisplayName, displayName);
  }

  // ── 同意授权状态 ──
  static bool get consented => _prefs.getBool(_keyConsented) ?? false;
  static Future<void> setConsented(bool v) => _prefs.setBool(_keyConsented, v);

  /// 清除所有登录信息
  static Future<void> clearAll() async {
    await _prefs.remove(_keyToken);
    await _prefs.remove(_keyUsername);
    await _prefs.remove(_keyRole);
    await _prefs.remove(_keyDisplayName);
  }

  // ── 主题模式 ──
  /// 返回存储的主题模式：'light' / 'dark' / 'system'，默认 'system'
  static String get themeMode => _prefs.getString(_keyThemeMode) ?? 'system';
  static Future<void> setThemeMode(String mode) =>
      _prefs.setString(_keyThemeMode, mode);
}
