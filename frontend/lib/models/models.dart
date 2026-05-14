import 'dart:convert';

/// AnswerCard 统一回答结构，对齐后端 model.AnswerCard
class AnswerCard {
  final String conclusion;
  final List<String> steps;
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
      sources: (json['sources'] as List?)
              ?.map((s) => Source.fromJson(s))
              .toList() ??
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

/// 来源引用
class Source {
  final String resourceId;
  final String title;
  final String version;
  final String sourceLink;
  final double relevanceScore;

  Source({
    required this.resourceId,
    required this.title,
    this.version = '',
    this.sourceLink = '',
    this.relevanceScore = 0,
  });

  factory Source.fromJson(Map<String, dynamic> json) {
    return Source(
      resourceId: json['resource_id'] ?? '',
      title: json['title'] ?? '',
      version: json['version'] ?? '',
      sourceLink: json['source_link'] ?? '',
      relevanceScore: (json['relevance_score'] ?? 0).toDouble(),
    );
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
      id: json['id']?.toString() ?? '',
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

  Message copyWith({bool? isFailed}) {
    return Message(
      id: id,
      role: role,
      content: content,
      answerCard: answerCard,
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

  UserProfile({
    required this.id,
    required this.username,
    required this.role,
    required this.displayName,
    this.college = '',
    this.major = '',
  });

  factory UserProfile.fromJson(Map<String, dynamic> json) {
    // 后端可能将 data 包裹在 data 字段中
    final data = json['data'] ?? json;
    return UserProfile(
      id: data['id'] ?? 0,
      username: data['username'] ?? '',
      role: data['role'] ?? 'student',
      displayName: data['display_name'] ?? data['username'] ?? '',
      college: data['college'] ?? '',
      major: data['major'] ?? '',
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
  final List<String> tags;
  final String sourceLink;

  KnowledgeCard({
    required this.resourceId,
    required this.resourceType,
    required this.title,
    this.summary = '',
    this.tags = const [],
    this.sourceLink = '',
  });

  factory KnowledgeCard.fromJson(Map<String, dynamic> json) {
    return KnowledgeCard(
      resourceId: json['resource_id'] ?? '',
      resourceType: json['resource_type'] ?? '',
      title: json['title'] ?? '',
      summary: json['summary'] ?? '',
      tags: _parseTags(json['tags']),
      sourceLink: json['source_link'] ?? '',
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
      'emotion': '情感分析',
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
  final String content;
  final String status;
  final String resolvedBy;
  final String? resolvedAt;
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
    this.content = '',
    this.status = 'pending',
    this.resolvedBy = '',
    this.resolvedAt,
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
      content: json['content'] ?? '',
      status: json['status'] ?? 'pending',
      resolvedBy: json['resolved_by'] ?? '',
      resolvedAt: json['resolved_at'],
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }

  String get categoryLabel {
    const map = {
      'answer_error': '回答有误',
      'suggestion': '功能建议',
      'other': '其他',
    };
    return map[category] ?? category;
  }

  String get statusLabel {
    const map = {
      'pending': '待处理',
      'resolved': '已处理',
      'dismissed': '已驳回',
    };
    return map[status] ?? status;
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
      courses: (json['courses'] as List?)?.map((e) => BriefingItem.fromJson(e)).toList() ?? [],
      deadlines: (json['deadlines'] as List?)?.map((e) => BriefingItem.fromJson(e)).toList() ?? [],
      activities: (json['activities'] as List?)?.map((e) => BriefingItem.fromJson(e)).toList() ?? [],
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

  BriefingItem({this.title = '', this.subtitle = '', this.time = '', this.icon = ''});

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
      quiz: (json['quiz'] as List?)?.map((e) => QuizItem.fromJson(e)).toList() ?? [],
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

  QuizItem({this.question = '', this.options = const [], this.correctIndex = 0, this.explanation = ''});

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

  DigitalTwinData({this.dimensions = const [], this.idealDimensions = const [], this.aiSummary = '', this.suggestions = const []});

  factory DigitalTwinData.fromJson(Map<String, dynamic> json) {
    return DigitalTwinData(
      dimensions: (json['dimensions'] as List?)?.map((e) => TwinDimension.fromJson(e)).toList() ?? [],
      idealDimensions: (json['ideal_dimensions'] as List?)?.map((e) => TwinDimension.fromJson(e)).toList() ?? [],
      aiSummary: json['ai_summary'] ?? '',
      suggestions: List<String>.from(json['suggestions'] ?? []),
    );
  }
}

class TwinDimension {
  final String name;
  final double score;
  final String label;

  TwinDimension({this.name = '', this.score = 0, this.label = ''});

  factory TwinDimension.fromJson(Map<String, dynamic> json) {
    return TwinDimension(
      name: json['name'] ?? '',
      score: (json['score'] ?? 0).toDouble(),
      label: json['label'] ?? '',
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

  CheckinRecord({this.date = '', this.streak = 0, this.totalDays = 0, this.longestStreak = 0, this.todayChecked = false, this.recentDates = const []});

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

  AchievementData({this.totalPoints = 0, this.level = 1, this.levelName = '青铜', this.nextLevelPoints = 100, this.badges = const [], this.weeklyRank = 0});

  factory AchievementData.fromJson(Map<String, dynamic> json) {
    return AchievementData(
      totalPoints: json['total_points'] ?? 0,
      level: json['level'] ?? 1,
      levelName: json['level_name'] ?? '青铜',
      nextLevelPoints: json['next_level_points'] ?? 100,
      badges: (json['badges'] as List?)?.map((e) => Achievement.fromJson(e)).toList() ?? [],
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

  Achievement({this.id = '', this.name = '', this.icon = '', this.description = '', this.unlocked = false, this.unlockedAt = ''});

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

  CourseNode({this.id = '', this.name = '', this.credits = 0, this.semester = 1, this.status = 'pending', this.prerequisites = const [], this.category = ''});

  factory CourseNode.fromJson(Map<String, dynamic> json) {
    return CourseNode(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      credits: json['credits'] ?? 0,
      semester: json['semester'] ?? 1,
      status: json['status'] ?? 'pending',
      prerequisites: List<String>.from(json['prerequisites'] ?? []),
      category: json['category'] ?? '',
    );
  }

  String get statusLabel => {'completed': '已修', 'current': '在修', 'pending': '待修', 'elective': '可选'}[status] ?? '待修';
}

/// 课程学情看板
class CourseAnalyticsData {
  final String courseName;
  final double progress;
  final int rankPercentile;
  final List<KnowledgePoint> knowledgePoints;
  final List<String> weakPoints;

  CourseAnalyticsData({this.courseName = '', this.progress = 0, this.rankPercentile = 50, this.knowledgePoints = const [], this.weakPoints = const []});

  factory CourseAnalyticsData.fromJson(Map<String, dynamic> json) {
    return CourseAnalyticsData(
      courseName: json['course_name'] ?? '',
      progress: (json['progress'] ?? 0).toDouble(),
      rankPercentile: json['rank_percentile'] ?? 50,
      knowledgePoints: (json['knowledge_points'] as List?)?.map((e) => KnowledgePoint.fromJson(e)).toList() ?? [],
      weakPoints: List<String>.from(json['weak_points'] ?? []),
    );
  }
}

class KnowledgePoint {
  final String name;
  final double mastery;

  KnowledgePoint({this.name = '', this.mastery = 0});

  factory KnowledgePoint.fromJson(Map<String, dynamic> json) {
    return KnowledgePoint(name: json['name'] ?? '', mastery: (json['mastery'] ?? 0).toDouble());
  }

  String get level => mastery >= 0.8 ? 'good' : mastery >= 0.5 ? 'medium' : 'weak';
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

  DailyFocusData({this.date = '', this.classHealthScore = 0, this.topStudents = const [], this.overview = const {}});

  factory DailyFocusData.fromJson(Map<String, dynamic> json) {
    return DailyFocusData(
      date: json['date'] ?? '',
      classHealthScore: (json['class_health_score'] ?? 0).toDouble(),
      topStudents: (json['top_students'] as List?)?.map((e) => FocusStudent.fromJson(e)).toList() ?? [],
      overview: Map<String, int>.from(json['overview'] ?? {}),
    );
  }
}

class FocusStudent {
  final String name;
  final String reason;
  final String riskLevel;
  final String suggestion;

  FocusStudent({this.name = '', this.reason = '', this.riskLevel = 'low', this.suggestion = ''});

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
    this.date = '', this.className = '', this.activeRate = 0, this.absentCount = 0,
    this.homeworkRate = 0, this.emotionAlertCount = 0, this.checkinRate = 0,
    this.anomalies = const [], this.aiNarrative = '',
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

  TalkRecord({this.id = '', this.studentName = '', this.date = '', this.topic = '', this.emotion = '', this.summary = '', this.followUps = const [], this.status = 'pending'});

  factory TalkRecord.fromJson(Map<String, dynamic> json) {
    return TalkRecord(
      id: json['id'] ?? '',
      studentName: json['student_name'] ?? '',
      date: json['date'] ?? '',
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

  LessonPlan({this.topic = '', this.outline = '', this.keyPoints = const [], this.difficulties = const [], this.strategies = const [], this.interactions = const [], this.homework = const []});

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

  ClassHeatmapData({this.courseName = '', this.points = const [], this.weakTopFive = const [], this.totalStudents = 0, this.anomalyCount = 0});

  factory ClassHeatmapData.fromJson(Map<String, dynamic> json) {
    return ClassHeatmapData(
      courseName: json['course_name'] ?? '',
      points: (json['points'] as List?)?.map((e) => KnowledgePoint.fromJson(e)).toList() ?? [],
      weakTopFive: List<String>.from(json['weak_top_five'] ?? []),
      totalStudents: json['total_students'] ?? 0,
      anomalyCount: json['anomaly_count'] ?? 0,
    );
  }
}
