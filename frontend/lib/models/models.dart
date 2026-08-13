import 'dart:convert';
import 'package:flutter/material.dart';

/// AnswerCard 统一回答结构，对齐后端 model.AnswerCard
class AnswerCard {
  final String conclusion;
  final List<String> steps;
  final List<ProcessStepDetail> stepDetails; // 富文本步骤详情（联系方式/地点/FAQ等）
  final List<Source> sources;
  final List<String> risks;
  final List<String> followUps;
  final List<CardAction> actions;
  final String traceId;
  final double confidence;
  final bool fallback;

  AnswerCard({
    required this.conclusion,
    this.steps = const [],
    this.stepDetails = const [],
    this.sources = const [],
    this.risks = const [],
    this.followUps = const [],
    this.actions = const [],
    this.traceId = '',
    this.confidence = 0,
    this.fallback = false,
  });

  factory AnswerCard.fromJson(Map<String, dynamic> json) {
    return AnswerCard(
      conclusion: json['conclusion'] ?? '',
      steps: List<String>.from(json['steps'] ?? []),
      stepDetails: (json['step_details'] as List?)
              ?.map((s) => ProcessStepDetail.fromJson(s))
              .toList() ??
          [],
      sources:
          (json['sources'] as List?)?.map((s) => Source.fromJson(s)).toList() ??
              [],
      risks: List<String>.from(json['risks'] ?? []),
      followUps: List<String>.from(json['follow_ups'] ?? []),
      actions: (json['actions'] as List?)
              ?.map((a) => CardAction.fromJson(a))
              .toList() ??
          [],
      traceId: json['trace_id'] ?? '',
      confidence: (json['confidence'] ?? 0).toDouble(),
      fallback: json['fallback'] ?? false,
    );
  }
}

/// 流程步骤详细信息（含联系人/地点/FAQ 等 6 类信息）
class ProcessStepDetail {
  final int step;
  final String title;
  final String status;
  final String contact;
  final String phone;
  final String contactWechat;
  final String location;
  final String officeHours;
  final String materials;
  final String entryUrl;
  final String deadline;
  final String notes;
  final List<String> mediaUrls;
  final List<ProcessFAQ> faq;

  ProcessStepDetail({
    required this.step,
    required this.title,
    this.status = 'pending',
    this.contact = '',
    this.phone = '',
    this.contactWechat = '',
    this.location = '',
    this.officeHours = '',
    this.materials = '',
    this.entryUrl = '',
    this.deadline = '',
    this.notes = '',
    this.mediaUrls = const [],
    this.faq = const [],
  });

  factory ProcessStepDetail.fromJson(Map<String, dynamic> json) {
    return ProcessStepDetail(
      step: json['step'] ?? 0,
      title: json['title'] ?? '',
      status: json['status'] ?? 'pending',
      contact: json['contact'] ?? '',
      phone: json['phone'] ?? '',
      contactWechat: json['contact_wechat'] ?? '',
      location: json['location'] ?? '',
      officeHours: json['office_hours'] ?? '',
      materials: json['materials'] ?? '',
      entryUrl: json['entry_url'] ?? '',
      deadline: json['deadline'] ?? '',
      notes: json['notes'] ?? '',
      mediaUrls: (json['media_urls'] as List?)?.cast<String>() ?? const [],
      faq:
          (json['faq'] as List?)?.map((f) => ProcessFAQ.fromJson(f)).toList() ??
              [],
    );
  }
}

/// 流程步骤的常见问题
class ProcessFAQ {
  final String q;
  final String a;

  ProcessFAQ({required this.q, required this.a});

  factory ProcessFAQ.fromJson(Map<String, dynamic> json) {
    return ProcessFAQ(
      q: json['q'] ?? '',
      a: json['a'] ?? '',
    );
  }
}

/// 流程步骤（管理端完整字段，对应后端 process_steps）
class ProcessStep {
  final int id;
  final String resourceId;
  final int stepOrder;
  final String title;
  final String materials; // JSON 数组字符串
  final String entryUrl;
  final String deadline;
  final String location;
  final String notes;
  final String contact;
  final String phone;
  final String contactWechat;
  final String officeHours;
  final double geoLat;
  final double geoLng;
  final String mediaUrls; // JSON 数组字符串
  final String faq; // JSON 数组字符串

  ProcessStep({
    this.id = 0,
    this.resourceId = '',
    this.stepOrder = 0,
    this.title = '',
    this.materials = '[]',
    this.entryUrl = '',
    this.deadline = '',
    this.location = '',
    this.notes = '',
    this.contact = '',
    this.phone = '',
    this.contactWechat = '',
    this.officeHours = '',
    this.geoLat = 0,
    this.geoLng = 0,
    this.mediaUrls = '[]',
    this.faq = '[]',
  });

  factory ProcessStep.fromJson(Map<String, dynamic> json) {
    return ProcessStep(
      id: json['id'] is int ? json['id'] : int.tryParse('${json['id']}') ?? 0,
      resourceId: json['resource_id'] ?? '',
      stepOrder: json['step_order'] ?? 0,
      title: json['title'] ?? '',
      materials: json['materials'] ?? '[]',
      entryUrl: json['entry_url'] ?? '',
      deadline: json['deadline'] ?? '',
      location: json['location'] ?? '',
      notes: json['notes'] ?? '',
      contact: json['contact'] ?? '',
      phone: json['phone'] ?? '',
      contactWechat: json['contact_wechat'] ?? '',
      officeHours: json['office_hours'] ?? '',
      geoLat: (json['geo_lat'] ?? 0).toDouble(),
      geoLng: (json['geo_lng'] ?? 0).toDouble(),
      mediaUrls: json['media_urls'] ?? '[]',
      faq: json['faq'] ?? '[]',
    );
  }

  List<String> get materialsList => _decodeStringList(materials);
  List<String> get mediaList => _decodeStringList(mediaUrls);

  List<ProcessFAQ> get faqList {
    final decoded = _decodeList(faq);
    return decoded
        .map((e) => ProcessFAQ.fromJson(Map<String, dynamic>.from(e)))
        .toList();
  }

  ProcessStepDetail toDetail() {
    return ProcessStepDetail(
      step: stepOrder,
      title: title,
      contact: contact,
      phone: phone,
      contactWechat: contactWechat,
      location: location,
      officeHours: officeHours,
      materials: materials,
      entryUrl: entryUrl,
      deadline: deadline,
      notes: notes,
      mediaUrls: mediaList,
      faq: faqList,
    );
  }
}

/// 流程提醒（对应后端 process_reminders）
class ProcessReminder {
  final int id;
  final String processId;
  final int stepOrder;
  final String remindAt;
  final String title;
  final String content;
  final bool isEnabled;
  final String createdAt;
  final String updatedAt;

  ProcessReminder({
    this.id = 0,
    this.processId = '',
    this.stepOrder = 0,
    this.remindAt = '',
    this.title = '',
    this.content = '',
    this.isEnabled = true,
    this.createdAt = '',
    this.updatedAt = '',
  });

  factory ProcessReminder.fromJson(Map<String, dynamic> json) {
    return ProcessReminder(
      id: json['id'] is int ? json['id'] : int.tryParse('${json['id']}') ?? 0,
      processId: json['process_id'] ?? '',
      stepOrder: json['step_order'] ?? 0,
      remindAt: json['remind_at'] ?? '',
      title: json['title'] ?? '',
      content: json['content'] ?? '',
      isEnabled: (json['is_enabled'] ?? 1) == 1,
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }
}

/// 办事流程完整定义（KB 资源 + 步骤 + 提醒）
class ProcessDefinition {
  final String resourceId;
  final String resourceType;
  final String ownerScope;
  final String ownerId;
  final String roleScope;
  final String version;
  final String status;
  final String title;
  final String summary;
  final String content;
  final String sourceLink;
  final String sourceVersion;
  final String effectiveAt;
  final String expiredAt;
  final List<String> tags;
  final String remark;
  final String updatedBy;
  final String createdAt;
  final String updatedAt;
  final List<ProcessStep> steps;
  final List<ProcessReminder> reminders;

  ProcessDefinition({
    required this.resourceId,
    required this.resourceType,
    required this.ownerScope,
    this.ownerId = '',
    this.roleScope = '',
    this.version = '',
    this.status = '',
    required this.title,
    this.summary = '',
    this.content = '',
    this.sourceLink = '',
    this.sourceVersion = '',
    this.effectiveAt = '',
    this.expiredAt = '',
    this.tags = const [],
    this.remark = '',
    this.updatedBy = '',
    this.createdAt = '',
    this.updatedAt = '',
    this.steps = const [],
    this.reminders = const [],
  });

  factory ProcessDefinition.fromJson(Map<String, dynamic> json) {
    return ProcessDefinition(
      resourceId: json['resource_id'] ?? '',
      resourceType: json['resource_type'] ?? 'Process',
      ownerScope: json['owner_scope'] ?? '',
      ownerId: json['owner_id'] ?? '',
      roleScope: json['role_scope'] ?? '',
      version: json['version'] ?? '',
      status: json['status'] ?? 'draft',
      title: json['title'] ?? '',
      summary: json['summary'] ?? '',
      content: json['content'] ?? '',
      sourceLink: json['source_link'] ?? '',
      sourceVersion: json['source_version'] ?? '',
      effectiveAt: json['effective_at'] ?? '',
      expiredAt: json['expired_at'] ?? '',
      tags: KnowledgeCard._parseTags(json['tags'] ?? ''),
      remark: json['remark'] ?? '',
      updatedBy: json['updated_by'] ?? '',
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
      steps: (json['steps'] as List?)
              ?.map((e) => ProcessStep.fromJson(Map<String, dynamic>.from(e)))
              .toList() ??
          const [],
      reminders: (json['reminders'] as List?)
              ?.map(
                  (e) => ProcessReminder.fromJson(Map<String, dynamic>.from(e)))
              .toList() ??
          const [],
    );
  }

  bool get isFreshmenRelated =>
      tags.any((t) => t.contains('新生') || t.contains('入学') || t.contains('报到'));

  /// 流程适用角色（role_scope JSON 解码），空表示不限制
  List<String> get roleCodes {
    if (roleScope.isEmpty || roleScope == '[]') return const [];
    try {
      final decoded = jsonDecode(roleScope);
      if (decoded is List) return decoded.map((e) => e.toString()).toList();
    } catch (_) {}
    return const [];
  }

  /// 面向群体（年级/身份维度），依据标题、摘要与 tags 判定
  String get audienceLabel {
    final s = '$title $summary ${tags.join(' ')}';
    if (s.contains('新生') || s.contains('入学') || s.contains('报到')) {
      return '新生';
    }
    if (s.contains('毕业') || s.contains('离校')) return '毕业生';
    if (s.contains('请假') ||
        s.contains('奖学金') ||
        s.contains('转专业') ||
        s.contains('助学贷款') ||
        s.contains('在校')) {
      return '在校生';
    }
    return '通用';
  }

  String get statusLabel {
    switch (status) {
      case 'published':
        return '已发布';
      case 'pending':
        return '待审核';
      case 'retired':
        return '已下架';
      default:
        return '草稿';
    }
  }
}

List<dynamic> _decodeList(String raw) {
  if (raw.isEmpty || raw == '[]') return [];
  try {
    final decoded = jsonDecode(raw);
    return decoded is List ? decoded : [];
  } catch (_) {
    return [];
  }
}

List<String> _decodeStringList(String raw) {
  return _decodeList(raw).map((e) => e.toString()).toList();
}

/// 来源引用
class Source {
  final String resourceId;
  final String title;
  final String version;
  final String sourceLink;
  final double relevanceScore;
  final String resourceType;
  final String snippet;
  final String effectiveDate;
  final String summary;

  Source({
    required this.resourceId,
    required this.title,
    this.version = '',
    this.sourceLink = '',
    this.relevanceScore = 0,
    this.resourceType = '',
    this.snippet = '',
    this.effectiveDate = '',
    this.summary = '',
  });

  factory Source.fromJson(Map<String, dynamic> json) {
    return Source(
      resourceId: json['resource_id'] ?? '',
      title: json['title'] ?? '',
      version: json['version'] ?? '',
      sourceLink: json['source_link'] ?? '',
      relevanceScore: (json['relevance_score'] ?? 0).toDouble(),
      resourceType: json['resource_type'] ?? json['type'] ?? '',
      snippet: json['snippet'] ?? json['content_preview'] ?? '',
      effectiveDate: json['effective_date'] ?? json['date'] ?? '',
      summary: json['summary'] ?? json['description'] ?? '',
    );
  }

  /// 资源类型中文标签
  String get typeLabel {
    const map = {
      'policy': '政策',
      'Policy': '政策',
      'process': '流程',
      'Process': '流程',
      'faq': '问答',
      'FAQ': '问答',
      'activity': '活动',
      'Activity': '活动',
    };
    return map[resourceType] ?? '资料';
  }

  /// 资源类型图标
  IconData get typeIcon {
    switch (resourceType.toLowerCase()) {
      case 'policy':
        return Icons.description;
      case 'process':
        return Icons.alt_route;
      case 'faq':
        return Icons.quiz;
      case 'activity':
        return Icons.event;
      default:
        return Icons.insert_drive_file;
    }
  }

  /// 资源类型颜色
  Color get typeColor {
    switch (resourceType.toLowerCase()) {
      case 'policy':
        return const Color(0xFFE53935);
      case 'process':
        return const Color(0xFF1E88E5);
      case 'faq':
        return const Color(0xFF43A047);
      case 'activity':
        return const Color(0xFFFB8C00);
      default:
        return const Color(0xFF757575);
    }
  }

  /// 相关度星级（0-5星）
  int get relevanceStars {
    if (relevanceScore >= 0.9) return 5;
    if (relevanceScore >= 0.75) return 4;
    if (relevanceScore >= 0.6) return 3;
    if (relevanceScore >= 0.4) return 2;
    if (relevanceScore >= 0.2) return 1;
    return 0;
  }
}

/// 可执行动作
class CardAction {
  final String label;
  final String url;

  CardAction({required this.label, required this.url});

  factory CardAction.fromJson(Map<String, dynamic> json) {
    return CardAction(
      label: json['label'] ?? '',
      url: json['url'] ?? '',
    );
  }
}

/// 对话请求
class ChatRequest {
  final String question;
  final String? sessionId;
  final String? agentId;

  ChatRequest({required this.question, this.sessionId, this.agentId});

  Map<String, dynamic> toJson() => {
        'question': question,
        if (sessionId != null) 'session_id': sessionId,
        if (agentId != null && agentId!.isNotEmpty) 'agent_id': agentId,
      };
}

/// 对话响应
class ChatResponse {
  final int code;
  final String message;
  final AnswerCard? data;
  final String sessionId;

  ChatResponse({
    required this.code,
    required this.message,
    this.data,
    this.sessionId = '',
  });

  factory ChatResponse.fromJson(Map<String, dynamic> json) {
    return ChatResponse(
      code: json['code'] ?? -1,
      message: json['message'] ?? '',
      data: json['data'] != null ? AnswerCard.fromJson(json['data']) : null,
      sessionId: json['session_id'] ?? '',
    );
  }
}

/// 会话
class Session {
  final String id;
  final String title;
  final String createdAt;
  final String updatedAt;

  Session({
    required this.id,
    required this.title,
    this.createdAt = '',
    this.updatedAt = '',
  });

  factory Session.fromJson(Map<String, dynamic> json) {
    return Session(
      // 后端 sessions 列表返回 {id: 自增整数, session_id: UUID,...}，
      // 加载/删除/重命名都按 session_id(UUID) 调用后端，故 id 必须取 session_id。
      id: json['session_id']?.toString() ?? json['id']?.toString() ?? '',
      title: json['title'] ?? '新对话',
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }
}

/// 消息（会话历史中的单条记录）
class Message {
  final String id;
  final String role; // user | assistant
  final String content;
  final AnswerCard? answerCard; // 仅 assistant 消息有
  final String createdAt;
  final bool isFailed; // 发送失败标记

  Message({
    required this.id,
    required this.role,
    required this.content,
    this.answerCard,
    this.createdAt = '',
    this.isFailed = false,
  });

  factory Message.fromJson(Map<String, dynamic> json) {
    return Message(
      id: json['id']?.toString() ?? '',
      role: json['role'] ?? 'user',
      content: json['content'] ?? '',
      answerCard: json['answer_card'] != null
          ? AnswerCard.fromJson(json['answer_card'])
          : null,
      createdAt: json['created_at'] ?? '',
      isFailed: json['is_failed'] ?? false,
    );
  }

  Message copyWith(
      {bool? isFailed, String? content, AnswerCard? answerCard, String? id}) {
    return Message(
      id: id ?? this.id,
      role: role,
      content: content ?? this.content,
      answerCard: answerCard ?? this.answerCard,
      createdAt: createdAt,
      isFailed: isFailed ?? this.isFailed,
    );
  }

  bool get isUser => role == 'user';
  bool get isAssistant => role == 'assistant';
}

/// 用户资料
class UserProfile {
  final int id;
  final String username;
  final String role;
  final String displayName;
  final String college;
  final String major;
  final String className;
  final String enrollmentDate;
  final String enrollmentYear;
  final String ownerScope;
  final String ownerId;
  final String status;
  final String gender;
  final String campus;
  final String educationLevel;
  final String studyDuration;
  final String expectedGrad;
  final String studyMode;
  final String ethnicity;
  final String politicalStatus;

  UserProfile({
    required this.id,
    required this.username,
    required this.role,
    required this.displayName,
    this.college = '',
    this.major = '',
    this.className = '',
    this.enrollmentDate = '',
    this.enrollmentYear = '',
    this.ownerScope = '',
    this.ownerId = '',
    this.status = 'active',
    this.gender = '',
    this.campus = '',
    this.educationLevel = '',
    this.studyDuration = '',
    this.expectedGrad = '',
    this.studyMode = '',
    this.ethnicity = '',
    this.politicalStatus = '',
  });

  factory UserProfile.fromJson(Map<String, dynamic> json) {
    // 后端可能将 data 包裹在 data 字段中
    final data = json['data'] ?? json;
    return UserProfile(
      id: data['id'] ?? 0,
      username: data['username'] ?? '',
      role: data['role'] ?? 'student',
      displayName: data['display_name'] ?? data['username'] ?? '',
      college: data['college'] ?? data['owner_id'] ?? '',
      major: data['major'] ?? '',
      className: data['class_name'] ?? '',
      enrollmentDate: data['enrollment_date'] ?? '',
      enrollmentYear: data['enrollment_year'] ?? '',
      ownerScope: data['owner_scope'] ?? '',
      ownerId: data['owner_id'] ?? '',
      status: data['status'] ?? 'active',
      gender: data['gender'] ?? '',
      campus: data['campus'] ?? '',
      educationLevel: data['education_level'] ?? '',
      studyDuration: data['study_duration'] ?? '',
      expectedGrad: data['expected_graduation_date'] ?? '',
      studyMode: data['study_mode'] ?? '',
      ethnicity: data['ethnicity'] ?? '',
      politicalStatus: data['political_status'] ?? '',
    );
  }

  /// 角色中文名
  String get roleLabel {
    const map = {
      'sys_admin': '系统管理员',
      'school_admin': '学校管理员',
      'college_admin': '学院管理员',
      'counselor': '辅导员',
      'student_union': '学生会',
      'student': '学生',
      'teacher': '教师',
      'assistant': '教辅',
    };
    return map[role] ?? role;
  }
}

/// 统一错误响应
class ApiError {
  final int code;
  final String message;
  final String traceId;

  ApiError({required this.code, required this.message, this.traceId = ''});

  factory ApiError.fromJson(Map<String, dynamic> json) {
    return ApiError(
      code: json['code'] ?? -1,
      message: json['message'] ?? '未知错误',
      traceId: json['trace_id'] ?? '',
    );
  }
}

/// 知识大厅卡片（轻量数据，不含正文）
class KnowledgeCard {
  final String resourceId;
  final String resourceType;
  final String title;
  final String summary;
  final String content;
  final String status;
  final List<String> tags;
  final String sourceLink;
  final String remark;
  final String updatedBy;
  final String createdAt;
  final String updatedAt;

  KnowledgeCard({
    required this.resourceId,
    required this.resourceType,
    required this.title,
    this.summary = '',
    this.content = '',
    this.status = '',
    this.tags = const [],
    this.sourceLink = '',
    this.remark = '',
    this.updatedBy = '',
    this.createdAt = '',
    this.updatedAt = '',
  });

  factory KnowledgeCard.fromJson(Map<String, dynamic> json) {
    return KnowledgeCard(
      resourceId: json['resource_id'] ?? '',
      resourceType: json['resource_type'] ?? '',
      title: json['title'] ?? '',
      summary: json['summary'] ?? '',
      content: json['content'] ?? '',
      status: json['status'] ?? '',
      tags: _parseTags(json['tags']),
      sourceLink: json['source_link'] ?? '',
      remark: json['remark'] ?? '',
      updatedBy: json['updated_by'] ?? '',
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }

  /// 解析 JSON 数组字符串 → `List<String>`
  static List<String> _parseTags(String raw) {
    if (raw.isEmpty || raw == '[]') return [];
    try {
      // 简单解析 JSON 数组，不引入 dart:convert 复杂度
      return raw
          .replaceAll('[', '')
          .replaceAll(']', '')
          .replaceAll('"', '')
          .split(',')
          .map((s) => s.trim())
          .where((s) => s.isNotEmpty)
          .toList();
    } catch (_) {
      return [];
    }
  }

  /// 类型中文标签
  String get typeLabel {
    const map = {
      'Policy': '政策',
      'Process': '流程',
      'FAQ': '问答',
      'Activity': '活动',
    };
    return map[resourceType] ?? resourceType;
  }
}

/// 情感分析日志（对齐后端 model.EmotionLog）
class EmotionLog {
  final int id;
  final int userId;
  final String username;
  final String sessionId;
  final String alertId;
  final String messageText;
  final double score;
  final String riskLevel;
  final String analysisJson;
  final int notified;
  final String status;
  final String acknowledgedBy;
  final String acknowledgedAt;
  final String createdAt;

  EmotionLog({
    required this.id,
    required this.userId,
    this.username = '',
    this.sessionId = '',
    this.alertId = '',
    this.messageText = '',
    this.score = 0,
    this.riskLevel = 'low',
    this.analysisJson = '',
    this.notified = 0,
    this.status = 'pending',
    this.acknowledgedBy = '',
    this.acknowledgedAt = '',
    this.createdAt = '',
  });

  factory EmotionLog.fromJson(Map<String, dynamic> json) {
    return EmotionLog(
      id: json['id'] ?? 0,
      userId: json['user_id'] ?? 0,
      username: json['username'] ?? '',
      sessionId: json['session_id'] ?? '',
      alertId: json['alert_id'] ?? '',
      messageText: json['message_text'] ?? '',
      score: (json['score'] ?? 0).toDouble(),
      riskLevel: json['risk_level'] ?? 'low',
      analysisJson: json['analysis_json'] ?? '',
      notified: json['notified'] ?? 0,
      status: json['status'] ?? 'pending',
      acknowledgedBy: json['acknowledged_by'] ?? '',
      acknowledgedAt: json['acknowledged_at'] ?? '',
      createdAt: json['created_at'] ?? '',
    );
  }

  /// 解析 analysis_json 获取结构化数据
  Map<String, dynamic> get analysis {
    if (analysisJson.isEmpty) return {};
    try {
      return jsonDecode(analysisJson) as Map<String, dynamic>;
    } catch (_) {
      return {};
    }
  }

  /// 风险等级中文标签
  String get riskLabel {
    const map = {
      'low': '低风险',
      'medium': '中风险',
      'high': '高风险',
      'urgent': '紧急',
    };
    return map[riskLevel] ?? riskLevel;
  }

  /// 状态中文标签
  String get statusLabel {
    const map = {
      'pending': '待处理',
      'acknowledged': '已确认',
      'resolved': '已处理',
    };
    return map[status] ?? status;
  }
}

/// 情感告警统计
class EmotionStats {
  final int pending;
  final int urgent;
  final int high;
  final int medium;
  final int low;

  EmotionStats({
    this.pending = 0,
    this.urgent = 0,
    this.high = 0,
    this.medium = 0,
    this.low = 0,
  });

  factory EmotionStats.fromJson(Map<String, dynamic> json) {
    return EmotionStats(
      pending: json['pending'] ?? 0,
      urgent: json['urgent'] ?? 0,
      high: json['high'] ?? 0,
      medium: json['medium'] ?? 0,
      low: json['low'] ?? 0,
    );
  }

  int get total => pending + urgent + high + medium + low;
}

/// 情感分析请求
class EmotionAnalyzeRequest {
  final String messageText;
  final String sessionId;

  EmotionAnalyzeRequest({required this.messageText, required this.sessionId});

  Map<String, dynamic> toJson() => {
        'message_text': messageText,
        'session_id': sessionId,
      };
}

/// 更新告警状态请求
class EmotionUpdateRequest {
  final String status;

  EmotionUpdateRequest({required this.status});

  Map<String, dynamic> toJson() => {'status': status};
}

// ── 智能体管理模型 ──

/// 智能体（对齐后端 model.Agent）
class Agent {
  final int id;
  final String agentId;
  final String name;
  final String description;
  final String agentType;
  final String systemPrompt;
  final String modelProvider;
  final String modelName;
  final double temperature;
  final int maxTokens;
  final String status;
  final String configJson;
  final String createdAt;
  final String updatedAt;

  Agent({
    required this.id,
    required this.agentId,
    required this.name,
    this.description = '',
    this.agentType = 'qa',
    this.systemPrompt = '',
    this.modelProvider = '',
    this.modelName = '',
    this.temperature = 0.7,
    this.maxTokens = 2048,
    this.status = 'active',
    this.configJson = '{}',
    this.createdAt = '',
    this.updatedAt = '',
  });

  factory Agent.fromJson(Map<String, dynamic> json) {
    return Agent(
      id: json['id'] ?? 0,
      agentId: json['agent_id'] ?? '',
      name: json['name'] ?? '',
      description: json['description'] ?? '',
      agentType: json['agent_type'] ?? 'qa',
      systemPrompt: json['system_prompt'] ?? '',
      modelProvider: json['model_provider'] ?? '',
      modelName: json['model_name'] ?? '',
      temperature: (json['temperature'] ?? 0.7).toDouble(),
      maxTokens: json['max_tokens'] ?? 2048,
      status: json['status'] ?? 'active',
      configJson: json['config_json'] ?? '{}',
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }

  /// 类型中文标签
  String get typeLabel {
    const map = {
      'qa': '通用问答',
      'policy': '政策解读',
      'process': '流程指引',
      'emotion': '心理疏导',
      'major': '学科专业',
      'custom': '自定义',
    };
    return map[agentType] ?? agentType;
  }

  /// 状态中文标签
  String get statusLabel => status == 'active' ? '启用' : '停用';

  bool get isActive => status == 'active';
}

// ── 管理端模型 ──

/// 质量看板指标（对齐后端 model.AdminMetrics）
class AdminMetrics {
  final double hitRate;
  final double fallbackRate;
  final double sourceCoverage;
  final int p95LatencyMs;
  final int totalQuestions;
  final int totalSessions;
  final int activeUsersToday;

  AdminMetrics({
    this.hitRate = 0,
    this.fallbackRate = 0,
    this.sourceCoverage = 0,
    this.p95LatencyMs = 0,
    this.totalQuestions = 0,
    this.totalSessions = 0,
    this.activeUsersToday = 0,
  });

  factory AdminMetrics.fromJson(Map<String, dynamic> json) {
    return AdminMetrics(
      hitRate: (json['hit_rate'] ?? 0).toDouble(),
      fallbackRate: (json['fallback_rate'] ?? 0).toDouble(),
      sourceCoverage: (json['source_coverage'] ?? 0).toDouble(),
      p95LatencyMs: json['p95_latency_ms'] ?? 0,
      totalQuestions: json['total_questions'] ?? 0,
      totalSessions: json['total_sessions'] ?? 0,
      activeUsersToday: json['active_users_today'] ?? 0,
    );
  }
}

/// 仪表盘统计数据
class DashboardStats {
  final UserStats users;
  final KnowledgeStats knowledge;
  final ChatStats chat;
  final FeedbackStats feedback;

  DashboardStats({
    required this.users,
    required this.knowledge,
    required this.chat,
    required this.feedback,
  });

  factory DashboardStats.fromJson(Map<String, dynamic> json) {
    return DashboardStats(
      users: UserStats.fromJson(json['users'] ?? {}),
      knowledge: KnowledgeStats.fromJson(json['knowledge'] ?? {}),
      chat: ChatStats.fromJson(json['chat'] ?? {}),
      feedback: FeedbackStats.fromJson(json['feedback'] ?? {}),
    );
  }
}

/// 用户统计
class UserStats {
  final int total;
  final int todayNew;
  final int monthNew;
  final Map<String, int> byRole;

  UserStats({
    this.total = 0,
    this.todayNew = 0,
    this.monthNew = 0,
    this.byRole = const {},
  });

  factory UserStats.fromJson(Map<String, dynamic> json) {
    Map<String, int> roleMap = {};
    if (json['by_role'] != null) {
      (json['by_role'] as Map<String, dynamic>).forEach((key, value) {
        roleMap[key] = value ?? 0;
      });
    }
    return UserStats(
      total: json['total'] ?? 0,
      todayNew: json['today_new'] ?? 0,
      monthNew: json['month_new'] ?? 0,
      byRole: roleMap,
    );
  }
}

/// 知识库统计
class KnowledgeStats {
  final int total;
  final int draft;
  final int pending;
  final int published;
  final int retired;
  final Map<String, int> byType;
  final int weekNew;

  KnowledgeStats({
    this.total = 0,
    this.draft = 0,
    this.pending = 0,
    this.published = 0,
    this.retired = 0,
    this.byType = const {},
    this.weekNew = 0,
  });

  factory KnowledgeStats.fromJson(Map<String, dynamic> json) {
    Map<String, int> typeMap = {};
    if (json['by_type'] != null) {
      (json['by_type'] as Map<String, dynamic>).forEach((key, value) {
        typeMap[key] = value ?? 0;
      });
    }
    return KnowledgeStats(
      total: json['total'] ?? 0,
      draft: json['draft'] ?? 0,
      pending: json['pending'] ?? 0,
      published: json['published'] ?? 0,
      retired: json['retired'] ?? 0,
      byType: typeMap,
      weekNew: json['week_new'] ?? 0,
    );
  }
}

/// 对话统计
class ChatStats {
  final int totalSessions;
  final int totalMessages;
  final int todaySessions;
  final int todayMessages;
  final List<DayTrendItem> weekTrend;

  ChatStats({
    this.totalSessions = 0,
    this.totalMessages = 0,
    this.todaySessions = 0,
    this.todayMessages = 0,
    this.weekTrend = const [],
  });

  factory ChatStats.fromJson(Map<String, dynamic> json) {
    List<DayTrendItem> trend = [];
    if (json['week_trend'] != null) {
      trend = (json['week_trend'] as List)
          .map((e) => DayTrendItem.fromJson(e))
          .toList();
    }
    return ChatStats(
      totalSessions: json['total_sessions'] ?? 0,
      totalMessages: json['total_messages'] ?? 0,
      todaySessions: json['today_sessions'] ?? 0,
      todayMessages: json['today_messages'] ?? 0,
      weekTrend: trend,
    );
  }
}

/// 每日趋势项
class DayTrendItem {
  final String date;
  final int sessions;
  final int messages;

  DayTrendItem({
    this.date = '',
    this.sessions = 0,
    this.messages = 0,
  });

  factory DayTrendItem.fromJson(Map<String, dynamic> json) {
    return DayTrendItem(
      date: json['date'] ?? '',
      sessions: json['sessions'] ?? 0,
      messages: json['messages'] ?? 0,
    );
  }
}

/// 审计日志（对齐后端 model.AuditLog）
class AuditLog {
  final int id;
  final int? userId;
  final String username;
  final String role;
  final String action;
  final String resource;
  final String detail;
  final String traceId;
  final String ip;
  final int durationMs;
  final int resultCode;
  final String createdAt;

  AuditLog({
    required this.id,
    this.userId,
    this.username = '',
    this.role = '',
    this.action = '',
    this.resource = '',
    this.detail = '',
    this.traceId = '',
    this.ip = '',
    this.durationMs = 0,
    this.resultCode = 0,
    this.createdAt = '',
  });

  factory AuditLog.fromJson(Map<String, dynamic> json) {
    return AuditLog(
      id: json['id'] ?? 0,
      userId: json['user_id'],
      username: json['username'] ?? '',
      role: json['role'] ?? '',
      action: json['action'] ?? '',
      resource: json['resource'] ?? '',
      detail: json['detail'] ?? '',
      traceId: json['trace_id'] ?? '',
      ip: json['ip'] ?? '',
      durationMs: json['duration_ms'] ?? 0,
      resultCode: json['result_code'] ?? 0,
      createdAt: json['created_at'] ?? '',
    );
  }

  String get actionLabel {
    const map = {
      'login': '登录',
      'chat': '对话',
      'knowledge_browse': '浏览知识',
      'export': '导出',
      'profile_update': '修改资料',
      'api_call': 'API 调用',
    };
    return map[action] ?? action;
  }
}

/// 用户反馈（对齐后端 model.Feedback）
class FeedbackEntry {
  final int id;
  final String feedbackId;
  final int userId;
  final String username;
  final String messageId;
  final String resourceId;
  final String category;
  final String module;
  final String content;
  final String screenshotUrl;
  final String status;
  final String resolvedBy;
  final String? resolvedAt;
  final String reply;
  final int rating;
  final String ratingComment;
  final String? ratedAt;
  final String linkedResourceNote;
  final String? linkedAt;
  final String linkedBy;
  final String createdAt;
  final String updatedAt;

  FeedbackEntry({
    required this.id,
    this.feedbackId = '',
    this.userId = 0,
    this.username = '',
    this.messageId = '',
    this.resourceId = '',
    this.category = 'answer_error',
    this.module = '',
    this.content = '',
    this.screenshotUrl = '',
    this.status = 'pending',
    this.resolvedBy = '',
    this.resolvedAt,
    this.reply = '',
    this.rating = 0,
    this.ratingComment = '',
    this.ratedAt,
    this.linkedResourceNote = '',
    this.linkedAt,
    this.linkedBy = '',
    this.createdAt = '',
    this.updatedAt = '',
  });

  factory FeedbackEntry.fromJson(Map<String, dynamic> json) {
    return FeedbackEntry(
      id: json['id'] ?? 0,
      feedbackId: json['feedback_id'] ?? '',
      userId: json['user_id'] ?? 0,
      username: json['username'] ?? '',
      messageId: json['message_id'] ?? '',
      resourceId: json['resource_id'] ?? '',
      category: json['category'] ?? 'answer_error',
      module: json['module'] ?? '',
      content: json['content'] ?? '',
      screenshotUrl: json['screenshot_url'] ?? '',
      status: json['status'] ?? 'pending',
      resolvedBy: json['resolved_by'] ?? '',
      resolvedAt: json['resolved_at'],
      reply: json['reply'] ?? '',
      rating: json['rating'] ?? 0,
      ratingComment: json['rating_comment'] ?? '',
      ratedAt: json['rated_at'],
      linkedResourceNote: json['linked_resource_note'] ?? '',
      linkedAt: json['linked_at'],
      linkedBy: json['linked_by'] ?? '',
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }

  String get categoryLabel {
    const map = {
      'answer_error': '回答有误',
      'feature_request': '功能建议',
      'bug': '系统问题',
      'other': '其他',
    };
    return map[category] ?? category;
  }

  String get statusLabel {
    const map = {
      'pending': '待处理',
      'processing': '处理中',
      'resolved': '已解决',
      'dismissed': '已驳回',
    };
    return map[status] ?? status;
  }
}

/// 反馈所属模块枚举（与 backend moduleFilesMap / 前端 feedback_repair._moduleMap 对应）
const feedbackModules = <String>[
  '登录 / 认证',
  '对话 / 问答',
  '知识库 / 检索',
  '办事流程',
  '报到 / 校园导航',
  '语音',
  '我的 / 个人中心',
  '反馈系统',
  '消息 / 通知',
  '学生服务',
  '教务 / 课表',
  '心理 / 情感',
  '管理端 / 数据',
];

/// AI 在线修复诊断结果（对应后端 model.AIRepairResponse）
class AIRepairResult {
  final String module;
  final String summary;
  final List<String> codeFiles;
  final String rootCause;
  final String repairHint;
  final String ocrText;
  final List<String> matchedFiles;

  AIRepairResult({
    this.module = '',
    this.summary = '',
    this.codeFiles = const [],
    this.rootCause = '',
    this.repairHint = '',
    this.ocrText = '',
    this.matchedFiles = const [],
  });

  factory AIRepairResult.fromJson(Map<String, dynamic> json) {
    return AIRepairResult(
      module: json['module'] ?? '',
      summary: json['summary'] ?? '',
      codeFiles: (json['code_files'] as List?)?.cast<String>() ?? const [],
      rootCause: json['root_cause'] ?? '',
      repairHint: json['repair_hint'] ?? '',
      ocrText: json['ocr_text'] ?? '',
      matchedFiles:
          (json['matched_files'] as List?)?.cast<String>() ?? const [],
    );
  }
}

/// 反馈统计数据
class FeedbackStats {
  final int total;
  final Map<String, int> byStatus;
  final Map<String, int> byCategory;
  final List<WeekTrendItem> weekTrend;
  final List<TopIssueItem> topIssues;
  final double avgResolveHours;

  FeedbackStats({
    this.total = 0,
    this.byStatus = const {},
    this.byCategory = const {},
    this.weekTrend = const [],
    this.topIssues = const [],
    this.avgResolveHours = 0,
  });

  factory FeedbackStats.fromJson(Map<String, dynamic> json) {
    Map<String, int> parseMap(dynamic data) {
      if (data is Map) {
        return data
            .map((k, v) => MapEntry(k.toString(), (v as num?)?.toInt() ?? 0));
      }
      return {};
    }

    List<WeekTrendItem> weekTrend = [];
    if (json['week_trend'] is List) {
      weekTrend = (json['week_trend'] as List)
          .map((e) => WeekTrendItem.fromJson(e as Map<String, dynamic>))
          .toList();
    }

    List<TopIssueItem> topIssues = [];
    if (json['top_issues'] is List) {
      topIssues = (json['top_issues'] as List)
          .map((e) => TopIssueItem.fromJson(e as Map<String, dynamic>))
          .toList();
    }

    return FeedbackStats(
      total: json['total'] ?? 0,
      byStatus: parseMap(json['by_status']),
      byCategory: parseMap(json['by_category']),
      weekTrend: weekTrend,
      topIssues: topIssues,
      avgResolveHours: (json['avg_resolve_hours'] as num?)?.toDouble() ?? 0,
    );
  }
}

/// 周趋势项
class WeekTrendItem {
  final String date;
  final int count;

  WeekTrendItem({this.date = '', this.count = 0});

  factory WeekTrendItem.fromJson(Map<String, dynamic> json) {
    return WeekTrendItem(
      date: json['date'] ?? '',
      count: json['count'] ?? 0,
    );
  }
}

/// 热门问题项
class TopIssueItem {
  final String keyword;
  final int count;

  TopIssueItem({this.keyword = '', this.count = 0});

  factory TopIssueItem.fromJson(Map<String, dynamic> json) {
    return TopIssueItem(
      keyword: json['keyword'] ?? '',
      count: json['count'] ?? 0,
    );
  }
}

/// 反馈处理记录
class FeedbackLog {
  final int id;
  final String feedbackId;
  final String action;
  final String operator;
  final String detail;
  final String createdAt;

  FeedbackLog({
    this.id = 0,
    this.feedbackId = '',
    this.action = '',
    this.operator = '',
    this.detail = '',
    this.createdAt = '',
  });

  factory FeedbackLog.fromJson(Map<String, dynamic> json) {
    return FeedbackLog(
      id: json['id'] ?? 0,
      feedbackId: json['feedback_id'] ?? '',
      action: json['action'] ?? '',
      operator: json['operator'] ?? '',
      detail: json['detail'] ?? '',
      createdAt: json['created_at'] ?? '',
    );
  }

  String get actionLabel {
    const map = {
      'submit': '提交反馈',
      'status_change': '状态变更',
      'link_resource': '关联知识',
      'rate': '用户评价',
    };
    return map[action] ?? action;
  }
}

/// 系统配置项（对齐后端 model.SystemSetting）
class SystemSetting {
  final int id;
  final String key;
  final String value;
  final String description;
  final String updatedBy;
  final String createdAt;
  final String updatedAt;

  SystemSetting({
    required this.id,
    this.key = '',
    this.value = '',
    this.description = '',
    this.updatedBy = '',
    this.createdAt = '',
    this.updatedAt = '',
  });

  factory SystemSetting.fromJson(Map<String, dynamic> json) {
    return SystemSetting(
      id: json['id'] ?? 0,
      key: json['key'] ?? '',
      value: json['value'] ?? '',
      description: json['description'] ?? '',
      updatedBy: json['updated_by'] ?? '',
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }
}

/// 创建/更新智能体请求
class AgentSaveRequest {
  final String agentId;
  final String name;
  final String? description;
  final String agentType;
  final String? systemPrompt;
  final String? modelProvider;
  final String? modelName;
  final double? temperature;
  final int? maxTokens;

  AgentSaveRequest({
    required this.agentId,
    required this.name,
    this.description,
    required this.agentType,
    this.systemPrompt,
    this.modelProvider,
    this.modelName,
    this.temperature,
    this.maxTokens,
  });

  Map<String, dynamic> toJson() => {
        'agent_id': agentId,
        'name': name,
        if (description != null) 'description': description,
        'agent_type': agentType,
        if (systemPrompt != null) 'system_prompt': systemPrompt,
        if (modelProvider != null) 'model_provider': modelProvider,
        if (modelName != null) 'model_name': modelName,
        if (temperature != null) 'temperature': temperature,
        if (maxTokens != null) 'max_tokens': maxTokens,
      };
}

// ═══════════════════════════════════════════════════════════════
// 学生 AI 功能模型
// ═══════════════════════════════════════════════════════════════

/// AI 今日速览
class DailyBriefing {
  final String date;
  final String greeting;
  final List<BriefingItem> courses;
  final List<BriefingItem> deadlines;
  final List<BriefingItem> activities;
  final String weather;
  final String motto;

  DailyBriefing({
    required this.date,
    this.greeting = '',
    this.courses = const [],
    this.deadlines = const [],
    this.activities = const [],
    this.weather = '',
    this.motto = '',
  });

  factory DailyBriefing.fromJson(Map<String, dynamic> json) {
    return DailyBriefing(
      date: json['date'] ?? '',
      greeting: json['greeting'] ?? '',
      courses: (json['courses'] as List?)
              ?.map((e) => BriefingItem.fromJson(e))
              .toList() ??
          [],
      deadlines: (json['deadlines'] as List?)
              ?.map((e) => BriefingItem.fromJson(e))
              .toList() ??
          [],
      activities: (json['activities'] as List?)
              ?.map((e) => BriefingItem.fromJson(e))
              .toList() ??
          [],
      weather: json['weather'] ?? '',
      motto: json['motto'] ?? '',
    );
  }
}

class BriefingItem {
  final String title;
  final String subtitle;
  final String time;
  final String icon;

  BriefingItem(
      {this.title = '', this.subtitle = '', this.time = '', this.icon = ''});

  factory BriefingItem.fromJson(Map<String, dynamic> json) {
    return BriefingItem(
      title: json['title'] ?? '',
      subtitle: json['subtitle'] ?? '',
      time: json['time'] ?? '',
      icon: json['icon'] ?? '',
    );
  }
}

/// AI 学习日记
class LearningDiary {
  final String date;
  final List<String> coursesStudied;
  final List<String> keyPoints;
  final int studyMinutes;
  final List<QuizItem> quiz;
  final String tomorrowPlan;
  final String encouragement;

  LearningDiary({
    required this.date,
    this.coursesStudied = const [],
    this.keyPoints = const [],
    this.studyMinutes = 0,
    this.quiz = const [],
    this.tomorrowPlan = '',
    this.encouragement = '',
  });

  factory LearningDiary.fromJson(Map<String, dynamic> json) {
    return LearningDiary(
      date: json['date'] ?? '',
      coursesStudied: List<String>.from(json['courses_studied'] ?? []),
      keyPoints: List<String>.from(json['key_points'] ?? []),
      studyMinutes: json['study_minutes'] ?? 0,
      quiz:
          (json['quiz'] as List?)?.map((e) => QuizItem.fromJson(e)).toList() ??
              [],
      tomorrowPlan: json['tomorrow_plan'] ?? '',
      encouragement: json['encouragement'] ?? '',
    );
  }
}

class QuizItem {
  final String question;
  final List<String> options;
  final int correctIndex;
  final String explanation;

  QuizItem(
      {this.question = '',
      this.options = const [],
      this.correctIndex = 0,
      this.explanation = ''});

  factory QuizItem.fromJson(Map<String, dynamic> json) {
    return QuizItem(
      question: json['question'] ?? '',
      options: List<String>.from(json['options'] ?? []),
      correctIndex: json['correct_index'] ?? 0,
      explanation: json['explanation'] ?? '',
    );
  }
}

/// 个人数字孪生
class DigitalTwinData {
  final List<TwinDimension> dimensions;
  final List<TwinDimension> idealDimensions;
  final String aiSummary;
  final List<String> suggestions;

  DigitalTwinData(
      {this.dimensions = const [],
      this.idealDimensions = const [],
      this.aiSummary = '',
      this.suggestions = const []});

  factory DigitalTwinData.fromJson(Map<String, dynamic> json) {
    return DigitalTwinData(
      dimensions: (json['dimensions'] as List?)
              ?.map((e) => TwinDimension.fromJson(e))
              .toList() ??
          [],
      idealDimensions: (json['ideal_dimensions'] as List?)
              ?.map((e) => TwinDimension.fromJson(e))
              .toList() ??
          [],
      // 后端真实孪生返回 interpretation / stage_advice（v1 字段对齐）
      aiSummary: json['ai_summary'] ?? json['interpretation'] ?? '',
      suggestions:
          List<String>.from(json['suggestions'] ?? json['stage_advice'] ?? []),
    );
  }
}

class TwinDimension {
  final String name;
  final double score;
  final String label;
  final bool dataAvailable;

  TwinDimension(
      {this.name = '',
      this.score = 0,
      this.label = '',
      this.dataAvailable = true});

  factory TwinDimension.fromJson(Map<String, dynamic> json) {
    return TwinDimension(
      name: json['name'] ?? '',
      score: (json['score'] ?? 0).toDouble(),
      label: json['label'] ?? json['level'] ?? '',
      // 后端 v1 无 data_available 字段时默认 true，兼容旧接口/兜底 mock
      dataAvailable: json['data_available'] ?? true,
    );
  }
}

/// 打卡记录
class CheckinRecord {
  final String date;
  final int streak;
  final int totalDays;
  final int longestStreak;
  final bool todayChecked;
  final List<String> recentDates;

  CheckinRecord(
      {this.date = '',
      this.streak = 0,
      this.totalDays = 0,
      this.longestStreak = 0,
      this.todayChecked = false,
      this.recentDates = const []});

  factory CheckinRecord.fromJson(Map<String, dynamic> json) {
    return CheckinRecord(
      date: json['date'] ?? '',
      streak: json['streak'] ?? 0,
      totalDays: json['total_days'] ?? 0,
      longestStreak: json['longest_streak'] ?? 0,
      todayChecked: json['today_checked'] ?? false,
      recentDates: List<String>.from(json['recent_dates'] ?? []),
    );
  }
}

/// 学习积分与成就
class AchievementData {
  final int totalPoints;
  final int level;
  final String levelName;
  final int nextLevelPoints;
  final List<Achievement> badges;
  final int weeklyRank;

  AchievementData(
      {this.totalPoints = 0,
      this.level = 1,
      this.levelName = '青铜',
      this.nextLevelPoints = 100,
      this.badges = const [],
      this.weeklyRank = 0});

  factory AchievementData.fromJson(Map<String, dynamic> json) {
    return AchievementData(
      totalPoints: json['total_points'] ?? 0,
      level: json['level'] ?? 1,
      levelName: json['level_name'] ?? '青铜',
      nextLevelPoints: json['next_level_points'] ?? 100,
      badges: (json['badges'] as List?)
              ?.map((e) => Achievement.fromJson(e))
              .toList() ??
          [],
      weeklyRank: json['weekly_rank'] ?? 0,
    );
  }
}

class Achievement {
  final String id;
  final String name;
  final String icon;
  final String description;
  final bool unlocked;
  final String unlockedAt;

  Achievement(
      {this.id = '',
      this.name = '',
      this.icon = '',
      this.description = '',
      this.unlocked = false,
      this.unlockedAt = ''});

  factory Achievement.fromJson(Map<String, dynamic> json) {
    return Achievement(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      icon: json['icon'] ?? '',
      description: json['description'] ?? '',
      unlocked: json['unlocked'] ?? false,
      unlockedAt: json['unlocked_at'] ?? '',
    );
  }
}

/// 课程地图节点
class CourseNode {
  final String id;
  final String name;
  final int credits;
  final int semester;
  final String status;
  final List<String> prerequisites;
  final String category;
  final double mastery;

  CourseNode(
      {this.id = '',
      this.name = '',
      this.credits = 0,
      this.semester = 1,
      this.status = 'pending',
      this.prerequisites = const [],
      this.category = '',
      this.mastery = 0});

  factory CourseNode.fromJson(Map<String, dynamic> json) {
    return CourseNode(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      credits: (json['credits'] ?? 0).toDouble().toInt(),
      semester: json['semester'] ?? 1,
      status: json['status'] ?? 'pending',
      prerequisites: List<String>.from(json['prerequisites'] ?? []),
      category: json['category'] ?? '',
      mastery: (json['mastery'] ?? 0).toDouble(),
    );
  }

  String get statusLabel =>
      {
        'completed': '已修',
        'current': '在修',
        'pending': '待修',
        'elective': '可选'
      }[status] ??
      '待修';
}

/// 课程学情看板
class CourseAnalyticsData {
  final String courseName;
  final double score;
  final double gpa;
  final String gradeLevel;
  final bool passed;
  final double credits;
  final String semester;
  final double progress;
  final int rankPercentile;
  final List<KnowledgePoint> knowledgePoints;
  final List<String> weakPoints;

  CourseAnalyticsData(
      {this.courseName = '',
      this.score = 0,
      this.gpa = 0,
      this.gradeLevel = '',
      this.passed = false,
      this.credits = 0,
      this.semester = '',
      this.progress = 0,
      this.rankPercentile = 50,
      this.knowledgePoints = const [],
      this.weakPoints = const []});

  factory CourseAnalyticsData.fromJson(Map<String, dynamic> json) {
    final score = (json['score'] ?? 0).toDouble();
    return CourseAnalyticsData(
      courseName: json['course_name'] ?? '',
      score: score,
      gpa: (json['gpa'] ?? 0).toDouble(),
      gradeLevel: json['grade_level'] ?? '',
      passed: json['passed'] ?? false,
      credits: (json['credits'] ?? 0).toDouble(),
      semester: (json['semester'] ?? '').toString(),
      progress:
          (json['progress'] ?? (score > 0 ? score / 100.0 : 0)).toDouble(),
      rankPercentile: json['rank_percentile'] ?? 50,
      knowledgePoints: (json['knowledge_points'] as List?)
              ?.map((e) => KnowledgePoint.fromJson(e))
              .toList() ??
          [],
      weakPoints: List<String>.from(json['weak_points'] ?? []),
    );
  }
}

class KnowledgePoint {
  final String name;
  final double mastery;

  KnowledgePoint({this.name = '', this.mastery = 0});

  factory KnowledgePoint.fromJson(Map<String, dynamic> json) {
    return KnowledgePoint(
        name: json['name'] ?? '', mastery: (json['mastery'] ?? 0).toDouble());
  }

  String get level => mastery >= 0.8
      ? 'good'
      : mastery >= 0.5
          ? 'medium'
          : 'weak';
}

// ═══════════════════════════════════════════════════════════════
// 辅导员 AI 功能模型
// ═══════════════════════════════════════════════════════════════

/// AI 今日关注
class DailyFocusData {
  final String date;
  final double classHealthScore;
  final List<FocusStudent> topStudents;
  final Map<String, int> overview;
  final String aiNarrative;
  final String dataSource;

  DailyFocusData({
    this.date = '',
    this.classHealthScore = 0,
    this.topStudents = const [],
    this.overview = const {},
    this.aiNarrative = '',
    this.dataSource = '',
  });

  factory DailyFocusData.fromJson(Map<String, dynamic> json) {
    return DailyFocusData(
      date: json['date'] ?? '',
      classHealthScore: (json['class_health_score'] ?? 0).toDouble(),
      topStudents: (json['top_students'] as List?)
              ?.map((e) => FocusStudent.fromJson(e))
              .toList() ??
          [],
      overview: Map<String, int>.from(json['overview'] ?? {}),
      aiNarrative: json['ai_narrative'] ?? '',
      dataSource: json['data_source'] ?? '',
    );
  }
}

class FocusStudent {
  final String name;
  final String reason;
  final String riskLevel;
  final String suggestion;

  FocusStudent(
      {this.name = '',
      this.reason = '',
      this.riskLevel = 'low',
      this.suggestion = ''});

  factory FocusStudent.fromJson(Map<String, dynamic> json) {
    return FocusStudent(
      name: json['name'] ?? '',
      reason: json['reason'] ?? '',
      riskLevel: json['risk_level'] ?? 'low',
      suggestion: json['suggestion'] ?? '',
    );
  }
}

/// 班级学情日报
class ClassReportData {
  final String date;
  final String className;
  final double activeRate;
  final int absentCount;
  final double homeworkRate;
  final int emotionAlertCount;
  final double checkinRate;
  final List<String> anomalies;
  final String aiNarrative;

  ClassReportData({
    this.date = '',
    this.className = '',
    this.activeRate = 0,
    this.absentCount = 0,
    this.homeworkRate = 0,
    this.emotionAlertCount = 0,
    this.checkinRate = 0,
    this.anomalies = const [],
    this.aiNarrative = '',
  });

  factory ClassReportData.fromJson(Map<String, dynamic> json) {
    return ClassReportData(
      date: json['date'] ?? '',
      className: json['class_name'] ?? '',
      activeRate: (json['active_rate'] ?? 0).toDouble(),
      absentCount: json['absent_count'] ?? 0,
      homeworkRate: (json['homework_rate'] ?? 0).toDouble(),
      emotionAlertCount: json['emotion_alert_count'] ?? 0,
      checkinRate: (json['checkin_rate'] ?? 0).toDouble(),
      anomalies: List<String>.from(json['anomalies'] ?? []),
      aiNarrative: json['ai_narrative'] ?? '',
    );
  }
}

/// 谈心谈话记录
class TalkRecord {
  final String id;
  final String studentName;
  final String date;
  final String topic;
  final String emotion;
  final String summary;
  final List<String> followUps;
  final String status;

  TalkRecord(
      {this.id = '',
      this.studentName = '',
      this.date = '',
      this.topic = '',
      this.emotion = '',
      this.summary = '',
      this.followUps = const [],
      this.status = 'pending'});

  factory TalkRecord.fromJson(Map<String, dynamic> json) {
    return TalkRecord(
      id: json['id'] ?? '',
      studentName: json['student_name'] ?? '',
      date: json['created_at'] ?? json['date'] ?? '',
      topic: json['topic'] ?? '',
      emotion: json['emotion'] ?? '',
      summary: json['summary'] ?? '',
      followUps: List<String>.from(json['follow_ups'] ?? []),
      status: json['status'] ?? 'pending',
    );
  }
}

// ═══════════════════════════════════════════════════════════════
// 教师 AI 功能模型
// ═══════════════════════════════════════════════════════════════

/// AI 备课助手输出
class LessonPlan {
  final String topic;
  final String outline;
  final List<String> keyPoints;
  final List<String> difficulties;
  final List<String> strategies;
  final List<String> interactions;
  final List<String> homework;

  LessonPlan(
      {this.topic = '',
      this.outline = '',
      this.keyPoints = const [],
      this.difficulties = const [],
      this.strategies = const [],
      this.interactions = const [],
      this.homework = const []});

  factory LessonPlan.fromJson(Map<String, dynamic> json) {
    return LessonPlan(
      topic: json['topic'] ?? '',
      outline: json['outline'] ?? '',
      keyPoints: List<String>.from(json['key_points'] ?? []),
      difficulties: List<String>.from(json['difficulties'] ?? []),
      strategies: List<String>.from(json['strategies'] ?? []),
      interactions: List<String>.from(json['interactions'] ?? []),
      homework: List<String>.from(json['homework'] ?? []),
    );
  }
}

/// 班级学情热力图数据
class ClassHeatmapData {
  final String courseName;
  final List<KnowledgePoint> points;
  final List<String> weakTopFive;
  final int totalStudents;
  final int anomalyCount;

  ClassHeatmapData(
      {this.courseName = '',
      this.points = const [],
      this.weakTopFive = const [],
      this.totalStudents = 0,
      this.anomalyCount = 0});

  factory ClassHeatmapData.fromJson(Map<String, dynamic> json) {
    return ClassHeatmapData(
      courseName: json['course_name'] ?? '',
      points: (json['points'] as List?)
              ?.map((e) => KnowledgePoint.fromJson(e))
              .toList() ??
          [],
      weakTopFive: List<String>.from(json['weak_top_five'] ?? []),
      totalStudents: json['total_students'] ?? 0,
      anomalyCount: json['anomaly_count'] ?? 0,
    );
  }
}

// ═══════════════════════════════════════════════════════════════
// AI 模型配置模型
// ═══════════════════════════════════════════════════════════════

/// 用户 AI 模型配置（对齐后端 model.UserModelConfig）
class ModelConfig {
  final int id;
  final int userId;
  final String deepseekKey;
  final String deepseekModel;
  final double deepseekTemp;
  final int deepseekMaxTokens;
  final String zhipuKey;
  final String zhipuModel;
  final double zhipuTemp;
  final int zhipuMaxTokens;
  final String xunfeiAppId;
  final String xunfeiKey;
  final String xunfeiSecret;
  final String xunfeiModel;
  final double xunfeiTemp;
  final int xunfeiMaxTokens;
  final String defaultProvider;
  final String createdAt;
  final String updatedAt;

  ModelConfig({
    this.id = 0,
    this.userId = 0,
    this.deepseekKey = '',
    this.deepseekModel = '',
    this.deepseekTemp = 0.7,
    this.deepseekMaxTokens = 2048,
    this.zhipuKey = '',
    this.zhipuModel = '',
    this.zhipuTemp = 0.7,
    this.zhipuMaxTokens = 2048,
    this.xunfeiAppId = '',
    this.xunfeiKey = '',
    this.xunfeiSecret = '',
    this.xunfeiModel = '',
    this.xunfeiTemp = 0.7,
    this.xunfeiMaxTokens = 2048,
    this.defaultProvider = 'deepseek',
    this.createdAt = '',
    this.updatedAt = '',
  });

  factory ModelConfig.fromJson(Map<String, dynamic> json) {
    return ModelConfig(
      id: json['id'] ?? 0,
      userId: json['user_id'] ?? 0,
      deepseekKey: json['deepseek_key'] ?? '',
      deepseekModel: json['deepseek_model'] ?? '',
      deepseekTemp: (json['deepseek_temp'] ?? 0.7).toDouble(),
      deepseekMaxTokens: json['deepseek_max_tokens'] ?? 2048,
      zhipuKey: json['zhipu_key'] ?? '',
      zhipuModel: json['zhipu_model'] ?? '',
      zhipuTemp: (json['zhipu_temp'] ?? 0.7).toDouble(),
      zhipuMaxTokens: json['zhipu_max_tokens'] ?? 2048,
      xunfeiAppId: json['xunfei_app_id'] ?? '',
      xunfeiKey: json['xunfei_key'] ?? '',
      xunfeiSecret: json['xunfei_secret'] ?? '',
      xunfeiModel: json['xunfei_model'] ?? '',
      xunfeiTemp: (json['xunfei_temp'] ?? 0.7).toDouble(),
      xunfeiMaxTokens: json['xunfei_max_tokens'] ?? 2048,
      defaultProvider: json['default_provider'] ?? 'deepseek',
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }

  /// 默认模型中文名
  String get defaultProviderLabel {
    const map = {'deepseek': 'DeepSeek', 'zhipu': '智谱清言', 'xunfei': '讯飞星火'};
    return map[defaultProvider] ?? defaultProvider;
  }

  Map<String, dynamic> toJson() => {
        'deepseek_key': deepseekKey,
        'deepseek_model': deepseekModel,
        'deepseek_temp': deepseekTemp,
        'deepseek_max_tokens': deepseekMaxTokens,
        'zhipu_key': zhipuKey,
        'zhipu_model': zhipuModel,
        'zhipu_temp': zhipuTemp,
        'zhipu_max_tokens': zhipuMaxTokens,
        'xunfei_app_id': xunfeiAppId,
        'xunfei_key': xunfeiKey,
        'xunfei_secret': xunfeiSecret,
        'xunfei_model': xunfeiModel,
        'xunfei_temp': xunfeiTemp,
        'xunfei_max_tokens': xunfeiMaxTokens,
        'default_provider': defaultProvider,
      };
}

// ═══════════════════════════════════════════════════════════════
// 词元统计模型
// ═══════════════════════════════════════════════════════════════

class TokenDailyPoint {
  final String date;
  final int promptTokens;
  final int outputTokens;
  final int totalTokens;

  TokenDailyPoint(
      {this.date = '',
      this.promptTokens = 0,
      this.outputTokens = 0,
      this.totalTokens = 0});

  factory TokenDailyPoint.fromJson(Map<String, dynamic> json) {
    return TokenDailyPoint(
      date: json['date'] ?? '',
      promptTokens: json['prompt_tokens'] ?? 0,
      outputTokens: json['output_tokens'] ?? 0,
      totalTokens: json['total_tokens'] ?? 0,
    );
  }
}

class TokenStatsSummary {
  final int totalPromptTokens;
  final int totalOutputTokens;
  final int totalTokens;
  final int todayTokens;

  const TokenStatsSummary(
      {this.totalPromptTokens = 0,
      this.totalOutputTokens = 0,
      this.totalTokens = 0,
      this.todayTokens = 0});

  factory TokenStatsSummary.fromJson(Map<String, dynamic> json) {
    return TokenStatsSummary(
      totalPromptTokens: json['total_prompt_tokens'] ?? 0,
      totalOutputTokens: json['total_output_tokens'] ?? 0,
      totalTokens: json['total_tokens'] ?? 0,
      todayTokens: json['today_tokens'] ?? 0,
    );
  }
}

class TokenStatsData {
  final TokenStatsSummary summary;
  final List<TokenDailyPoint> daily;

  TokenStatsData({TokenStatsSummary? summary, List<TokenDailyPoint>? daily})
      : summary = summary ?? const TokenStatsSummary(),
        daily = daily ?? const [];

  factory TokenStatsData.fromJson(Map<String, dynamic> json) {
    return TokenStatsData(
      summary: json['summary'] != null
          ? TokenStatsSummary.fromJson(json['summary'])
          : null,
      daily: (json['daily'] as List?)
          ?.map((e) => TokenDailyPoint.fromJson(e))
          .toList(),
    );
  }
}

class SubordinateTokenStats {
  final int userId;
  final String username;
  final String displayName;
  final int totalTokens;
  final int promptTokens;
  final int outputTokens;

  SubordinateTokenStats(
      {this.userId = 0,
      this.username = '',
      this.displayName = '',
      this.totalTokens = 0,
      this.promptTokens = 0,
      this.outputTokens = 0});

  factory SubordinateTokenStats.fromJson(Map<String, dynamic> json) {
    return SubordinateTokenStats(
      userId: json['user_id'] ?? 0,
      username: json['username'] ?? '',
      displayName: json['display_name'] ?? '',
      totalTokens: json['total_tokens'] ?? 0,
      promptTokens: json['prompt_tokens'] ?? 0,
      outputTokens: json['output_tokens'] ?? 0,
    );
  }
}

/// 办事流程办理记录（与后端 process_records 表对应）
class ProcessRecord {
  final int id;
  final String recordId;
  final int userId;
  final String flowType;
  final String flowLabel;
  final int currentStep;
  final List<int> completedSteps;
  final int totalSteps;
  final String status; // in_progress / completed / abandoned
  final String notes;
  final String createdAt;
  final String updatedAt;

  ProcessRecord({
    this.id = 0,
    this.recordId = '',
    this.userId = 0,
    this.flowType = '',
    this.flowLabel = '',
    this.currentStep = 0,
    this.completedSteps = const [],
    this.totalSteps = 0,
    this.status = 'in_progress',
    this.notes = '',
    this.createdAt = '',
    this.updatedAt = '',
  });

  factory ProcessRecord.fromJson(Map<String, dynamic> json) {
    // completed_steps 从后端是 JSON 字符串（例如 "[0,1,3]"），需要解码
    List<int> parseSteps(dynamic raw) {
      if (raw is List) {
        return raw.map((e) => (e as num).toInt()).toList();
      }
      if (raw is String && raw.isNotEmpty) {
        try {
          final decoded = jsonDecode(raw);
          if (decoded is List) {
            return decoded.map((e) => (e as num).toInt()).toList();
          }
        } catch (_) {}
      }
      return const [];
    }

    return ProcessRecord(
      id: json['id'] is int ? json['id'] : int.tryParse('${json['id']}') ?? 0,
      recordId: json['record_id'] ?? '',
      userId: json['user_id'] is int
          ? json['user_id']
          : int.tryParse('${json['user_id']}') ?? 0,
      flowType: json['flow_type'] ?? '',
      flowLabel: json['flow_label'] ?? '',
      currentStep: json['current_step'] ?? 0,
      completedSteps: parseSteps(json['completed_steps']),
      totalSteps: json['total_steps'] ?? 0,
      status: json['status'] ?? 'in_progress',
      notes: json['notes'] ?? '',
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }

  String get statusLabel {
    switch (status) {
      case 'completed':
        return '已完成';
      case 'abandoned':
        return '已放弃';
      default:
        return '进行中';
    }
  }

  double get progressRatio {
    if (totalSteps <= 0) return 0;
    return completedSteps.length / totalSteps;
  }
}

/// 导入结果数据
class ImportResultData {
  final int total;
  final int success;
  final int failed;
  final List<ImportResultDetail> details;

  const ImportResultData({
    required this.total,
    required this.success,
    required this.failed,
    required this.details,
  });

  factory ImportResultData.fromJson(Map<String, dynamic> json) {
    return ImportResultData(
      total: json['total'] ?? 0,
      success: json['success'] ?? 0,
      failed: json['failed'] ?? 0,
      details: (json['details'] as List?)
              ?.map(
                  (e) => ImportResultDetail.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }
}

/// 导入结果明细
class ImportResultDetail {
  final String username;
  final String displayName;
  final bool success;
  final String error;

  const ImportResultDetail({
    required this.username,
    required this.displayName,
    required this.success,
    required this.error,
  });

  factory ImportResultDetail.fromJson(Map<String, dynamic> json) {
    return ImportResultDetail(
      username: json['username'] ?? '',
      displayName: json['display_name'] ?? '',
      success: json['success'] ?? false,
      error: json['error'] ?? '',
    );
  }
}

// ═══════════════════════════════════════════════════════════════
// 第三方应用（应用中心）
// ═══════════════════════════════════════════════════════════════

/// 第三方应用（GET /api/v1/apps 返回的可见应用）
class ExternalAppItem {
  final String id;
  final String name;
  final String icon;
  final String category; // study | culture | service | admin | external
  final String summary;
  final String version;
  final String type; // external_link | webview | reverse_proxy
  final String url;
  final String openIn; // _self | _blank | _native

  const ExternalAppItem({
    this.id = '',
    this.name = '',
    this.icon = '',
    this.category = 'external',
    this.summary = '',
    this.version = '',
    this.type = 'external_link',
    this.url = '',
    this.openIn = '_blank',
  });

  factory ExternalAppItem.fromJson(Map<String, dynamic> json) {
    return ExternalAppItem(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      icon: json['icon'] ?? '',
      category: json['category'] ?? 'external',
      summary: json['summary'] ?? '',
      version: json['version'] ?? '',
      type: json['type'] ?? 'external_link',
      url: json['url'] ?? '',
      openIn: json['open_in'] ?? '_blank',
    );
  }
}

// ═══════════════════════════════════════════════════════════════
// AI 简讯
// ═══════════════════════════════════════════════════════════════

/// AI 简讯（首页资讯）
class AIBriefing {
  final int id;
  final String source;
  final String category;
  final String topic;
  final String summary;
  final String content;
  final String link;
  final String keyword;
  final int heat;
  final String reason;
  final String publishedAt;
  final String fetchedAt;
  final int status;
  final bool favorited;

  const AIBriefing({
    this.id = 0,
    this.source = '',
    this.category = 'ai_teaching',
    this.topic = '',
    this.summary = '',
    this.content = '',
    this.link = '',
    this.keyword = '',
    this.heat = 0,
    this.reason = '',
    this.publishedAt = '',
    this.fetchedAt = '',
    this.status = 1,
    this.favorited = false,
  });

  factory AIBriefing.fromJson(Map<String, dynamic> json) {
    return AIBriefing(
      id: json['id'] ?? 0,
      source: json['source'] ?? '',
      category: json['category'] ?? 'ai_teaching',
      topic: json['topic'] ?? '',
      summary: json['summary'] ?? '',
      content: json['content'] ?? '',
      link: json['link'] ?? '',
      keyword: json['keyword'] ?? '',
      heat: json['heat'] ?? 0,
      reason: json['reason'] ?? '',
      publishedAt: json['published_at'] ?? '',
      fetchedAt: json['fetched_at'] ?? '',
      status: json['status'] ?? 1,
      favorited: json['favorited'] == true,
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'source': source,
        'category': category,
        'topic': topic,
        'summary': summary,
        'content': content,
        'link': link,
        'keyword': keyword,
        'heat': heat,
        'reason': reason,
        'published_at': publishedAt,
        'status': status,
      };

  AIBriefing copyWith({bool? favorited}) {
    return AIBriefing(
      id: id,
      source: source,
      category: category,
      topic: topic,
      summary: summary,
      content: content,
      link: link,
      keyword: keyword,
      heat: heat,
      reason: reason,
      publishedAt: publishedAt,
      fetchedAt: fetchedAt,
      status: status,
      favorited: favorited ?? this.favorited,
    );
  }
}

/// AI 简讯来源配置
class AIBriefingSource {
  final int id;
  final String name;
  final String url;
  final String category;
  final int enabled;
  final int fetchEnabled;
  final String fetchTime;
  final String lastFetchAt;

  const AIBriefingSource({
    this.id = 0,
    this.name = '',
    this.url = '',
    this.category = 'ai_teaching',
    this.enabled = 1,
    this.fetchEnabled = 1,
    this.fetchTime = '08:00',
    this.lastFetchAt = '',
  });

  factory AIBriefingSource.fromJson(Map<String, dynamic> json) {
    return AIBriefingSource(
      id: json['id'] ?? 0,
      name: json['name'] ?? '',
      url: json['url'] ?? '',
      category: json['category'] ?? 'ai_teaching',
      enabled: json['enabled'] ?? 1,
      fetchEnabled: json['fetch_enabled'] ?? 1,
      fetchTime: json['fetch_time'] ?? '08:00',
      lastFetchAt: json['last_fetch_at'] ?? '',
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'url': url,
        'category': category,
        'enabled': enabled,
        'fetch_enabled': fetchEnabled,
        'fetch_time': fetchTime,
      };
}

/// AI 简讯分类展示名
class AIBriefingCategory {
  final String key;
  final String label;
  const AIBriefingCategory(this.key, this.label);

  static const all = [
    AIBriefingCategory('ai_teaching', 'AI 辅助教学'),
    AIBriefingCategory('ai_tool', 'AI 工具'),
    AIBriefingCategory('ai_version', 'AI 版本'),
    AIBriefingCategory('ai_industry', 'AI 行业热点'),
  ];

  static String labelOf(String key) {
    for (final c in all) {
      if (c.key == key) return c.label;
    }
    return key;
  }
}

// ═══════════════════════════════════════════════════════════════
// 数字孪生画像
// ═══════════════════════════════════════════════════════════════

/// 数字孪生画像（CogView 生成，蔚小芯风格）
class TwinPortrait {
  final int id;
  final int userId;
  final String prototypeType; // photo | chao_xing
  final String promptVersion;
  final String imageBase64;
  final String imageMime;
  final bool hasPhoto;
  final String createdAt;

  const TwinPortrait({
    this.id = 0,
    this.userId = 0,
    this.prototypeType = 'chao_xing',
    this.promptVersion = '',
    this.imageBase64 = '',
    this.imageMime = 'image/png',
    this.hasPhoto = false,
    this.createdAt = '',
  });

  factory TwinPortrait.fromJson(Map<String, dynamic> json) {
    return TwinPortrait(
      id: json['id'] ?? 0,
      userId: json['user_id'] ?? 0,
      prototypeType: json['prototype_type'] ?? 'chao_xing',
      promptVersion: json['prompt_version'] ?? '',
      imageBase64: json['image_base64'] ?? '',
      imageMime: json['image_mime'] ?? 'image/png',
      hasPhoto: json['has_photo'] ?? false,
      createdAt: json['created_at'] ?? '',
    );
  }
}

// ═══════════════════════════════════════════════════════════════
// 个人详细信息（基本信息 + 联系方式 + 组织关系 + 门户凭证）
// ═══════════════════════════════════════════════════════════════

/// 组织关系联系人（辅导员/领导）
class ContactPerson {
  final int id;
  final String name;
  final String role;
  final String roleName;
  final String phone;
  final String wechat;
  final String email;

  const ContactPerson({
    this.id = 0,
    this.name = '',
    this.role = '',
    this.roleName = '',
    this.phone = '',
    this.wechat = '',
    this.email = '',
  });

  factory ContactPerson.fromJson(Map<String, dynamic> json) {
    return ContactPerson(
      id: json['id'] ?? 0,
      name: json['name'] ?? '',
      role: json['role'] ?? '',
      roleName: json['role_name'] ?? '',
      phone: json['phone'] ?? '',
      wechat: json['wechat'] ?? '',
      email: json['email'] ?? '',
    );
  }
}

/// 个人详细信息
class PersonalDetail {
  // 基本信息
  final int userId;
  final String username;
  final String displayName;
  final String role;
  final String college;
  final String major;
  final String className;
  final String enrollmentDate;
  final String enrollmentYear;
  // 联系方式
  final String phone;
  final String wechat;
  final String qq;
  final String email;
  // 头像
  final String avatarBase64;
  final String avatarMime;
  // 组织关系
  final List<ContactPerson> supervisors;
  final int subordinates;

  const PersonalDetail({
    this.userId = 0,
    this.username = '',
    this.displayName = '',
    this.role = '',
    this.college = '',
    this.major = '',
    this.className = '',
    this.enrollmentDate = '',
    this.enrollmentYear = '',
    this.phone = '',
    this.wechat = '',
    this.qq = '',
    this.email = '',
    this.avatarBase64 = '',
    this.avatarMime = 'image/png',
    this.supervisors = const [],
    this.subordinates = 0,
  });

  factory PersonalDetail.fromJson(Map<String, dynamic> json) {
    return PersonalDetail(
      userId: json['user_id'] ?? 0,
      username: json['username'] ?? '',
      displayName: json['display_name'] ?? '',
      role: json['role'] ?? '',
      college: json['college'] ?? '',
      major: json['major'] ?? '',
      className: json['class_name'] ?? '',
      enrollmentDate: json['enrollment_date'] ?? '',
      enrollmentYear: json['enrollment_year'] ?? '',
      phone: json['phone'] ?? '',
      wechat: json['wechat'] ?? '',
      qq: json['qq'] ?? '',
      email: json['email'] ?? '',
      avatarBase64: json['avatar_base64'] ?? '',
      avatarMime: json['avatar_mime'] ?? 'image/png',
      supervisors: (json['supervisors'] as List? ?? [])
          .whereType<Map>()
          .map((e) => ContactPerson.fromJson(Map<String, dynamic>.from(e)))
          .toList(),
      subordinates: json['subordinates'] ?? 0,
    );
  }
}

/// 学校门户凭证绑定状态（不含密码明文）
class PortalCredential {
  final int userId;
  final String portalUrl;
  final String portalAccount;
  final bool bound;
  final String updatedAt;

  const PortalCredential({
    this.userId = 0,
    this.portalUrl = 'https://my0.chzu.edu.cn/',
    this.portalAccount = '',
    this.bound = false,
    this.updatedAt = '',
  });

  factory PortalCredential.fromJson(Map<String, dynamic> json) {
    return PortalCredential(
      userId: json['user_id'] ?? 0,
      portalUrl: json['portal_url'] ?? 'https://my0.chzu.edu.cn/',
      portalAccount: json['portal_account'] ?? '',
      bound: json['bound'] ?? false,
      updatedAt: json['updated_at'] ?? '',
    );
  }
}

// ═══════════════════════════════════════════════════════════════
// 审计恢复快照
// ═══════════════════════════════════════════════════════════════

/// 审计恢复快照（可恢复的写操作前后状态）
class AuditSnapshot {
  final int id;
  final int auditId;
  final String opTable;
  final String recordId;
  final String operation;
  final String beforeJson;
  final String afterJson;
  final int restored;
  final String restoredAt;
  final String restoredBy;
  final String createdAt;

  const AuditSnapshot({
    this.id = 0,
    this.auditId = 0,
    this.opTable = '',
    this.recordId = '',
    this.operation = '',
    this.beforeJson = '',
    this.afterJson = '',
    this.restored = 0,
    this.restoredAt = '',
    this.restoredBy = '',
    this.createdAt = '',
  });

  factory AuditSnapshot.fromJson(Map<String, dynamic> json) {
    return AuditSnapshot(
      id: json['id'] ?? 0,
      auditId: json['audit_id'] ?? 0,
      opTable: json['op_table'] ?? '',
      recordId: json['record_id'] ?? '',
      operation: json['operation'] ?? '',
      beforeJson: json['before_json'] ?? '',
      afterJson: json['after_json'] ?? '',
      restored: json['restored'] ?? 0,
      restoredAt: json['restored_at'] ?? '',
      restoredBy: json['restored_by'] ?? '',
      createdAt: json['created_at'] ?? '',
    );
  }
}
