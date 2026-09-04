// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of '../models.dart';

class ImportResultData {
  final int total;
  final int success;
  final int failed;
  final List<ImportResultDetail> details;

  const ImportResultData({
    required this.total,
    required this.success,
    required this.failed,
    required this.details,
  });

  factory ImportResultData.fromJson(Map<String, dynamic> json) {
    return ImportResultData(
      total: json['total'] ?? 0,
      success: json['success'] ?? 0,
      failed: json['failed'] ?? 0,
      details: (json['details'] as List?)
              ?.map(
                  (e) => ImportResultDetail.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }
}

/// 导入结果明细

class ImportResultDetail {
  final String username;
  final String displayName;
  final bool success;
  final String error;

  const ImportResultDetail({
    required this.username,
    required this.displayName,
    required this.success,
    required this.error,
  });

  factory ImportResultDetail.fromJson(Map<String, dynamic> json) {
    return ImportResultDetail(
      username: json['username'] ?? '',
      displayName: json['display_name'] ?? '',
      success: json['success'] ?? false,
      error: json['error'] ?? '',
    );
  }
}

// ═══════════════════════════════════════════════════════════════
// 第三方应用（应用中心）
// ═══════════════════════════════════════════════════════════════

/// 第三方应用（GET /api/v1/apps 返回的可见应用）

class ExternalAppItem {
  final String id;
  final String name;
  final String icon;
  final String category; // study | culture | service | admin | external
  final String summary;
  final String version;
  final String type; // external_link | webview | reverse_proxy
  final String url;
  final String openIn; // _self | _blank | _native

  const ExternalAppItem({
    this.id = '',
    this.name = '',
    this.icon = '',
    this.category = 'external',
    this.summary = '',
    this.version = '',
    this.type = 'external_link',
    this.url = '',
    this.openIn = '_blank',
  });

  factory ExternalAppItem.fromJson(Map<String, dynamic> json) {
    return ExternalAppItem(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      icon: json['icon'] ?? '',
      category: json['category'] ?? 'external',
      summary: json['summary'] ?? '',
      version: json['version'] ?? '',
      type: json['type'] ?? 'external_link',
      url: json['url'] ?? '',
      openIn: json['open_in'] ?? '_blank',
    );
  }
}

// ═══════════════════════════════════════════════════════════════
// AI 简讯
// ═══════════════════════════════════════════════════════════════

/// AI 简讯（首页资讯）



