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

/// 督办工单模型（D5-3「洞察→工单」治理回环，与后端 GovTicket 对应）
class GovTicket {
  final int id;
  final String ticketNo;
  final String title;
  final String category; // insight / supplement
  final String sourceKey;
  final String sourceDesc;
  final String dataSource;
  final String priority;
  final String status; // pending/processing/completed/closed
  final String college;
  final String assigneeRole;
  final int assigneeId;
  final String assigneeName;
  final String deadline;
  final String remark;
  final String createdAt;

  const GovTicket({
    this.id = 0,
    this.ticketNo = '',
    this.title = '',
    this.category = 'insight',
    this.sourceKey = '',
    this.sourceDesc = '',
    this.dataSource = 'not_available',
    this.priority = 'normal',
    this.status = 'pending',
    this.college = '',
    this.assigneeRole = '',
    this.assigneeId = 0,
    this.assigneeName = '',
    this.deadline = '',
    this.remark = '',
    this.createdAt = '',
  });

  factory GovTicket.fromJson(Map<String, dynamic> j) {
    return GovTicket(
      id: (j['id'] ?? 0).toInt(),
      ticketNo: j['ticket_no'] ?? '',
      title: j['title'] ?? '',
      category: j['category'] ?? 'insight',
      sourceKey: j['source_key'] ?? '',
      sourceDesc: j['source_desc'] ?? '',
      dataSource: j['data_source'] ?? 'not_available',
      priority: j['priority'] ?? 'normal',
      status: j['status'] ?? 'pending',
      college: j['college'] ?? '',
      assigneeRole: j['assignee_role'] ?? '',
      assigneeId: (j['assignee_id'] ?? 0).toInt(),
      assigneeName: j['assignee_name'] ?? '',
      deadline: j['deadline'] ?? '',
      remark: j['remark'] ?? '',
      createdAt: j['created_at'] ?? '',
    );
  }

  String get statusLabel {
    return switch (status) {
      'pending' => '待办',
      'processing' => '处理中',
      'completed' => '已完成',
      'closed' => '已关闭',
      _ => status,
    };
  }

  String get categoryLabel => category == 'supplement' ? '补料督办' : '治理督办';
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

  // 协同育人总览（书记视角，2026-08-16，蓝图第2块）
  Map<String, dynamic>? collabDashboard;
  bool collabDashboardLoading = false;

  // 育人成效 KPI 指标卡（D5-1 功能补齐，2026-08-16）：量化 KPI + 诚实数据来源标注
  List<Map<String, dynamic>> nurtureKPIs = [];
  bool nurtureKPILoading = false;

  // 督办工单（D5-3「洞察→工单」治理回环，2026-08-16）
  List<GovTicket> tickets = [];
  List<GovTicket> myTickets = [];
  bool ticketsLoading = false;
  Map<String, int> ticketStats = {};

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

  /// 拉取协同育人总览（书记视角，2026-08-16，蓝图第2块）
  Future<void> fetchCollabDashboard() async {
    collabDashboardLoading = true;
    collabDashboard = null;
    error = '';
    notifyListeners();
    final data = await _guard(() async {
      final r = await _api.get('${ApiConfig.apiPrefix}/college/collab-dashboard');
      final body = r.data;
      if (body is Map && body['data'] != null) return body['data'] as Map<String, dynamic>;
      return null;
    });
    collabDashboard = data;
    collabDashboardLoading = false;
    notifyListeners();
  }

  /// 拉取育人成效 KPI 指标卡（D5-1 功能补齐，2026-08-16）。
  /// 列表每项：{ key, label, value, unit, data_source(real/not_available), source_desc, upload_target, upload_hint }。
  /// real → 显示真实数值；not_available → 显示「数据待补充」+ 上传入口（不伪造数字）。
  Future<void> fetchNurtureKPI() async {
    nurtureKPILoading = true;
    nurtureKPIs = [];
    error = '';
    notifyListeners();
    try {
      final r = await _api.get('${ApiConfig.apiPrefix}/college/nurture-kpi');
      final body = r.data;
      if (body is Map && body['list'] is List) {
        nurtureKPIs = (body['list'] as List)
            .map((e) => Map<String, dynamic>.from(e as Map))
            .toList();
      }
    } catch (e) {
      error = e.toString().replaceAll('Exception: ', '');
    }
    nurtureKPILoading = false;
    notifyListeners();
  }

  /// 上传支撑材料（复制 kb 上传心智：由人工补料后转 real，绝不本地伪造数字）。
  /// 返回知识库上传目标（upload_target=kb 时前端据此渲染「上传材料」按钮）。
  String nurtureUploadTarget() => '${ApiConfig.apiPrefix}/kb/upload';

  // ════════ 督办工单（D5-3「洞察→工单」治理回环，2026-08-16）════════
  /// 从育人 KPI 生成补料督办工单（D5-1 联动）：仅对 data_source=not_available 且
  /// upload_target=kb 的真实缺失指标生成，后端校验，绝不伪造数字。
  /// 返回 (是否成功, 工单 id/错误信息)。
  Future<({bool ok, int id, String msg})> createTicketFromKPI({
    required String kpiKey,
    String? ownerId,
    String priority = 'normal',
    int assigneeId = 0,
    String assigneeName = '',
    String assigneeRole = '',
  }) async {
    try {
      final r = await _api.post('${ApiConfig.apiPrefix}/college/tickets/from-kpi', data: {
        'kpi_key': kpiKey,
        'owner_id': ownerId ?? '',
        'priority': priority,
        if (assigneeId > 0)
          'assignee': {
            'assignee_id': assigneeId,
            'assignee_name': assigneeName,
            'assignee_role': assigneeRole,
          },
      });
      final body = r.data;
      if (body is Map && (body['code'] ?? -1) == 0) {
        final newId = ((body['id'] ?? 0) as num).toInt();
        return (ok: true, id: newId, msg: (body['message'] ?? '').toString());
      }
      return (ok: false, id: 0, msg: (body?['message'] ?? '').toString());
    } catch (e) {
      return (ok: false, id: 0, msg: e.toString().replaceAll('Exception: ', ''));
    }
  }

  /// 拉取督办工单列表（书记/学院管理端）。
  Future<void> fetchTickets({String status = '', String category = ''}) async {
    ticketsLoading = true;
    error = '';
    notifyListeners();
    final params = <String, dynamic>{};
    if (status.isNotEmpty) params['status'] = status;
    if (category.isNotEmpty) params['category'] = category;
    try {
      final r = await _api.get('${ApiConfig.apiPrefix}/college/tickets',
          params: params.isEmpty ? null : params);
      final body = r.data;
      if (body is Map && body['list'] is List) {
        tickets = (body['list'] as List)
            .map((e) => GovTicket.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList();
      }
    } catch (e) {
      error = e.toString().replaceAll('Exception: ', '');
    }
    ticketsLoading = false;
    notifyListeners();
  }

  /// 拉取分派给本人的督办工单（辅导员/教辅/党群责任人视角）。
  Future<void> fetchMyTickets() async {
    ticketsLoading = true;
    error = '';
    notifyListeners();
    try {
      final r = await _api.get('${ApiConfig.apiPrefix}/college/tickets/mine');
      final body = r.data;
      if (body is Map && body['list'] is List) {
        myTickets = (body['list'] as List)
            .map((e) => GovTicket.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList();
      }
    } catch (e) {
      error = e.toString().replaceAll('Exception: ', '');
    }
    ticketsLoading = false;
    notifyListeners();
  }

  /// 分派/改派责任人（书记）：需 assignee_id>0，name/role 落库为责任人展示。
  Future<({bool ok, String msg})> assignTicket({
    required int id,
    required int assigneeId,
    required String assigneeName,
    required String assigneeRole,
    String deadline = '',
  }) async {
    try {
      final r = await _api.put('${ApiConfig.apiPrefix}/college/tickets/$id/assign',
          data: {
            'assignee_id': assigneeId,
            'assignee_name': assigneeName,
            'assignee_role': assigneeRole,
            'deadline': deadline,
          });
      final body = r.data;
      if (body is Map && (body['code'] ?? -1) == 0) {
        return (ok: true, msg: (body['message'] ?? '').toString());
      }
      return (ok: false, msg: (body?['message'] ?? '分派失败').toString());
    } catch (e) {
      return (ok: false, msg: e.toString().replaceAll('Exception: ', ''));
    }
  }

  /// 推进督办工单状态（责任人推进本人分派，书记可任意）。
  Future<({bool ok, String msg})> updateTicketStatus({
    required int id,
    required String status,
    String detail = '',
    bool asManager = false,
  }) async {
    try {
      final path = asManager
          ? '${ApiConfig.apiPrefix}/college/tickets/$id/status'
          : '${ApiConfig.apiPrefix}/college/tickets/mine/$id/status';
      final r = await _api.put(path, data: {'status': status, 'detail': detail});
      final body = r.data;
      if (body is Map && (body['code'] ?? -1) == 0) {
        return (ok: true, msg: (body['message'] ?? '').toString());
      }
      return (ok: false, msg: (body?['message'] ?? '更新失败').toString());
    } catch (e) {
      return (ok: false, msg: e.toString().replaceAll('Exception: ', ''));
    }
  }

  /// 督办总览（书记/学院管理端）。
  Future<Map<String, int>?> fetchTicketStats() async {
    try {
      final r = await _api.get('${ApiConfig.apiPrefix}/college/tickets/stats');
      final body = r.data;
      if (body is Map && body['stats'] is Map) {
        ticketStats = (body['stats'] as Map)
            .map((k, v) => MapEntry(k.toString(), (v as num).toInt()));
        notifyListeners();
        return ticketStats;
      }
    } catch (e) {
      error = e.toString().replaceAll('Exception: ', '');
    }
    return null;
  }

  /// 登记党课/活动（教师/教辅，蓝图第3块）。studentIds 空=未指定具体学生
  Future<dynamic> registerPartyRecord({
    required String title,
    required String studyType,
    String content = '',
    int duration = 0,
    required String studyDate,
    List<int>? studentIds,
  }) async {
    try {
      final r = await _api.post('${ApiConfig.apiPrefix}/teacher/party/register', data: {
        'title': title,
        'study_type': studyType,
        'content': content,
        'duration': duration,
        'study_date': studyDate,
        'student_ids': studentIds ?? [],
      });
      return r.data;
    } catch (e) {
      error = e.toString().replaceAll('Exception: ', '');
      notifyListeners();
      return null;
    }
  }

  /// 拉取本人的党课/活动登记（教师/教辅）
  Future<List<dynamic>?> fetchMyPartyRecords() async {
    try {
      final r = await _api.get('${ApiConfig.apiPrefix}/teacher/party/records');
      final body = r.data;
      if (body is Map && body['list'] is List) return body['list'] as List;
      return null;
    } catch (e) {
      error = e.toString().replaceAll('Exception: ', '');
      notifyListeners();
      return null;
    }
  }

  /// 删除本人的党课/活动登记
  Future<dynamic> deletePartyRecord(int id) async {
    try {
      final r = await _api.delete('${ApiConfig.apiPrefix}/teacher/party/records/$id');
      return r.data;
    } catch (e) {
      error = e.toString().replaceAll('Exception: ', '');
      notifyListeners();
      return null;
    }
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
