import 'package:flutter/foundation.dart';

import '../models/models.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';

/// 教师 AI 功能状态管理
class TeacherFeatureProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String _error = '';
  bool get loading => _loading;
  String get error => _error;

  // ── AI 今日授课概览 ──
  Map<String, dynamic>? _dailyOverview;
  Map<String, dynamic>? get dailyOverview => _dailyOverview;

  Future<void> fetchDailyOverview() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.teacherDailyOverview);
      if (res.statusCode == 200 && res.data != null) {
        _dailyOverview = res.data is Map<String, dynamic> ? res.data : {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── AI 备课助手 ──
  LessonPlan? _lessonPlan;
  LessonPlan? get lessonPlan => _lessonPlan;

  Future<void> generateLessonPlan(String topic, {String? courseId}) async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.post(ApiConfig.teacherLessonPrep, data: {'topic': topic, if (courseId != null) 'course_id': courseId});
      if (res.statusCode == 200 && res.data != null) {
        _lessonPlan = LessonPlan.fromJson(res.data is Map ? res.data : res.data['data'] ?? {});
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── AI 考试出题 ──
  Map<String, dynamic>? _examPaper;
  Map<String, dynamic>? get examPaper => _examPaper;

  Future<void> generateExam(Map<String, dynamic> params) async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.post(ApiConfig.teacherExamGen, data: params);
      if (res.statusCode == 200 && res.data != null) {
        _examPaper = res.data is Map<String, dynamic> ? res.data : {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── AI 课堂互动 ──
  Map<String, dynamic>? _interactionData;
  Map<String, dynamic>? get interactionData => _interactionData;

  Future<void> startInteraction(Map<String, dynamic> params) async {
    _loading = true;
    notifyListeners();
    try {
      final res = await _api.post(ApiConfig.teacherClassInteract, data: params);
      if (res.statusCode == 200 && res.data != null) {
        _interactionData = res.data is Map<String, dynamic> ? res.data : {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── AI 作业批改 ──
  Map<String, dynamic>? _gradingResult;
  Map<String, dynamic>? get gradingResult => _gradingResult;

  Future<void> submitGrading(Map<String, dynamic> params) async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.post(ApiConfig.teacherGrading, data: params);
      if (res.statusCode == 200 && res.data != null) {
        _gradingResult = res.data is Map<String, dynamic> ? res.data : {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 班级学情热力图 ──
  ClassHeatmapData? _heatmap;
  ClassHeatmapData? get heatmap => _heatmap;

  Future<void> fetchHeatmap({String? courseId}) async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.teacherHeatmap, params: {if (courseId != null) 'course_id': courseId});
      if (res.statusCode == 200 && res.data != null) {
        _heatmap = ClassHeatmapData.fromJson(res.data is Map ? res.data : res.data['data'] ?? {});
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── AI 教学反思 ──
  Map<String, dynamic>? _reflection;
  Map<String, dynamic>? get reflection => _reflection;

  Future<void> fetchReflection() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.teacherReflection);
      if (res.statusCode == 200 && res.data != null) {
        _reflection = res.data is Map<String, dynamic> ? res.data : {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 学生学习风格分布 ──
  Map<String, dynamic>? _styleDist;
  Map<String, dynamic>? get styleDist => _styleDist;

  Future<void> fetchStyleDistribution({String? courseId}) async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.teacherStyleDist, params: {if (courseId != null) 'course_id': courseId});
      if (res.statusCode == 200 && res.data != null) {
        _styleDist = res.data is Map<String, dynamic> ? res.data : {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }
}
