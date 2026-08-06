import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../config/api_config.dart';
import '../../widgets/md_text.dart';

class PartyProgressPage extends StatefulWidget {
  const PartyProgressPage({super.key});
  @override
  State<PartyProgressPage> createState() => _PartyProgressPageState();
}

class _PartyProgressPageState extends State<PartyProgressPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().askAI(ApiConfig.partyProgress);
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('党建进度')),
      body: RefreshIndicator(
        onRefresh: () => provider.askAI(ApiConfig.partyProgress),
        child: provider.aiLoading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  Container(
                    padding: const EdgeInsets.all(20),
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        colors: [theme.colorScheme.primary, theme.colorScheme.primary.withOpacity( 0.7)],
                        begin: Alignment.topLeft, end: Alignment.bottomRight,
                      ),
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Row(children: [
                      Icon(Icons.flag, color: theme.colorScheme.onPrimary, size: 32),
                      const SizedBox(width: 16),
                      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Text('党建进度', style: TextStyle(color: theme.colorScheme.onPrimary, fontSize: 18, fontWeight: FontWeight.bold)),
                        const SizedBox(height: 4),
                        Text('入党流程进度跟踪', style: TextStyle(color: theme.colorScheme.onPrimary.withOpacity( 0.8), fontSize: 13)),
                      ])),
                    ]),
                  ),
                  const SizedBox(height: 16),
                  if (provider.aiResponse.isNotEmpty)
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(16),
                        child: MdText(provider.aiResponse),
                      ),
                    )
                  else
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(20),
                        child: Column(children: [
                          Icon(Icons.flag, size: 48, color: theme.colorScheme.primary.withOpacity( 0.5)),
                          const SizedBox(height: 12),
                          Text('暂无内容', style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                        ]),
                      ),
                    ),
                ],
              ),
      ),
    );
  }
}
