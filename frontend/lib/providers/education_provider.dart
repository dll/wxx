import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../services/api_service.dart';

/// 就业数据管理
class CareerProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String _error = '';
  bool get loading => _loading;
  String get error => _error;

  List<dynamic> _policies = [];
  List<dynamic> _jobs = [];
  List<dynamic> _sessions = [];
  List<dynamic> _interviewQuestions = [];

  List<dynamic> get policies => _policies;
  List<dynamic> get jobs => _jobs;
  List<dynamic> get sessions => _sessions;
  List<dynamic> get interviewQuestions => _interviewQuestions;

  Map<String, dynamic>? _policyDetail;
  Map<String, dynamic>? _jobDetail;
  Map<String, dynamic>? get policyDetail => _policyDetail;
  Map<String, dynamic>? get jobDetail => _jobDetail;

  Future<void> fetchPolicies() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.careerPolicies);
      if (res.statusCode == 200 && res.data != null) {
        _policies = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchJobs() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.careerJobs);
      if (res.statusCode == 200 && res.data != null) {
        final raw = res.data is List
            ? res.data as List
            : (res.data['data'] as List?) ?? const [];
        // 过滤非法元素，避免后端单条异常数据使整列崩溃
        _jobs = raw.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchSessions() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.careerSessions);
      if (res.statusCode == 200 && res.data != null) {
        _sessions = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchInterviewQuestions() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.careerInterviewQuestions);
      if (res.statusCode == 200 && res.data != null) {
        _interviewQuestions = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchPolicyDetail(String id) async {
    _loading = true;
    _error = '';
    _policyDetail = null;
    notifyListeners();
    try {
      final res = await _api.get('${ApiConfig.careerPolicies}/$id');
      if (res.statusCode == 200 && res.data != null) {
        _policyDetail = res.data is Map<String, dynamic>
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

  Future<void> fetchJobDetail(String id) async {
    _loading = true;
    _error = '';
    _jobDetail = null;
    notifyListeners();
    try {
      final res = await _api.get('${ApiConfig.careerJobs}/$id');
      if (res.statusCode == 200 && res.data != null) {
        _jobDetail = res.data is Map<String, dynamic>
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

  void clearDetail() {
    _policyDetail = null;
    _jobDetail = null;
  }
}

/// 学业数据管理
class StudyProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String _error = '';
  bool get loading => _loading;
  String get error => _error;

  List<dynamic> _courses = [];
  List<dynamic> _grades = [];
  Map<String, dynamic>? _gradesSummary;
  List<dynamic> _resources = [];
  List<dynamic> _exams = [];

  List<dynamic> get courses => _courses;
  List<dynamic> get grades => _grades;
  Map<String, dynamic>? get gradesSummary => _gradesSummary;
  List<dynamic> get resources => _resources;
  List<dynamic> get exams => _exams;

  Map<String, dynamic>? _courseDetail;
  Map<String, dynamic>? _resourceDetail;
  Map<String, dynamic>? get courseDetail => _courseDetail;
  Map<String, dynamic>? get resourceDetail => _resourceDetail;

  Future<void> fetchCourses() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.studyCourses);
      if (res.statusCode == 200 && res.data != null) {
        _courses = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchGrades() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.studyGrades);
      if (res.statusCode == 200 && res.data != null) {
        _grades = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchGradesSummary() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.studyGradesSummary);
      if (res.statusCode == 200 && res.data != null) {
        _gradesSummary = res.data is Map<String, dynamic>
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

  Future<void> fetchResources() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.studyResources);
      if (res.statusCode == 200 && res.data != null) {
        _resources = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchExams() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.studyExams);
      if (res.statusCode == 200 && res.data != null) {
        _exams = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchCourseDetail(String id) async {
    _loading = true;
    _error = '';
    _courseDetail = null;
    notifyListeners();
    try {
      final res = await _api.get('${ApiConfig.studyCourses}/$id');
      if (res.statusCode == 200 && res.data != null) {
        _courseDetail = res.data is Map<String, dynamic>
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

  Future<void> fetchResourceDetail(String id) async {
    _loading = true;
    _error = '';
    _resourceDetail = null;
    notifyListeners();
    try {
      final res = await _api.get('${ApiConfig.studyResources}/$id');
      if (res.statusCode == 200 && res.data != null) {
        _resourceDetail = res.data is Map<String, dynamic>
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

  void clearDetail() {
    _courseDetail = null;
    _resourceDetail = null;
  }
}

/// 心理健康数据管理
class MentalProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String _error = '';
  bool get loading => _loading;
  String get error => _error;

  List<dynamic> _scales = [];
  List<dynamic> _assessments = [];
  List<dynamic> _counselors = [];
  List<dynamic> _appointments = [];
  List<dynamic> _articles = [];
  List<dynamic> _hotlines = [];
  List<dynamic> _moodRecords = [];

  List<dynamic> get scales => _scales;
  List<dynamic> get assessments => _assessments;
  List<dynamic> get counselors => _counselors;
  List<dynamic> get appointments => _appointments;
  List<dynamic> get articles => _articles;
  List<dynamic> get hotlines => _hotlines;
  List<dynamic> get moodRecords => _moodRecords;

  Map<String, dynamic>? _scaleDetail;
  Map<String, dynamic>? _articleDetail;
  Map<String, dynamic>? get scaleDetail => _scaleDetail;
  Map<String, dynamic>? get articleDetail => _articleDetail;

  Future<void> fetchScales() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.mentalScales);
      if (res.statusCode == 200 && res.data != null) {
        _scales = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchAssessments() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.mentalAssessments);
      if (res.statusCode == 200 && res.data != null) {
        _assessments = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchCounselors() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.mentalCounselors);
      if (res.statusCode == 200 && res.data != null) {
        _counselors = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchAppointments() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.mentalAppointments);
      if (res.statusCode == 200 && res.data != null) {
        _appointments = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchArticles() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.mentalArticles);
      if (res.statusCode == 200 && res.data != null) {
        _articles = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchHotlines() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.mentalHotlines);
      if (res.statusCode == 200 && res.data != null) {
        _hotlines = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchMoodRecords() async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      final res = await _api.get(ApiConfig.mentalMood);
      if (res.statusCode == 200 && res.data != null) {
        _moodRecords = res.data is List ? res.data : (res.data['data'] as List?) ?? [];
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> fetchScaleDetail(String id) async {
    _loading = true;
    _error = '';
    _scaleDetail = null;
    notifyListeners();
    try {
      final res = await _api.get('${ApiConfig.mentalScales}/$id');
      if (res.statusCode == 200 && res.data != null) {
        _scaleDetail = res.data is Map<String, dynamic>
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

  Future<void> fetchArticleDetail(String id) async {
    _loading = true;
    _error = '';
    _articleDetail = null;
    notifyListeners();
    try {
      final res = await _api.get('${ApiConfig.mentalArticles}/$id');
      if (res.statusCode == 200 && res.data != null) {
        _articleDetail = res.data is Map<String, dynamic>
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

  Future<bool> submitMoodRecord(Map<String, dynamic> data) async {
    try {
      final res = await _api.post(ApiConfig.mentalMood, data: data);
      if (res.statusCode == 200) {
        await fetchMoodRecords();
        return true;
      }
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    }
    return false;
  }

  Future<bool> submitAppointment(Map<String, dynamic> data) async {
    try {
      final res = await _api.post(ApiConfig.mentalAppointments, data: data);
      if (res.statusCode == 200) {
        await fetchAppointments();
        return true;
      }
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    }
    return false;
  }

  Future<bool> submitAssessment(Map<String, dynamic> data) async {
    try {
      final res = await _api.post(ApiConfig.mentalAssessments, data: data);
      if (res.statusCode == 200) {
        await fetchAssessments();
        return true;
      }
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    }
    return false;
  }

  void clearDetail() {
    _scaleDetail = null;
    _articleDetail = null;
  }
}
