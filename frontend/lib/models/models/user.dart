// 由 models.dart 拆分（P4-c）；part 文件不可被直接 import，请使用 models.dart barrel
part of wxx_models;

class UserProfile {
  final int id;
  final String username;
  final String role;
  final List<String> roles;
  final String displayName;
  final String college;
  final String major;
  final String className;
  final String enrollmentDate;
  final String enrollmentYear;
  final String ownerScope;
  final String ownerId;
  final String status;
  final String gender;
  final String campus;
  final String educationLevel;
  final String studyDuration;
  final String expectedGrad;
  final String studyMode;
  final String ethnicity;
  final String politicalStatus;
  final String position;

  UserProfile({
    required this.id,
    required this.username,
    required this.role,
    this.roles = const [],
    required this.displayName,
    this.college = '',
    this.major = '',
    this.className = '',
    this.enrollmentDate = '',
    this.enrollmentYear = '',
    this.ownerScope = '',
    this.ownerId = '',
    this.status = 'active',
    this.gender = '',
    this.campus = '',
    this.educationLevel = '',
    this.studyDuration = '',
    this.expectedGrad = '',
    this.studyMode = '',
    this.ethnicity = '',
    this.politicalStatus = '',
    this.position = '',
  });

  factory UserProfile.fromJson(Map<String, dynamic> json) {
    // 后端可能将 data 包裹在 data 字段中
    final data = json['data'] ?? json;
    return UserProfile(
      id: data['id'] ?? 0,
      username: data['username'] ?? '',
      role: data['role'] ?? 'student',
      roles: (data['roles'] as List?)?.cast<String>() ?? const [],
      displayName: data['display_name'] ?? data['username'] ?? '',
      college: data['college'] ?? data['owner_id'] ?? '',
      major: data['major'] ?? '',
      className: data['class_name'] ?? '',
      enrollmentDate: data['enrollment_date'] ?? '',
      enrollmentYear: data['enrollment_year'] ?? '',
      ownerScope: data['owner_scope'] ?? '',
      ownerId: data['owner_id'] ?? '',
      status: data['status'] ?? 'active',
      gender: data['gender'] ?? '',
      campus: data['campus'] ?? '',
      educationLevel: data['education_level'] ?? '',
      studyDuration: data['study_duration'] ?? '',
      expectedGrad: data['expected_graduation_date'] ?? '',
      studyMode: data['study_mode'] ?? '',
      ethnicity: data['ethnicity'] ?? '',
      politicalStatus: data['political_status'] ?? '',
      position: data['position'] ?? '',
    );
  }

  /// 角色中文名
  String get roleLabel {
    const map = {
      'sys_admin': '系统管理员',
      'school_admin': '学校管理员',
      'college_admin': '学院管理员',
      'counselor': '辅导员',
      'student_union': '学生会',
      'student': '学生',
      'teacher': '教师',
      'assistant': '教辅',
    };
    return map[role] ?? role;
  }
}

/// 统一错误响应

class ApiError {
  final int code;
  final String message;
  final String traceId;

  ApiError({required this.code, required this.message, this.traceId = ''});

  factory ApiError.fromJson(Map<String, dynamic> json) {
    return ApiError(
      code: json['code'] ?? -1,
      message: json['message'] ?? '未知错误',
      traceId: json['trace_id'] ?? '',
    );
  }
}

/// 知识大厅卡片（轻量数据，不含正文）

