// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of '../models.dart';

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



