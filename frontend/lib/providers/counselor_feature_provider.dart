import 'package:flutter/foundation.dart';

import '../models/models.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';
import '../utils/api_error.dart';

/// 辅导员 AI 功能状态管理
class CounselorFeatureProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String _error = '';
  bool get loading => _loading;
  String get error => _error;

  // ── AI 今日关注 ──
  DailyFocusData? _dailyFocus;
  DailyFocusData? get dailyFocus => _dailyFocus;

  Future<void> fetchDailyFocus() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.counselorDailyFocus);
      if (res.statusCode == 200 && res.data != null) {
        _dailyFocus = DailyFocusData.fromJson(res.data is Map ? res.data : res.data['data'] ?? {});
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 班级学情日报 ──
  ClassReportData? _classReport;
  ClassReportData? get classReport => _classReport;

  Future<void> fetchClassReport() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.counselorClassReport);
      if (res.statusCode == 200 && res.data != null) {
        _classReport = ClassReportData.fromJson(res.data is Map ? res.data : res.data['data'] ?? {});
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 学生数字孪生看板 ──
  List<Map<String, dynamic>> _twinBoard = [];
  List<Map<String, dynamic>> get twinBoard => _twinBoard;

  Future<void> fetchTwinBoard() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.counselorTwinBoard);
      if (res.statusCode == 200 && res.data != null) {
        final list = res.data is List ? res.data : res.data['data'] ?? [];
        _twinBoard = List<Map<String, dynamic>>.from(list);
      }
    } catch (e) {
      _error = friendlyApiError(e);
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 预测性预警 ──
  List<Map<String, dynamic>> _predictions = [];
  List<Map<String, dynamic>> get predictions => _predictions;

  Future<void> fetchPredictions() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.counselorPrediction);
      if (res.statusCode == 200 && res.data != null) {
        final list = res.data is List ? res.data : res.data['data'] ?? [];
        _predictions = List<Map<String, dynamic>>.from(list);
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 干预方案 ──
  Map<String, dynamic>? _intervention;
  Map<String, dynamic>? get intervention => _intervention;

  Future<void> generateIntervention(String studentId) async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.post(ApiConfig.counselorIntervention, data: {'student_id': studentId});
      if (res.statusCode == 200 && res.data != null) {
        _intervention = res.data is Map<String, dynamic> ? res.data : {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 谈心谈话记录 ──
  List<TalkRecord> _talkRecords = [];
  List<TalkRecord> get talkRecords => _talkRecords;

  Future<void> fetchTalkRecords() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.counselorTalkRecord);
      if (res.statusCode == 200 && res.data != null) {
        final list = res.data is List ? res.data : res.data['data'] ?? [];
        _talkRecords = (list as List).map((e) => TalkRecord.fromJson(e)).toList();
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<bool> saveTalkRecord(Map<String, dynamic> data) async {
    try {
      final res = await _api.post(ApiConfig.counselorTalkRecord, data: data);
      if (res.statusCode == 200) {
        await fetchTalkRecords();
        return true;
      }
    } catch (e) {
      _error = e.toString();
    }
    return false;
  }

  // ── 谈话话术推荐 ──
  List<String> _talkTips = [];
  List<String> get talkTips => _talkTips;

  Future<void> fetchTalkTips({String scene = '', String studentType = ''}) async {
    _loading = true;
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.counselorTalkTips, params: {'scene': scene, 'type': studentType});
      if (res.statusCode == 200 && res.data != null) {
        final list = res.data is List ? res.data : res.data['tips'] ?? [];
        _talkTips = List<String>.from(list);
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 学生思想档案 ──
  Map<String, dynamic>? _ideologicalData;
  Map<String, dynamic>? get ideologicalData => _ideologicalData;

  Future<void> fetchIdeological() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.counselorIdeological);
      if (res.statusCode == 200 && res.data != null) {
        _ideologicalData = res.data is Map<String, dynamic> ? res.data : {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 班级性格画像 ──
  Map<String, dynamic>? _classProfile;
  Map<String, dynamic>? get classProfile => _classProfile;

  Future<void> fetchClassProfile() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.counselorClassProfile);
      if (res.statusCode == 200 && res.data != null) {
        _classProfile = res.data is Map<String, dynamic> ? res.data : {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 学生列表 ──
  List<Map<String, dynamic>> _studentList = [];
  List<Map<String, dynamic>> get studentList => _studentList;

  Future<void> fetchStudentList() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.counselorStudentList);
      if (res.statusCode == 200 && res.data != null) {
        final list = res.data is List ? res.data : res.data['students'] ?? [];
        _studentList = List<Map<String, dynamic>>.from(list);
      }
    } catch (e) {
      _error = friendlyApiError(e);
    } finally {
      _loading = false;
      notifyListeners();
    }
  }
}
