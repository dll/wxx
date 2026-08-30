import 'package:flutter_test/flutter_test.dart';

import 'package:wxx_app/config/api_config.dart';
import 'package:wxx_app/models/models.dart';

void main() {
  group('AnswerCard/Source 反序列化', () {
    test('snippet 优先命中片段，缺失回退空串', () {
      final card = AnswerCard.fromJson({
        'conclusion': '结论',
        'sources': [
          {'title': 'A', 'snippet': '命中段落'},
          {'title': 'B', 'summary': '摘要'},
        ],
      });
      expect(card.sources[0].snippet, '命中段落');
      expect(card.sources[1].snippet, '');
      expect(card.sources[1].summary, '摘要');
    });

    test('fallback 缺省为 false，confidence 缺省合理', () {
      final card = AnswerCard.fromJson({'conclusion': 'x'});
      expect(card.fallback, isFalse);
      expect(card.confidence, greaterThanOrEqualTo(0));
      expect(card.sources, isEmpty);
    });

    test('followUps 空数组安全', () {
      final card = AnswerCard.fromJson({'conclusion': 'x', 'followUps': null});
      expect(card.followUps, isEmpty);
    });
  });

  group('PortalCredential 门户默认值', () {
    test('默认 portalUrl 走 ApiConfig 常量（域名收敛单点）', () {
      const cred = PortalCredential();
      expect(cred.portalUrl, ApiConfig.schoolPortalUrl);
      expect(cred.portalUrl, startsWith('https://'));
    });

    test('fromJson 缺 portal_url 回退常量', () {
      final cred = PortalCredential.fromJson({'user_id': 7});
      expect(cred.portalUrl, ApiConfig.schoolPortalUrl);
      expect(cred.userId, 7);
    });

    test('fromJson 保留服务端值', () {
      final cred = PortalCredential.fromJson({'portal_url': 'https://portal.example.edu/'});
      expect(cred.portalUrl, 'https://portal.example.edu/');
    });
  });
}
