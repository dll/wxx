// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of wxx_models;

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

