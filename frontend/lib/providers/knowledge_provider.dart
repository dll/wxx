import 'package:flutter/material.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 知识大厅状态管理
class KnowledgeProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  Map<String, List<KnowledgeCard>> _categories = {};
  bool _loading = false;
  String? _error;
  String _selectedType = ''; // 空=全部

  Map<String, List<KnowledgeCard>> get categories => _categories;
  bool get loading => _loading;
  String? get error => _error;
  String get selectedType => _selectedType;

  /// 是否有数据
  bool get isEmpty => _categories.isEmpty && !_loading;

  /// 总卡片数
  int get totalCount {
    int count = 0;
    for (final list in _categories.values) {
      count += list.length;
    }
    return count;
  }

  /// 加载知识大厅数据
  Future<void> load({String? type}) async {
    _loading = true;
    _error = null;
    _selectedType = type ?? '';
    notifyListeners();

    try {
      final queryParams = <String, dynamic>{};
      if (_selectedType.isNotEmpty) {
        queryParams['type'] = _selectedType;
      }
      final response = await _api.get(ApiConfig.knowledge, params: queryParams);

      final data = response.data['data'] as Map<String, dynamic>?;
      if (data == null) {
        _categories = {};
      } else {
        _categories = {};
        data.forEach((type, cards) {
          final list = (cards as List)
              .map((c) => KnowledgeCard.fromJson(c as Map<String, dynamic>))
              .toList();
          _categories[type] = list;
        });
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// 切换类型过滤
  void filterByType(String? type) {
    load(type: type);
  }

  /// 类型顺序（Policy → Process → FAQ → Activity）
  static const List<String> typeOrder = ['Policy', 'Process', 'FAQ', 'Activity'];

  // ── 知识管理（counselor+）──

  final List<KnowledgeCard> _resources = [];
  bool _resourcesLoading = false;
  int _resourcePage = 1;
  int _resourceTotal = 0;
  String _resourceError = '';

  List<KnowledgeCard> get resources => _resources;
  bool get resourcesLoading => _resourcesLoading;
  int get resourceTotal => _resourceTotal;
  String get resourceError => _resourceError;

  Future<void> listResources({bool refresh = false}) async {
    if (_resourcesLoading) return;
    if (refresh) {
      _resourcePage = 1;
      _resources.clear();
    }
    _resourcesLoading = true;
    _resourceError = '';
    notifyListeners();

    try {
      final response = await _api.get(ApiConfig.kbResources,
          params: {'page': _resourcePage, 'page_size': 20});
      if (response.data['code'] == 0) {
        final list = (response.data['data'] as List?)
                ?.map((e) => KnowledgeCard.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [];
        _resources.addAll(list);
        _resourceTotal = response.data['total'] ?? 0;
        _resourcePage++;
      }
    } catch (e) {
      _resourceError = '获取资源列表失败: $e';
    } finally {
      _resourcesLoading = false;
      notifyListeners();
    }
  }

  // ── 审核（counselor+）──

  final List<KnowledgeCard> _pendingReviews = [];
  bool _reviewsLoading = false;
  int _reviewPage = 1;
  int _reviewTotal = 0;

  List<KnowledgeCard> get pendingReviews => _pendingReviews;
  bool get reviewsLoading => _reviewsLoading;
  int get reviewTotal => _reviewTotal;

  Future<void> listPendingReviews({bool refresh = false}) async {
    if (_reviewsLoading) return;
    if (refresh) {
      _reviewPage = 1;
      _pendingReviews.clear();
    }
    _reviewsLoading = true;
    notifyListeners();

    try {
      final response = await _api.get(ApiConfig.reviewPending,
          params: {'page': _reviewPage, 'page_size': 20});
      if (response.data['code'] == 0) {
        final list = (response.data['data'] as List?)
                ?.map((e) => KnowledgeCard.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [];
        _pendingReviews.addAll(list);
        _reviewTotal = response.data['total'] ?? 0;
        _reviewPage++;
      }
    } catch (e) {
      // ignore
    } finally {
      _reviewsLoading = false;
      notifyListeners();
    }
  }

  Future<bool> approveResource(String resourceId) async {
    try {
      final response = await _api.post(ApiConfig.kbApprove(resourceId));
      return response.data['code'] == 0;
    } catch (_) {
      return false;
    }
  }

  Future<bool> rejectResource(String resourceId) async {
    try {
      final response = await _api.post(ApiConfig.kbReject(resourceId));
      return response.data['code'] == 0;
    } catch (_) {
      return false;
    }
  }

  Future<bool> submitForReview(String resourceId) async {
    try {
      final response = await _api.post(ApiConfig.kbSubmitReview(resourceId));
      return response.data['code'] == 0;
    } catch (_) {
      return false;
    }
  }

  Future<bool> createResource(Map<String, dynamic> data) async {
    try {
      final response = await _api.post(ApiConfig.kbResources, data: data);
      return response.data['code'] == 0;
    } catch (_) {
      return false;
    }
  }

  /// 按固定顺序返回分类列表
  List<MapEntry<String, List<KnowledgeCard>>> get orderedCategories {
    final result = <MapEntry<String, List<KnowledgeCard>>>[];
    for (final type in typeOrder) {
      if (_categories.containsKey(type) && _categories[type]!.isNotEmpty) {
        result.add(MapEntry(type, _categories[type]!));
      }
    }
    return result;
  }
}
