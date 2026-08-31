// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of wxx_models;

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

