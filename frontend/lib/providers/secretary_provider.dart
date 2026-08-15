import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../services/api_service.dart';

/// 毕业去向记录模型（与后端 GraduationOutcome 对应）
class OutcomeRecord {
  final int id;
  final int studentId;
  final String studentName;
  final String college;
  final String major;
  final int graduateYear;
  final String outcomeType;
  final String employerName;
  final String position;
  final String remark;
  final String status; // pending/approved/rejected
  final String submittedRole;
  final String reviewedName;
  final String reviewNote;

  const OutcomeRecord({
    this.id = 0,
    this.studentId = 0,
    this.studentName = '',
    this.college = '',
    this.major = '',
    this.graduateYear = 0,
    this.outcomeType = '',
    this.employerName = '',
    this.position = '',
    this.remark = '',
    this.status = 'pending',
    this.submittedRole = '',
    this.reviewedName = '',
    this.reviewNote = '',
  });

  factory OutcomeRecord.fromJson(Map<String, dynamic> j) {
    return OutcomeRecord(
      id: j['id'] ?? 0,
      studentId: (j['student_id'] ?? 0).toInt(),
      studentName: j['student_name'] ?? '',
      college: j['college'] ?? '',
      major: j['major'] ?? '',
      graduateYear: (j['graduate_year'] ?? 0).toInt(),
      outcomeType: j['outcome_type'] ?? '',
      employerName: j['employer_name'] ?? '',
      position: j['position'] ?? '',
      remark: j['remark'] ?? '',
      status: j['status'] ?? 'pending',
      submittedRole: j['submitted_role'] ?? '',
      reviewedName: j['reviewed_name'] ?? '',
      reviewNote: j['review_note'] ?? '',
    );
  }
}

/// 书记教育成果 Provider：毕业去向登记/审核 + 教育成果大屏
class SecretaryProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool loading = false;
  String error = '';

  // 教育成果大屏数据
  Map<String, dynamic>? dashboard;
  bool dashboardLoading = false;

  // 党建育人聚合看板（书记接线）
  Map<String, dynamic>? partyDashboard;
  bool partyDashboardLoading = false;

  // 毕业去向列表
  List<OutcomeRecord> outcomes = [];
  bool outcomesLoading = false;
  int pendingCount = 0;

  // 去向类型元信息
  Map<String, dynamic> outcomeTypes = {};

  Future<Map<String, dynamic>?> _guard(Future<Map<String, dynamic>?> Function() fn) async {
    try {
      return await fn();
    } catch (e) {
      error = e.toString().replaceAll('Exception: ', '');
      notifyListeners();
      return null;
    }
  }

  /// 拉取去向类型下拉
  Future<void> fetchOutcomeTypes() async {
    final resp = await _guard(() async {
      final r = await _api.get('${ApiConfig.apiPrefix}/assistant/outcome/meta');
      return r.data;
    });
    if (resp != null && resp['outcome_types'] is Map) {
      outcomeTypes = Map<String, dynamic>.from(resp['outcome_types'] as Map);
      notifyListeners();
    }
  }

  /// 拉取教育成果大屏（college 空=全校）
  Future<void> fetchDashboard({String college = ''}) async {
    dashboardLoading = true;
    dashboard = null;
    error = '';
    notifyListeners();
    final data = await _guard(() async {
      final r = await _api.get('${ApiConfig.apiPrefix}/college/education-outcome',
          params: college.isEmpty ? null : {'college': college});
      final body = r.data;
      if (body is Map && body['data'] != null) return body['data'] as Map<String, dynamic>;
      return null;
    });
    dashboard = data;
    dashboardLoading = false;
    notifyListeners();
  }

  /// 拉取党建育人聚合看板（owner_id 空=全校，非空=本院；后端自动按角色归属）
  Future<void> fetchPartyDashboard() async {
    partyDashboardLoading = true;
    partyDashboard = null;
    error = '';
    notifyListeners();
    final data = await _guard(() async {
      final r = await _api.get('${ApiConfig.apiPrefix}/college/party-dashboard');
      final body = r.data;
      if (body is Map && body['data'] != null) return body['data'] as Map<String, dynamic>;
      return null;
    });
    partyDashboard = data;
    partyDashboardLoading = false;
    notifyListeners();
  }

  /// 拉取毕业去向列表
  Future<void> fetchOutcomes({String? status, String? college, int? studentId}) async {
    outcomesLoading = true;
    error = '';
    notifyListeners();
    final params = <String, dynamic>{};
    if (status != null && status.isNotEmpty) params['status'] = status;
    if (college != null && college.isNotEmpty) params['college'] = college;
    if (studentId != null && studentId > 0) params['student_id'] = studentId;
    final data = await _guard(() async {
      final r = await _api.get('${ApiConfig.apiPrefix}/assistant/outcome/records',
          params: params.isEmpty ? null : params);
      return r.data;
    });
    if (data != null && data['list'] is List) {
      outcomes = (data['list'] as List)
          .map((e) => OutcomeRecord.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();
    }
    outcomesLoading = false;
    notifyListeners();
  }

  /// 拉取待审核数
  Future<void> fetchPendingCount() async {
    final data = await _guard(() async {
      final r = await _api.get('${ApiConfig.apiPrefix}/assistant/outcome/pending');
      return r.data;
    });
    if (data != null && data['pending'] != null) {
      pendingCount = (data['pending'] as num).toInt();
      notifyListeners();
    }
  }

  /// 学生自报 / 教辅录入毕业去向
  Future<String?> submitOutcome({
    required int studentId,
    String studentName = '',
    String college = '',
    String major = '',
    int graduateYear = 0,
    required String outcomeType,
    String employerName = '',
    String position = '',
    String remark = '',
  }) async {
    final data = await _guard(() async {
      final r = await _api.post('${ApiConfig.apiPrefix}/assistant/outcome/record', data: {
        'student_id': studentId,
        'student_name': studentName,
        'college': college,
        'major': major,
        'graduate_year': graduateYear,
        'outcome_type': outcomeType,
        'employer_name': employerName,
        'position': position,
        'remark': remark,
      });
      return r.data;
    });
    if (data == null) return error.isNotEmpty ? error : '提交失败';
    return null; // null=成功
  }

  /// 学生自报（走 /student/outcome/self-report）
  Future<String?> selfReport({
    required String outcomeType,
    String employerName = '',
    String position = '',
    String remark = '',
  }) async {
    final data = await _guard(() async {
      final r = await _api.post('${ApiConfig.apiPrefix}/student/outcome/self-report', data: {
        'outcome_type': outcomeType,
        'employer_name': employerName,
        'position': position,
        'remark': remark,
      });
      return r.data;
    });
    if (data == null) return error.isNotEmpty ? error : '提交失败';
    return null;
  }

  /// 教辅审核
  Future<String?> reviewOutcome(int id, String status, String note) async {
    final data = await _guard(() async {
      final r = await _api.put('${ApiConfig.apiPrefix}/assistant/outcome/review/$id',
          data: {'status': status, 'note': note});
      return r.data;
    });
    if (data == null) return error.isNotEmpty ? error : '审核失败';
    return null;
  }
}
