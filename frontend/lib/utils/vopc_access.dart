import 'capability_utils.dart';
import 'storage.dart';

/// vOPC v2.0「现实延伸（L4）」外部站点跳转地址。
///
/// 虚拟OPC 网站即将开发上线；当前为占位域名，后续替换为真实域名。
const String vopcSiteUrl =
    'https://qcnbzr4eeji5.feishu.cn/page/PQeimb5jwdPt6GaaIbIcymkYnjg'; // 虚拟OPC 官网（飞书 page；待 ccit-agent.cn 域名备案接入生效后切回）

/// vOPC 前端准入判定。后端 CollegeAccess 仍是最终安全边界；这里负责入口与路由门禁。
class VopcAccess {
  VopcAccess._();

  static bool evaluate({
    required bool loggedIn,
    required String? role,
    required String? status,
    required String? ownerScope,
    required String? ownerId,
    required Iterable<String> capabilities,
    String collegeId = 'cs',
  }) {
    if (!loggedIn || role == 'guest' || status != 'active') return false;

    // 系统管理员负责全局运维与审计，账号作用域通常为 school/system；并且
    // 登录后能力清单可能仍在异步刷新，不能因缓存暂缺而隐藏管理员主菜单。
    // 后端仍会依据 JWT 中的真实角色和账号状态执行最终授权。
    if (role == 'sys_admin') return true;
    return ownerScope == 'college' &&
        ownerId?.toLowerCase() == collegeId.toLowerCase() &&
        capabilities.contains(Capability.vopcRead);
  }

  static bool get allowed => evaluate(
        loggedIn: Storage.isLoggedIn,
        role: Storage.role,
        status: Storage.userStatus,
        ownerScope: Storage.ownerScope,
        ownerId: Storage.ownerId,
        capabilities: Storage.capabilities,
      );
}
