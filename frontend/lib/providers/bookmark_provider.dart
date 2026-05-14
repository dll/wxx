import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

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
class BookmarkProvider extends ChangeNotifier {
  static const String _key = 'bookmarks';

  final List<BookmarkEntry> _bookmarks = [];
  final Set<String> _conclusionIndex = {};
  SharedPreferences? _prefs;
  bool _loaded = false;

  List<BookmarkEntry> get bookmarks => List.unmodifiable(_bookmarks);
  bool get loaded => _loaded;

  Future<SharedPreferences> get _sharedPrefs async {
    _prefs ??= await SharedPreferences.getInstance();
    return _prefs!;
  }

  /// 从 SharedPreferences 加载收藏列表
  Future<void> load() async {
    if (_loaded) return;
    final prefs = await _sharedPrefs;
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
