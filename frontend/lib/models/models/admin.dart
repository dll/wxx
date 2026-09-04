// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of '../models.dart';

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
    final byRole = json['by_role'];
    if (byRole is Map) {
      byRole.forEach((key, value) {
        roleMap['$key'] = (value is num) ? value.toInt() : 0;
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
    final byType = json['by_type'];
    if (byType is Map) {
      byType.forEach((key, value) {
        typeMap['$key'] = (value is num) ? value.toInt() : 0;
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



