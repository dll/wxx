// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of '../models.dart';

class DigitalTwinData {
  final List<TwinDimension> dimensions;
  final List<TwinDimension> idealDimensions;
  final String aiSummary;
  final List<String> suggestions;
  final String growthStage;
  final String profileTag;
  final double dataCoverage;

  DigitalTwinData(
      {this.dimensions = const [],
      this.idealDimensions = const [],
      this.aiSummary = '',
      this.suggestions = const [],
      this.growthStage = '',
      this.profileTag = '',
      this.dataCoverage = 0});

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
      growthStage: json['growth_stage'] ?? '',
      profileTag: json['profile_tag'] ?? '',
      dataCoverage: (json['data_coverage'] ?? 0).toDouble(),
    );
  }
}

class TwinDimension {
  final String name;
  final double score;
  final String label;
  final bool dataAvailable;
  final List<String> evidence;

  TwinDimension(
      {this.name = '',
      this.score = 0,
      this.label = '',
      this.dataAvailable = true,
      this.evidence = const []});

  factory TwinDimension.fromJson(Map<String, dynamic> json) {
    return TwinDimension(
      name: json['name'] ?? '',
      score: (json['score'] ?? 0).toDouble(),
      label: json['label'] ?? json['level'] ?? '',
      // 后端 v1 无 data_available 字段时默认 true，兼容旧接口/兜底 mock
      dataAvailable: json['data_available'] ?? true,
      evidence: List<String>.from(json['evidence'] ?? const []),
    );
  }
}

/// 打卡记录

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



