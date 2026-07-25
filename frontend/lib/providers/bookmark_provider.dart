import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../utils/storage.dart';

/// 收藏的消息条目（可序列化）
class BookmarkEntry {
  final String id;
  final String question;
  final String conclusion;
  final List<String> sources;   // 来源标题列表
  final List<String> followUps; // 追问建议列表
  final String createdAt;

  BookmarkEntry({
    required this.id,
    required this.question,
    required this.conclusion,
    this.sources = const [],
    this.followUps = const [],
    required this.createdAt,
  });

  Map<String, dynamic> toJson() => {
        'id': id,
        'question': question,
        'conclusion': conclusion,
        'sources': sources,
        'followUps': followUps,
        'createdAt': createdAt,
      };

  factory BookmarkEntry.fromJson(Map<String, dynamic> json) => BookmarkEntry(
        id: json['id'] as String,
        question: json['question'] as String,
        conclusion: json['conclusion'] as String,
        sources: (json['sources'] as List<dynamic>?)
                ?.map((e) => e.toString())
                .toList() ?? [],
        followUps: (json['followUps'] as List<dynamic>?)
                ?.map((e) => e.toString())
                .toList() ?? [],
        createdAt: json['createdAt'] as String,
      );
}

/// 收藏状态管理 — 基于 SharedPreferences 持久化
///
/// 存储键按用户维度隔离（`bookmarks:<username>`），避免不同账号在同一设备上
/// 共用同一份收藏而造成跨账号数据泄露（Q-08）。退出登录须调用 [reset]
/// 清空内存态，防止下一个登录用户看到上一个用户的收藏。
class BookmarkProvider extends ChangeNotifier {
  static const String _legacyKey = 'bookmarks'; // 历史全局键，仅用于一次性迁移
  static const String _keyPrefix = 'bookmarks:';

  final List<BookmarkEntry> _bookmarks = [];
  final Set<String> _conclusionIndex = {};
  SharedPreferences? _prefs;
  bool _loaded = false;
  String? _scopeUser; // 当前已加载数据所属用户

  List<BookmarkEntry> get bookmarks => List.unmodifiable(_bookmarks);
  bool get loaded => _loaded;

  Future<SharedPreferences> get _sharedPrefs async {
    _prefs ??= await SharedPreferences.getInstance();
    return _prefs!;
  }

  /// 当前登录用户的存储键；未登录时用匿名键，互不干扰
  String get _key => '$_keyPrefix${Storage.username ?? "_anon"}';

  /// 退出登录时清空内存态与加载标记，切换账号后重新按新用户加载
  void reset() {
    _bookmarks.clear();
    _conclusionIndex.clear();
    _loaded = false;
    _scopeUser = null;
    notifyListeners();
  }

  /// 从 SharedPreferences 加载收藏列表
  Future<void> load() async {
    final currentUser = Storage.username ?? '_anon';
    // 已加载且仍是同一用户则跳过；用户变化则强制重载
    if (_loaded && _scopeUser == currentUser) return;
    _bookmarks.clear();
    _conclusionIndex.clear();
    _scopeUser = currentUser;

    final prefs = await _sharedPrefs;
    // 一次性迁移：把历史全局键的数据并入当前用户键后清除旧键
    final legacy = prefs.getString(_legacyKey);
    if (legacy != null && legacy.isNotEmpty && prefs.getString(_key) == null) {
      await prefs.setString(_key, legacy);
      await prefs.remove(_legacyKey);
    }
    final raw = prefs.getString(_key);
    if (raw != null && raw.isNotEmpty) {
      final list = json.decode(raw) as List;
      _bookmarks.clear();
      _conclusionIndex.clear();
      for (final item in list) {
        final entry = BookmarkEntry.fromJson(item as Map<String, dynamic>);
        _bookmarks.add(entry);
        _conclusionIndex.add(entry.conclusion);
      }
    }
    _loaded = true;
    notifyListeners();
  }

  /// 添加收藏
  Future<void> add({
    required String question,
    required String conclusion,
    List<String> sources = const [],
    List<String> followUps = const [],
  }) async {
    final id = DateTime.now().millisecondsSinceEpoch.toString();
    final entry = BookmarkEntry(
      id: id,
      question: question,
      conclusion: conclusion,
      sources: sources,
      followUps: followUps,
      createdAt: DateTime.now().toIso8601String(),
    );
    _bookmarks.insert(0, entry);
    _conclusionIndex.add(conclusion);
    await _persist();
  }

  /// 移除收藏
  Future<void> remove(String id) async {
    final idx = _bookmarks.indexWhere((b) => b.id == id);
    if (idx != -1) {
      _conclusionIndex.remove(_bookmarks[idx].conclusion);
      _bookmarks.removeAt(idx);
    }
    await _persist();
  }

  /// 是否已收藏（O(1) 查找）
  bool isBookmarked(String conclusion) {
    return _conclusionIndex.contains(conclusion);
  }

  /// 切换收藏状态
  Future<void> toggle({
    required String question,
    required String conclusion,
    List<String> sources = const [],
    List<String> followUps = const [],
  }) async {
    if (isBookmarked(conclusion)) {
      final entry = _bookmarks.firstWhere((b) => b.conclusion == conclusion);
      await remove(entry.id);
    } else {
      await add(
        question: question,
        conclusion: conclusion,
        sources: sources,
        followUps: followUps,
      );
    }
  }

  Future<void> _persist() async {
    final prefs = await _sharedPrefs;
    final list = _bookmarks.map((b) => b.toJson()).toList();
    await prefs.setString(_key, json.encode(list));
    notifyListeners();
  }
}
