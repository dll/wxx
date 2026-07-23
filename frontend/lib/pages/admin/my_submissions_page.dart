import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/knowledge_provider.dart';
import '../../widgets/error_view.dart';

/// 我的提交页面（student_union 及以上可访问）
class MySubmissionsPage extends StatefulWidget {
  const MySubmissionsPage({super.key});

  @override
  State<MySubmissionsPage> createState() => _MySubmissionsPageState();
}

class _MySubmissionsPageState extends State<MySubmissionsPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<KnowledgeProvider>().listResources();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('我的提交'),
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            tooltip: '创建知识资源',
            onPressed: () => _showCreateDialog(),
          ),
        ],
      ),
      body: Consumer<KnowledgeProvider>(
        builder: (_, provider, __) {
          if (provider.resourcesLoading && provider.resources.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.resourceError.isNotEmpty &&
              provider.resources.isEmpty) {
            return ErrorView.error(
              message: provider.resourceError,
              onRetry: () => provider.listResources(),
            );
          }
          if (provider.resources.isEmpty) {
            return ErrorView.empty(
                message: '暂无提交记录',
                subtitle: '点击右上角 + 创建知识资源',
                icon: Icons.note_add_outlined);
          }
          return RefreshIndicator(
            onRefresh: () => provider.listResources(refresh: true),
            child: ListView.builder(
              padding: const EdgeInsets.all(12),
              itemCount: provider.resources.length + 1,
              itemBuilder: (context, index) {
                if (index == provider.resources.length) {
                  if (provider.resources.length < provider.resourceTotal) {
                    provider.listResources();
                    return const Padding(
                      padding: EdgeInsets.all(16),
                      child: Center(child: CircularProgressIndicator()),
                    );
                  }
                  return const SizedBox.shrink();
                }
                final r = provider.resources[index];
                return Card(
                  margin: const EdgeInsets.only(bottom: 8),
                  child: ListTile(
                    leading: CircleAvatar(
                      backgroundColor: theme.colorScheme.secondaryContainer,
                      child: Icon(Icons.article_outlined,
                          color: theme.colorScheme.onSecondaryContainer),
                    ),
                    title: Text(r.title, style: theme.textTheme.titleSmall),
                    subtitle: Text('${r.typeLabel} · ${r.resourceId}',
                        style: theme.textTheme.bodySmall),
                    trailing: FilledButton.tonal(
                      onPressed: () => _handleSubmit(provider, r.resourceId),
                      style: FilledButton.styleFrom(
                        visualDensity: VisualDensity.compact,
                        textStyle: const TextStyle(fontSize: 12),
                      ),
                      child: const Text('提交审核'),
                    ),
                  ),
                );
              },
            ),
          );
        },
      ),
    );
  }

  Future<void> _handleSubmit(
      KnowledgeProvider provider, String resourceId) async {
    final ok = await provider.submitForReview(resourceId);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '已提交审核' : '提交失败')),
      );
      if (ok) provider.listResources(refresh: true);
    }
  }

  void _showCreateDialog() {
    showDialog(
      context: context,
      builder: (ctx) => _CreateResourceDialog(),
    ).then((_) {
      if (mounted) {
        context.read<KnowledgeProvider>().listResources(refresh: true);
      }
    });
  }
}

class _CreateResourceDialog extends StatefulWidget {
  @override
  State<_CreateResourceDialog> createState() => _CreateResourceDialogState();
}

class _CreateResourceDialogState extends State<_CreateResourceDialog> {
  final _formKey = GlobalKey<FormState>();
  final _titleCtrl = TextEditingController();
  final _summaryCtrl = TextEditingController();
  final _contentCtrl = TextEditingController();
  final _tagsCtrl = TextEditingController();
  String _type = 'FAQ';
  String _scope = 'school';
  final String _roleScope = 'student';
  bool _saving = false;

  @override
  void dispose() {
    _titleCtrl.dispose();
    _summaryCtrl.dispose();
    _contentCtrl.dispose();
    _tagsCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('创建知识资源'),
      content: SizedBox(
        width: 500,
        child: Form(
          key: _formKey,
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextFormField(
                  controller: _titleCtrl,
                  decoration: const InputDecoration(
                    labelText: '标题 *',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  validator: (v) =>
                      (v == null || v.trim().isEmpty) ? '必填' : null,
                ),
                const SizedBox(height: 10),
                TextFormField(
                  controller: _summaryCtrl,
                  decoration: const InputDecoration(
                    labelText: '摘要',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  maxLines: 2,
                ),
                const SizedBox(height: 10),
                TextFormField(
                  controller: _contentCtrl,
                  decoration: const InputDecoration(
                    labelText: '正文 *',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  maxLines: 4,
                  validator: (v) =>
                      (v == null || v.trim().isEmpty) ? '必填' : null,
                ),
                const SizedBox(height: 10),
                Row(
                  children: [
                    Expanded(
                      child: DropdownButtonFormField<String>(
                        value: _type,
                        decoration: const InputDecoration(
                          labelText: '类型',
                          border: OutlineInputBorder(),
                          isDense: true,
                        ),
                        items: const [
                          DropdownMenuItem(value: 'Policy', child: Text('政策')),
                          DropdownMenuItem(value: 'Process', child: Text('流程')),
                          DropdownMenuItem(value: 'FAQ', child: Text('问答')),
                          DropdownMenuItem(value: 'Activity', child: Text('活动')),
                        ],
                        onChanged: (v) {
                          if (v != null) setState(() => _type = v);
                        },
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: DropdownButtonFormField<String>(
                        value: _scope,
                        decoration: const InputDecoration(
                          labelText: '范围',
                          border: OutlineInputBorder(),
                          isDense: true,
                        ),
                        items: const [
                          DropdownMenuItem(value: 'school', child: Text('学校')),
                          DropdownMenuItem(value: 'college', child: Text('学院')),
                          DropdownMenuItem(value: 'class', child: Text('班级')),
                        ],
                        onChanged: (v) {
                          if (v != null) setState(() => _scope = v);
                        },
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 10),
                TextFormField(
                  controller: _tagsCtrl,
                  decoration: const InputDecoration(
                    labelText: '标签（逗号分隔）',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('取消'),
        ),
        FilledButton(
          onPressed: _saving ? null : _handleSave,
          child: _saving
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2))
              : const Text('创建'),
        ),
      ],
    );
  }

  Future<void> _handleSave() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _saving = true);

    final ok = await context.read<KnowledgeProvider>().createResource({
          'title': _titleCtrl.text.trim(),
          'summary': _summaryCtrl.text.trim(),
          'content': _contentCtrl.text.trim(),
          'resource_type': _type,
          'owner_scope': _scope,
          'owner_id': '',
          'role_scope': '["$_roleScope"]',
          'tags': _tagsCtrl.text.trim(),
        });

    if (mounted) {
      if (ok) {
        Navigator.pop(context);
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('创建成功')));
      } else {
        setState(() => _saving = false);
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('创建失败')));
      }
    }
  }
}
