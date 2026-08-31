// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of wxx_models;

class ContactPerson {
  final int id;
  final String name;
  final String role;
  final String roleName;
  final String phone;
  final String wechat;
  final String email;

  const ContactPerson({
    this.id = 0,
    this.name = '',
    this.role = '',
    this.roleName = '',
    this.phone = '',
    this.wechat = '',
    this.email = '',
  });

  factory ContactPerson.fromJson(Map<String, dynamic> json) {
    return ContactPerson(
      id: json['id'] ?? 0,
      name: json['name'] ?? '',
      role: json['role'] ?? '',
      roleName: json['role_name'] ?? '',
      phone: json['phone'] ?? '',
      wechat: json['wechat'] ?? '',
      email: json['email'] ?? '',
    );
  }
}

/// 个人详细信息

class PersonalDetail {
  // 基本信息
  final int userId;
  final String username;
  final String displayName;
  final String role;
  final String college;
  final String major;
  final String className;
  final String enrollmentDate;
  final String enrollmentYear;
  // 联系方式
  final String phone;
  final String wechat;
  final String qq;
  final String email;
  // 头像
  final String avatarBase64;
  final String avatarMime;
  // 组织关系
  final List<ContactPerson> supervisors;
  final int subordinates;

  const PersonalDetail({
    this.userId = 0,
    this.username = '',
    this.displayName = '',
    this.role = '',
    this.college = '',
    this.major = '',
    this.className = '',
    this.enrollmentDate = '',
    this.enrollmentYear = '',
    this.phone = '',
    this.wechat = '',
    this.qq = '',
    this.email = '',
    this.avatarBase64 = '',
    this.avatarMime = 'image/png',
    this.supervisors = const [],
    this.subordinates = 0,
  });

  factory PersonalDetail.fromJson(Map<String, dynamic> json) {
    return PersonalDetail(
      userId: json['user_id'] ?? 0,
      username: json['username'] ?? '',
      displayName: json['display_name'] ?? '',
      role: json['role'] ?? '',
      college: json['college'] ?? '',
      major: json['major'] ?? '',
      className: json['class_name'] ?? '',
      enrollmentDate: json['enrollment_date'] ?? '',
      enrollmentYear: json['enrollment_year'] ?? '',
      phone: json['phone'] ?? '',
      wechat: json['wechat'] ?? '',
      qq: json['qq'] ?? '',
      email: json['email'] ?? '',
      avatarBase64: json['avatar_base64'] ?? '',
      avatarMime: json['avatar_mime'] ?? 'image/png',
      supervisors: (json['supervisors'] as List? ?? [])
          .whereType<Map>()
          .map((e) => ContactPerson.fromJson(Map<String, dynamic>.from(e)))
          .toList(),
      subordinates: json['subordinates'] ?? 0,
    );
  }
}

/// 学校门户凭证绑定状态（不含密码明文）

class PortalCredential {
  final int userId;
  final String portalUrl;
  final String portalAccount;
  final bool bound;
  final String updatedAt;

  const PortalCredential({
    this.userId = 0,
    this.portalUrl = ApiConfig.schoolPortalUrl,
    this.portalAccount = '',
    this.bound = false,
    this.updatedAt = '',
  });

  factory PortalCredential.fromJson(Map<String, dynamic> json) {
    return PortalCredential(
      userId: json['user_id'] ?? 0,
      portalUrl: json['portal_url'] ?? ApiConfig.schoolPortalUrl,
      portalAccount: json['portal_account'] ?? '',
      bound: json['bound'] ?? false,
      updatedAt: json['updated_at'] ?? '',
    );
  }
}

// ═══════════════════════════════════════════════════════════════
// 审计恢复快照
// ═══════════════════════════════════════════════════════════════

/// 审计恢复快照（可恢复的写操作前后状态）

class AuditSnapshot {
  final int id;
  final int auditId;
  final String opTable;
  final String recordId;
  final String operation;
  final String beforeJson;
  final String afterJson;
  final int restored;
  final String restoredAt;
  final String restoredBy;
  final String createdAt;

  const AuditSnapshot({
    this.id = 0,
    this.auditId = 0,
    this.opTable = '',
    this.recordId = '',
    this.operation = '',
    this.beforeJson = '',
    this.afterJson = '',
    this.restored = 0,
    this.restoredAt = '',
    this.restoredBy = '',
    this.createdAt = '',
  });

  factory AuditSnapshot.fromJson(Map<String, dynamic> json) {
    return AuditSnapshot(
      id: json['id'] ?? 0,
      auditId: json['audit_id'] ?? 0,
      opTable: json['op_table'] ?? '',
      recordId: json['record_id'] ?? '',
      operation: json['operation'] ?? '',
      beforeJson: json['before_json'] ?? '',
      afterJson: json['after_json'] ?? '',
      restored: json['restored'] ?? 0,
      restoredAt: json['restored_at'] ?? '',
      restoredBy: json['restored_by'] ?? '',
      createdAt: json['created_at'] ?? '',
    );
  }
}

