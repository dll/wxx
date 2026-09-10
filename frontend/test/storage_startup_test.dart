import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:wxx_app/utils/storage.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const secureStorageChannel =
      MethodChannel('plugins.it_nomads.com/flutter_secure_storage');

  setUp(() {
    SharedPreferences.setMockInitialValues(const {});
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(secureStorageChannel, null);
  });

  test('安全存储插件未注册时应用存储仍可初始化', () async {
    await Storage.init();

    expect(Storage.isLoggedIn, isFalse);
  });

  test('安全存储插件未注册时登录会话保留在内存中', () async {
    await Storage.init();

    await Storage.setToken('test-token');
    expect(Storage.token, 'test-token');

    await Storage.clearToken();
    expect(Storage.token, isNull);
  });
}
