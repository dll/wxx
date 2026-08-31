// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of wxx_models;

class AnswerCard {
  final String conclusion;
  final List<String> steps;
  final List<ProcessStepDetail> stepDetails; // 富文本步骤详情（联系方式/地点/FAQ等）
  final List<Source> sources;
  final List<String> risks;
  final List<String> followUps;
  final List<CardAction> actions;
  final String traceId;
  final String model; // 回答所用大模型名（如 deepseek-v4-flash）
  final double confidence;
  final bool fallback;
  final List<String> agents; // 参与本次回答的智能体名称列表（D4-3，不足时为空）

  AnswerCard({
    required this.conclusion,
    this.steps = const [],
    this.stepDetails = const [],
    this.sources = const [],
    this.risks = const [],
    this.followUps = const [],
    this.actions = const [],
    this.traceId = '',
    this.model = '',
    this.confidence = 0,
    this.fallback = false,
    this.agents = const [],
  });

  factory AnswerCard.fromJson(Map<String, dynamic> json) {
    return AnswerCard(
      conclusion: json['conclusion'] ?? '',
      steps: List<String>.from(json['steps'] ?? []),
      stepDetails: (json['step_details'] as List?)
              ?.map((s) => ProcessStepDetail.fromJson(s))
              .toList() ??
          [],
      sources:
          (json['sources'] as List?)?.map((s) => Source.fromJson(s)).toList() ??
              [],
      risks: List<String>.from(json['risks'] ?? []),
      followUps: List<String>.from(json['follow_ups'] ?? []),
      actions: (json['actions'] as List?)
              ?.map((a) => CardAction.fromJson(a))
              .toList() ??
          [],
      traceId: json['trace_id'] ?? '',
      model: json['model'] ?? '',
      confidence: (json['confidence'] ?? 0).toDouble(),
      fallback: json['fallback'] ?? false,
      agents: List<String>.from(json['agents'] ?? []),
    );
  }
}

/// 流程步骤详细信息（含联系人/地点/FAQ 等 6 类信息）

class ProcessStepDetail {
  final int step;
  final String title;
  final String status;
  final String contact;
  final String phone;
  final String contactWechat;
  final String location;
  final String officeHours;
  final String materials;
  final String entryUrl;
  final String deadline;
  final String notes;
  final List<String> mediaUrls;
  final List<ProcessFAQ> faq;

  ProcessStepDetail({
    required this.step,
    required this.title,
    this.status = 'pending',
    this.contact = '',
    this.phone = '',
    this.contactWechat = '',
    this.location = '',
    this.officeHours = '',
    this.materials = '',
    this.entryUrl = '',
    this.deadline = '',
    this.notes = '',
    this.mediaUrls = const [],
    this.faq = const [],
  });

  factory ProcessStepDetail.fromJson(Map<String, dynamic> json) {
    return ProcessStepDetail(
      step: json['step'] ?? 0,
      title: json['title'] ?? '',
      status: json['status'] ?? 'pending',
      contact: json['contact'] ?? '',
      phone: json['phone'] ?? '',
      contactWechat: json['contact_wechat'] ?? '',
      location: json['location'] ?? '',
      officeHours: json['office_hours'] ?? '',
      materials: json['materials'] ?? '',
      entryUrl: json['entry_url'] ?? '',
      deadline: json['deadline'] ?? '',
      notes: json['notes'] ?? '',
      mediaUrls: (json['media_urls'] as List?)?.cast<String>() ?? const [],
      faq:
          (json['faq'] as List?)?.map((f) => ProcessFAQ.fromJson(f)).toList() ??
              [],
    );
  }
}

/// 流程步骤的常见问题

class ProcessFAQ {
  final String q;
  final String a;

  ProcessFAQ({required this.q, required this.a});

  factory ProcessFAQ.fromJson(Map<String, dynamic> json) {
    return ProcessFAQ(
      q: json['q'] ?? '',
      a: json['a'] ?? '',
    );
  }
}

/// 流程步骤（管理端完整字段，对应后端 process_steps）

class ProcessStep {
  final int id;
  final String resourceId;
  final int stepOrder;
  final String title;
  final String materials; // JSON 数组字符串
  final String entryUrl;
  final String deadline;
  final String location;
  final String notes;
  final String contact;
  final String phone;
  final String contactWechat;
  final String officeHours;
  final double geoLat;
  final double geoLng;
  final String mediaUrls; // JSON 数组字符串
  final String faq; // JSON 数组字符串

  ProcessStep({
    this.id = 0,
    this.resourceId = '',
    this.stepOrder = 0,
    this.title = '',
    this.materials = '[]',
    this.entryUrl = '',
    this.deadline = '',
    this.location = '',
    this.notes = '',
    this.contact = '',
    this.phone = '',
    this.contactWechat = '',
    this.officeHours = '',
    this.geoLat = 0,
    this.geoLng = 0,
    this.mediaUrls = '[]',
    this.faq = '[]',
  });

  factory ProcessStep.fromJson(Map<String, dynamic> json) {
    return ProcessStep(
      id: json['id'] is int ? json['id'] : int.tryParse('${json['id']}') ?? 0,
      resourceId: json['resource_id'] ?? '',
      stepOrder: json['step_order'] ?? 0,
      title: json['title'] ?? '',
      materials: json['materials'] ?? '[]',
      entryUrl: json['entry_url'] ?? '',
      deadline: json['deadline'] ?? '',
      location: json['location'] ?? '',
      notes: json['notes'] ?? '',
      contact: json['contact'] ?? '',
      phone: json['phone'] ?? '',
      contactWechat: json['contact_wechat'] ?? '',
      officeHours: json['office_hours'] ?? '',
      geoLat: (json['geo_lat'] ?? 0).toDouble(),
      geoLng: (json['geo_lng'] ?? 0).toDouble(),
      mediaUrls: json['media_urls'] ?? '[]',
      faq: json['faq'] ?? '[]',
    );
  }

  List<String> get materialsList => _decodeStringList(materials);
  List<String> get mediaList => _decodeStringList(mediaUrls);

  List<ProcessFAQ> get faqList {
    final decoded = _decodeList(faq);
    return decoded
        .map((e) => ProcessFAQ.fromJson(Map<String, dynamic>.from(e)))
        .toList();
  }

  ProcessStepDetail toDetail() {
    return ProcessStepDetail(
      step: stepOrder,
      title: title,
      contact: contact,
      phone: phone,
      contactWechat: contactWechat,
      location: location,
      officeHours: officeHours,
      materials: materials,
      entryUrl: entryUrl,
      deadline: deadline,
      notes: notes,
      mediaUrls: mediaList,
      faq: faqList,
    );
  }
}

/// 流程提醒（对应后端 process_reminders）

class ProcessReminder {
  final int id;
  final String processId;
  final int stepOrder;
  final String remindAt;
  final String title;
  final String content;
  final bool isEnabled;
  final String createdAt;
  final String updatedAt;

  ProcessReminder({
    this.id = 0,
    this.processId = '',
    this.stepOrder = 0,
    this.remindAt = '',
    this.title = '',
    this.content = '',
    this.isEnabled = true,
    this.createdAt = '',
    this.updatedAt = '',
  });

  factory ProcessReminder.fromJson(Map<String, dynamic> json) {
    return ProcessReminder(
      id: json['id'] is int ? json['id'] : int.tryParse('${json['id']}') ?? 0,
      processId: json['process_id'] ?? '',
      stepOrder: json['step_order'] ?? 0,
      remindAt: json['remind_at'] ?? '',
      title: json['title'] ?? '',
      content: json['content'] ?? '',
      isEnabled: (json['is_enabled'] ?? 1) == 1,
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }
}

/// 办事流程完整定义（KB 资源 + 步骤 + 提醒）

class ProcessDefinition {
  final String resourceId;
  final String resourceType;
  final String ownerScope;
  final String ownerId;
  final String roleScope;
  final String version;
  final String status;
  final String title;
  final String summary;
  final String content;
  final String sourceLink;
  final String sourceVersion;
  final String effectiveAt;
  final String expiredAt;
  final List<String> tags;
  final String remark;
  final String updatedBy;
  final String createdAt;
  final String updatedAt;
  final List<ProcessStep> steps;
  final List<ProcessReminder> reminders;

  ProcessDefinition({
    required this.resourceId,
    required this.resourceType,
    required this.ownerScope,
    this.ownerId = '',
    this.roleScope = '',
    this.version = '',
    this.status = '',
    required this.title,
    this.summary = '',
    this.content = '',
    this.sourceLink = '',
    this.sourceVersion = '',
    this.effectiveAt = '',
    this.expiredAt = '',
    this.tags = const [],
    this.remark = '',
    this.updatedBy = '',
    this.createdAt = '',
    this.updatedAt = '',
    this.steps = const [],
    this.reminders = const [],
  });

  factory ProcessDefinition.fromJson(Map<String, dynamic> json) {
    return ProcessDefinition(
      resourceId: json['resource_id'] ?? '',
      resourceType: json['resource_type'] ?? 'Process',
      ownerScope: json['owner_scope'] ?? '',
      ownerId: json['owner_id'] ?? '',
      roleScope: json['role_scope'] ?? '',
      version: json['version'] ?? '',
      status: json['status'] ?? 'draft',
      title: json['title'] ?? '',
      summary: json['summary'] ?? '',
      content: json['content'] ?? '',
      sourceLink: json['source_link'] ?? '',
      sourceVersion: json['source_version'] ?? '',
      effectiveAt: json['effective_at'] ?? '',
      expiredAt: json['expired_at'] ?? '',
      tags: KnowledgeCard._parseTags(json['tags'] ?? ''),
      remark: json['remark'] ?? '',
      updatedBy: json['updated_by'] ?? '',
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
      steps: (json['steps'] as List?)
              ?.map((e) => ProcessStep.fromJson(Map<String, dynamic>.from(e)))
              .toList() ??
          const [],
      reminders: (json['reminders'] as List?)
              ?.map(
                  (e) => ProcessReminder.fromJson(Map<String, dynamic>.from(e)))
              .toList() ??
          const [],
    );
  }

  bool get isFreshmenRelated =>
      tags.any((t) => t.contains('新生') || t.contains('入学') || t.contains('报到'));

  /// 流程适用角色（role_scope JSON 解码），空表示不限制
  List<String> get roleCodes {
    if (roleScope.isEmpty || roleScope == '[]') return const [];
    try {
      final decoded = jsonDecode(roleScope);
      if (decoded is List) return decoded.map((e) => e.toString()).toList();
    } catch (_) {}
    return const [];
  }

  /// 面向群体（年级/身份维度），依据标题、摘要与 tags 判定
  String get audienceLabel {
    final s = '$title $summary ${tags.join(' ')}';
    if (s.contains('新生') || s.contains('入学') || s.contains('报到')) {
      return '新生';
    }
    if (s.contains('毕业') || s.contains('离校')) return '毕业生';
    if (s.contains('请假') ||
        s.contains('奖学金') ||
        s.contains('转专业') ||
        s.contains('助学贷款') ||
        s.contains('在校')) {
      return '在校生';
    }
    return '通用';
  }

  String get statusLabel {
    switch (status) {
      case 'published':
        return '已发布';
      case 'pending':
        return '待审核';
      case 'retired':
        return '已下架';
      default:
        return '草稿';
    }
  }
}

List<dynamic> _decodeList(String raw) {
  if (raw.isEmpty || raw == '[]') return [];
  try {
    final decoded = jsonDecode(raw);
    return decoded is List ? decoded : [];
  } catch (_) {
    return [];
  }
}

List<String> _decodeStringList(String raw) {
  return _decodeList(raw).map((e) => e.toString()).toList();
}

/// 来源引用

class Source {
  final String resourceId;
  final String title;
  final String version;
  final String sourceLink;
  final double relevanceScore;
  final String resourceType;
  final String snippet;
  final String effectiveDate;
  final String summary;

  Source({
    required this.resourceId,
    required this.title,
    this.version = '',
    this.sourceLink = '',
    this.relevanceScore = 0,
    this.resourceType = '',
    this.snippet = '',
    this.effectiveDate = '',
    this.summary = '',
  });

  factory Source.fromJson(Map<String, dynamic> json) {
    return Source(
      resourceId: json['resource_id'] ?? '',
      title: json['title'] ?? '',
      version: json['version'] ?? '',
      sourceLink: json['source_link'] ?? '',
      relevanceScore: (json['relevance_score'] ?? 0).toDouble(),
      resourceType: json['resource_type'] ?? json['type'] ?? '',
      snippet: json['snippet'] ?? json['content_preview'] ?? '',
      effectiveDate: json['effective_date'] ?? json['date'] ?? '',
      summary: json['summary'] ?? json['description'] ?? '',
    );
  }

  /// 资源类型中文标签
  String get typeLabel {
    const map = {
      'policy': '政策',
      'Policy': '政策',
      'process': '流程',
      'Process': '流程',
      'faq': '问答',
      'FAQ': '问答',
      'activity': '活动',
      'Activity': '活动',
    };
    return map[resourceType] ?? '资料';
  }

  /// 资源类型图标
  IconData get typeIcon {
    switch (resourceType.toLowerCase()) {
      case 'policy':
        return Icons.description;
      case 'process':
        return Icons.alt_route;
      case 'faq':
        return Icons.quiz;
      case 'activity':
        return Icons.event;
      default:
        return Icons.insert_drive_file;
    }
  }

  /// 资源类型颜色
  Color get typeColor {
    switch (resourceType.toLowerCase()) {
      case 'policy':
        return const Color(0xFFE53935);
      case 'process':
        return const Color(0xFF1E88E5);
      case 'faq':
        return const Color(0xFF43A047);
      case 'activity':
        return const Color(0xFFFB8C00);
      default:
        return const Color(0xFF757575);
    }
  }

  /// 相关度星级（0-5星）
  int get relevanceStars {
    if (relevanceScore >= 0.9) return 5;
    if (relevanceScore >= 0.75) return 4;
    if (relevanceScore >= 0.6) return 3;
    if (relevanceScore >= 0.4) return 2;
    if (relevanceScore >= 0.2) return 1;
    return 0;
  }
}

/// 可执行动作

class CardAction {
  final String label;
  final String url;

  CardAction({required this.label, required this.url});

  factory CardAction.fromJson(Map<String, dynamic> json) {
    return CardAction(
      label: json['label'] ?? '',
      url: json['url'] ?? '',
    );
  }
}

/// 对话请求

