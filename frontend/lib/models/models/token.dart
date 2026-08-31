// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of wxx_models;

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

