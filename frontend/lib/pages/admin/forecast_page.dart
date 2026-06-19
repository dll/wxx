import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/forecast_provider.dart';
import '../../widgets/error_view.dart';

/// 问题预案页面（sys_admin、college_admin 可访问）
class ForecastPage extends StatefulWidget {
  const ForecastPage({super.key});

  @override
  State<ForecastPage> createState() => _ForecastPageState();
}

class _ForecastPageState extends State<ForecastPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<ForecastProvider>().fetchForecasts(refresh: true);
      context.read<ForecastProvider>().fetchStatistics();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('问题预案'),
        actions: [
          IconButton(
            icon: const Icon(Icons.analytics),
            tooltip: '查看统计',
            onPressed: () => _showStatistics(context),
          ),
          IconButton(
            icon: const Icon(Icons.auto_awesome),
            tooltip: '执行分析',
            onPressed: () => _analyzeIssues(context),
          ),
        ],
      ),
      body: Column(
        children: [
          _buildFilterBar(),
          Expanded(child: _buildForecastList()),
        ],
      ),
    );
  }

  Widget _buildFilterBar() {
    return Consumer<ForecastProvider>(
      builder: (_, provider, __) {
        return Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          child: Column(
            children: [
              Row(
                children: [
                  Expanded(
                    child: SegmentedButton<String>(
                      selected: {provider.categoryFilter.isEmpty ? '' : provider.categoryFilter},
                      onSelectionChanged: (v) {
                        provider.setCategoryFilter(v.first == '' ? '' : v.first);
                      },
                      segments: const [
                        ButtonSegment(value: '', label: Text('全部')),
                        ButtonSegment(value: 'emotion', label: Text('情感')),
                        ButtonSegment(value: 'process', label: Text('流程')),
                        ButtonSegment(value: 'graduation', label: Text('毕业')),
                        ButtonSegment(value: 'feedback', label: Text('反馈')),
                      ],
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Expanded(
                    child: SegmentedButton<String>(
                      selected: {provider.riskLevelFilter.isEmpty ? '' : provider.riskLevelFilter},
                      onSelectionChanged: (v) {
                        provider.setRiskLevelFilter(v.first == '' ? '' : v.first);
                      },
                      segments: const [
                        ButtonSegment(value: '', label: Text('全部风险')),
                        ButtonSegment(value: 'urgent', label: Text('紧急')),
                        ButtonSegment(value: 'high', label: Text('高')),
                        ButtonSegment(value: 'medium', label: Text('中')),
                        ButtonSegment(value: 'low', label: Text('低')),
                      ],
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Expanded(
                    child: SegmentedButton<String>(
                      selected: {provider.statusFilter.isEmpty ? '' : provider.statusFilter},
                      onSelectionChanged: (v) {
                        provider.setStatusFilter(v.first == '' ? '' : v.first);
                      },
                      segments: const [
                        ButtonSegment(value: '', label: Text('全部状态')),
                        ButtonSegment(value: 'pending', label: Text('待处理')),
                        ButtonSegment(value: 'processing', label: Text('处理中')),
                        ButtonSegment(value: 'resolved', label: Text('已解决')),
                        ButtonSegment(value: 'archived', label: Text('已归档')),
                      ],
                    ),
                  ),
                ],
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildForecastList() {
    return Consumer<ForecastProvider>(
      builder: (_, provider, __) {
        if (provider.loading && provider.forecasts.isEmpty) {
          return const Center(child: CircularProgressIndicator());
        }
        if (provider.error.isNotEmpty && provider.forecasts.isEmpty) {
          return ErrorView.error(
            message: provider.error,
            onRetry: () => provider.fetchForecasts(refresh: true),
          );
        }
        if (provider.forecasts.isEmpty) {
          return ErrorView.empty(
            message: '暂无问题预案',
            icon: Icons.warning_amber_outlined,
          );
        }
        return RefreshIndicator(
          onRefresh: () => provider.fetchForecasts(refresh: true),
          child: ListView.builder(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            itemCount: provider.forecasts.length + 1,
            itemBuilder: (context, index) {
              if (index == provider.forecasts.length) {
                if (provider.forecasts.length < provider.total) {
                  provider.fetchForecasts();
                  return const Padding(
                    padding: EdgeInsets.all(16),
                    child: Center(child: CircularProgressIndicator()),
                  );
                }
                return const SizedBox.shrink();
              }
              return _ForecastCard(
                forecast: provider.forecasts[index],
                onTap: () => _showDetail(context, provider.forecasts[index]),
              );
            },
          ),
        );
      },
    );
  }

  void _showStatistics(BuildContext context) {
    final provider = context.read<ForecastProvider>();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('问题统计'),
        content: SizedBox(
          width: 400,
          height: 300,
          child: provider.statistics == null
              ? const Center(child: CircularProgressIndicator())
              : Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('风险分布', style: Theme.of(context).textTheme.titleMedium),
                    const SizedBox(height: 8),
                    ...provider.statistics!.riskDistribution.entries.map(
                      (e) => Padding(
                        padding: const EdgeInsets.symmetric(vertical: 4),
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Text(_riskLevelLabel(e.key)),
                            Text('${e.value}个'),
                          ],
                        ),
                      ),
                    ),
                    const Divider(),
                    Text('分类分布', style: Theme.of(context).textTheme.titleMedium),
                    const SizedBox(height: 8),
                    ...provider.statistics!.categoryDistribution.entries.map(
                      (e) => Padding(
                        padding: const EdgeInsets.symmetric(vertical: 4),
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Text(_categoryLabel(e.key)),
                            Text('${e.value}个'),
                          ],
                        ),
                      ),
                    ),
                  ],
                ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('关闭'),
          ),
        ],
      ),
    );
  }

  Future<void> _analyzeIssues(BuildContext context) async {
    final provider = context.read<ForecastProvider>();
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('执行问题分析'),
        content: const Text('将汇总多源数据并生成问题预案，确定执行？'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('确认执行'),
          ),
        ],
      ),
    );

    if (confirmed == true && context.mounted) {
      final ok = await provider.analyzeIssues();
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(ok ? '分析完成' : '分析失败')),
        );
      }
    }
  }

  void _showDetail(BuildContext context, IssueForecast forecast) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(forecast.title),
        content: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              _buildInfoRow('分类', forecast.categoryLabel),
              _buildInfoRow('风险等级', forecast.riskLevelLabel),
              _buildInfoRow('状态', forecast.statusLabel),
              _buildInfoRow('负责人', forecast.ownerName),
              _buildInfoRow('创建时间', forecast.createdAt),
              const Divider(),
              if (forecast.summary.isNotEmpty) ...[
                const Text('摘要', style: TextStyle(fontWeight: FontWeight.bold)),
                const SizedBox(height: 4),
                Text(forecast.summary),
                const SizedBox(height: 12),
              ],
              if (forecast.rootCause.isNotEmpty) ...[
                const Text('原因分析', style: TextStyle(fontWeight: FontWeight.bold)),
                const SizedBox(height: 4),
                Text(forecast.rootCause),
                const SizedBox(height: 12),
              ],
              if (forecast.recommendedAction.isNotEmpty) ...[
                const Text('解决预案', style: TextStyle(fontWeight: FontWeight.bold)),
                const SizedBox(height: 4),
                Text(forecast.recommendedAction),
              ],
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('关闭'),
          ),
          if (forecast.status == 'pending') ...[
            FilledButton(
              onPressed: () async {
                Navigator.pop(ctx);
                final provider = context.read<ForecastProvider>();
                await provider.updateStatus(forecast.forecastId, 'processing');
                if (context.mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('状态已更新为处理中')),
                  );
                }
              },
              child: const Text('开始处理'),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildInfoRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label, style: const TextStyle(color: Colors.grey)),
          Text(value),
        ],
      ),
    );
  }

  String _riskLevelLabel(String key) {
    switch (key) {
      case 'urgent': return '紧急';
      case 'high': return '高';
      case 'medium': return '中';
      case 'low': return '低';
      default: return key;
    }
  }

  String _categoryLabel(String key) {
    switch (key) {
      case 'emotion': return '情感问题';
      case 'process': return '流程问题';
      case 'graduation': return '毕业问题';
      case 'feedback': return '反馈问题';
      case 'comprehensive': return '综合问题';
      default: return key;
    }
  }
}

class _ForecastCard extends StatelessWidget {
  final IssueForecast forecast;
  final VoidCallback onTap;

  const _ForecastCard({required this.forecast, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: _getRiskColor(forecast.riskLevel).withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      forecast.riskLevelLabel,
                      style: TextStyle(
                        fontSize: 11,
                        color: _getRiskColor(forecast.riskLevel),
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: theme.colorScheme.primaryContainer.withValues(alpha: 0.3),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      forecast.categoryLabel,
                      style: TextStyle(
                        fontSize: 11,
                        color: theme.colorScheme.primary,
                      ),
                    ),
                  ),
                  const Spacer(),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: _getStatusColor(forecast.status).withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      forecast.statusLabel,
                      style: TextStyle(
                        fontSize: 11,
                        color: _getStatusColor(forecast.status),
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              Text(
                forecast.title,
                style: theme.textTheme.titleSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                forecast.summary,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Icon(Icons.person_outline, size: 14, color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 4),
                  Text(
                    forecast.ownerName.isNotEmpty ? forecast.ownerName : '系统',
                    style: theme.textTheme.labelSmall,
                  ),
                  const Spacer(),
                  Icon(Icons.access_time, size: 14, color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 4),
                  Text(
                    forecast.createdAt,
                    style: theme.textTheme.labelSmall,
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Color _getRiskColor(String riskLevel) {
    switch (riskLevel) {
      case 'urgent':
        return Colors.red;
      case 'high':
        return Colors.orange;
      case 'medium':
        return Colors.yellow.shade700;
      case 'low':
        return Colors.green;
      default:
        return Colors.grey;
    }
  }

  Color _getStatusColor(String status) {
    switch (status) {
      case 'pending':
        return Colors.orange;
      case 'processing':
        return Colors.blue;
      case 'resolved':
        return Colors.green;
      case 'archived':
        return Colors.grey;
      default:
        return Colors.grey;
    }
  }
}
