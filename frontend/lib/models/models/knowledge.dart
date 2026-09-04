// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of '../models.dart';

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



