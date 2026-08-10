import 'package:flutter/foundation.dart';
import 'package:dio/dio.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 办事流程学生端状态管理（动态定义 + 步骤进度持久化）
class EnrollmentProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  List<ProcessDefinition> _definitions = [];
  bool _definitionsLoading = false;
  final Set<String> _roleFilter = {};
  String? _audienceFilter;

  String _flowType = 'process-registration-2026';
  bool _loading = false;
  String? _error;
  AnswerCard? _answerCard;
  List<ProcessStepDetail> _stepDetails = [];
  List<ProcessReminder> _reminders = [];
  String _recordId = '';
  String _recordStatus = 'in_progress';
  final Set<int> _completedSteps = {};

  List<ProcessDefinition> get definitions => _definitions;
  bool get definitionsLoading => _definitionsLoading;
  Set<String> get roleFilter => Set.unmodifiable(_roleFilter);
  String? get audienceFilter => _audienceFilter;
  String get flowType => _flowType;
  bool get loading => _loading;
  String? get error => _error;
  AnswerCard? get answerCard => _answerCard;
  List<String> get steps {
    if (_stepDetails.isNotEmpty) {
      return _stepDetails
          .map((s) => s.title)
          .where((s) => s.isNotEmpty)
          .toList();
    }
    return _answerCard?.steps ?? [];
  }

  List<ProcessStepDetail> get stepDetails => _stepDetails;
  List<ProcessReminder> get reminders => _reminders;
  Set<int> get completedSteps => Set.unmodifiable(_completedSteps);
  String get recordId => _recordId;
  String get recordStatus => _recordStatus;

  int get totalSteps {
    if (_stepDetails.isNotEmpty) return _stepDetails.length;
    return steps.length;
  }

  int get completedCount => _completedSteps.length;
  double get progress => totalSteps > 0 ? completedCount / totalSteps : 0;

  static const Map<String, String> _legacyFlowMap = {
    'enrollment': 'process-registration-2026',
    'graduation': 'process-graduation-2026',
    'major_change': 'process-major-change-2026',
    'student_loan': 'process-student-loan-2026',
    'leave': 'process-leave-2026',
    'scholarship': 'process-scholarship-2026',
  };

  Future<void> loadCatalog({bool refresh = false}) async {
    if (_definitionsLoading && !refresh) return;
    _definitionsLoading = true;
    notifyListeners();
    try {
      final resp = await _api.get(ApiConfig.processDefinitions,
          params: {'page': 1, 'page_size': 200});
      final data = resp.data;
      Map<String, dynamic>? body;
      if (data is Map<String, dynamic>) {
        if (data['data'] is List) body = data;
      }
      if (body != null && body['data'] is List) {
        _definitions = (body['data'] as List)
            .map(
                (e) => ProcessDefinition.fromJson(Map<String, dynamic>.from(e)))
            .toList();
        if (_definitions.isNotEmpty &&
            !_definitions.any((d) => d.resourceId == _flowType)) {
          final freshmen =
              _definitions.where((d) => d.isFreshmenRelated).toList();
          _flowType =
              (freshmen.isNotEmpty ? freshmen.first : _definitions.first)
                  .resourceId;
        }
      }
    } catch (e) {
      _error = '加载办事流程列表失败: $e';
    } finally {
      _definitionsLoading = false;
      notifyListeners();
    }
  }

  /// 按角色过滤（多选，再次点击取消）
  void toggleRoleFilter(String role) {
    if (!_roleFilter.add(role)) _roleFilter.remove(role);
    notifyListeners();
  }

  /// 按面向群体过滤（单选，传 null 清除）
  void setAudienceFilter(String? audience) {
    _audienceFilter = audience;
    notifyListeners();
  }

  void clearFilters() {
    _roleFilter.clear();
    _audienceFilter = null;
    notifyListeners();
  }

  /// 组合过滤后的流程列表
  List<ProcessDefinition> get filteredDefinitions {
    return _definitions.where((d) {
      if (_audienceFilter != null && d.audienceLabel != _audienceFilter) {
        return false;
      }
      if (_roleFilter.isNotEmpty &&
          d.roleCodes.isNotEmpty &&
          !d.roleCodes.any(_roleFilter.contains)) {
        return false;
      }
      return true;
    }).toList();
  }

  /// 切换流程；兼容旧的 flow_type 名（enrollment/graduation 等）和资源 ID
  void setFlowType(String type) {
    final id = _legacyFlowMap[type] ?? type;
    if (id == _flowType) {
      loadFlow();
      return;
    }
    _flowType = id;
    _answerCard = null;
    _stepDetails = [];
    _reminders = [];
    _completedSteps.clear();
    _recordId = '';
    _recordStatus = 'in_progress';
    _error = null;
    notifyListeners();
    loadFlow();
  }

  Future<void> loadFlow([String? overrideId]) async {
    if (overrideId != null && overrideId != _flowType) {
      _flowType = overrideId;
    }
    if (_definitions.isEmpty) {
      await loadCatalog();
    }
    _loading = true;
    _error = null;
    notifyListeners();

    try {
      final resp = await _api.get(ApiConfig.processDefinition(_flowType));
      final data = resp.data;
      Map<String, dynamic>? body;
      if (data is Map<String, dynamic>) {
        if (data['data'] is Map) body = data;
      }
      if (body == null || body['data'] is! Map) {
        _error = '流程数据不存在或未发布';
        _loading = false;
        notifyListeners();
        return;
      }

      final def = ProcessDefinition.fromJson(
          Map<String, dynamic>.from(body['data'] as Map));
      _answerCard = AnswerCard(
        conclusion:
            def.summary.isEmpty ? '${def.title}已整理，请按下列步骤办理。' : def.summary,
        steps: def.steps.map((s) => s.title).toList(),
        stepDetails: def.steps.map((s) => s.toDetail()).toList(),
        sources: [
          Source(
            resourceId: def.resourceId,
            title: def.title,
            resourceType: def.resourceType,
            version: def.version,
            sourceLink: def.sourceLink,
            snippet: def.summary,
            summary: def.summary,
          ),
        ],
        risks: const [],
      );
      _stepDetails = def.steps.map((s) => s.toDetail()).toList();
      _reminders = def.reminders.where((r) => r.isEnabled).toList();
      await _restoreFromBackend(def);
      _loading = false;
      notifyListeners();
    } catch (e) {
      if (e is DioException && e.response?.statusCode == 403) {
        _error = '暂无权限访问，请先登录';
      } else {
        _error = '加载流程失败：网络异常，请检查后重试';
      }
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> _restoreFromBackend(ProcessDefinition def) async {
    try {
      final resp = await _api.post(
        ApiConfig.processRecordStart(_flowType),
        data: {
          'flow_label': def.title,
          'total_steps': def.steps.length,
        },
      );
      if (resp.data['code'] == 0 && resp.data['data'] != null) {
        final rec = ProcessRecord.fromJson(resp.data['data']);
        _recordId = rec.recordId;
        _recordStatus = rec.status;
        _completedSteps
          ..clear()
          ..addAll(rec.completedSteps);
      }
    } catch (_) {
      // 恢复进度失败不阻塞主流程
    }
  }

  Future<void> _persistProgress() async {
    if (_flowType.isEmpty) return;
    try {
      await _api.post(
        ApiConfig.processRecordProgress(_flowType),
        data: {
          'current_step': _completedSteps.isEmpty
              ? 0
              : (_completedSteps.reduce((a, b) => a > b ? a : b) + 1),
          'completed_steps': _completedSteps.toList()..sort(),
        },
      );
    } catch (_) {
      // 静默失败，本地状态保留
    }
  }

  Future<void> toggleStep(int index) async {
    if (_completedSteps.contains(index)) {
      _completedSteps.remove(index);
    } else {
      _completedSteps.add(index);
    }
    notifyListeners();
    await _persistProgress();
  }

  Future<void> completeAll() async {
    for (var i = 0; i < totalSteps; i++) {
      _completedSteps.add(i);
    }
    notifyListeners();
    await _persistProgress();
  }

  Future<void> resetProgress() async {
    _completedSteps.clear();
    notifyListeners();
    await _persistProgress();
  }
}
