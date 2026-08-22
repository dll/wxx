import 'capability_utils.dart';
import 'storage.dart';

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
    return loggedIn &&
        role != 'guest' &&
        status == 'active' &&
        ownerScope == 'college' &&
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
