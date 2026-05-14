/// 角色权限判断工具 — 单一来源，避免各页面重复定义
class RoleUtils {
  RoleUtils._();

  static const _emotionRoles = {'sys_admin', 'school_admin', 'college_admin', 'counselor'};
  static const _adminRoles = {'sys_admin', 'school_admin', 'college_admin'};
  static const _agentRoles = {'sys_admin', 'school_admin'};
  static const _kbSubmitRoles = {'sys_admin', 'school_admin', 'college_admin', 'counselor', 'student_union'};

  /// 情感预警（counselor 及以上）
  static bool canAccessEmotion(String? role) => role != null && _emotionRoles.contains(role);

  /// 管理端（college_admin 及以上）
  static bool canAccessAdmin(String? role) => role != null && _adminRoles.contains(role);

  /// 智能体管理（school_admin 及以上）
  static bool canAccessAgents(String? role) => role != null && _agentRoles.contains(role);

  /// 知识提交/反馈管理（student_union 及以上）
  static bool canSubmitKB(String? role) => role != null && _kbSubmitRoles.contains(role);
}
