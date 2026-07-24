import 'package:shared_preferences/shared_preferences.dart';

/// 本地存储工具，封装 SharedPreferences
/// 用于持久化 JWT Token、用户信息、能力清单等
class Storage {
  static const String _keyToken = 'jwt_token';
  static const String _keyUsername = 'username';
  static const String _keyRole = 'role';
  static const String _keyDisplayName = 'display_name';
  static const String _keyConsented = 'consented';
  static const String _keyFirstLaunch = 'first_launch_done';
  static const String _keyThemeMode = 'theme_mode';
  static const String _keyCapabilities = 'capabilities';
  static const String _keyListedFeatures = 'listed_features';
  static const String _keyEnabledFeatures = 'enabled_features';

  static late SharedPreferences _prefs;

  /// 初始化（在 main 中调用）
  static Future<void> init() async {
    _prefs = await SharedPreferences.getInstance();
  }

  // ── Token ──
  static String? get token => _prefs.getString(_keyToken);
  static Future<void> setToken(String token) =>
      _prefs.setString(_keyToken, token);
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

  // ── 首次启动 ──
  /// 是否已完成首次启动的隐私同意流程
  static bool get firstLaunchDone => _prefs.getBool(_keyFirstLaunch) ?? false;
  static Future<void> setFirstLaunchDone() =>
      _prefs.setBool(_keyFirstLaunch, true);

  // ── 同意授权状态 ──
  static bool get consented => _prefs.getBool(_keyConsented) ?? false;
  static Future<void> setConsented(bool v) => _prefs.setBool(_keyConsented, v);

  // ── 能力清单（来自后端 /user/capabilities）──
  /// 当前用户拥有的能力 ID 列表，含继承
  static List<String> get capabilities =>
      _prefs.getStringList(_keyCapabilities) ?? const [];

  static Future<void> setCapabilities(List<String> caps) =>
      _prefs.setStringList(_keyCapabilities, caps);

  static List<String> get listedFeatures =>
      _prefs.getStringList(_keyListedFeatures) ?? const [];
  static Future<void> setListedFeatures(List<String> keys) =>
      _prefs.setStringList(_keyListedFeatures, keys);

  static List<String> get enabledFeatures =>
      _prefs.getStringList(_keyEnabledFeatures) ?? const [];
  static Future<void> setEnabledFeatures(List<String> keys) =>
      _prefs.setStringList(_keyEnabledFeatures, keys);

  /// 清除所有登录信息
  static Future<void> clearAll() async {
    await _prefs.remove(_keyToken);
    await _prefs.remove(_keyUsername);
    await _prefs.remove(_keyRole);
    await _prefs.remove(_keyDisplayName);
    await _prefs.remove(_keyCapabilities);
  }

  // ── 主题模式 ──
  /// 返回存储的主题模式：'light' / 'dark' / 'system'，默认 'system'
  static String get themeMode => _prefs.getString(_keyThemeMode) ?? 'system';
  static Future<void> setThemeMode(String mode) =>
      _prefs.setString(_keyThemeMode, mode);
}
