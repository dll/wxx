// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of '../models.dart';

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



