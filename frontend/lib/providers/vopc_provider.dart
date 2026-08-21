import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import '../config/api_config.dart';
import '../services/api_service.dart';

abstract class VopcApiClient {
  Future<Response> get(String path);
  Future<Response> post(String path, {dynamic data});
  Future<Response> put(String path, {dynamic data});
}

class _VopcApiServiceClient implements VopcApiClient {
  final ApiService _api;

  _VopcApiServiceClient(this._api);

  @override
  Future<Response> get(String path) => _api.get(path);

  @override
  Future<Response> post(String path, {dynamic data}) =>
      _api.post(path, data: data);

  @override
  Future<Response> put(String path, {dynamic data}) =>
      _api.put(path, data: data);
}

class VopcProject {
  final int id;
  final String name;
  final String summary;
  final String stage;
  final String status;
  final String projectType;
  final String riskLevel;
  final String problem;
  final String targetUsers;
  final String expectedOutcome;
  final String validationPlan;
  final String productForm;
  final String projectCycle;
  final String acceptanceCriteria;
  final int ownerUserId;

  const VopcProject(
      {required this.id,
      required this.name,
      required this.summary,
      required this.stage,
      required this.status,
      required this.projectType,
      required this.riskLevel,
      this.problem = '',
      this.targetUsers = '',
      this.expectedOutcome = '',
      this.validationPlan = '',
      this.productForm = '',
      this.projectCycle = '',
      this.acceptanceCriteria = '',
      this.ownerUserId = 0});

  factory VopcProject.fromJson(Map<String, dynamic> json) => VopcProject(
      id: (json['id'] as num).toInt(),
      name: json['name'] ?? '',
      summary: json['summary'] ?? '',
      stage: json['stage'] ?? '',
      status: json['status'] ?? '',
      projectType: json['project_type'] ?? '',
      riskLevel: json['risk_level'] ?? '',
      problem: json['problem_statement'] ?? '',
      targetUsers: json['target_users'] ?? '',
      expectedOutcome: json['expected_outcome'] ?? '',
      validationPlan: json['validation_plan'] ?? '',
      productForm: json['product_form'] ?? '',
      projectCycle: json['project_cycle'] ?? '',
      acceptanceCriteria: json['acceptance_criteria'] ?? '',
      ownerUserId: (json['owner_user_id'] as num?)?.toInt() ?? 0);
}

class VopcDecision {
  final int id;
  final String title;
  final String background;
  final String options;
  final String decision;
  final String rationale;
  final String status;
  final int? decidedBy;
  final String createdAt;
  final String? decidedAt;
  const VopcDecision(
      {required this.id,
      required this.title,
      required this.background,
      required this.options,
      required this.decision,
      required this.rationale,
      required this.status,
      required this.decidedBy,
      required this.createdAt,
      required this.decidedAt});
  factory VopcDecision.fromJson(Map<String, dynamic> j) => VopcDecision(
      id: (j['id'] as num).toInt(),
      title: j['title']?.toString() ?? '',
      background: j['background']?.toString() ?? '',
      options: j['options']?.toString() ?? '',
      decision: j['decision']?.toString() ?? '',
      rationale: j['rationale']?.toString() ?? '',
      status: j['status']?.toString() ?? 'pending',
      decidedBy: (j['decided_by'] as num?)?.toInt(),
      createdAt: j['created_at']?.toString() ?? '',
      decidedAt: j['decided_at']?.toString());
}

class VopcTask {
  final int id;
  final String title;
  final String description;
  final int? assigneeUserId;
  final String? assigneeAiRole;
  final String acceptanceCriteria;
  final String priority;
  final String status;
  final String? dueAt;

  const VopcTask({
    required this.id,
    required this.title,
    required this.description,
    required this.assigneeUserId,
    required this.assigneeAiRole,
    required this.acceptanceCriteria,
    required this.priority,
    required this.status,
    required this.dueAt,
  });

  factory VopcTask.fromJson(Map<String, dynamic> json) => VopcTask(
        id: (json['id'] as num).toInt(),
        title: json['title']?.toString() ?? '',
        description: json['description']?.toString() ?? '',
        assigneeUserId: (json['assignee_user_id'] as num?)?.toInt(),
        assigneeAiRole: json['assignee_ai_role']?.toString(),
        acceptanceCriteria: json['acceptance_criteria']?.toString() ?? '',
        priority: json['priority']?.toString() ?? 'normal',
        status: json['status']?.toString() ?? 'todo',
        dueAt: json['due_at']?.toString(),
      );
}

class VopcArtifact {
  final int id;
  final String name;
  final String type;
  final int versionCount;
  const VopcArtifact(this.id, this.name, this.type, this.versionCount);
  factory VopcArtifact.fromJson(Map<String, dynamic> j) => VopcArtifact(
      (j['id'] as num).toInt(),
      j['name']?.toString() ?? '',
      j['artifact_type']?.toString() ?? '',
      (j['version_count'] as num?)?.toInt() ?? 0);
}

class VopcMilestoneSubmission {
  final int id;
  final String stage;
  final String evidence;
  final String status;
  final int? reviewerUserId;
  const VopcMilestoneSubmission(
      this.id, this.stage, this.evidence, this.status, this.reviewerUserId);
  factory VopcMilestoneSubmission.fromJson(Map<String, dynamic> j) =>
      VopcMilestoneSubmission(
          (j['id'] as num).toInt(),
          j['stage']?.toString() ?? '',
          j['evidence']?.toString() ?? '',
          j['status']?.toString() ?? '',
          (j['reviewer_user_id'] as num?)?.toInt());
}

class VopcProvider extends ChangeNotifier {
  final VopcApiClient _api;
  VopcProvider([VopcApiClient? api])
      : _api = api ?? _VopcApiServiceClient(ApiService());

  List<VopcProject> projects = const [];
  VopcProject? detail;
  List<VopcTask> tasks = const [];
  List<VopcDecision> decisions = const [];
  List<Map<String, dynamic>> members = const [];
  List<Map<String, dynamic>> invitations = const [];
  List<VopcArtifact> artifacts = const [];
  List<VopcMilestoneSubmission> milestoneSubmissions = const [];
  bool decisionsLoading = false;
  bool decisionMutating = false;
  bool tasksLoading = false;
  bool taskMutating = false;
  bool loading = false;
  bool accessChecked = false;
  bool allowed = false;
  String? error;
  int? statusCode;

  Future<bool> checkAccess() async {
    loading = true;
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final response = await _api.get(ApiConfig.vopcAccess);
      allowed = response.statusCode == 200 &&
          response.data?['data']?['allowed'] == true;
      accessChecked = true;
      if (!allowed) error = '当前账号未获 vOPC 准入';
      return allowed;
    } catch (e) {
      _setError(e, '准入校验失败');
      allowed = false;
      accessChecked = true;
      return false;
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  Future<void> loadProjects() async {
    loading = true;
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final response = await _api.get(ApiConfig.vopcProjects);
      final data = response.data?['data'] as List? ?? const [];
      projects = data
          .map((e) => VopcProject.fromJson(Map<String, dynamic>.from(e)))
          .toList();
    } catch (e) {
      _setError(e, '项目列表加载失败');
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  Future<int?> createProject(Map<String, dynamic> data) async {
    error = null;
    statusCode = null;
    try {
      final response = await _api.post(ApiConfig.vopcProjects, data: data);
      final id = (response.data?['data']?['id'] as num?)?.toInt();
      if (id != null) await loadProjects();
      return id;
    } catch (e) {
      _setError(e, '项目创建失败');
      notifyListeners();
      return null;
    }
  }

  Future<bool> advanceProject(int projectId, String currentStage,
      {String evidence = '', String reviewNote = ''}) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final current = int.tryParse(currentStage.replaceFirst('S', '')) ?? -1;
      if (current < 0 || current >= 9) return false;
      final path = current == 0
          ? ApiConfig.vopcProjectSubmit(projectId)
          : ApiConfig.vopcProjectAdvance(projectId, 'S${current + 1}');
      await _api
          .post(path, data: {'evidence': evidence, 'review_note': reviewNote});
      await loadDetail(projectId);
      await loadProjects();
      return true;
    } catch (e) {
      _setError(e, '项目阶段推进失败');
      notifyListeners();
      return false;
    }
  }

  Future<void> loadDetail(int id) async {
    loading = true;
    detail = null;
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final response = await _api.get(ApiConfig.vopcProject(id));
      detail = VopcProject.fromJson(
          Map<String, dynamic>.from(response.data['data']));
      await loadTasks(id);
      await loadDecisions(id);
      await loadMembers(id);
      await loadArtifacts(id);
      await loadMilestoneSubmissions(id);
    } catch (e) {
      _setError(e, '项目详情加载失败');
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  Future<void> loadDecisions(int projectId) async {
    decisionsLoading = true;
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final response =
          await _api.get(ApiConfig.vopcProjectDecisions(projectId));
      final data = response.data?['data'] as List? ?? const [];
      decisions = data
          .map(
              (e) => VopcDecision.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();
    } catch (e) {
      _setError(e, '决策列表加载失败');
    } finally {
      decisionsLoading = false;
      notifyListeners();
    }
  }

  Future<bool> createDecision(int projectId, Map<String, dynamic> data) async {
    decisionMutating = true;
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final response = await _api
          .post(ApiConfig.vopcProjectDecisions(projectId), data: data);
      final id = (response.data?['data']?['id'] as num?)?.toInt();
      if (id == null) {
        error = '决策创建响应无效';
        return false;
      }
      await loadDecisions(projectId);
      return true;
    } catch (e) {
      _setError(e, '决策创建失败');
      return false;
    } finally {
      decisionMutating = false;
      notifyListeners();
    }
  }

  Future<bool> actDecision(int projectId, int decisionId, String action,
      {String? decision, String? rationale}) async {
    decisionMutating = true;
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final response = await _api
          .put(ApiConfig.vopcProjectDecision(projectId, decisionId), data: {
        'action': action,
        if (decision != null) 'decision': decision,
        if (rationale != null) 'rationale': rationale
      });
      if (response.data?['data']?['status'] == null) {
        error = '决策处理响应无效';
        return false;
      }
      await loadDecisions(projectId);
      return true;
    } catch (e) {
      _setError(e, '决策处理失败');
      return false;
    } finally {
      decisionMutating = false;
      notifyListeners();
    }
  }

  Future<void> loadMembers(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcProjectMembers(projectId));
      members = (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      _setError(e, '成员列表加载失败');
    }
    notifyListeners();
  }

  Future<bool> inviteMember(
      int projectId, int userId, String role, String message) async {
    try {
      final r = await _api.post(ApiConfig.vopcProjectMembers(projectId), data: {
        'user_id': userId,
        'project_role': role,
        'message': message,
      });
      if (r.data?['data']?['id'] == null) return false;
      return true;
    } catch (e) {
      _setError(e, '邀请成员失败');
      notifyListeners();
      return false;
    }
  }

  Future<void> loadInvitations() async {
    try {
      final r = await _api.get(ApiConfig.vopcInvitations);
      invitations = (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      _setError(e, '邀请列表加载失败');
    }
    notifyListeners();
  }

  Future<bool> respondInvitation(int id, String action) async {
    try {
      final r = await _api
          .post(ApiConfig.vopcInvitationRespond(id), data: {'action': action});
      if (r.data?['data']?['status'] == null) return false;
      await loadInvitations();
      return true;
    } catch (e) {
      _setError(e, '邀请处理失败');
      notifyListeners();
      return false;
    }
  }

  Future<void> loadArtifacts(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcProjectArtifacts(projectId));
      artifacts = (r.data?['data'] as List? ?? const [])
          .map(
              (e) => VopcArtifact.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();
    } catch (e) {
      _setError(e, '成果仓库加载失败');
    }
    notifyListeners();
  }

  Future<bool> createArtifact(int projectId, Map<String, dynamic> data) async {
    try {
      final r = await _api.post(ApiConfig.vopcProjectArtifacts(projectId),
          data: data);
      if (r.data?['data']?['id'] == null) return false;
      await loadArtifacts(projectId);
      return true;
    } catch (e) {
      _setError(e, '成果创建失败');
      notifyListeners();
      return false;
    }
  }

  Future<bool> createArtifactVersion(
      int projectId, int artifactId, Map<String, dynamic> data) async {
    try {
      final r = await _api.post(
          ApiConfig.vopcArtifactVersions(projectId, artifactId),
          data: data);
      if (r.data?['data']?['id'] == null) return false;
      await loadArtifacts(projectId);
      return true;
    } catch (e) {
      _setError(e, '成果版本创建失败');
      notifyListeners();
      return false;
    }
  }

  Future<void> loadMilestoneSubmissions(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcMilestoneSubmissions(projectId));
      milestoneSubmissions = (r.data?['data'] as List? ?? const [])
          .map((e) => VopcMilestoneSubmission.fromJson(
              Map<String, dynamic>.from(e as Map)))
          .toList();
    } catch (e) {
      _setError(e, '里程碑材料加载失败');
    }
    notifyListeners();
  }

  Future<bool> submitMilestone(int projectId, Map<String, dynamic> data) async {
    try {
      final r = await _api.post(ApiConfig.vopcMilestoneSubmissions(projectId),
          data: data);
      if (r.data?['data']?['id'] == null) return false;
      await loadMilestoneSubmissions(projectId);
      return true;
    } catch (e) {
      _setError(e, '里程碑提交失败');
      notifyListeners();
      return false;
    }
  }

  Future<bool> reviewMilestone(
      int projectId, int submissionId, String result, String note) async {
    try {
      final r = await _api.post(
          ApiConfig.vopcMilestoneReview(projectId, submissionId),
          data: {'result': result, 'note': note});
      if (r.data?['data']?['status'] == null) return false;
      await loadMilestoneSubmissions(projectId);
      await loadDetail(projectId);
      return true;
    } catch (e) {
      _setError(e, '里程碑评审失败');
      notifyListeners();
      return false;
    }
  }

  Future<void> loadTasks(int projectId) async {
    tasksLoading = true;
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final response = await _api.get(ApiConfig.vopcProjectTasks(projectId));
      final data = response.data?['data'] as List? ?? const [];
      tasks = data
          .map((e) => VopcTask.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();
    } catch (e) {
      _setError(e, '任务列表加载失败');
    } finally {
      tasksLoading = false;
      notifyListeners();
    }
  }

  Future<bool> createTask(int projectId, Map<String, dynamic> data) async {
    taskMutating = true;
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final response = await _api.post(
        ApiConfig.vopcProjectTasks(projectId),
        data: data,
      );
      final id = (response.data?['data']?['id'] as num?)?.toInt();
      if (id == null) {
        error = '任务创建响应无效';
        return false;
      }
      await loadTasks(projectId);
      return true;
    } catch (e) {
      _setError(e, '任务创建失败');
      return false;
    } finally {
      taskMutating = false;
      notifyListeners();
    }
  }

  Future<bool> updateTaskStatus(
      int projectId, int taskId, String status) async {
    taskMutating = true;
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final response = await _api.put(
        ApiConfig.vopcProjectTask(projectId, taskId),
        data: {'status': status},
      );
      if (response.data?['data']?['status'] != status) {
        error = '任务状态更新响应无效';
        return false;
      }
      await loadTasks(projectId);
      return true;
    } catch (e) {
      _setError(e, '任务状态更新失败');
      return false;
    } finally {
      taskMutating = false;
      notifyListeners();
    }
  }

  void _setError(Object value, String fallback) {
    if (value is DioException) {
      statusCode = value.response?.statusCode;
      error = value.response?.data?['message'] ??
          _messageFor(statusCode) ??
          fallback;
    } else {
      error = fallback;
    }
  }

  String? _messageFor(int? code) => switch (code) {
        401 => '登录已过期，请重新登录',
        403 => '没有访问该功能或项目的权限',
        404 => '项目不存在或无权访问',
        409 => '当前任务状态或项目阶段不允许此操作，请刷新后重试',
        422 => '提交内容未通过校验',
        500 => '服务暂时不可用，请稍后重试',
        _ => null,
      };
}
