import 'package:flutter/foundation.dart';
import '../services/api_service.dart';
import '../config/api_config.dart';

// ── 数据模型 ──

class Advisor {
  final int id;
  final String name;
  final String advisorId;
  final String title;
  final String college;
  final String department;
  final List<String> researchAreas;
  final int maxStudents;
  final int currentStudents;

  Advisor({
    required this.id, required this.name, required this.advisorId,
    required this.title, required this.college, required this.department,
    this.researchAreas = const [], this.maxStudents = 5, this.currentStudents = 0,
  });

  factory Advisor.fromJson(Map<String, dynamic> j) => Advisor(
    id: j['id'] ?? 0, name: j['name'] ?? '', advisorId: j['advisor_id'] ?? '',
    title: j['title'] ?? '', college: j['college'] ?? '', department: j['department'] ?? '',
    researchAreas: List<String>.from(j['research_areas'] ?? []),
    maxStudents: j['max_students'] ?? 5, currentStudents: j['current_students'] ?? 0,
  );
}

class ThesisTopic {
  final int id;
  final String title;
  final String advisorName;
  final String major;
  final String topicType;
  final String difficulty;
  final String description;
  final String keywords;
  final int maxStudents;
  final int selectedCount;
  final String batch;

  ThesisTopic({
    required this.id, required this.title, this.advisorName = '',
    this.major = '', this.topicType = '', this.difficulty = '',
    this.description = '', this.keywords = '', this.maxStudents = 1,
    this.selectedCount = 0, this.batch = '',
  });

  factory ThesisTopic.fromJson(Map<String, dynamic> j) => ThesisTopic(
    id: j['id'] ?? 0, title: j['title'] ?? '',
    advisorName: j['advisor_name'] ?? '', major: j['major'] ?? '',
    topicType: j['topic_type'] ?? '', difficulty: j['difficulty'] ?? '',
    description: j['description'] ?? '', keywords: j['keywords'] ?? '',
    maxStudents: j['max_students'] ?? 1, selectedCount: j['selected_count'] ?? 0,
    batch: j['batch']?.toString() ?? '',
  );

  String get difficultyLabel {
    switch (difficulty) {
      case 'easy': return '简单';
      case 'medium': return '中等';
      case 'hard': return '困难';
      default: return difficulty;
    }
  }
}

class CompetitionItem {
  final int id;
  final String name;
  final String level;
  final String category;
  final String description;
  final String startDate;
  final String endDate;
  final String status;
  final int registrationCount;

  CompetitionItem({
    required this.id, required this.name, this.level = '',
    this.category = '', this.description = '', this.startDate = '',
    this.endDate = '', this.status = '', this.registrationCount = 0,
  });

  factory CompetitionItem.fromJson(Map<String, dynamic> j) => CompetitionItem(
    id: j['id'] ?? 0, name: j['name'] ?? '', level: j['level'] ?? '',
    category: j['category'] ?? '', description: j['description'] ?? '',
    startDate: j['start_date'] ?? '', endDate: j['end_date'] ?? '',
    status: j['status'] ?? 'upcoming', registrationCount: j['registration_count'] ?? 0,
  );

  String get levelLabel {
    switch (level) {
      case 'international': return '国际级';
      case 'national': return '国家级';
      case 'provincial': return '省级';
      case 'municipal': return '市级';
      case 'school': return '校级';
      case 'college': return '院级';
      default: return level;
    }
  }
}

class PlanTemplate {
  final int id;
  final String name;
  final String category;
  final String description;
  final String content;
  final int applicableYear;

  PlanTemplate({
    required this.id, required this.name, this.category = '',
    this.description = '', this.content = '', this.applicableYear = 1,
  });

  factory PlanTemplate.fromJson(Map<String, dynamic> j) => PlanTemplate(
    id: j['id'] ?? 0, name: j['name'] ?? '', category: j['category'] ?? '',
    description: j['description'] ?? '', content: j['content'] ?? '',
    applicableYear: j['applicable_year'] ?? 1,
  );
}

class StudentPlan {
  final int id;
  final String title;
  final String category;
  final String content;
  final int templateId;
  final String status;
  final String reviewComment;
  final String createdAt;

  StudentPlan({
    required this.id, required this.title, this.category = '',
    this.content = '', this.templateId = 0, this.status = 'draft',
    this.reviewComment = '', this.createdAt = '',
  });

  factory StudentPlan.fromJson(Map<String, dynamic> j) => StudentPlan(
    id: j['id'] ?? 0, title: j['title'] ?? '', category: j['category'] ?? '',
    content: j['content'] ?? '', templateId: j['template_id'] ?? 0,
    status: j['status'] ?? 'draft', reviewComment: j['review_comment'] ?? '',
    createdAt: j['created_at'] ?? '',
  );

  String get statusLabel {
    switch (status) {
      case 'draft': return '草稿';
      case 'submitted': return '待审核';
      case 'approved': return '已通过';
      case 'in_progress': return '执行中';
      case 'completed': return '已完成';
      case 'rejected': return '已驳回';
      default: return status;
    }
  }
}

class PartyStage {
  final int id;
  final String code;
  final String name;
  final String description;
  final int sortOrder;

  PartyStage({
    required this.id, required this.code, required this.name,
    this.description = '', this.sortOrder = 0,
  });

  factory PartyStage.fromJson(Map<String, dynamic> j) => PartyStage(
    id: j['id'] ?? 0, code: j['code'] ?? '', name: j['name'] ?? '',
    description: j['description'] ?? '', sortOrder: j['sort_order'] ?? 0,
  );
}

class ClubItem {
  final int id;
  final String name;
  final String category;
  final String description;
  final String presidentName;
  final int memberCount;
  final String status;

  ClubItem({
    required this.id, required this.name, this.category = '',
    this.description = '', this.presidentName = '',
    this.memberCount = 0, this.status = 'active',
  });

  factory ClubItem.fromJson(Map<String, dynamic> j) => ClubItem(
    id: j['id'] ?? 0, name: j['name'] ?? '', category: j['category'] ?? '',
    description: j['description'] ?? '', presidentName: j['president_name'] ?? '',
    memberCount: j['member_count'] ?? 0, status: j['status'] ?? 'active',
  );
}

class ClubActivity {
  final int id;
  final String title;
  final String clubName;
  final String description;
  final String location;
  final String startTime;
  final String endTime;
  final int maxParticipants;
  final int registeredCount;

  ClubActivity({
    required this.id, required this.title, this.clubName = '',
    this.description = '', this.location = '', this.startTime = '',
    this.endTime = '', this.maxParticipants = 0, this.registeredCount = 0,
  });

  factory ClubActivity.fromJson(Map<String, dynamic> j) => ClubActivity(
    id: j['id'] ?? 0, title: j['title'] ?? '', clubName: j['club_name'] ?? '',
    description: j['description'] ?? '', location: j['location'] ?? '',
    startTime: j['start_time'] ?? '', endTime: j['end_time'] ?? '',
    maxParticipants: j['max_participants'] ?? 0, registeredCount: j['registered_count'] ?? 0,
  );
}

// ── Provider ──

class StudentNewFeaturesProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  bool _loading = false;
  String _error = '';
  bool get loading => _loading;
  String get error => _error;

  // ── 毕设选题 ──
  List<Advisor> _advisors = [];
  List<ThesisTopic> _topics = [];
  ThesisTopic? _mySelection;
  List<Map<String, dynamic>> _milestones = [];
  Map<String, dynamic>? _graduationStats;
  List<Advisor> get advisors => _advisors;
  List<ThesisTopic> get topics => _topics;
  ThesisTopic? get mySelection => _mySelection;
  List<Map<String, dynamic>> get milestones => _milestones;
  Map<String, dynamic>? get graduationStats => _graduationStats;

  // ── 学科竞赛 ──
  List<CompetitionItem> _competitions = [];
  List<CompetitionItem> get competitions => _competitions;
  Map<String, dynamic>? _competitionStats;
  Map<String, dynamic>? get competitionStats => _competitionStats;

  // ── 大学规划 ──
  List<PlanTemplate> _planTemplates = [];
  List<StudentPlan> _myPlans = [];
  List<PlanTemplate> get planTemplates => _planTemplates;
  List<StudentPlan> get myPlans => _myPlans;

  // ── 入党教育 ──
  List<PartyStage> _partyStages = [];
  Map<String, dynamic>? _myPartyProgress;
  List<Map<String, dynamic>> _studyRecords = [];
  Map<String, dynamic>? _partyStats;
  List<PartyStage> get partyStages => _partyStages;
  Map<String, dynamic>? get myPartyProgress => _myPartyProgress;
  List<Map<String, dynamic>> get studyRecords => _studyRecords;
  Map<String, dynamic>? get partyStats => _partyStats;

  // ── 社团生活 ──
  List<ClubItem> _clubs = [];
  List<ClubItem> _myClubs = [];
  List<ClubActivity> _clubActivities = [];
  List<ClubItem> get clubs => _clubs;
  List<ClubItem> get myClubs => _myClubs;
  List<ClubActivity> get clubActivities => _clubActivities;

  Future<void> _safeLoad(Future<void> Function() fn) async {
    _loading = true;
    _error = '';
    notifyListeners();
    try {
      await fn();
    } catch (e) {
      _error = e.toString();
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  // ── 毕设选题 ──

  Future<void> fetchAdvisors() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.graduationAdvisors);
    if (res.data['code'] == 0) {
      _advisors = (res.data['data'] as List? ?? []).map((e) => Advisor.fromJson(e)).toList();
    }
  });

  Future<void> fetchTopics({int page = 1, int pageSize = 20, String? keyword}) => _safeLoad(() async {
    final params = <String, dynamic>{'page': page, 'page_size': pageSize};
    if (keyword != null && keyword.isNotEmpty) params['keyword'] = keyword;
    final res = await _api.get(ApiConfig.graduationTopics, params: params);
    if (res.data['code'] == 0) {
      _topics = (res.data['data'] as List? ?? []).map((e) => ThesisTopic.fromJson(e)).toList();
    }
  });

  Future<bool> selectTopic(int topicId) async {
    try {
      final res = await _api.post(ApiConfig.graduationSelect, data: {'topic_id': topicId});
      if (res.data['code'] == 0) {
        await fetchMySelection();
        return true;
      }
      _error = res.data['message'] ?? '选题失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  Future<void> fetchMySelection() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.graduationMySelection);
    if (res.data['code'] == 0 && res.data['data'] != null) {
      _mySelection = ThesisTopic.fromJson(res.data['data']);
    }
  });

  Future<void> fetchMilestones() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.graduationMilestones);
    if (res.data['code'] == 0) {
      _milestones = List<Map<String, dynamic>>.from(res.data['data'] ?? []);
    }
  });

  Future<void> fetchGraduationStats() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.graduationStats);
    if (res.data['code'] == 0) {
      _graduationStats = res.data['data'];
    }
  });

  // ── 学科竞赛 ──

  Future<void> fetchCompetitions({int page = 1, int pageSize = 20}) => _safeLoad(() async {
    final res = await _api.get(ApiConfig.competitionList, params: {'page': page, 'page_size': pageSize});
    if (res.data['code'] == 0) {
      _competitions = (res.data['data'] as List? ?? []).map((e) => CompetitionItem.fromJson(e)).toList();
    }
  });

  Future<bool> registerCompetition(int competitionId, {String teamMembers = ''}) async {
    try {
      final data = <String, dynamic>{'competition_id': competitionId};
      if (teamMembers.isNotEmpty) data['team_members'] = teamMembers;
      final res = await _api.post(ApiConfig.competitionRegister, data: data);
      if (res.data['code'] == 0) return true;
      _error = res.data['message'] ?? '报名失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  Future<void> fetchCompetitionStats() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.competitionStats);
    if (res.data['code'] == 0) {
      _competitionStats = res.data['data'];
    }
  });

  // ── 大学规划 ──

  Future<void> fetchPlanTemplates() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.planTemplates);
    if (res.data['code'] == 0) {
      _planTemplates = (res.data['data'] as List? ?? []).map((e) => PlanTemplate.fromJson(e)).toList();
    }
  });

  Future<void> fetchMyPlans() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.planMyPlans);
    if (res.data['code'] == 0) {
      _myPlans = (res.data['data'] as List? ?? []).map((e) => StudentPlan.fromJson(e)).toList();
    }
  });

  Future<bool> createPlan({required String title, required String content, int? templateId, String category = 'custom'}) async {
    try {
      final data = <String, dynamic>{'title': title, 'content': content, 'category': category};
      if (templateId != null) data['template_id'] = templateId;
      final res = await _api.post(ApiConfig.planCreate, data: data);
      if (res.data['code'] == 0) {
        await fetchMyPlans();
        return true;
      }
      _error = res.data['message'] ?? '创建失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  Future<bool> submitPlan(int planId) async {
    try {
      final res = await _api.put(ApiConfig.planSubmit(planId.toString()));
      if (res.data['code'] == 0) {
        await fetchMyPlans();
        return true;
      }
      _error = res.data['message'] ?? '提交失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  // ── 入党教育 ──

  Future<void> fetchPartyStages() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.partyStages);
    if (res.data['code'] == 0) {
      _partyStages = (res.data['data'] as List? ?? []).map((e) => PartyStage.fromJson(e)).toList();
    }
  });

  Future<void> fetchMyPartyProgress() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.partyMyProgress);
    if (res.data['code'] == 0) {
      _myPartyProgress = res.data['data'];
    }
  });

  Future<void> fetchStudyRecords() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.partyStudyRecords);
    if (res.data['code'] == 0) {
      _studyRecords = List<Map<String, dynamic>>.from(res.data['data'] ?? []);
    }
  });

  Future<void> fetchPartyStats() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.partyStats);
    if (res.data['code'] == 0) {
      _partyStats = res.data['data'];
    }
  });

  // ── 社团生活 ──

  Future<void> fetchClubs({int page = 1, int pageSize = 20}) => _safeLoad(() async {
    final res = await _api.get(ApiConfig.clubList, params: {'page': page, 'page_size': pageSize});
    if (res.data['code'] == 0) {
      _clubs = (res.data['data'] as List? ?? []).map((e) => ClubItem.fromJson(e)).toList();
    }
  });

  Future<bool> joinClub(int clubId) async {
    try {
      final res = await _api.post(ApiConfig.clubJoin, data: {'club_id': clubId});
      if (res.data['code'] == 0) {
        await fetchMyClubs();
        return true;
      }
      _error = res.data['message'] ?? '加入失败';
      notifyListeners();
      return false;
    } catch (e) {
      _error = '网络错误: $e';
      notifyListeners();
      return false;
    }
  }

  Future<void> fetchMyClubs() => _safeLoad(() async {
    final res = await _api.get(ApiConfig.clubMyClubs);
    if (res.data['code'] == 0) {
      _myClubs = (res.data['data'] as List? ?? []).map((e) => ClubItem.fromJson(e)).toList();
    }
  });

  Future<void> fetchClubActivities({int page = 1, int pageSize = 20}) => _safeLoad(() async {
    final res = await _api.get(ApiConfig.clubActivities, params: {'page': page, 'page_size': pageSize});
    if (res.data['code'] == 0) {
      _clubActivities = (res.data['data'] as List? ?? []).map((e) => ClubActivity.fromJson(e)).toList();
    }
  });
}
