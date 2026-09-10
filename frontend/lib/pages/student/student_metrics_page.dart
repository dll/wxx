import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../providers/student_feature_provider.dart';

/// 学习数据列表：保留可解释的指标，移除数字人、动画和图片生成效果。
class StudentMetricsPage extends StatefulWidget {
  const StudentMetricsPage({super.key});

  @override
  State<StudentMetricsPage> createState() => _StudentMetricsPageState();
}

class _StudentMetricsPageState extends State<StudentMetricsPage> {
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
    final twin = provider.twin;
    return Scaffold(
      appBar: AppBar(title: const Text('学习数据')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchDigitalTwin(),
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Text('成长指标', style: theme.textTheme.titleLarge),
            const SizedBox(height: 8),
            if (provider.loading && twin == null)
              const LinearProgressIndicator()
            else if (twin == null || twin.dimensions.isEmpty)
              const ListTile(
                leading: Icon(Icons.info_outline),
                title: Text('暂无可展示的数据'),
                subtitle: Text('完成课程、打卡或活动后，相关数据会在这里显示。'),
              )
            else
              ...twin.dimensions.map((item) => Card(
                    margin: const EdgeInsets.only(bottom: 8),
                    child: ListTile(
                      leading: CircleAvatar(
                        child: Text(item.score.toStringAsFixed(0)),
                      ),
                      title: Text(item.name.isEmpty ? '未命名指标' : item.name),
                      subtitle: Text(item.label.isEmpty
                          ? (item.dataAvailable ? '已记录数据' : '暂无真实记录')
                          : item.label),
                      trailing: item.dataAvailable
                          ? const Icon(Icons.check_circle_outline,
                              color: Colors.green)
                          : const Icon(Icons.remove_circle_outline),
                    ),
                  )),
            if (twin != null && twin.suggestions.isNotEmpty) ...[
              const SizedBox(height: 16),
              Text('建议', style: theme.textTheme.titleMedium),
              ...twin.suggestions.map((text) => ListTile(
                    dense: true,
                    leading: const Icon(Icons.arrow_right),
                    title: Text(text),
                  )),
            ],
          ],
        ),
      ),
    );
  }
}
