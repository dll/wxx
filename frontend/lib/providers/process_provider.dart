import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 办事流程定义状态管理（学生端动态列表 + 管理端 CRUD/审核）
class ProcessProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  List<ProcessDefinition> _definitions = [];
  bool _definitionsLoading = false;
  String? _error;
  ProcessDefinition? _current;

  List<ProcessDefinition> _adminResources = [];
  bool _adminLoading = false;
  int _adminTotal = 0;
  String _adminStatus = '';
  String _adminKeyword = '';

  List<ProcessDefinition> _pendingReviews = [];
  bool _reviewLoading = false;
  int _reviewTotal = 0;

  List<ProcessDefinition> get definitions => _definitions;
  bool get definitionsLoading => _definitionsLoading;
  String? get error => _error;
  ProcessDefinition? get current => _current;

  List<ProcessDefinition> get adminResources => _adminResources;
  bool get adminLoading => _adminLoading;
  int get adminTotal => _adminTotal;
  String get adminStatus => _adminStatus;
  String get adminKeyword => _adminKeyword;

  List<ProcessDefinition> get pendingReviews => _pendingReviews;
  bool get reviewLoading => _reviewLoading;
  int get reviewTotal => _reviewTotal;

  Future<void> loadDefinitions({bool refresh = false}) async {
    if (_definitionsLoading && !refresh) return;
    _definitionsLoading = true;
    _error = null;
    if (refresh) _definitions = [];
    notifyListeners();
    try {
      final resp = await _api.get(ApiConfig.processDefinitions,
          params: {'page': 1, 'page_size': 200});
      if (resp.data['code'] == 0 && resp.data['data'] is List) {
        _definitions = (resp.data['data'] as List)
            .map(
                (e) => ProcessDefinition.fromJson(Map<String, dynamic>.from(e)))
            .toList();
      } else {
        _error = resp.data['message'] ?? '加载办事流程失败';
      }
    } catch (e) {
      _error = '加载办事流程失败: $e';
    } finally {
      _definitionsLoading = false;
      notifyListeners();
    }
  }

  Future<ProcessDefinition?> loadDefinition(String resourceId) async {
    _error = null;
    notifyListeners();
    try {
      final resp = await _api.get(ApiConfig.processDefinition(resourceId));
      if (resp.data['code'] == 0 && resp.data['data'] is Map) {
        _current = ProcessDefinition.fromJson(
            Map<String, dynamic>.from(resp.data['data']));
        notifyListeners();
        return _current;
      }
      _error = resp.data['message'] ?? '加载流程失败';
    } catch (e) {
      _error = '加载流程失败: $e';
    }
    notifyListeners();
    return null;
  }

  Future<void> loadAdmin({bool refresh = false}) async {
    if (_adminLoading && !refresh) return;
    if (refresh) {
      _adminResources = [];
      _adminTotal = 0;
    }
    _adminLoading = true;
    _error = null;
    notifyListeners();
    try {
      final params = <String, dynamic>{
        'page': 1,
        'page_size': 100,
        if (_adminStatus.isNotEmpty) 'status': _adminStatus,
        if (_adminKeyword.isNotEmpty) 'keyword': _adminKeyword,
      };
      final resp = await _api.get(ApiConfig.processAdmin, params: params);
      if (resp.data['code'] == 0 && resp.data['data'] is List) {
        _adminResources = (resp.data['data'] as List)
            .map(
                (e) => ProcessDefinition.fromJson(Map<String, dynamic>.from(e)))
            .toList();
        _adminTotal = resp.data['total'] ?? 0;
      } else {
        _error = resp.data['message'] ?? '获取流程列表失败';
      }
    } catch (e) {
      _error = '获取流程列表失败: $e';
    } finally {
      _adminLoading = false;
      notifyListeners();
    }
  }

  void setAdminStatus(String status) {
    _adminStatus = status;
    loadAdmin(refresh: true);
  }

  void setAdminKeyword(String keyword) {
    _adminKeyword = keyword.trim();
    loadAdmin(refresh: true);
  }

  Future<void> loadPending({bool refresh = false}) async {
    if (_reviewLoading && !refresh) return;
    if (refresh) {
      _pendingReviews = [];
      _reviewTotal = 0;
    }
    _reviewLoading = true;
    _error = null;
    notifyListeners();
    try {
      final resp = await _api
          .get(ApiConfig.processPending, params: {'page': 1, 'page_size': 100});
      if (resp.data['code'] == 0 && resp.data['data'] is List) {
        _pendingReviews = (resp.data['data'] as List)
            .map(
                (e) => ProcessDefinition.fromJson(Map<String, dynamic>.from(e)))
            .toList();
        _reviewTotal = resp.data['total'] ?? 0;
      } else {
        _error = resp.data['message'] ?? '获取待审核流程失败';
      }
    } catch (e) {
      _error = '获取待审核流程失败: $e';
    } finally {
      _reviewLoading = false;
      notifyListeners();
    }
  }

  Future<bool> createProcess(Map<String, dynamic> payload) async {
    try {
      final resp = await _api.post(ApiConfig.processAdmin, data: payload);
      if (resp.data['code'] == 0) {
        await loadAdmin(refresh: true);
        return true;
      }
      _error = resp.data['message'] ?? '创建失败';
    } catch (e) {
      _error = '创建失败: $e';
    }
    notifyListeners();
    return false;
  }

  Future<bool> updateProcess(
      String resourceId, Map<String, dynamic> payload) async {
    try {
      final resp = await _api.put(ApiConfig.processAdminResource(resourceId),
          data: payload);
      if (resp.data['code'] == 0) {
        await loadAdmin(refresh: true);
        return true;
      }
      _error = resp.data['message'] ?? '更新失败';
    } catch (e) {
      _error = '更新失败: $e';
    }
    notifyListeners();
    return false;
  }

  Future<bool> deleteProcess(String resourceId) async {
    try {
      final resp =
          await _api.delete(ApiConfig.processAdminResource(resourceId));
      if (resp.data['code'] == 0) {
        await loadAdmin(refresh: true);
        return true;
      }
      _error = resp.data['message'] ?? '删除失败';
    } catch (e) {
      _error = '删除失败: $e';
    }
    notifyListeners();
    return false;
  }

  Future<bool> submitForReview(String resourceId) async {
    try {
      final resp = await _api.post(ApiConfig.processSubmit(resourceId));
      if (resp.data['code'] == 0) {
        await loadAdmin(refresh: true);
        return true;
      }
      _error = resp.data['message'] ?? '提交失败';
    } catch (e) {
      _error = '提交失败: $e';
    }
    notifyListeners();
    return false;
  }

  Future<bool> approveProcess(String resourceId) async {
    try {
      final resp = await _api.post(ApiConfig.processApprove(resourceId));
      if (resp.data['code'] == 0) {
        await loadPending(refresh: true);
        return true;
      }
      _error = resp.data['message'] ?? '审核失败';
    } catch (e) {
      _error = '审核失败: $e';
    }
    notifyListeners();
    return false;
  }

  Future<bool> rejectProcess(String resourceId, String reason) async {
    try {
      final resp = await _api
          .post(ApiConfig.processReject(resourceId), data: {'reason': reason});
      if (resp.data['code'] == 0) {
        await loadPending(refresh: true);
        return true;
      }
      _error = resp.data['message'] ?? '驳回失败';
    } catch (e) {
      _error = '驳回失败: $e';
    }
    notifyListeners();
    return false;
  }

  Future<bool> retireProcess(String resourceId) async {
    try {
      final resp = await _api.post(ApiConfig.processRetire(resourceId));
      if (resp.data['code'] == 0) {
        await loadAdmin(refresh: true);
        return true;
      }
      _error = resp.data['message'] ?? '下架失败';
    } catch (e) {
      _error = '下架失败: $e';
    }
    notifyListeners();
    return false;
  }
}
