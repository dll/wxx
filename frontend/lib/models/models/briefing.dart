// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of '../models.dart';

class AIBriefing {
  final int id;
  final String source;
  final String category;
  final String topic;
  final String summary;
  final String content;
  final String link;
  final String keyword;
  final int heat;
  final String reason;
  final String publishedAt;
  final String fetchedAt;
  final int status;
  final bool favorited;

  const AIBriefing({
    this.id = 0,
    this.source = '',
    this.category = 'ai_teaching',
    this.topic = '',
    this.summary = '',
    this.content = '',
    this.link = '',
    this.keyword = '',
    this.heat = 0,
    this.reason = '',
    this.publishedAt = '',
    this.fetchedAt = '',
    this.status = 1,
    this.favorited = false,
  });

  factory AIBriefing.fromJson(Map<String, dynamic> json) {
    return AIBriefing(
      id: json['id'] ?? 0,
      source: json['source'] ?? '',
      category: json['category'] ?? 'ai_teaching',
      topic: json['topic'] ?? '',
      summary: json['summary'] ?? '',
      content: json['content'] ?? '',
      link: json['link'] ?? '',
      keyword: json['keyword'] ?? '',
      heat: json['heat'] ?? 0,
      reason: json['reason'] ?? '',
      publishedAt: json['published_at'] ?? '',
      fetchedAt: json['fetched_at'] ?? '',
      status: json['status'] ?? 1,
      favorited: json['favorited'] == true,
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'source': source,
        'category': category,
        'topic': topic,
        'summary': summary,
        'content': content,
        'link': link,
        'keyword': keyword,
        'heat': heat,
        'reason': reason,
        'published_at': publishedAt,
        'status': status,
      };

  AIBriefing copyWith({bool? favorited}) {
    return AIBriefing(
      id: id,
      source: source,
      category: category,
      topic: topic,
      summary: summary,
      content: content,
      link: link,
      keyword: keyword,
      heat: heat,
      reason: reason,
      publishedAt: publishedAt,
      fetchedAt: fetchedAt,
      status: status,
      favorited: favorited ?? this.favorited,
    );
  }
}

/// AI 简讯来源配置

class AIBriefingSource {
  final int id;
  final String name;
  final String url;
  final String category;
  final int enabled;
  final int fetchEnabled;
  final String fetchTime;
  final String lastFetchAt;

  const AIBriefingSource({
    this.id = 0,
    this.name = '',
    this.url = '',
    this.category = 'ai_teaching',
    this.enabled = 1,
    this.fetchEnabled = 1,
    this.fetchTime = '08:00',
    this.lastFetchAt = '',
  });

  factory AIBriefingSource.fromJson(Map<String, dynamic> json) {
    return AIBriefingSource(
      id: json['id'] ?? 0,
      name: json['name'] ?? '',
      url: json['url'] ?? '',
      category: json['category'] ?? 'ai_teaching',
      enabled: json['enabled'] ?? 1,
      fetchEnabled: json['fetch_enabled'] ?? 1,
      fetchTime: json['fetch_time'] ?? '08:00',
      lastFetchAt: json['last_fetch_at'] ?? '',
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'url': url,
        'category': category,
        'enabled': enabled,
        'fetch_enabled': fetchEnabled,
        'fetch_time': fetchTime,
      };
}

/// AI 简讯分类展示名

class AIBriefingCategory {
  final String key;
  final String label;
  const AIBriefingCategory(this.key, this.label);

  static const all = [
    AIBriefingCategory('ai_teaching', 'AI 辅助教学'),
    AIBriefingCategory('ai_tool', 'AI 工具'),
    AIBriefingCategory('ai_version', 'AI 版本'),
    AIBriefingCategory('ai_industry', 'AI 行业热点'),
  ];

  static String labelOf(String key) {
    for (final c in all) {
      if (c.key == key) return c.label;
    }
    return key;
  }
}

// ═══════════════════════════════════════════════════════════════
// 数字孪生画像
// ═══════════════════════════════════════════════════════════════

/// 数字孪生画像（CogView 生成，蔚小芯风格）



