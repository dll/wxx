import 'capability_utils.dart';

/// 角色权限判断工具 — 已升级为 capability 委托
///
/// 此类签名保留以兼容已有 30+ 调用点；新代码请直接使用 CapabilityUtils。
/// 内部全部委托给 CapabilityUtils.has(...)，依赖后端登录后下发的能力清单。
class RoleUtils {
  RoleUtils._();

  /// 情感预警（counselor 及以上由能力继承自动获得）
  static bool canAccessEmotion(String? role) =>
      CapabilityUtils.has(Capability.counselorAlertRead);

  /// 管理端（college_admin 及以上）
  static bool canAccessAdmin(String? role) =>
      CapabilityUtils.has(Capability.collegeMetricsRead);

  /// 智能体管理（school_admin 及以上）
  static bool canAccessAgents(String? role) =>
      CapabilityUtils.has(Capability.schoolAgentWrite);

  /// 知识提交/反馈管理（student_union 及以上）
  static bool canSubmitKB(String? role) =>
      CapabilityUtils.has(Capability.unionKbSubmit);
}
