import 'package:flutter/foundation.dart';

import '../models/models.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';

/// 学生 AI 功能状态管理
class StudentFeatureProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  // ── 通用状态 ──
  bool _loading = false;
  String _error = '';
  bool get loading => _loading;
  String get error => _error;

  // ── 今日速览 ──
  DailyBriefing? _briefing;
  DailyBriefing? get briefing => _briefing;

  Future<void> fetchDailyBriefing() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.dailyBriefing);
      if (res.statusCode == 200 && res.data != null) {
        _briefing = DailyBriefing.fromJson(res.data is Map ? res.data : res.data['data'] ?? {});
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 学习日记 ──
  LearningDiary? _diary;
  LearningDiary? get diary => _diary;

  Future<void> fetchLearningDiary() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.learningDiary);
      if (res.statusCode == 200 && res.data != null) {
        _diary = LearningDiary.fromJson(res.data is Map ? res.data : res.data['data'] ?? {});
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 打卡 ──
  CheckinRecord? _checkin;
  CheckinRecord? get checkin => _checkin;

  Future<void> fetchCheckin() async {
    try {
      final res = await _api.get(ApiConfig.checkinHistory);
      if (res.statusCode == 200 && res.data != null) {
        _checkin = CheckinRecord.fromJson(res.data is Map ? res.data : res.data['data'] ?? {});
      }
    } catch (e) {
      _error = e.toString();
    }
    notifyListeners();
  }

  Future<bool> doCheckin() async {
    try {
      final res = await _api.post(ApiConfig.checkin, data: {});
      if (res.statusCode == 200) {
        await fetchCheckin();
        return true;
      }
    } catch (e) {
      _error = e.toString();
    }
    return false;
  }

  // ── 数字孪生 ──
  DigitalTwinData? _twin;
  DigitalTwinData? get twin => _twin;

  Future<void> fetchDigitalTwin() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.digitalTwin);
      if (res.statusCode == 200 && res.data != null) {
        _twin = DigitalTwinData.fromJson(res.data is Map ? res.data : res.data['data'] ?? {});
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 积分成就 ──
  AchievementData? _achievements;
  AchievementData? get achievements => _achievements;

  Future<void> fetchAchievements() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.achievements);
      if (res.statusCode == 200 && res.data != null) {
        _achievements = AchievementData.fromJson(res.data is Map ? res.data : res.data['data'] ?? {});
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 课程地图（知识图谱） ──
  Map<String, dynamic>? _courseGraph;
  Map<String, dynamic>? get courseGraph => _courseGraph;
  List<CourseNode> _courseNodes = [];
  List<CourseNode> get courseNodes => _courseNodes;

  Future<void> fetchCourseMap() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.courseMap);
      if (res.statusCode == 200 && res.data != null) {
        // 后端返回 KnowledgeGraph 对象 {course_name,nodes,edges}；兜底返回数组
        if (res.data is Map) {
          final raw = res.data as Map;
          final data = raw['data'] is Map ? raw['data'] as Map : raw;
          _courseGraph = Map<String, dynamic>.from(data);
          final nodes = (data['nodes'] as List?)?.cast<Map<String, dynamic>>() ?? [];
          _courseNodes = nodes.map((e) => CourseNode.fromJson(e)).toList();
        } else {
          _courseGraph = null;
          final list = res.data['data'] ?? res.data;
          _courseNodes = (list as List).map((e) => CourseNode.fromJson(e)).toList();
        }
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 课程学情 ──
  List<CourseAnalyticsData> _courseAnalytics = [];
  List<CourseAnalyticsData> get courseAnalytics => _courseAnalytics;
  Map<String, dynamic>? _courseAnalyticsSummary;
  Map<String, dynamic>? get courseAnalyticsSummary => _courseAnalyticsSummary;

  Future<void> fetchCourseAnalytics() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.courseAnalytics);
      if (res.statusCode == 200 && res.data != null) {
        // 真实路径返回 CourseAnalyticsResult 对象（含 courses 数组）；兜底返回数组
        if (res.data is Map) {
          final raw = res.data as Map;
          final data = raw['data'] is Map ? raw['data'] as Map : raw;
          _courseAnalyticsSummary = Map<String, dynamic>.from(data);
          final courses = (data['courses'] as List?)?.cast<Map<String, dynamic>>() ?? [];
          _courseAnalytics = courses.map((e) => CourseAnalyticsData.fromJson(e)).toList();
        } else {
          _courseAnalyticsSummary = null;
          final list = res.data['data'] ?? res.data;
          _courseAnalytics = (list as List).map((e) => CourseAnalyticsData.fromJson(e)).toList();
        }
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 性格洞察 ──
  Map<String, dynamic>? _personality;
  Map<String, dynamic>? get personality => _personality;

  Future<void> fetchPersonality() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.personalityInsight);
      if (res.statusCode == 200 && res.data != null) {
        _personality = res.data is Map<String, dynamic> ? res.data : (res.data['data'] as Map<String, dynamic>?) ?? {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 学习周报 ──
  Map<String, dynamic>? _weeklyReport;
  Map<String, dynamic>? get weeklyReport => _weeklyReport;

  Future<void> fetchWeeklyReport() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.weeklyReport);
      if (res.statusCode == 200 && res.data != null) {
        _weeklyReport = res.data is Map<String, dynamic> ? res.data : (res.data['data'] as Map<String, dynamic>?) ?? {};
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 通用 AI 对话请求（用于多个 AI 功能） ──
  String _aiResponse = '';
  String get aiResponse => _aiResponse;
  bool _aiLoading = false;
  bool get aiLoading => _aiLoading;

  Future<String> askAI(String endpoint, {Map<String, dynamic>? data}) async {
    _aiLoading = true;
    _aiResponse = '';
    notifyListeners();
    try {
      final res = await _api.get(endpoint);
      if (res.statusCode == 200 && res.data != null) {
        _aiResponse = res.data is String ? res.data : (res.data['response'] ?? res.data['content'] ?? '');
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _aiLoading = false;
      notifyListeners();
    }
    return _aiResponse;
  }

  // ── 问答广场（结构化） ──
  List<Map<String, dynamic>> _qaQuestions = [];
  List<Map<String, dynamic>> get qaQuestions => _qaQuestions;

  Future<void> fetchQAPlaza() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.qaPlaza);
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map ? res.data : (res.data['data'] ?? {});
        _qaQuestions = (data['hot_questions'] as List?)?.cast<Map<String, dynamic>>() ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// 发布问题（真实落库）
  Future<bool> createQAPost(String title, String content, String category) async {
    try {
      final res = await _api.post(ApiConfig.qaPosts,
          data: {'title': title, 'content': content, 'category': category});
      if (res.statusCode == 200) {
        await fetchQAPlaza();
        return true;
      }
    } catch (e) {
      _error = e.toString();
    }
    return false;
  }

  // ── 热点关注（结构化） ──
  List<Map<String, dynamic>> _hotTopics = [];
  List<Map<String, dynamic>> get hotTopics => _hotTopics;

  Future<void> fetchHotTopics() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.hotTopics);
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map ? res.data : (res.data['data'] ?? {});
        _hotTopics = (data['topics'] as List?)?.cast<Map<String, dynamic>>() ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 问答排行榜（结构化） ──
  Map<String, dynamic>? _leaderboard;
  Map<String, dynamic>? get leaderboard => _leaderboard;

  Future<void> fetchQALeaderboard() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.qaLeaderboard);
      if (res.statusCode == 200 && res.data != null) {
        _leaderboard = res.data is Map<String, dynamic>
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

  // ── 成长路径（结构化） ──
  Map<String, dynamic>? _growthPath;
  Map<String, dynamic>? get growthPath => _growthPath;

  Future<void> fetchGrowthPath() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.growthPath);
      if (res.statusCode == 200 && res.data != null) {
        _growthPath = res.data is Map<String, dynamic>
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
}
