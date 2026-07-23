import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../config/api_config.dart';

class HotTopicSensePage extends StatefulWidget {
  const HotTopicSensePage({super.key});
  @override
  State<HotTopicSensePage> createState() => _HotTopicSensePageState();
}

class _HotTopicSensePageState extends State<HotTopicSensePage> {
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    try {
      await context.read<StudentFeatureProvider>().askAI(ApiConfig.counselorHotTopicSense);
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('热点感知')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                Card(
                  color: Colors.red.withOpacity( 0.05),
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Row(children: [
                      const Icon(Icons.warning_amber, color: Colors.red, size: 32),
                      const SizedBox(width: 12),
                      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Text('舆情预警', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                        const SizedBox(height: 4),
                        Text('当前无异常舆情', style: theme.textTheme.bodyMedium?.copyWith(color: Colors.green)),
                      ])),
                    ]),
                  ),
                ),
                const SizedBox(height: 16),
                _topicCard(theme, '期中考试安排', 95, 'rising', '本学期期中考试集中在第10-11周'),
                _topicCard(theme, '暑期实习招聘', 82, 'rising', '多家互联网公司开放暑期实习岗位'),
                _topicCard(theme, '校园网升级', 68, 'stable', '校园网将于下周升级至千兆'),
                _topicCard(theme, '宿舍调整', 45, 'falling', '下学期宿舍调整方案已公布'),
              ],
            ),
    );
  }

  Widget _topicCard(ThemeData theme, String title, int heat, String trend, String summary) {
    final trendIcon = trend == 'rising' ? Icons.trending_up : (trend == 'falling' ? Icons.trending_down : Icons.trending_flat);
    final trendColor = trend == 'rising' ? Colors.red : (trend == 'falling' ? Colors.blue : Colors.grey);
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor: trendColor.withOpacity( 0.1),
          child: Text('$heat', style: TextStyle(color: trendColor, fontWeight: FontWeight.bold, fontSize: 13)),
        ),
        title: Text(title),
        subtitle: Text(summary, maxLines: 1, overflow: TextOverflow.ellipsis),
        trailing: Icon(trendIcon, color: trendColor),
      ),
    );
  }
}
