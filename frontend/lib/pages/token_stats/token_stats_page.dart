import 'dart:math';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:fl_chart/fl_chart.dart';
import '../../models/models.dart';
import '../../providers/token_stats_provider.dart';
import '../../utils/capability_utils.dart';
import '../../widgets/error_view.dart';

class TokenStatsPage extends StatefulWidget {
  const TokenStatsPage({super.key});

  @override
  State<TokenStatsPage> createState() => _TokenStatsPageState();
}

class _TokenStatsPageState extends State<TokenStatsPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<TokenStatsProvider>().fetchAll();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('词元统计')),
      body: Consumer<TokenStatsProvider>(
        builder: (_, provider, __) {
          if (provider.loading && provider.myStats == null) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.error.isNotEmpty && provider.myStats == null) {
            return ErrorView.error(
              message: provider.error,
              onRetry: () => provider.fetchAll(),
            );
          }
          final stats = provider.myStats;
          if (stats == null) {
            return ErrorView.empty(message: '暂无数据');
          }
          return RefreshIndicator(
            onRefresh: () => provider.fetchAll(),
            child: ListView(
              padding: const EdgeInsets.all(16),
              children: [
                _buildTimeRangeSelector(provider),
                const SizedBox(height: 16),
                _buildSummaryCards(stats.summary),
                const SizedBox(height: 24),
                _buildTrendChart(stats.daily),
                if (CapabilityUtils.has(Capability.counselorTokenSubordinates)) ...[
                  const SizedBox(height: 24),
                  _buildSubordinateSection(provider),
                ],
              ],
            ),
          );
        },
      ),
    );
  }

  Widget _buildTimeRangeSelector(TokenStatsProvider provider) {
    return SegmentedButton<int>(
      segments: const [
        ButtonSegment(value: 7, label: Text('7天')),
        ButtonSegment(value: 30, label: Text('30天')),
        ButtonSegment(value: 90, label: Text('90天')),
      ],
      selected: {provider.days},
      onSelectionChanged: (values) => provider.setDays(values.first),
    );
  }

  Widget _buildSummaryCards(TokenStatsSummary summary) {
    return LayoutBuilder(
      builder: (_, constraints) {
        final crossAxisCount = constraints.maxWidth > 600 ? 4 : 2;
        return GridView.extent(
          maxCrossAxisExtent: constraints.maxWidth / crossAxisCount,
          mainAxisSpacing: 12,
          crossAxisSpacing: 12,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          children: [
            _StatCard(
              label: '总词元',
              value: _formatNumber(summary.totalTokens),
              icon: Icons.data_usage,
              color: Colors.blue,
            ),
            _StatCard(
              label: '输入词元',
              value: _formatNumber(summary.totalPromptTokens),
              icon: Icons.arrow_upward,
              color: Colors.green,
            ),
            _StatCard(
              label: '输出词元',
              value: _formatNumber(summary.totalOutputTokens),
              icon: Icons.arrow_downward,
              color: Colors.orange,
            ),
            _StatCard(
              label: '今日消耗',
              value: _formatNumber(summary.todayTokens),
              icon: Icons.today,
              color: Colors.purple,
            ),
          ],
        );
      },
    );
  }

  Widget _buildTrendChart(List<TokenDailyPoint> daily) {
    if (daily.isEmpty) {
      return const Card(
        child: Padding(
          padding: EdgeInsets.all(32),
          child: Center(child: Text('暂无趋势数据')),
        ),
      );
    }

    final maxTotal = daily.map((e) => e.totalTokens.toDouble()).reduce(max);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('消耗趋势', style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 16),
            SizedBox(
              height: 200,
              child: LineChart(
                LineChartData(
                  gridData: FlGridData(show: true, drawVerticalLine: false),
                  titlesData: FlTitlesData(
                    leftTitles: AxisTitles(
                      sideTitles: SideTitles(
                        showTitles: true,
                        reservedSize: 50,
                        getTitlesWidget: (value, meta) {
                          return Text(_formatNumber(value.toInt()), style: const TextStyle(fontSize: 10));
                        },
                      ),
                    ),
                    bottomTitles: AxisTitles(
                      sideTitles: SideTitles(
                        showTitles: true,
                        interval: max((daily.length / 5).ceilToDouble(), 1),
                        getTitlesWidget: (value, meta) {
                          final idx = value.toInt();
                          if (idx < 0 || idx >= daily.length) return const SizedBox();
                          return Padding(
                            padding: const EdgeInsets.only(top: 8),
                            child: Text(daily[idx].date.substring(5), style: const TextStyle(fontSize: 10)),
                          );
                        },
                      ),
                    ),
                    rightTitles: AxisTitles(sideTitles: SideTitles(showTitles: false)),
                    topTitles: AxisTitles(sideTitles: SideTitles(showTitles: false)),
                  ),
                  borderData: FlBorderData(show: false),
                  lineBarsData: [
                    LineChartBarData(
                      spots: daily.asMap().entries.map((e) => FlSpot(e.key.toDouble(), e.value.totalTokens.toDouble())).toList(),
                      isCurved: true,
                      color: Colors.blue,
                      barWidth: 2,
                      dotData: FlDotData(show: daily.length <= 14),
                      belowBarData: BarAreaData(
                        show: true,
                        color: Colors.blue.withValues(alpha: 0.1),
                      ),
                    ),
                    LineChartBarData(
                      spots: daily.asMap().entries.map((e) => FlSpot(e.key.toDouble(), e.value.promptTokens.toDouble())).toList(),
                      isCurved: true,
                      color: Colors.green,
                      barWidth: 1.5,
                      dotData: FlDotData(show: false),
                    ),
                    LineChartBarData(
                      spots: daily.asMap().entries.map((e) => FlSpot(e.key.toDouble(), e.value.outputTokens.toDouble())).toList(),
                      isCurved: true,
                      color: Colors.orange,
                      barWidth: 1.5,
                      dotData: FlDotData(show: false),
                    ),
                  ],
                  minY: 0,
                  maxY: maxTotal * 1.2,
                ),
              ),
            ),
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                _chartLegend(Colors.blue, '总计'),
                const SizedBox(width: 16),
                _chartLegend(Colors.green, '输入'),
                const SizedBox(width: 16),
                _chartLegend(Colors.orange, '输出'),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildSubordinateSection(TokenStatsProvider provider) {
    final subs = provider.subordinateStats;
    if (subs.isEmpty) {
      return const Card(
        child: Padding(
          padding: EdgeInsets.all(16),
          child: Text('暂无下级用户数据'),
        ),
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('下级用户词元用量', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 12),
        _buildSubordinateBarChart(subs),
        const SizedBox(height: 16),
        ...subs.map((s) => Card(
          margin: const EdgeInsets.only(bottom: 8),
          child: ListTile(
            leading: CircleAvatar(
              child: Text(s.displayName.isNotEmpty ? s.displayName[0] : '?'),
            ),
            title: Text(s.displayName.isNotEmpty ? s.displayName : s.username),
            subtitle: Text('输入: ${_formatNumber(s.promptTokens)} / 输出: ${_formatNumber(s.outputTokens)}'),
            trailing: Text(
              _formatNumber(s.totalTokens),
              style: const TextStyle(fontWeight: FontWeight.bold),
            ),
          ),
        )),
      ],
    );
  }

  Widget _buildSubordinateBarChart(List<SubordinateTokenStats> subs) {
    final displaySubs = subs.length > 10 ? subs.sublist(0, 10) : subs;
    final maxTokens = displaySubs.map((e) => e.totalTokens.toDouble()).reduce(max);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: SizedBox(
          height: 200,
          child: BarChart(
            BarChartData(
              alignment: BarChartAlignment.spaceAround,
              maxY: maxTokens * 1.2,
              gridData: FlGridData(show: true, drawVerticalLine: false),
              titlesData: FlTitlesData(
                leftTitles: AxisTitles(
                  sideTitles: SideTitles(
                    showTitles: true,
                    reservedSize: 50,
                    getTitlesWidget: (value, meta) {
                      return Text(_formatNumber(value.toInt()), style: const TextStyle(fontSize: 10));
                    },
                  ),
                ),
                bottomTitles: AxisTitles(
                  sideTitles: SideTitles(
                    showTitles: true,
                    getTitlesWidget: (value, meta) {
                      final idx = value.toInt();
                      if (idx < 0 || idx >= displaySubs.length) return const SizedBox();
                      final name = displaySubs[idx].displayName;
                      return Padding(
                        padding: const EdgeInsets.only(top: 8),
                        child: Text(name.length > 4 ? '${name.substring(0, 4)}…' : name, style: const TextStyle(fontSize: 10)),
                      );
                    },
                  ),
                ),
                rightTitles: AxisTitles(sideTitles: SideTitles(showTitles: false)),
                topTitles: AxisTitles(sideTitles: SideTitles(showTitles: false)),
              ),
              borderData: FlBorderData(show: false),
              barGroups: displaySubs.asMap().entries.map((e) => BarChartGroupData(
                x: e.key,
                barRods: [
                  BarChartRodData(
                    toY: e.value.totalTokens.toDouble(),
                    color: Colors.blue,
                    width: 16,
                    borderRadius: const BorderRadius.vertical(top: Radius.circular(4)),
                  ),
                ],
              )).toList(),
            ),
          ),
        ),
      ),
    );
  }

  Widget _chartLegend(Color color, String label) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(width: 12, height: 12, decoration: BoxDecoration(color: color, shape: BoxShape.circle)),
        const SizedBox(width: 4),
        Text(label, style: const TextStyle(fontSize: 12)),
      ],
    );
  }

  String _formatNumber(int n) {
    if (n >= 1000000) return '${(n / 1000000).toStringAsFixed(1)}M';
    if (n >= 1000) return '${(n / 1000).toStringAsFixed(1)}K';
    return n.toString();
  }
}

class _StatCard extends StatelessWidget {
  final String label;
  final String value;
  final IconData icon;
  final Color color;
  const _StatCard({required this.label, required this.value, required this.icon, required this.color});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 28, color: color),
            const SizedBox(height: 8),
            Text(value, style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold, color: color)),
            const SizedBox(height: 4),
            Text(label, style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ],
        ),
      ),
    );
  }
}
