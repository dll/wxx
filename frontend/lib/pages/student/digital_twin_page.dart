import 'dart:math';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';

class DigitalTwinPage extends StatefulWidget {
  const DigitalTwinPage({super.key});
  @override
  State<DigitalTwinPage> createState() => _DigitalTwinPageState();
}

class _DigitalTwinPageState extends State<DigitalTwinPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchDigitalTwin();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('数字孪生画像')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchDigitalTwin(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty
                ? ErrorView.error(
                    message: provider.error,
                    onRetry: () => provider.fetchDigitalTwin())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final t = provider.twin;
    if (t == null) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        if (t.dimensions.isNotEmpty) ...[
          Text('能力维度', style: theme.textTheme.titleMedium),
          const SizedBox(height: 12),
          // 五维雷达图
          SizedBox(
            height: 240,
            child: _RadarChart(
              dimensions: t.dimensions,
              idealDimensions: t.idealDimensions,
              color: theme.colorScheme.primary,
              secondaryColor: theme.colorScheme.tertiary,
            ),
          ),
          const SizedBox(height: 16),
          // 各维度详情
          ...t.dimensions.map((d) {
            final normalized = d.score > 1 ? d.score / 100.0 : d.score;
            return Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child:
                  Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                  Text(d.name),
                  Text(
                    d.label.isNotEmpty
                        ? d.label
                        : '${(normalized * 100).toInt()}%',
                    style: theme.textTheme.bodySmall,
                  ),
                ]),
                const SizedBox(height: 4),
                LinearProgressIndicator(
                  value: normalized.clamp(0.0, 1.0),
                  backgroundColor:
                      theme.colorScheme.surfaceContainerHighest,
                  color: normalized >= 0.8
                      ? Colors.green
                      : normalized >= 0.5
                          ? Colors.orange
                          : Colors.red,
                ),
              ]),
            );
          }),
        ],
        if (t.aiSummary.isNotEmpty) ...[
          const SizedBox(height: 16),
          Card(
            color: theme.colorScheme.primaryContainer,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(children: [
                      Icon(Icons.psychology,
                          color: theme.colorScheme.onPrimaryContainer),
                      const SizedBox(width: 8),
                      Text('AI 分析', style: theme.textTheme.titleSmall),
                    ]),
                    const SizedBox(height: 8),
                    Text(t.aiSummary),
                  ]),
            ),
          ),
        ],
        if (t.suggestions.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('成长建议', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...t.suggestions.asMap().entries.map(
                (e) => Card(
                  child: ListTile(
                    leading: CircleAvatar(
                      backgroundColor:
                          theme.colorScheme.secondaryContainer,
                      child: Text('${e.key + 1}'),
                    ),
                    title: Text(e.value),
                  ),
                ),
              ),
        ],
      ],
    );
  }
}

/// 五维雷达图绘制组件
class _RadarChart extends StatelessWidget {
  final List<dynamic> dimensions; // TwinDimension 列表
  final List<dynamic> idealDimensions; // 理想值（可选）
  final Color color;
  final Color secondaryColor;

  const _RadarChart({
    required this.dimensions,
    required this.idealDimensions,
    required this.color,
    required this.secondaryColor,
  });

  @override
  Widget build(BuildContext context) {
    return CustomPaint(
      size: const Size(double.infinity, 240),
      painter: _RadarChartPainter(
        labels: dimensions.map((d) => d.name as String).toList(),
        values: dimensions.map((d) {
          final s = (d.score as num).toDouble();
          return s > 1 ? s / 100.0 : s;
        }).toList(),
        idealValues: idealDimensions.map((d) {
          final s = (d.score as num).toDouble();
          return s > 1 ? s / 100.0 : s;
        }).toList(),
        color: color,
        secondaryColor: secondaryColor,
      ),
    );
  }
}

class _RadarChartPainter extends CustomPainter {
  final List<String> labels;
  final List<double> values;
  final List<double> idealValues;
  final Color color;
  final Color secondaryColor;

  _RadarChartPainter({
    required this.labels,
    required this.values,
    required this.idealValues,
    required this.color,
    required this.secondaryColor,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    final radius = min(size.width, size.height) / 2 - 30;
    final n = labels.length;
    if (n == 0) return;

    final angleStep = 2 * pi / n;
    // 从顶部开始 (-pi/2)
    final startAngle = -pi / 2;

    // 绘制网格（5 层同心多边形）
    for (int level = 1; level <= 5; level++) {
      final levelRadius = radius * level / 5.0;
      final path = Path();
      for (int i = 0; i < n; i++) {
        final angle = startAngle + i * angleStep;
        final x = center.dx + levelRadius * cos(angle);
        final y = center.dy + levelRadius * sin(angle);
        if (i == 0) {
          path.moveTo(x, y);
        } else {
          path.lineTo(x, y);
        }
      }
      path.close();
      canvas.drawPath(
        path,
        Paint()
          ..color = Colors.grey.withOpacity( 0.15)
          ..style = PaintingStyle.stroke
          ..strokeWidth = 1,
      );
    }

    // 绘制轴线
    for (int i = 0; i < n; i++) {
      final angle = startAngle + i * angleStep;
      final x = center.dx + radius * cos(angle);
      final y = center.dy + radius * sin(angle);
      canvas.drawLine(
        center,
        Offset(x, y),
        Paint()
          ..color = Colors.grey.withOpacity( 0.3)
          ..strokeWidth = 1,
      );
    }

    // 绘制理想值（淡色虚线多边形）
    if (idealValues.length == n) {
      final idealPath = Path();
      for (int i = 0; i < n; i++) {
        final angle = startAngle + i * angleStep;
        final value = idealValues[i].clamp(0.0, 1.0);
        final vRadius = radius * value;
        final x = center.dx + vRadius * cos(angle);
        final y = center.dy + vRadius * sin(angle);
        if (i == 0) {
          idealPath.moveTo(x, y);
        } else {
          idealPath.lineTo(x, y);
        }
      }
      idealPath.close();
      canvas.drawPath(
        idealPath,
        Paint()
          ..color = secondaryColor.withOpacity( 0.4)
          ..style = PaintingStyle.stroke
          ..strokeWidth = 2,
      );
      // 理想值数据点
      for (int i = 0; i < n; i++) {
        final angle = startAngle + i * angleStep;
        final value = idealValues[i].clamp(0.0, 1.0);
        final vRadius = radius * value;
        canvas.drawCircle(
          Offset(center.dx + vRadius * cos(angle),
              center.dy + vRadius * sin(angle)),
          3,
          Paint()..color = secondaryColor.withOpacity( 0.6),
        );
      }
    }

    // 绘制实际值（填充 + 描边）
    final dataPath = Path();
    for (int i = 0; i < n; i++) {
      final angle = startAngle + i * angleStep;
      final value = values[i].clamp(0.0, 1.0);
      final vRadius = radius * value;
      final x = center.dx + vRadius * cos(angle);
      final y = center.dy + vRadius * sin(angle);
      if (i == 0) {
        dataPath.moveTo(x, y);
      } else {
        dataPath.lineTo(x, y);
      }
    }
    dataPath.close();

    // 填充
    canvas.drawPath(
      dataPath,
      Paint()
        ..color = color.withOpacity( 0.2)
        ..style = PaintingStyle.fill,
    );

    // 描边
    canvas.drawPath(
      dataPath,
      Paint()
        ..color = color
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2.5,
    );

    // 数据点
    for (int i = 0; i < n; i++) {
      final angle = startAngle + i * angleStep;
      final value = values[i].clamp(0.0, 1.0);
      final vRadius = radius * value;
      final point = Offset(
        center.dx + vRadius * cos(angle),
        center.dy + vRadius * sin(angle),
      );
      canvas.drawCircle(
        point,
        5,
        Paint()..color = color,
      );
      canvas.drawCircle(
        point,
        2.5,
        Paint()..color = Colors.white,
      );
    }

    // 标签
    for (int i = 0; i < n; i++) {
      final angle = startAngle + i * angleStep;
      // 标签放在多边形外侧
      final labelRadius = radius + 22;
      final x = center.dx + labelRadius * cos(angle);
      final y = center.dy + labelRadius * sin(angle);

      final textPainter = TextPainter(
        text: TextSpan(
          text: labels[i],
          style: TextStyle(
            color: Colors.grey.shade700,
            fontSize: 12,
            fontWeight: FontWeight.w500,
          ),
        ),
        textDirection: TextDirection.ltr,
        textAlign: TextAlign.center,
      );
      textPainter.layout(maxWidth: 80);
      textPainter.paint(
        canvas,
        Offset(x - textPainter.width / 2, y - textPainter.height / 2),
      );
    }
  }

  @override
  bool shouldRepaint(covariant _RadarChartPainter oldDelegate) {
    return values != oldDelegate.values ||
        idealValues != oldDelegate.idealValues;
  }
}
