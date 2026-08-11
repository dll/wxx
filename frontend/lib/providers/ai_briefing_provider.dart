import 'package:flutter/foundation.dart';

import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// AI 简讯 Provider — 用户列表 + 管理 CRUD + 来源设置 + 导出
class AIBriefingProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  // ── 用户端 ──
  bool _userLoading = false;
  bool _userLoaded = false; // 是否已尝试过首次加载（防失败后无限自动重试触发限流）
  String _userError = '';
  List<AIBriefing> _userBriefings = const [];
  List<AIBriefing> _hotBriefings = const [];
  List<AIBriefing> _favoriteBriefings = const [];
  bool _hotLoading = false;
  bool _favoritesLoading = false;
  String _userCategory = '';
  String _userQ = '';

  bool get userLoading => _userLoading;
  String get userError => _userError;
  List<AIBriefing> get userBriefings => _userBriefings;
  List<AIBriefing> get hotBriefings => _hotBriefings;
  List<AIBriefing> get favoriteBriefings => _favoriteBriefings;
  bool get hotLoading => _hotLoading;
  bool get favoritesLoading => _favoritesLoading;
  String get userCategory => _userCategory;

  /// 是否已完成首次加载（首页卡片据此决定是否自动触发请求）
  bool get userLoaded => _userLoaded;

  Future<void> fetchUserBriefings({String? category, String? q}) async {
    _userLoading = true;
    _userError = '';
    _userLoaded = true;
    if (category != null) _userCategory = category;
    if (q != null) _userQ = q;
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.aiBriefings, params: {
        if (_userCategory.isNotEmpty) 'category': _userCategory,
        if (_userQ.isNotEmpty) 'q': _userQ,
      });
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map ? (res.data as Map)['data'] : res.data;
        if (data is List) {
          _userBriefings = data
              .whereType<Map>()
              .map((e) => AIBriefing.fromJson(Map<String, dynamic>.from(e)))
              .toList();
        }
      }
    } catch (e) {
      _userError = e.toString();
    } finally {
      _userLoading = false;
      notifyListeners();
    }
  }

  /// 热度榜
  Future<void> fetchHotBriefings() async {
    _hotLoading = true;
    notifyListeners();
    try {
      final res =
          await _api.get(ApiConfig.aiBriefingsHot, params: {'limit': 50});
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map ? (res.data as Map)['data'] : res.data;
        if (data is List) {
          _hotBriefings = data
              .whereType<Map>()
              .map((e) => AIBriefing.fromJson(Map<String, dynamic>.from(e)))
              .toList();
        }
      }
    } catch (_) {
    } finally {
      _hotLoading = false;
      notifyListeners();
    }
  }

  /// 我的收藏
  Future<void> fetchFavoriteBriefings() async {
    _favoritesLoading = true;
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.aiBriefingsFavorites);
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map ? (res.data as Map)['data'] : res.data;
        if (data is List) {
          _favoriteBriefings = data
              .whereType<Map>()
              .map((e) => AIBriefing.fromJson(Map<String, dynamic>.from(e)))
              .toList();
        }
      }
    } catch (_) {
    } finally {
      _favoritesLoading = false;
      notifyListeners();
    }
  }

  /// 收藏/取消收藏并同步三份列表的收藏态
  Future<bool> toggleFavorite(AIBriefing b) async {
    final target = !b.favorited;
    try {
      final res = target
          ? await _api.post(ApiConfig.aiBriefingFavorite('${b.id}'))
          : await _api.delete(ApiConfig.aiBriefingFavorite('${b.id}'));
      if (res.statusCode != 200) return false;
    } catch (_) {
      return false;
    }
    _applyFavoriteState(b.id, target);
    if (target) {
      _favoriteBriefings = [
        b.copyWith(favorited: true),
        ..._favoriteBriefings.where((e) => e.id != b.id),
      ];
    } else {
      _favoriteBriefings =
          _favoriteBriefings.where((e) => e.id != b.id).toList();
    }
    notifyListeners();
    return true;
  }

  void _applyFavoriteState(int id, bool favorited) {
    _userBriefings = [
      for (final e in _userBriefings)
        e.id == id ? e.copyWith(favorited: favorited) : e,
    ];
    _hotBriefings = [
      for (final e in _hotBriefings)
        e.id == id ? e.copyWith(favorited: favorited) : e,
    ];
  }

  // ── 管理端 ──
  bool _adminLoading = false;
  String _adminError = '';
  List<AIBriefing> _adminBriefings = const [];
  int _total = 0;
  Map<String, dynamic>? _stats;
  List<AIBriefingSource> _sources = const [];

  bool get adminLoading => _adminLoading;
  String get adminError => _adminError;
  List<AIBriefing> get adminBriefings => _adminBriefings;
  int get total => _total;
  Map<String, dynamic>? get stats => _stats;
  List<AIBriefingSource> get sources => _sources;

  Future<void> fetchAdminBriefings({
    String status = '',
    String category = '',
    String q = '',
    int page = 1,
    int pageSize = 20,
  }) async {
    _adminLoading = true;
    _adminError = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.adminAIBriefings, params: {
        if (status.isNotEmpty) 'status': status,
        if (category.isNotEmpty) 'category': category,
        if (q.isNotEmpty) 'q': q,
        'page': page,
        'page_size': pageSize,
      });
      if (res.statusCode == 200 && res.data != null) {
        final data = (res.data as Map)['data'];
        _adminBriefings = (data['list'] as List? ?? [])
            .whereType<Map>()
            .map((e) => AIBriefing.fromJson(Map<String, dynamic>.from(e)))
            .toList();
        _total = data['total'] ?? 0;
      }
    } catch (e) {
      _adminError = e.toString();
    } finally {
      _adminLoading = false;
      notifyListeners();
    }
  }

  Future<void> fetchStats() async {
    try {
      final res = await _api.get(ApiConfig.adminAIBriefingStats);
      if (res.statusCode == 200 && res.data != null) {
        _stats = (res.data as Map)['data'] as Map<String, dynamic>?;
        notifyListeners();
      }
    } catch (_) {}
  }

  Future<void> fetchSources() async {
    try {
      final res = await _api.get(ApiConfig.adminAIBriefingSources);
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map ? (res.data as Map)['data'] : res.data;
        if (data is List) {
          _sources = data
              .whereType<Map>()
              .map((e) =>
                  AIBriefingSource.fromJson(Map<String, dynamic>.from(e)))
              .toList();
          notifyListeners();
        }
      }
    } catch (_) {}
  }

  Future<bool> createBriefing(AIBriefing b) async {
    try {
      final res = await _api.post(ApiConfig.adminAIBriefings, data: b.toJson());
      return res.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  Future<bool> updateBriefing(AIBriefing b) async {
    try {
      final res = await _api.put(ApiConfig.adminAIBriefing('${b.id}'),
          data: b.toJson());
      return res.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  Future<bool> updateBriefingStatus(int id, int status) async {
    try {
      final res = await _api.put(ApiConfig.adminAIBriefingStatus('$id'),
          data: {'status': status});
      return res.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  Future<bool> deleteBriefing(int id) async {
    try {
      final res = await _api.delete(ApiConfig.adminAIBriefing('$id'));
      return res.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  Future<bool> deleteManyBriefings(List<int> ids) async {
    try {
      final res = await _api.post('${ApiConfig.adminAIBriefings}/batch-delete',
          data: {'ids': ids});
      return res.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  Future<bool> clearAll() async {
    try {
      final res = await _api.delete(ApiConfig.adminAIBriefingClear);
      return res.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  Future<bool> createSource(AIBriefingSource s) async {
    try {
      final res =
          await _api.post(ApiConfig.adminAIBriefingSources, data: s.toJson());
      return res.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  Future<bool> updateSource(AIBriefingSource s) async {
    try {
      final res = await _api.put(ApiConfig.adminAIBriefingSource('${s.id}'),
          data: s.toJson());
      return res.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  Future<bool> deleteSource(int id) async {
    try {
      final res = await _api.delete(ApiConfig.adminAIBriefingSource('$id'));
      return res.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  Future<Map<String, dynamic>?> fetchNow() async {
    try {
      final res = await _api.post(ApiConfig.adminAIBriefingFetch, data: {});
      if (res.statusCode == 200) {
        final data = res.data is Map ? (res.data as Map)['data'] : null;
        return data is Map ? Map<String, dynamic>.from(data) : {};
      }
    } catch (_) {}
    return null;
  }

  /// 导出：format=md|pdf, ids=[] 且 all=true 时导出全部
  /// 返回下载 URL（GET 直接触发浏览器下载）
  String exportUrl(
      {required String format, List<int> ids = const [], bool all = false}) {
    final sb = StringBuffer()
      ..write(ApiConfig.adminAIBriefingExport)
      ..write('?format=$format');
    if (all) {
      sb.write('&all=1');
    } else if (ids.isNotEmpty) {
      sb.write('&ids=${ids.join(",")}');
    }
    return sb.toString();
  }
}
