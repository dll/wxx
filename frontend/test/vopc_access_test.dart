import 'package:flutter_test/flutter_test.dart';
import 'package:wxx_app/utils/capability_utils.dart';
import 'package:wxx_app/utils/vopc_access.dart';

void main() {
  test('vOPC entry allows active sys admin and authorized college users', () {
    bool allowed({
      String role = 'student',
      String status = 'active',
      String scope = 'college',
      String owner = 'cs',
      List<String> caps = const [Capability.vopcRead],
    }) =>
        VopcAccess.evaluate(
          loggedIn: true,
          role: role,
          status: status,
          ownerScope: scope,
          ownerId: owner,
          capabilities: caps,
        );

    expect(allowed(owner: 'CS'), isTrue);
    expect(allowed(role: 'sys_admin', scope: 'system', owner: ''), isTrue);
    expect(allowed(role: 'guest'), isFalse);
    expect(allowed(status: 'inactive'), isFalse);
    expect(
        allowed(
            role: 'sys_admin', status: 'inactive', scope: 'system', owner: ''),
        isFalse);
    // 管理员菜单不依赖异步能力缓存，避免登录后能力接口尚未返回时被隐藏。
    expect(
        allowed(
            role: 'sys_admin', scope: 'school', owner: 'all', caps: const []),
        isTrue);
    expect(allowed(scope: 'school'), isFalse);
    expect(allowed(owner: 'math'), isFalse);
    expect(allowed(caps: const []), isFalse);
    expect(
        VopcAccess.evaluate(
          loggedIn: false,
          role: 'student',
          status: 'active',
          ownerScope: 'college',
          ownerId: 'cs',
          capabilities: const [Capability.vopcRead],
        ),
        isFalse);
  });
}
