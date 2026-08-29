import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// 本地存储工具，封装 SharedPreferences
/// 用于持久化 JWT Token、用户信息、能力清单等
class Storage {
  static const String _keyToken = 'jwt_token';
  static const FlutterSecureStorage _secureStorage = FlutterSecureStorage();
  static String? _token;
  static const String _keyUsername = 'username';
  static const String _keyRole = 'role';
  static const String _keyDisplayName = 'display_name';
  static const String _keyOwnerScope = 'owner_scope';
  static const String _keyOwnerId = 'owner_id';
  static const String _keyUserStatus = 'user_status';
  static const String _keyConsented = 'consented';
  static const String _keyFirstLaunch = 'first_launch_done';
  static const String _keyFreshmanGuideSeen = 'freshman_guide_seen';
  static const String _keyThemeMode = 'theme_mode';
  static const String _keyCapabilities = 'capabilities';
  static const String _keyListedFeatures = 'listed_features';
  static const String _keyEnabledFeatures = 'enabled_features';
  static const String _keyGlobalFeatureSwitches = 'global_feature_switches';
  static const String _keyShowAvatar = 'show_avatar';
  static const String _keyGradeThemeEnabled = 'grade_theme_enabled';
  static const String _keyEnrollmentYear = 'enrollment_year';
  static const String _keyStudentInterests =
      'student_interests'; // 学生关注内容（逗号分隔的多选兴趣）

  // ── 反馈草稿 ──
  static const String _keyFeedbackDraft = 'feedback_draft';
  static const String _keyFeedbackDraftCategory = 'feedback_draft_category';
  static const String _keyFeedbackDraftModule = 'feedback_draft_module';

  static late SharedPreferences _prefs;

  /// 初始化（在 main 中调用）
  static Future<void> init() async {
    _prefs = await SharedPreferences.getInstance();
    _token = await _secureStorage.read(key: _keyToken);
    // 一次性迁移旧版本 SharedPreferences 中的 bearer token，然后删除明文副本。
    final legacyToken = _prefs.getString(_keyToken);
    if ((_token == null || _token!.isEmpty) && legacyToken != null && legacyToken.isNotEmpty) {
      await _secureStorage.write(key: _keyToken, value: legacyToken);
      _token = legacyToken;
    }
    if (legacyToken != null) {
      await _prefs.remove(_keyToken);
    }
  }

  // ── Token ──
  static String? get token => _token;
  static Future<void> setToken(String token) async {
    await _secureStorage.write(key: _keyToken, value: token);
    _token = token;
  }
  static Future<void> clearToken() async {
    await _secureStorage.delete(key: _keyToken);
    _token = null;
  }
  static bool get isLoggedIn => token != null && token!.isNotEmpty;

  // ── 用户信息 ──
  static String? get username => _prefs.getString(_keyUsername);
  static String? get role => _prefs.getString(_keyRole);
  static String? get displayName => _prefs.getString(_keyDisplayName);
  static String? get ownerScope => _prefs.getString(_keyOwnerScope);
  static String? get ownerId => _prefs.getString(_keyOwnerId);
  static String? get userStatus => _prefs.getString(_keyUserStatus);

  static Future<void> setUserInfo({
    required String username,
    required String role,
    required String displayName,
    String ownerScope = '',
    String ownerId = '',
    String status = 'active',
  }) async {
    await _prefs.setString(_keyUsername, username);
    await _prefs.setString(_keyRole, role);
    await _prefs.setString(_keyDisplayName, displayName);
    await _prefs.setString(_keyOwnerScope, ownerScope);
    await _prefs.setString(_keyOwnerId, ownerId);
    await _prefs.setString(_keyUserStatus, status);
  }

  // ── 首次启动 ──
  /// 是否已完成首次启动的隐私同意流程
  static bool get firstLaunchDone => _prefs.getBool(_keyFirstLaunch) ?? false;
  static Future<void> setFirstLaunchDone() =>
      _prefs.setBool(_keyFirstLaunch, true);

  // ── 新生引导状态 ──
  /// 是否已看过新生应用内引导（onboarding）
  static bool get freshmanGuideSeen =>
      _prefs.getBool(_keyFreshmanGuideSeen) ?? false;
  static Future<void> setFreshmanGuideSeen() =>
      _prefs.setBool(_keyFreshmanGuideSeen, true);

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

  /// 管理员全局功能开关（key=feature.*，缓存自后端 /public/feature-switches）
  static Map<String, String> get globalFeatureSwitches =>
      (_prefs.getStringList(_keyGlobalFeatureSwitches) ?? const []).fold({},
          (map, kv) {
        final idx = kv.indexOf('=');
        if (idx > 0) {
          map[kv.substring(0, idx)] = kv.substring(idx + 1);
        }
        return map;
      });
  static Future<void> setGlobalFeatureSwitches(Map<String, String> switches) =>
      _prefs.setStringList(_keyGlobalFeatureSwitches,
          switches.entries.map((e) => '${e.key}=${e.value}').toList());

  /// 清除所有登录信息
  static Future<void> clearAll() async {
    await clearToken();
    await _prefs.remove(_keyUsername);
    await _prefs.remove(_keyRole);
    await _prefs.remove(_keyOwnerScope);
    await _prefs.remove(_keyOwnerId);
    await _prefs.remove(_keyUserStatus);
    await _prefs.remove(_keyDisplayName);
    await _prefs.remove(_keyCapabilities);
  }

  // ── 主题模式 ──
  /// 返回存储的主题模式：'light' / 'dark' / 'system'，默认 'light'
  static String get themeMode => _prefs.getString(_keyThemeMode) ?? 'light';
  static Future<void> setThemeMode(String mode) =>
      _prefs.setString(_keyThemeMode, mode);

  // ── 数字人形象显示开关 ──
  /// 是否显示数字人形象（默认开启）
  static bool get showAvatar => _prefs.getBool(_keyShowAvatar) ?? true;
  static Future<void> setShowAvatar(bool v) =>
      _prefs.setBool(_keyShowAvatar, v);

  // ── 年级主题自动切换 ──
  /// 是否按入学年份自动切换年级主题（默认开启）
  static bool get gradeThemeEnabled =>
      _prefs.getBool(_keyGradeThemeEnabled) ?? true;
  static Future<void> setGradeThemeEnabled(bool v) =>
      _prefs.setBool(_keyGradeThemeEnabled, v);

  /// 缓存入学年份（如 2025）
  static int? get enrollmentYear => _prefs.getInt(_keyEnrollmentYear);
  static Future<void> setEnrollmentYear(int v) =>
      _prefs.setInt(_keyEnrollmentYear, v);

  // ── 学生关注内容（兴趣，逗号分隔）──
  static List<String> get studentInterests {
    final raw = _prefs.getString(_keyStudentInterests) ?? '';
    if (raw.isEmpty) return const [];
    return raw
        .split(',')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();
  }

  static bool get studentInterestsCollected =>
      _prefs.getString(_keyStudentInterests) != null;

  static Future<void> setStudentInterests(List<String> interests) =>
      _prefs.setString(_keyStudentInterests, interests.join(','));

  static Future<void> clearStudentInterests() =>
      _prefs.remove(_keyStudentInterests);

  /// 当前学生年级（1~4）：由入学年份推导；未知返回 0。
  static int get grade {
    final y = enrollmentYear;
    if (y == null || y <= 0) return 0;
    final g = DateTime.now().year - y + 1;
    return g.clamp(1, 4);
  }

  // ── 反馈草稿（提交失败/关闭时保存，下次打开恢复）──
  static String get feedbackDraft => _prefs.getString(_keyFeedbackDraft) ?? '';
  static String get feedbackDraftCategory =>
      _prefs.getString(_keyFeedbackDraftCategory) ?? 'answer_error';
  static String get feedbackDraftModule =>
      _prefs.getString(_keyFeedbackDraftModule) ?? '';
  static Future<void> saveFeedbackDraft(
          {String content = '', String category = '', String module = ''}) =>
      Future.wait([
        _prefs.setString(_keyFeedbackDraft, content),
        _prefs.setString(_keyFeedbackDraftCategory, category),
        _prefs.setString(_keyFeedbackDraftModule, module),
      ]);
  static Future<void> clearFeedbackDraft() => Future.wait([
        _prefs.remove(_keyFeedbackDraft),
        _prefs.remove(_keyFeedbackDraftCategory),
        _prefs.remove(_keyFeedbackDraftModule),
      ]);
}
