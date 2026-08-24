import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import '../config/api_config.dart';
import '../services/api_service.dart';

abstract class VopcApiClient {
  Future<Response> get(String path, {Map<String, dynamic>? params});
  Future<Response> post(String path, {dynamic data});
  Future<Response> put(String path, {dynamic data});
  Future<Response> delete(String path);
  /// 以字节流接收（用于私有文件鉴权下载）。
  Future<Response> getBytes(String path);
  /// 上传单个文件（multipart，字段名 file）。
  Future<Response> uploadFile(String path,
      {required List<int> bytes, required String filename});
}

class _VopcApiServiceClient implements VopcApiClient {
  final ApiService _api;

  _VopcApiServiceClient(this._api);

  @override
  Future<Response> get(String path, {Map<String, dynamic>? params}) =>
      _api.get(path, params: params);

  @override
  Future<Response> post(String path, {dynamic data}) =>
      _api.post(path, data: data);

  @override
  Future<Response> put(String path, {dynamic data}) =>
      _api.put(path, data: data);

  @override
  Future<Response> delete(String path) => _api.delete(path);

  @override
  Future<Response> getBytes(String path) => _api.getBytes(path);

  @override
  Future<Response> uploadFile(String path,
          {required List<int> bytes, required String filename}) =>
      _api.uploadBytes(path,
          bytes: bytes, filename: filename, fieldName: 'file');
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
  final String projectSource;
  final String dataType;
  final String mentorNeeds;
  final String resourceNeeds;
  final bool realUserTrial;
  final bool externalPublish;
  final bool fundsInvolved;
  final int ownerUserId;
  final bool canManage;
  final int complexityLayer;
  final String visibility;

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
      this.projectSource = 'self_proposed',
      this.dataType = '公开数据',
      this.mentorNeeds = '',
      this.resourceNeeds = '',
      this.realUserTrial = false,
      this.externalPublish = false,
      this.fundsInvolved = false,
      this.ownerUserId = 0,
      this.canManage = false,
      this.complexityLayer = 2,
      this.visibility = 'private'});

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
      projectSource: json['project_source'] ?? 'self_proposed',
      dataType: json['data_type'] ?? '公开数据',
      mentorNeeds: json['mentor_needs'] ?? '',
      resourceNeeds: json['resource_needs'] ?? '',
      realUserTrial: json['real_user_trial'] == true,
      externalPublish: json['external_publish'] == true,
      fundsInvolved: json['funds_involved'] == true,
      ownerUserId: (json['owner_user_id'] as num?)?.toInt() ?? 0,
      canManage: json['can_manage'] == true,
      complexityLayer:
          (json['complexity_layer'] as num?)?.toInt() ?? 2,
      visibility: json['visibility']?.toString() ?? 'private');
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

/// 虚拟向导任务（v2.0 四态审阅载体）。
class VopcAITask {
  final int id;
  final String roleKey;
  final String model;
  final String status;
  final String outputContent;
  final String? finalDecision;
  final String decisionNote;
  final String revision;
  final double modificationRate;
  const VopcAITask({
    required this.id,
    required this.roleKey,
    required this.model,
    required this.status,
    required this.outputContent,
    required this.finalDecision,
    required this.decisionNote,
    required this.revision,
    required this.modificationRate,
  });
  factory VopcAITask.fromJson(Map<String, dynamic> j) => VopcAITask(
        id: (j['id'] as num).toInt(),
        roleKey: j['role_key']?.toString() ?? '',
        model: j['model']?.toString() ?? '',
        status: j['status']?.toString() ?? '',
        outputContent: j['output_content']?.toString() ?? '',
        finalDecision: j['final_decision']?.toString(),
        decisionNote: j['decision_note']?.toString() ?? '',
        revision: j['revision']?.toString() ?? '',
        modificationRate: (j['modification_rate'] as num?)?.toDouble() ?? 0.0,
      );
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
  List<Map<String, dynamic>> aiRoles = const [];
  List<Map<String, dynamic>> invitations = const [];
  List<VopcArtifact> artifacts = const [];
  List<VopcMilestoneSubmission> milestoneSubmissions = const [];
  // v2.0 新增：L1 概念层内容与 L2 虚拟向导模板
  Map<String, dynamic> learning = const {};
  Map<String, dynamic> guides = const {};
  // v2.0 L3 治理层数据（结项记录/风险/专项审批/豁免/甲方证据/虚拟向导任务/评分量表/私有文件）
  List<Map<String, dynamic>> closeRecords = const [];
  List<Map<String, dynamic>> risks = const [];
  List<Map<String, dynamic>> riskAppeals = const [];
  List<Map<String, dynamic>> specialApprovals = const [];
  List<Map<String, dynamic>> milestoneWaivers = const [];
  List<Map<String, dynamic>> clientEvidence = const [];
  List<Map<String, dynamic>> rubrics = const [];
  List<Map<String, dynamic>> files = const [];
  List<VopcAITask> aiTasks = const [];
  // B1 成果与复盘聚合（跨项目汇总）
  Map<String, dynamic> outcomesSummary = const {};
  bool outcomesLoading = false;
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

  Future<bool> updateProject(int id, Map<String, dynamic> data) async {
    error = null;
    statusCode = null;
    try {
      await _api.put(ApiConfig.vopcProject(id), data: data);
      await loadDetail(id);
      await loadProjects();
      return true;
    } catch (e) {
      _setError(e, '草稿保存失败');
      notifyListeners();
      return false;
    }
  }

  /// 删除项目（仅 S0 草稿可删）。
  Future<bool> deleteProject(int id) async {
    error = null;
    statusCode = null;
    try {
      await _api.post(ApiConfig.vopcProjectDelete(id));
      await loadProjects();
      return true;
    } catch (e) {
      _setError(e, '项目删除失败');
      notifyListeners();
      return false;
    }
  }

  /// 搜索 vOPC 内可邀用户。
  Future<List<Map<String, dynamic>>> searchUsers(String keyword,
      {int excludeProjectId = 0}) async {
    try {
      final response = await _api.get(ApiConfig.vopcUsersSearch, params: {
        'q': keyword,
        if (excludeProjectId > 0) 'exclude': excludeProjectId,
      });
      final list = (response.data?['data'] as List?) ?? const [];
      return list.cast<Map<String, dynamic>>();
    } catch (e) {
      error = '成员查找失败';
      statusCode = 0;
      notifyListeners();
      return const [];
    }
  }

  Future<bool> submitProject(int projectId) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      await _api.post(ApiConfig.vopcProjectSubmit(projectId));
      await loadDetail(projectId);
      await loadProjects();
      return true;
    } catch (e) {
      _setError(e, '项目立项提交失败');
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

  Future<void> loadAIRoles(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcAiRoles(projectId));
      aiRoles = (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      // AI 岗位加载失败不阻塞页面
    }
    notifyListeners();
  }

  /// L1 概念层内容（知识卡 + 核心流程图 + 自测问卷）。零项目依赖。
  Future<void> loadLearning() async {
    try {
      final r = await _api.get(ApiConfig.vopcLearning);
      learning = Map<String, dynamic>.from(r.data?['data'] as Map? ?? {});
    } catch (e) {
      // 概念层内容加载失败不阻塞页面
    }
    notifyListeners();
  }

  /// L2 虚拟向导模板 + 角色扮演提示。
  Future<void> loadGuides() async {
    try {
      final r = await _api.get(ApiConfig.vopcGuides);
      guides = Map<String, dynamic>.from(r.data?['data'] as Map? ?? {});
    } catch (e) {
      // 向导模板加载失败不阻塞页面
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

  Future<List<Map<String, dynamic>>> loadArtifactVersions(
      int projectId, int artifactId) async {
    try {
      final r =
          await _api.get(ApiConfig.vopcArtifactVersions(projectId, artifactId));
      return (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      _setError(e, '成果版本加载失败');
      notifyListeners();
      return const [];
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

  /// B1 成果与复盘：跨项目汇总 artifacts 数量与 close-records（结项/复盘）。
  /// 复用已有 projects 列表，逐个读 artifacts + close-records，失败项目最佳努力跳过。
  Future<void> loadOutcomesSummary() async {
    outcomesLoading = true;
    notifyListeners();
    var totalArtifacts = 0;
    var totalVersions = 0;
    final closeItems = <Map<String, dynamic>>[];
    final projectNames = <int, String>{};
    for (final prj in projects) {
      projectNames[prj.id] = prj.name;
      try {
        final r = await _api.get(ApiConfig.vopcProjectArtifacts(prj.id));
        final arts = (r.data?['data'] as List? ?? const [])
            .map((e) => Map<String, dynamic>.from(e as Map))
            .toList();
        totalArtifacts += arts.length;
        for (final a in arts) {
          totalVersions += (a['version_count'] as num?)?.toInt() ?? 0;
        }
      } catch (_) {
        // 单个项目读取失败跳过
      }
      try {
        final rr = await _api.get(ApiConfig.vopcProjectCloseRecords(prj.id));
        final records = (rr.data?['data'] as List? ?? const [])
            .map((e) => Map<String, dynamic>.from(e as Map))
            .toList();
        for (final rec in records) {
          closeItems.add({...rec, 'project_name': prj.name});
        }
      } catch (_) {
        // 单个项目关闭记录读取失败跳过
      }
    }
    outcomesSummary = {
      'artifact_count': totalArtifacts,
      'version_count': totalVersions,
      'close_count': closeItems.length,
      'close_records': closeItems,
    };
    outcomesLoading = false;
    notifyListeners();
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

  // ── v2.0 L3 治理层：结项/风险/里程碑门禁/私有文件/虚拟向导 ──

  /// 加载结项与异常状态记录。
  Future<void> loadCloseRecords(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcProjectCloseRecords(projectId));
      closeRecords = (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      // 结项记录加载失败不阻塞工作台
    }
    notifyListeners();
  }

  /// 结项/终止/暂停/转向/归档统一入口。
  Future<bool> closeProject(int projectId, Map<String, dynamic> data) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.post(ApiConfig.vopcProjectClose(projectId), data: data);
      if (r.data?['data']?['status'] == null) return false;
      await loadCloseRecords(projectId);
      await loadDetail(projectId);
      await loadProjects();
      return true;
    } catch (e) {
      _setError(e, '项目状态流转失败');
      notifyListeners();
      return false;
    }
  }

  Future<void> loadRisks(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcProjectRisks(projectId));
      risks = (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      // 风险列表加载失败不阻塞
    }
    notifyListeners();
  }

  Future<bool> createRisk(int projectId, Map<String, dynamic> data) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.post(ApiConfig.vopcProjectRisks(projectId), data: data);
      if (r.data?['data']?['id'] == null) return false;
      await loadRisks(projectId);
      return true;
    } catch (e) {
      _setError(e, '风险登记失败');
      notifyListeners();
      return false;
    }
  }

  Future<bool> approveRisk(
      int projectId, int riskId, String decision, String reason) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.post(ApiConfig.vopcRiskApprove(projectId, riskId),
          data: {'decision': decision, 'reason': reason});
      if (r.data?['data']?['status'] == null) return false;
      await loadRisks(projectId);
      return true;
    } catch (e) {
      _setError(e, '风险审批失败');
      notifyListeners();
      return false;
    }
  }

  Future<void> loadSpecialApprovals(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcSpecialApprovals(projectId));
      specialApprovals = (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      // 专项审批加载失败不阻塞
    }
    notifyListeners();
  }

  Future<bool> createSpecialApproval(
      int projectId, Map<String, dynamic> data) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api
          .post(ApiConfig.vopcSpecialApprovals(projectId), data: data);
      if (r.data?['data']?['id'] == null) return false;
      await loadSpecialApprovals(projectId);
      return true;
    } catch (e) {
      _setError(e, '专项审批登记失败');
      notifyListeners();
      return false;
    }
  }

  Future<bool> freezeProject(
      int projectId, String action, String reason) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.post(ApiConfig.vopcProjectFreeze(projectId),
          data: {'action': action, 'reason': reason});
      if (r.data?['data']?['status'] == null) return false;
      await loadDetail(projectId);
      await loadProjects();
      return true;
    } catch (e) {
      _setError(e, '项目冻结/解冻失败');
      notifyListeners();
      return false;
    }
  }

  Future<void> loadRiskAppeals(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcRiskAppeals(projectId));
      riskAppeals = (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      // 申诉列表加载失败不阻塞
    }
    notifyListeners();
  }

  Future<bool> createRiskAppeal(int projectId, String reason) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.post(ApiConfig.vopcRiskAppeals(projectId),
          data: {'reason': reason});
      if (r.data?['data']?['id'] == null) return false;
      await loadRiskAppeals(projectId);
      return true;
    } catch (e) {
      _setError(e, '风险申诉提交失败');
      notifyListeners();
      return false;
    }
  }

  Future<bool> resolveRiskAppeal(
      int projectId, int appealId, String decision, String resolution) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.post(
          ApiConfig.vopcRiskAppealResolve(projectId, appealId),
          data: {'decision': decision, 'resolution': resolution});
      if (r.data?['data']?['status'] == null) return false;
      await loadRiskAppeals(projectId);
      return true;
    } catch (e) {
      _setError(e, '申诉裁定失败');
      notifyListeners();
      return false;
    }
  }

  Future<void> loadMilestoneWaivers(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcMilestoneWaivers(projectId));
      milestoneWaivers = (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      // 豁免列表加载失败不阻塞
    }
    notifyListeners();
  }

  Future<bool> createMilestoneWaiver(
      int projectId, Map<String, dynamic> data) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api
          .post(ApiConfig.vopcMilestoneWaivers(projectId), data: data);
      if (r.data?['data']?['id'] == null) return false;
      await loadMilestoneWaivers(projectId);
      return true;
    } catch (e) {
      _setError(e, '豁免申请失败');
      notifyListeners();
      return false;
    }
  }

  Future<bool> reviewMilestoneWaiver(
      int projectId, int waiverId, String action, String note) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.post(
          ApiConfig.vopcMilestoneWaiverReview(projectId, waiverId),
          data: {'action': action, 'note': note});
      if (r.data?['data']?['status'] == null) return false;
      await loadMilestoneWaivers(projectId);
      return true;
    } catch (e) {
      _setError(e, '豁免审批失败');
      notifyListeners();
      return false;
    }
  }

  Future<void> loadClientEvidence(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcClientEvidence(projectId));
      clientEvidence = (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      // 甲方证据加载失败不阻塞
    }
    notifyListeners();
  }

  Future<void> loadRubrics(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcRubrics(projectId));
      rubrics = (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      // 评分量表加载失败不阻塞
    }
    notifyListeners();
  }

  /// 读取单个里程碑提交的评审详情（评分维度 + 条件）。
  Future<Map<String, dynamic>> loadSubmissionReview(
      int projectId, int submissionId) async {
    try {
      final r = await _api
          .get(ApiConfig.vopcSubmissionReview(projectId, submissionId));
      return Map<String, dynamic>.from(r.data?['data'] as Map? ?? {});
    } catch (e) {
      _setError(e, '评审详情加载失败');
      notifyListeners();
      return const {};
    }
  }

  Future<bool> markConditionSatisfied(
      int projectId, int submissionId, int conditionId) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.put(
          ApiConfig.vopcMarkCondition(projectId, submissionId, conditionId));
      if (r.data?['data']?['satisfied'] != true) return false;
      return true;
    } catch (e) {
      _setError(e, '条件标记失败');
      notifyListeners();
      return false;
    }
  }

  Future<bool> finalizeMilestone(int projectId, int submissionId) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api
          .post(ApiConfig.vopcFinalizeMilestone(projectId, submissionId));
      if (r.data?['data']?['status'] == null) return false;
      await loadMilestoneSubmissions(projectId);
      await loadDetail(projectId);
      return true;
    } catch (e) {
      _setError(e, '里程碑闭环失败');
      notifyListeners();
      return false;
    }
  }

  Future<void> loadFiles(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcProjectFiles(projectId));
      files = (r.data?['data'] as List? ?? const [])
          .map((e) => Map<String, dynamic>.from(e as Map))
          .toList();
    } catch (e) {
      // 私有文件列表加载失败不阻塞
    }
    notifyListeners();
  }

  /// 上传私有文件（受控鉴权，字段名 file）。
  Future<bool> uploadProjectFile(int projectId,
      {required String filename, required List<int> bytes}) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.uploadFile(ApiConfig.vopcProjectFiles(projectId),
          filename: filename, bytes: bytes);
      if (r.data?['data']?['object_key'] == null) return false;
      await loadFiles(projectId);
      return true;
    } catch (e) {
      _setError(e, '私有文件上传失败');
      notifyListeners();
      return false;
    }
  }

  /// 下载私有文件（返回原始字节，受控鉴权由后端执行）。
  Future<List<int>?> downloadProjectFile(int projectId, String objectKey) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.getBytes(ApiConfig.vopcProjectFile(projectId, objectKey));
      final data = r.data;
      if (data is List<int>) return data;
      return null;
    } catch (e) {
      _setError(e, '私有文件下载失败');
      notifyListeners();
      return null;
    }
  }
  Future<void> loadAITasks(int projectId) async {
    try {
      final r = await _api.get(ApiConfig.vopcAITasks(projectId));
      aiTasks = (r.data?['data'] as List? ?? const [])
          .map((e) => VopcAITask.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList();
    } catch (e) {
      // 虚拟向导任务加载失败不阻塞
    }
    notifyListeners();
  }

  Future<VopcAITask?> createAITask(
      int projectId, Map<String, dynamic> data) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.post(ApiConfig.vopcAITasks(projectId), data: data);
      final map = Map<String, dynamic>.from(r.data?['data'] as Map? ?? {});
      if (map['id'] == null) return null;
      await loadAITasks(projectId);
      return VopcAITask.fromJson(map);
    } catch (e) {
      _setError(e, '虚拟向导引导失败');
      notifyListeners();
      return null;
    }
  }

  /// 审阅虚拟向导草稿（accept/revise/reject/overrule）。
  Future<bool> reviewAITask(int projectId, int taskId, String decision,
      {String note = '', String revision = ''}) async {
    error = null;
    statusCode = null;
    notifyListeners();
    try {
      final r = await _api.post(ApiConfig.vopcAITaskReview(projectId, taskId),
          data: {
            'decision': decision,
            'note': note,
            if (decision == 'revise') 'revision': revision,
          });
      if (r.data?['data']?['final_decision'] == null) return false;
      await loadAITasks(projectId);
      return true;
    } catch (e) {
      _setError(e, '虚拟向导审阅失败');
      notifyListeners();
      return false;
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
