import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_new_features_provider.dart';
import '../../widgets/md_text.dart';

/// 大学规划页面
class PlanPage extends StatefulWidget {
  const PlanPage({super.key});
  @override
  State<PlanPage> createState() => _PlanPageState();
}

class _PlanPageState extends State<PlanPage> with SingleTickerProviderStateMixin {
  late TabController _tabCtrl;

  @override
  void initState() {
    super.initState();
    _tabCtrl = TabController(length: 2, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<StudentNewFeaturesProvider>();
      p.fetchPlanTemplates();
      p.fetchMyPlans();
    });
  }

  @override
  void dispose() {
    _tabCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('大学规划'),
        bottom: TabBar(
          controller: _tabCtrl,
          tabs: const [Tab(text: '规划模板'), Tab(text: '我的规划')],
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            tooltip: '创建规划',
            onPressed: () => _showCreatePlanDialog(context),
          ),
        ],
      ),
      body: Consumer<StudentNewFeaturesProvider>(
        builder: (_, p, __) {
          if (p.loading) return const Center(child: CircularProgressIndicator());
          return TabBarView(
            controller: _tabCtrl,
            children: [_buildTemplatesTab(p), _buildMyPlansTab(context, p)],
          );
        },
      ),
    );
  }

  Widget _buildTemplatesTab(StudentNewFeaturesProvider p) {
    if (p.planTemplates.isEmpty) return const Center(child: Text('暂无模板'));
    return ListView.builder(
      padding: const EdgeInsets.all(12),
      itemCount: p.planTemplates.length,
      itemBuilder: (_, i) {
        final t = p.planTemplates[i];
        return Card(
          margin: const EdgeInsets.only(bottom: 10),
          child: ListTile(
            leading: CircleAvatar(
              child: Text(t.category == 'academic' ? '学' : (t.category == 'career' ? '职' : '自')),
            ),
            title: Text(t.name),
            subtitle: Text(t.description, maxLines: 2, overflow: TextOverflow.ellipsis),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => _showTemplateDetail(context, t),
          ),
        );
      },
    );
  }

  Widget _buildMyPlansTab(BuildContext context, StudentNewFeaturesProvider p) {
    if (p.myPlans.isEmpty) return const Center(child: Text('暂无规划，点击右上角创建'));
    return ListView.builder(
      padding: const EdgeInsets.all(12),
      itemCount: p.myPlans.length,
      itemBuilder: (_, i) {
        final plan = p.myPlans[i];
        final statusColor = _statusColor(plan.status);
        return Card(
          margin: const EdgeInsets.only(bottom: 10),
          child: ListTile(
            title: Text(plan.title),
            subtitle: Text(plan.content, maxLines: 2, overflow: TextOverflow.ellipsis),
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(color: statusColor.withOpacity(0.1), borderRadius: BorderRadius.circular(4)),
                  child: Text(plan.statusLabel, style: TextStyle(color: statusColor, fontSize: 11)),
                ),
                if (plan.status == 'draft') ...[
                  const SizedBox(width: 4),
                  IconButton(
                    icon: const Icon(Icons.send, size: 18),
                    tooltip: '提交审核',
                    onPressed: () async {
                      final ok = await p.submitPlan(plan.id);
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(
                          SnackBar(content: Text(ok ? '已提交' : p.error)),
                        );
                      }
                    },
                  ),
                ],
              ],
            ),
          ),
        );
      },
    );
  }

  void _showTemplateDetail(BuildContext context, dynamic t) {
    showDialog(
      context: context,
      builder: (_) => AlertDialog(
        title: Text(t.name),
        content: SingleChildScrollView(child: MdText(t.content.isNotEmpty ? t.content : t.description)),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text('关闭')),
          FilledButton(
            onPressed: () {
              Navigator.pop(context);
              _showCreatePlanDialog(context, templateId: t.id, title: t.name, category: t.category);
            },
            child: const Text('基于此模板创建'),
          ),
        ],
      ),
    );
  }

  void _showCreatePlanDialog(BuildContext context, {int? templateId, String? title, String category = 'custom'}) {
    final titleCtrl = TextEditingController(text: title ?? '');
    final contentCtrl = TextEditingController();
    final formKey = GlobalKey<FormState>();

    showDialog(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('创建大学规划'),
        content: Form(
          key: formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextFormField(
                controller: titleCtrl,
                decoration: const InputDecoration(labelText: '规划名称', border: OutlineInputBorder()),
                validator: (v) => v == null || v.isEmpty ? '请输入名称' : null,
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: contentCtrl,
                maxLines: 4,
                decoration: const InputDecoration(labelText: '规划内容', border: OutlineInputBorder()),
                validator: (v) => v == null || v.isEmpty ? '请输入内容' : null,
              ),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text('取消')),
          FilledButton(
            onPressed: () async {
              if (!formKey.currentState!.validate()) return;
              final ok = await context.read<StudentNewFeaturesProvider>().createPlan(
                title: titleCtrl.text,
                content: contentCtrl.text,
                templateId: templateId,
                category: category,
              );
              if (context.mounted) {
                Navigator.pop(context);
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(ok ? '创建成功' : '创建失败')),
                );
              }
            },
            child: const Text('创建'),
          ),
        ],
      ),
    );
  }

  Color _statusColor(String s) {
    switch (s) {
      case 'draft': return Colors.grey;
      case 'submitted': return Colors.orange;
      case 'approved': return Colors.green;
      case 'in_progress': return Colors.blue;
      case 'completed': return Colors.teal;
      case 'rejected': return Colors.red;
      default: return Colors.grey;
    }
  }
}
