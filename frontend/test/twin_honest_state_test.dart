import 'package:flutter_test/flutter_test.dart';

import 'package:wxx_app/models/models.dart';

void main() {
  group('DigitalTwin 画像契约（诚实空态）', () {
    test('缺 data_available 的维度按不可用处理，不得把占位 0 当真实分数', () {
      final dim = TwinDimension.fromJson({
        'name': '学业',
        'score': 0,
        'level': '数据积累中',
      });
      expect(dim.dataAvailable, isFalse);
      expect(dim.label, '数据积累中');
    });

    test('显式 data_available=true 的维度保留真实分数与档位', () {
      final dim = TwinDimension.fromJson({
        'name': '学业',
        'score': 82.5,
        'level': '优秀',
        'data_available': true,
        'evidence': ['成绩记录 12 条'],
      });
      expect(dim.dataAvailable, isTrue);
      expect(dim.score, 82.5);
      expect(dim.label, '优秀');
      expect(dim.evidence, isNotEmpty);
    });

    test('全空画像：无可用维度时 coverage/overall 保持 0，fallback 透传', () {
      final twin = DigitalTwinData.fromJson({
        'overall_score': 0,
        'data_coverage': 0,
        'fallback': true,
        'dimensions': [
          {'name': '学业', 'score': 0, 'level': '数据积累中'},
        ],
      });
      expect(twin.overallScore, 0);
      expect(twin.dataCoverage, 0);
      expect(twin.fallback, isTrue);
      expect(twin.dimensions.every((d) => !d.dataAvailable), isTrue);
    });

    test('interpretation/stage_advice 字段对齐（v1 兼容）', () {
      final twin = DigitalTwinData.fromJson({
        'interpretation': '你的优势维度是学业',
        'stage_advice': ['保持节奏'],
      });
      expect(twin.aiSummary, '你的优势维度是学业');
      expect(twin.suggestions, ['保持节奏']);
    });
  });
}
