import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../services/api_service.dart';

/// 学习计划与校历课表数据管理
///
/// 管理：当前校历、课表、计划列表、计划详情（含任务）、各维度概览。
/// 同时提供创建/更新/删除计划、添加任务、更新任务状态、AI 一键生成等操作。
class StudyPlanProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String _error = '';
  bool get loading => _loading;
  String get error => _error;

  /// 细分操作状态：用于按钮 loading、避免与列表 loading 互相干扰
  bool _mutating = false;
  bool get mutating => _mutating;

  bool _aiGenerating = false;
  bool get aiGenerating => _aiGenerating;

  // ── 校历 ──
  Map<String, dynamic>? _calendarData;
  Map<String, dynamic>? get calendarData => _calendarData;

  // ── 课表 ──
  Map<String, dynamic>? _timetable;
  Map<String, dynamic>? get timetable => _timetable;
  List<dynamic> get schedule =>
      _timetable == null ? [] : (_timetable!['schedule'] as List?) ?? [];

  // ── 计划列表（按 plan_type 缓存）──
  /// key: plan_type（weekly/monthly/...），value: 该类型下的计划列表
  final Map<String, List<dynamic>> _plansByType = {};
  List<dynamic> plansOf(String type) => _plansByType[type] ?? [];

  // ── 当前选中的 plan_type（用于 Tab 切换）──
  String _currentType = 'weekly';
  String get currentType => _currentType;
  void setCurrentType(String type) {
    _currentType = type;
    notifyListeners();
  }

  // ── 计划详情（含 plan 与 tasks）──
  Map<String, dynamic>? _currentPlan;
  Map<String, dynamic>? get currentPlan =>
      _currentPlan == null ? null : _currentPlan!['plan'] as Map<String, dynamic>?;
  List<dynamic> get currentTasks =>
      _currentPlan == null ? [] : (_currentPlan!['tasks'] as List?) ?? [];

  // ── 各维度概览 ──
  Map<String, dynamic>? _overview;
  Map<String, dynamic>? get overview => _overview;

  // ── 校历 ──
  Future<void> fetchCalendar() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.studyCalendarCurrent);
      if (res.statusCode == 200 && res.data != null) {
        _calendarData = res.data is Map<String, dynamic>
            ? res.data
            : (res.data['data'] as Map<String, dynamic>?) ?? {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// 按学期代码加载校历
  Future<void> fetchCalendarByCode(String code) async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.studyCalendar(code));
      if (res.statusCode == 200 && res.data != null) {
        _calendarData = res.data is Map<String, dynamic>
            ? res.data
            : (res.data['data'] as Map<String, dynamic>?) ?? {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 课表 ──
  Future<void> fetchTimetable({String? semesterCode}) async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final params = semesterCode == null ? null : {'semester_code': semesterCode};
      final res = await _api.get(ApiConfig.studyTimetable, params: params);
      if (res.statusCode == 200 && res.data != null) {
        _timetable = res.data is Map<String, dynamic>
            ? res.data
            : (res.data['data'] as Map<String, dynamic>?) ?? {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 计划列表 ──
  Future<void> fetchPlans(String planType, {bool force = true}) async {
    if (!force && _plansByType.containsKey(planType)) return;
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(
        ApiConfig.studyPlans,
        params: {'plan_type': planType},
      );
      if (res.statusCode == 200 && res.data != null) {
        final list = res.data is List
            ? res.data
            : (res.data['plans'] as List?) ??
                (res.data['data'] as List?) ??
                [];
        _plansByType[planType] = list;
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 计划详情 ──
  Future<void> fetchPlanDetail(String id) async {
    _loading = true;
    _error = '';
    _currentPlan = null;
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.studyPlanDetail(id));
      if (res.statusCode == 200 && res.data != null) {
        _currentPlan = res.data is Map<String, dynamic>
            ? res.data
            : (res.data['data'] as Map<String, dynamic>?) ?? {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 概览 ──
  Future<void> fetchOverview() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.studyPlansOverview);
      if (res.statusCode == 200 && res.data != null) {
        _overview = res.data is Map<String, dynamic>
            ? res.data
            : (res.data['data'] as Map<String, dynamic>?) ?? {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 创建计划 ──
  Future<bool> createPlan(Map<String, dynamic> data) async {
    _mutating = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.post(ApiConfig.studyPlans, data: data);
      if (res.statusCode == 200 || res.statusCode == 201) {
        // 创建后刷新对应类型的列表与概览
        final type = (data['plan_type'] as String?) ?? _currentType;
        await fetchPlans(type, force: true);
        await fetchOverview();
        return true;
      }
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    } finally {
      _mutating = false;
      notifyListeners();
    }
    return false;
  }

  // ── 更新计划 ──
  Future<bool> updatePlan(String id, Map<String, dynamic> data) async {
    _mutating = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.put(ApiConfig.studyPlanDetail(id), data: data);
      if (res.statusCode == 200) {
        await fetchPlanDetail(id);
        final type = (data['plan_type'] as String?) ??
            currentPlan?['plan_type'] as String? ??
            _currentType;
        await fetchPlans(type, force: true);
        await fetchOverview();
        return true;
      }
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    } finally {
      _mutating = false;
      notifyListeners();
    }
    return false;
  }

  // ── 删除计划 ──
  Future<bool> deletePlan(String id, {String? planType}) async {
    _mutating = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.delete(ApiConfig.studyPlanDetail(id));
      if (res.statusCode == 200 || res.statusCode == 204) {
        final type = planType ?? _currentType;
        await fetchPlans(type, force: true);
        await fetchOverview();
        return true;
      }
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    } finally {
      _mutating = false;
      notifyListeners();
    }
    return false;
  }

  // ── 添加任务 ──
  Future<bool> addTask(String planId, Map<String, dynamic> data) async {
    _mutating = true;
    _error = '';
    notifyListeners();
    try {
      // POST /study/plans/:id/tasks —— 复用 plans/:id 子资源（约定与详情同级）
      final res = await _api.post(
        '${ApiConfig.studyPlanDetail(planId)}/tasks',
        data: data,
      );
      if (res.statusCode == 200 || res.statusCode == 201) {
        await fetchPlanDetail(planId);
        return true;
      }
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    } finally {
      _mutating = false;
      notifyListeners();
    }
    return false;
  }

  // ── 更新任务状态 ──
  Future<bool> updateTaskStatus(
    String planId,
    String taskId, {
    required String status,
    int? actualDuration,
    String? reflection,
  }) async {
    _mutating = true;
    _error = '';
    notifyListeners();
    try {
      final body = <String, dynamic>{'status': status};
      if (actualDuration != null) body['actual_duration'] = actualDuration;
      if (reflection != null) body['reflection'] = reflection;
      final res = await _api.patch(
        ApiConfig.studyPlanTask(planId, taskId),
        data: body,
      );
      if (res.statusCode == 200) {
        await fetchPlanDetail(planId);
        // 同步刷新该计划所属类型的列表（进度会变化）
        final type = currentPlan?['plan_type'] as String? ?? _currentType;
        await fetchPlans(type, force: true);
        await fetchOverview();
        return true;
      }
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    } finally {
      _mutating = false;
      notifyListeners();
    }
    return false;
  }

  // ── AI 一键生成计划 ──
  ///
  /// 入参示例：
  /// ```json
  /// {"plan_type":"weekly","start_date":"...","end_date":"...","hint":"..."}
  /// ```
  /// 返回 AI 生成的计划草稿（含 title/goals_json/tasks），由调用方填表后提交。
  Future<Map<String, dynamic>?> aiGeneratePlan(Map<String, dynamic> data) async {
    _aiGenerating = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.post(ApiConfig.studyPlanAIGenerate, data: data);
      if (res.statusCode == 200 && res.data != null) {
        return res.data is Map<String, dynamic>
            ? res.data
            : (res.data['data'] as Map<String, dynamic>?);
      }
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    } finally {
      _aiGenerating = false;
      notifyListeners();
    }
    return null;
  }

  /// 清理详情缓存（详情页退出时调用）
  void clearDetail() {
    _currentPlan = null;
    notifyListeners();
  }

  /// 清理全部缓存（用户切换账号时可选调用）
  void clearAll() {
    _calendarData = null;
    _timetable = null;
    _plansByType.clear();
    _currentPlan = null;
    _overview = null;
    _error = '';
    _loading = false;
    _mutating = false;
    _aiGenerating = false;
    notifyListeners();
  }
}
