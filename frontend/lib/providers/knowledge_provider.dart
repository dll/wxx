import 'package:flutter/material.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';
import '../utils/capability_utils.dart';
import '../utils/storage.dart';

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
  /// 已登录调用认证接口（返回个性化内容），未登录调用公开接口（仅全校公开内容）
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
      // 根据登录状态选择接口
      final apiPath = Storage.isLoggedIn
          ? ApiConfig.knowledge
          : ApiConfig.knowledgePublic;
      final response = await _api.get(apiPath, params: queryParams);

      final data = response.data['data'] as Map<String, dynamic>?;
      if (data == null) {
        _categories = {};
      } else {
        _categories = {};
        data.forEach((type, cards) {
          if (cards is! List) return;
          final list = cards
              .whereType<Map>()
              .map((c) => KnowledgeCard.fromJson(Map<String, dynamic>.from(c)))
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
  static const List<String> typeOrder = [
    'Policy',
    'Process',
    'FAQ',
    'Activity'
  ];

  // ── 知识管理（counselor+）──

  final List<KnowledgeCard> _resources = [];
  bool _resourcesLoading = false;
  int _resourcePage = 1;
  int _resourceTotal = 0;
  String _resourceError = '';

  // 高级筛选条件
  String _keyword = '';
  String _resourceTypeFilter = '';
  String _statusFilter = '';
  String _ownerScopeFilter = '';
  String _sortBy = 'updated_at';
  String _sortOrder = 'desc';

  // 字典值
  List<String> _resourceTypeList = [];
  List<String> _statusList = [];
  List<String> _ownerScopeList = [];

  // 批量选择
  final Set<String> _selectedResourceIds = {};

  // 统计数据
  Map<String, dynamic>? _kbStats;

  List<KnowledgeCard> get resources => _resources;
  bool get resourcesLoading => _resourcesLoading;
  int get resourceTotal => _resourceTotal;
  String get resourceError => _resourceError;
  String get keyword => _keyword;
  String get resourceTypeFilter => _resourceTypeFilter;
  String get statusFilter => _statusFilter;
  String get ownerScopeFilter => _ownerScopeFilter;
  List<String> get resourceTypeList => _resourceTypeList;
  List<String> get statusList => _statusList;
  List<String> get ownerScopeList => _ownerScopeList;
  Set<String> get selectedResourceIds => _selectedResourceIds;
  int get selectedCount => _selectedResourceIds.length;
  Map<String, dynamic>? get kbStats => _kbStats;

  /// 搜索知识资源（高级查询，分页刷新）
  Future<void> searchResources({bool refresh = false}) async {
    if (_resourcesLoading && !refresh) return;
    if (!CapabilityUtils.has(Capability.counselorKbWrite)) {
      _resourceError = '当前角色无权访问知识资源管理';
      notifyListeners();
      return;
    }
    if (refresh) {
      _resourcePage = 1;
      _resources.clear();
      _selectedResourceIds.clear();
    }
    _resourceError = '';
    _resourcesLoading = true;
    notifyListeners();

    try {
      final params = <String, dynamic>{
        'page': _resourcePage,
        'page_size': 20,
      };
      if (_keyword.isNotEmpty) params['keyword'] = _keyword;
      if (_resourceTypeFilter.isNotEmpty) {
        params['resource_type'] = _resourceTypeFilter;
      }
      if (_statusFilter.isNotEmpty) params['status'] = _statusFilter;
      if (_ownerScopeFilter.isNotEmpty) {
        params['owner_scope'] = _ownerScopeFilter;
      }
      params['sort_by'] = _sortBy;
      params['sort_order'] = _sortOrder;

      final response = await _api.get(ApiConfig.kbResourcesAdvanced,
          params: params);
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

  /// 旧方法兼容
  Future<void> listResources({bool refresh = false}) async {
    await searchResources(refresh: refresh);
  }

  /// 设置搜索关键词
  void setKeyword(String value) {
    _keyword = value.trim();
    searchResources(refresh: true);
  }

  /// 设置筛选条件
  void setResourceFilter({
    String? resourceType,
    String? status,
    String? ownerScope,
    String? sortBy,
    String? sortOrder,
  }) {
    if (resourceType != null) _resourceTypeFilter = resourceType;
    if (status != null) _statusFilter = status;
    if (ownerScope != null) _ownerScopeFilter = ownerScope;
    if (sortBy != null) _sortBy = sortBy;
    if (sortOrder != null) _sortOrder = sortOrder;
    searchResources(refresh: true);
  }

  /// 重置筛选条件
  void resetResourceFilters() {
    _keyword = '';
    _resourceTypeFilter = '';
    _statusFilter = '';
    _ownerScopeFilter = '';
    _sortBy = 'updated_at';
    _sortOrder = 'desc';
    searchResources(refresh: true);
  }

  /// 获取字典值
  Future<List<String>> fetchDictValues(String column) async {
    try {
      final response = await _api.get(ApiConfig.kbDict, params: {'column': column});
      if (response.data['code'] == 0 && response.data['data'] != null) {
        final list = (response.data['data'] as List)
            .map((e) => e.toString())
            .toList();
        switch (column) {
          case 'resource_type':
            _resourceTypeList = list;
            break;
          case 'status':
            _statusList = list;
            break;
          case 'owner_scope':
            _ownerScopeList = list;
            break;
        }
        notifyListeners();
        return list;
      }
      return [];
    } catch (e) {
      _resourceError = '获取字典值失败: $e';
      return [];
    }
  }

  /// 获取统计数据
  Future<void> fetchStats() async {
    try {
      final response = await _api.get(ApiConfig.kbStats);
      if (response.data['code'] == 0 && response.data['data'] != null) {
        _kbStats = Map<String, dynamic>.from(response.data['data'] as Map);
        notifyListeners();
      }
    } catch (e) {
      // 忽略统计获取失败
    }
  }

  // ── 批量选择 ──

  void toggleSelectResource(String resourceId) {
    if (_selectedResourceIds.contains(resourceId)) {
      _selectedResourceIds.remove(resourceId);
    } else {
      _selectedResourceIds.add(resourceId);
    }
    notifyListeners();
  }

  void selectAllVisibleResources() {
    for (final r in _resources) {
      _selectedResourceIds.add(r.resourceId);
    }
    notifyListeners();
  }

  void deselectAllResources() {
    _selectedResourceIds.clear();
    notifyListeners();
  }

  // ── 批量操作 ──

  Future<bool> batchApprove() async {
    if (_selectedResourceIds.isEmpty) return false;
    try {
      final response = await _api.post(
        ApiConfig.kbBatchApprove,
        data: {'ids': _selectedResourceIds.toList()},
      );
      if (response.data['code'] == 0) {
        _selectedResourceIds.clear();
        await searchResources(refresh: true);
        fetchStats();
        return true;
      }
      _resourceError = response.data['message'] ?? '操作失败';
      notifyListeners();
      return false;
    } catch (e) {
      _resourceError = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  Future<bool> batchReject() async {
    if (_selectedResourceIds.isEmpty) return false;
    try {
      final response = await _api.post(
        ApiConfig.kbBatchReject,
        data: {'ids': _selectedResourceIds.toList()},
      );
      if (response.data['code'] == 0) {
        _selectedResourceIds.clear();
        await searchResources(refresh: true);
        fetchStats();
        return true;
      }
      _resourceError = response.data['message'] ?? '操作失败';
      notifyListeners();
      return false;
    } catch (e) {
      _resourceError = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  Future<bool> batchRetire() async {
    if (_selectedResourceIds.isEmpty) return false;
    try {
      final response = await _api.post(
        ApiConfig.kbBatchRetire,
        data: {'ids': _selectedResourceIds.toList()},
      );
      if (response.data['code'] == 0) {
        _selectedResourceIds.clear();
        await searchResources(refresh: true);
        fetchStats();
        return true;
      }
      _resourceError = response.data['message'] ?? '操作失败';
      notifyListeners();
      return false;
    } catch (e) {
      _resourceError = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  Future<bool> batchDelete() async {
    if (_selectedResourceIds.isEmpty) return false;
    try {
      final response = await _api.post(
        ApiConfig.kbBatchDelete,
        data: {'ids': _selectedResourceIds.toList()},
      );
      if (response.data['code'] == 0) {
        _selectedResourceIds.clear();
        await searchResources(refresh: true);
        fetchStats();
        return true;
      }
      _resourceError = response.data['message'] ?? '删除失败';
      notifyListeners();
      return false;
    } catch (e) {
      _resourceError = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  /// 批量 AI 精修选中资源元数据（标题/摘要/标签）。
  /// 返回汇总信息供结果展示：{total, success, failed, results[]}。
  Future<Map<String, dynamic>?> batchRefine() async {
    if (_selectedResourceIds.isEmpty) return null;
    try {
      final response = await _api.post(
        ApiConfig.kbBatchRefine,
        data: {'ids': _selectedResourceIds.toList()},
      );
      if (response.data['code'] == 0) {
        final data =
            Map<String, dynamic>.from(response.data['data'] as Map);
        _selectedResourceIds.clear();
        await searchResources(refresh: true);
        fetchStats();
        return data;
      }
      _resourceError = response.data['message']?.toString() ?? '批量精修失败';
      notifyListeners();
      return null;
    } catch (e) {
      _resourceError = '网络错误: $e';
      notifyListeners();
      return null;
    }
  }

  Future<bool> deleteResource(String resourceId) async {
    try {
      final response = await _api.post(
        ApiConfig.kbBatchDelete,
        data: {'ids': [resourceId]},
      );
      if (response.data['code'] == 0) {
        _selectedResourceIds.remove(resourceId);
        await searchResources(refresh: true);
        fetchStats();
        return true;
      }
      _resourceError = response.data['message'] ?? '删除失败';
      notifyListeners();
      return false;
    } catch (e) {
      _resourceError = '网络错误: $e';
      notifyListeners();
      return false;
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

  Future<KnowledgeCard?> getResource(String resourceId) async {
    try {
      final response = await _api.get(ApiConfig.kbResource(resourceId));
      final data = response.data['data'];
      if (response.data['code'] == 0 && data is Map<String, dynamic>) {
        return KnowledgeCard.fromJson(data);
      }
    } catch (_) {}
    return null;
  }

  Future<bool> createResource(Map<String, dynamic> data) async {
    try {
      final response = await _api.post(ApiConfig.kbResources, data: data);
      return response.data['code'] == 0;
    } catch (_) {
      return false;
    }
  }

  Future<bool> updateResource(
      String resourceId, Map<String, dynamic> data) async {
    try {
      final response =
          await _api.put(ApiConfig.kbResource(resourceId), data: data);
      return response.data['code'] == 0;
    } catch (_) {
      return false;
    }
  }

  Future<Map<String, dynamic>?> uploadKnowledgeDocument({
    required List<int> bytes,
    required String filename,
    required String resourceType,
  }) async {
    try {
      final response = await _api.uploadBytes(
        ApiConfig.kbUpload,
        bytes: bytes,
        filename: filename,
        fieldName: 'file',
        fields: {'resource_type': resourceType},
      );
      if (response.statusCode == 200) {
        return Map<String, dynamic>.from(response.data as Map);
      }
      _resourceError = response.data?['error']?.toString() ?? '上传失败 (${response.statusCode})';
    } catch (e) {
      _resourceError = '上传失败: $e';
    }
    return null;
  }

  /// 文档解析（仅解析，不入库）
  /// refine=true 时后端自动调用 LLM 精修标题/摘要/关键词，解析结果可直接回填表单。
  Future<Map<String, dynamic>?> parseDocument({
    required List<int> bytes,
    required String filename,
  }) async {
    try {
      final response = await _api.uploadBytes(
        '${ApiConfig.documentParse}?refine=1',
        bytes: bytes,
        filename: filename,
        fieldName: 'file',
      );
      if (response.data['code'] == 0) {
        return Map<String, dynamic>.from(response.data['data'] as Map);
      }
      _resourceError = response.data['message']?.toString() ?? '解析失败';
    } catch (e) {
      _resourceError = '解析失败: $e';
    }
    return null;
  }

  /// 文档元数据 AI 精修（标题/摘要/关键词）
  /// content 为正文；title/summary/keywords 作为精修失败时的兜底值回传。
  Future<Map<String, dynamic>?> refineDocument({
    required String content,
    String title = '',
    String summary = '',
    List<String> keywords = const [],
  }) async {
    try {
      final response = await _api.post(
        ApiConfig.documentRefine,
        data: {
          'content': content,
          'title': title,
          'summary': summary,
          'keywords': keywords,
        },
      );
      if (response.data['code'] == 0) {
        return Map<String, dynamic>.from(response.data['data'] as Map);
      }
      _resourceError = response.data['message']?.toString() ?? '精修失败';
    } catch (e) {
      _resourceError = '精修失败: $e';
    }
    return null;
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
