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

  ChatRequest({required this.question, this.sessionId});

  Map<String, dynamic> toJson() => {
        'question': question,
        if (sessionId != null) 'session_id': sessionId,
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

  Message({
    required this.id,
    required this.role,
    required this.content,
    this.answerCard,
    this.createdAt = '',
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
