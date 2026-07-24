import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/admin_provider.dart';
import '../../widgets/error_view.dart';
import '../../models/models.dart';

/// 管理员数据仪表盘页面
class AdminDashboardPage extends StatefulWidget {
  const AdminDashboardPage({super.key});

  @override
  State<AdminDashboardPage> createState() => _AdminDashboardPageState();
}

class _AdminDashboardPageState extends State<AdminDashboardPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AdminProvider>().fetchDashboard();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('数据概览'),
      ),
      body: Consumer<AdminProvider>(
        builder: (_, provider, __) {
          if (provider.dashboardLoading && provider.dashboard == null) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.error.isNotEmpty && provider.dashboard == null) {
            return ErrorView.error(
              message: provider.error,
              onRetry: () => provider.fetchDashboard(),
            );
          }
          final data = provider.dashboard;
          if (data == null) {
            return ErrorView.empty(message: '暂无数据');
          }
          return RefreshIndicator(
            onRefresh: () => provider.fetchDashboard(),
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // 第一行：4 个统计卡片
                  _StatGrid(children: [
                    _StatCard(
                      label: '总用户数',
                      value: '${data.users.total}',
                      subValue: '今日新增 ${data.users.todayNew}',
                      icon: Icons.people,
                      color: Colors.blue,
                    ),
                    _StatCard(
                      label: '知识资源',
                      value: '${data.knowledge.total}',
                      subValue: '本周新增 ${data.knowledge.weekNew}',
                      icon: Icons.menu_book,
                      color: Colors.green,
                    ),
                    _StatCard(
                      label: '对话总数',
                      value: '${data.chat.totalSessions}',
                      subValue: '今日 ${data.chat.todaySessions} 会话',
                      icon: Icons.chat,
                      color: Colors.orange,
                    ),
                    _StatCard(
                      label: '反馈总数',
                      value: '${data.feedback.total}',
                      subValue: '待处理 ${data.feedback.byStatus['pending'] ?? 0}',
                      icon: Icons.feedback,
                      color: Colors.red,
                    ),
                  ]),
                  const SizedBox(height: 16),
                  // 第二行：近 7 天对话趋势图
                  _buildTrendCard(data.chat),
                  const SizedBox(height: 16),
                  // 第三行：两个分布卡片
                  _DistGrid(children: [
                    _buildKnowledgeDistCard(data.knowledge),
                    _buildUserRoleDistCard(data.users),
                  ]),
                  const SizedBox(height: 16),
                  // 底部：最新待处理反馈
                  _buildFeedbackCard(data.feedback),
                ],
              ),
            ),
          );
        },
      ),
    );
  }

  /// 构建近 7 天对话趋势卡片
  Widget _buildTrendCard(ChatStats chat) {
    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  '近 7 天对话趋势',
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                ),
                const Row(
                  children: [
                    _LegendItem(color: Colors.blue, label: '会话'),
                    SizedBox(width: 12),
                    _LegendItem(color: Colors.orange, label: '消息'),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 16),
            SizedBox(
              height: 200,
              child: _BarChart(
                data: chat.weekTrend,
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// 构建知识库类型分布卡片
  Widget _buildKnowledgeDistCard(KnowledgeStats knowledge) {
    final items = [
      _DistItem('政策', knowledge.byType['policy'] ?? 0, Colors.blue),
      _DistItem('流程', knowledge.byType['process'] ?? 0, Colors.green),
      _DistItem('FAQ', knowledge.byType['faq'] ?? 0, Colors.orange),
      _DistItem('活动', knowledge.byType['activity'] ?? 0, Colors.purple),
    ];
    final total = items.fold<int>(0, (sum, item) => sum + item.value);

    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '知识库类型分布',
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
            ),
            const SizedBox(height: 16),
            _SimplePieChart(items: items, total: total),
            const SizedBox(height: 12),
            ...items.map((item) => Padding(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  child: Row(
                    children: [
                      Container(
                        width: 12,
                        height: 12,
                        decoration: BoxDecoration(
                          color: item.color,
                          shape: BoxShape.circle,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(child: Text(item.label)),
                      Text(
                        '${item.value} (${total > 0 ? ((item.value / total) * 100).toStringAsFixed(1) : 0}%)',
                        style: const TextStyle(fontWeight: FontWeight.w500),
                      ),
                    ],
                  ),
                )),
          ],
        ),
      ),
    );
  }

  /// 构建用户角色分布卡片
  Widget _buildUserRoleDistCard(UserStats users) {
    final roleLabels = {
      'student': '学生',
      'counselor': '辅导员',
      'college_admin': '学院管理员',
      'school_admin': '学校管理员',
      'sys_admin': '系统管理员',
      'student_union': '学生会',
      'teacher': '教师',
      'assistant': '教辅',
    };
    final colors = [
      Colors.blue,
      Colors.green,
      Colors.orange,
      Colors.purple,
      Colors.red,
      Colors.teal,
      Colors.indigo,
      Colors.pink,
    ];

    final items = <_DistItem>[];
    var colorIndex = 0;
    users.byRole.forEach((role, count) {
      items.add(_DistItem(
        roleLabels[role] ?? role,
        count,
        colors[colorIndex % colors.length],
      ));
      colorIndex++;
    });
    final total = users.total;

    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '用户角色分布',
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
            ),
            const SizedBox(height: 16),
            _SimplePieChart(items: items, total: total),
            const SizedBox(height: 12),
            ...items.map((item) => Padding(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  child: Row(
                    children: [
                      Container(
                        width: 12,
                        height: 12,
                        decoration: BoxDecoration(
                          color: item.color,
                          shape: BoxShape.circle,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(child: Text(item.label)),
                      Text(
                        '${item.value} (${total > 0 ? ((item.value / total) * 100).toStringAsFixed(1) : 0}%)',
                        style: const TextStyle(fontWeight: FontWeight.w500),
                      ),
                    ],
                  ),
                )),
          ],
        ),
      ),
    );
  }

  /// 构建反馈统计卡片
  Widget _buildFeedbackCard(FeedbackStats feedback) {
    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '反馈统计',
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: _FeedbackStatItem(
                    label: '待处理',
                    value: feedback.byStatus['pending'] ?? 0,
                    color: Colors.orange,
                  ),
                ),
                Expanded(
                  child: _FeedbackStatItem(
                    label: '处理中',
                    value: feedback.byStatus['processing'] ?? 0,
                    color: Colors.blue,
                  ),
                ),
                Expanded(
                  child: _FeedbackStatItem(
                    label: '已解决',
                    value: feedback.byStatus['resolved'] ?? 0,
                    color: Colors.green,
                  ),
                ),
                Expanded(
                  child: _FeedbackStatItem(
                    label: '已忽略',
                    value: feedback.byStatus['dismissed'] ?? 0,
                    color: Colors.grey,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Colors.orange.withOpacity(0.1),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                children: [
                  const Icon(Icons.warning_amber_rounded,
                      color: Colors.orange, size: 20),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      '本周新增反馈 ${feedback.weekTrend.fold<int>(0, (sum, e) => sum + e.count)} 条，当前待处理 ${feedback.byStatus['pending'] ?? 0} 条',
                      style: const TextStyle(color: Colors.orange),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// 统计卡片网格
class _StatGrid extends StatelessWidget {
  final List<Widget> children;
  const _StatGrid({required this.children});

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (_, constraints) {
        final crossAxisCount = constraints.maxWidth > 600 ? 4 : 2;
        return GridView.extent(
          maxCrossAxisExtent: constraints.maxWidth / crossAxisCount,
          mainAxisSpacing: 12,
          crossAxisSpacing: 12,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          children: children,
        );
      },
    );
  }
}

/// 分布卡片网格
class _DistGrid extends StatelessWidget {
  final List<Widget> children;
  const _DistGrid({required this.children});

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (_, constraints) {
        if (constraints.maxWidth > 800) {
          return Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(child: children[0]),
              const SizedBox(width: 12),
              Expanded(child: children[1]),
            ],
          );
        }
        return Column(children: children);
      },
    );
  }
}

/// 统计卡片
class _StatCard extends StatelessWidget {
  final String label;
  final String value;
  final String subValue;
  final IconData icon;
  final Color color;

  const _StatCard({
    required this.label,
    required this.value,
    required this.subValue,
    required this.icon,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(label, style: theme.textTheme.bodyMedium),
                Icon(icon, size: 24, color: color),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              value,
              style: theme.textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.bold,
                    color: color,
                  ),
            ),
            const SizedBox(height: 4),
            Text(
              subValue,
              style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.outline,
                  ),
            ),
          ],
        ),
      ),
    );
  }
}

/// 图例项
class _LegendItem extends StatelessWidget {
  final Color color;
  final String label;
  const _LegendItem({required this.color, required this.label});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(width: 12, height: 12, color: color),
        const SizedBox(width: 4),
        Text(label, style: Theme.of(context).textTheme.bodySmall),
      ],
    );
  }
}

/// 自定义柱状图
class _BarChart extends StatelessWidget {
  final List<DayTrendItem> data;

  const _BarChart({required this.data});

  @override
  Widget build(BuildContext context) {
    if (data.isEmpty) {
      return const Center(child: Text('暂无数据'));
    }

    final maxSessions = data.fold<int>(
        0, (max, item) => item.sessions > max ? item.sessions : max);
    final maxMessages = data.fold<int>(
        0, (max, item) => item.messages > max ? item.messages : max);
    final maxVal =
        maxMessages > maxSessions ? maxMessages : maxSessions;
    final chartMax = maxVal == 0 ? 1 : (maxVal * 1.2).ceil();

    return LayoutBuilder(
      builder: (context, constraints) {
        final barWidth = constraints.maxWidth / data.length - 16;
        return Row(
          crossAxisAlignment: CrossAxisAlignment.end,
          mainAxisAlignment: MainAxisAlignment.spaceAround,
          children: data.map((item) {
            final sessionHeight = maxVal == 0
                ? 0.0
                : (item.sessions / chartMax) * constraints.maxHeight * 0.85;
            final messageHeight = maxVal == 0
                ? 0.0
                : (item.messages / chartMax) * constraints.maxHeight * 0.85;
            return Column(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                SizedBox(
                  width: barWidth < 20 ? 20 : barWidth,
                  height: constraints.maxHeight * 0.85,
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      // 会话数柱子
                      Container(
                        width: (barWidth < 20 ? 20 : barWidth) / 2 - 2,
                        height: sessionHeight,
                        decoration: BoxDecoration(
                          color: Colors.blue.withOpacity(0.8),
                          borderRadius: const BorderRadius.vertical(
                              top: Radius.circular(4)),
                        ),
                      ),
                      const SizedBox(width: 4),
                      // 消息数柱子
                      Container(
                        width: (barWidth < 20 ? 20 : barWidth) / 2 - 2,
                        height: messageHeight,
                        decoration: BoxDecoration(
                          color: Colors.orange.withOpacity(0.8),
                          borderRadius: const BorderRadius.vertical(
                              top: Radius.circular(4)),
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  item.date.substring(5),
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            );
          }).toList(),
        );
      },
    );
  }
}

/// 分布项
class _DistItem {
  final String label;
  final int value;
  final Color color;
  _DistItem(this.label, this.value, this.color);
}

/// 简易饼图（环形图）
class _SimplePieChart extends StatelessWidget {
  final List<_DistItem> items;
  final int total;

  const _SimplePieChart({required this.items, required this.total});

  @override
  Widget build(BuildContext context) {
    if (total == 0) {
      return const SizedBox(
        height: 150,
        child: Center(child: Text('暂无数据')),
      );
    }

    return SizedBox(
      height: 150,
      child: Center(
        child: CustomPaint(
          size: const Size(120, 120),
          painter: _PieChartPainter(items: items, total: total),
        ),
      ),
    );
  }
}

/// 饼图画师
class _PieChartPainter extends CustomPainter {
  final List<_DistItem> items;
  final int total;

  _PieChartPainter({required this.items, required this.total});

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    final radius = size.width / 2;
    final innerRadius = radius * 0.55;

    var startAngle = -90.0 * (3.1415926 / 180);

    for (final item in items) {
      if (item.value == 0) continue;
      final sweepAngle = (item.value / total) * 2 * 3.1415926;

      final paint = Paint()
        ..color = item.color
        ..style = PaintingStyle.fill;

      canvas.drawArc(
        Rect.fromCircle(center: center, radius: radius),
        startAngle,
        sweepAngle,
        true,
        paint,
      );

      startAngle += sweepAngle;
    }

    // 内圆（挖空成环形）
    final innerPaint = Paint()
      ..color = Colors.white
      ..style = PaintingStyle.fill;
    canvas.drawCircle(center, innerRadius, innerPaint);

    // 中间文字
    final textPainter = TextPainter(
      text: TextSpan(
        text: '$total',
        style: const TextStyle(
          color: Colors.black87,
          fontSize: 18,
          fontWeight: FontWeight.bold,
        ),
      ),
      textDirection: TextDirection.ltr,
    );
    textPainter.layout();
    textPainter.paint(
      canvas,
      Offset(center.dx - textPainter.width / 2,
          center.dy - textPainter.height / 2),
    );
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => true;
}

/// 反馈统计项
class _FeedbackStatItem extends StatelessWidget {
  final String label;
  final int value;
  final Color color;

  const _FeedbackStatItem({
    required this.label,
    required this.value,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text(
          '$value',
          style: Theme.of(context).textTheme.titleLarge?.copyWith(
                fontWeight: FontWeight.bold,
                color: color,
              ),
        ),
        const SizedBox(height: 4),
        Text(
          label,
          style: Theme.of(context).textTheme.bodySmall,
        ),
      ],
    );
  }
}
