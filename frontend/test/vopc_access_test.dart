import 'package:flutter_test/flutter_test.dart';
import 'package:wxx_app/utils/capability_utils.dart';
import 'package:wxx_app/utils/vopc_access.dart';

void main() {
  test('vOPC entry requires active authorized computer-college user', () {
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
    expect(allowed(role: 'guest'), isFalse);
    expect(allowed(status: 'inactive'), isFalse);
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
