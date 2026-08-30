import 'package:flutter_test/flutter_test.dart';

import 'package:wxx_app/config/release_config.dart';

void main() {
  group('ReleaseConfig.qrCodeUrl 二维码 URL 生成', () {
    test('包含原始内容并做 URI 编码', () {
      final url = ReleaseConfig.qrCodeUrl('https://wxx-agent.online/qr-login?qr=abc123');
      expect(url, startsWith('https://api.qrserver.com/v1/create-qr-code/'));
      expect(url, contains(Uri.encodeComponent('https://wxx-agent.online/qr-login?qr=abc123')));
      expect(url, isNot(contains('?qr='))); // 原文不应裸露（须编码）
    });

    test('默认尺寸 220，可自定义', () {
      expect(ReleaseConfig.qrCodeUrl('x'), contains('size=220x220'));
      expect(ReleaseConfig.qrCodeUrl('x', size: 240), contains('size=240x240'));
    });

    test('中文内容正确编码', () {
      final url = ReleaseConfig.qrCodeUrl('蔚小芯');
      expect(url, contains(Uri.encodeComponent('蔚小芯')));
    });
  });
}
