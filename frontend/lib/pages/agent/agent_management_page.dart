import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/agent_provider.dart';
import '../../models/models.dart';

/// 智能体管理页面（sys_admin / school_admin 可访问）
class AgentManagementPage extends StatefulWidget {
  const AgentManagementPage({super.key});

  @override
  State<AgentManagementPage> createState() => _AgentManagementPageState();
}

class _AgentManagementPageState extends State<AgentManagementPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AgentProvider>().loadAgents();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('智能体管理'),
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            tooltip: '创建智能体',
            onPressed: () => _openEditDialog(context),
          ),
        ],
      ),
      body: Consumer<AgentProvider>(
        builder: (_, provider, __) {
          if (provider.loading && provider.agents.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.error.isNotEmpty && provider.agents.isEmpty) {
            return _buildError(provider);
          }
          if (provider.agents.isEmpty) {
            return const Center(child: Text('暂无智能体'));
          }
          return RefreshIndicator(
            onRefresh: () => provider.loadAgents(),
            child: ListView.builder(
              padding: const EdgeInsets.all(12),
              itemCount: provider.agents.length,
              itemBuilder: (context, index) =>
                  _AgentCard(agent: provider.agents[index]),
            ),
          );
        },
      ),
    );
  }

  Widget _buildError(AgentProvider provider) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.error_outline,
                size: 48, color: Theme.of(context).colorScheme.error),
            const SizedBox(height: 12),
            Text(provider.error),
            const SizedBox(height: 16),
            FilledButton.icon(
              onPressed: () => provider.loadAgents(),
              icon: const Icon(Icons.refresh),
              label: const Text('重试'),
            ),
          ],
        ),
      ),
    );
  }

  /// 打开创建/编辑对话框
  void _openEditDialog(BuildContext context, {Agent? agent}) {
    showDialog(
      context: context,
      builder: (ctx) => _AgentEditDialog(agent: agent),
    ).then((_) {
      if (mounted) context.read<AgentProvider>().loadAgents();
    });
  }
}

/// 智能体卡片
class _AgentCard extends StatelessWidget {
  final Agent agent;
  const _AgentCard({required this.agent});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      clipBehavior: Clip.antiAlias,
      child: ExpansionTile(
        leading: _typeIcon(agent.agentType),
        title: Row(
          children: [
            Expanded(child: Text(agent.name,
                style: theme.textTheme.titleSmall)),
            const SizedBox(width: 8),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              decoration: BoxDecoration(
                color: agent.isActive
                    ? Colors.green.withValues(alpha: 0.1)
                    : Colors.grey.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Text(agent.statusLabel,
                  style: TextStyle(
                      fontSize: 11,
                      color: agent.isActive ? Colors.green : Colors.grey,
                      fontWeight: FontWeight.w600)),
            ),
          ],
        ),
        subtitle: Text(
          '${agent.typeLabel} · ${agent.agentId}',
          style: theme.textTheme.bodySmall,
        ),
        childrenPadding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
        expandedCrossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (agent.description.isNotEmpty) ...[
            Text(agent.description, style: theme.textTheme.bodySmall),
            const SizedBox(height: 8),
          ],
          if (agent.systemPrompt.isNotEmpty) ...[
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerLow,
                borderRadius: BorderRadius.circular(8),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('系统提示词',
                      style: theme.textTheme.labelMedium
                          ?.copyWith(fontWeight: FontWeight.w600)),
                  const SizedBox(height: 4),
                  Text(agent.systemPrompt,
                      style: const TextStyle(fontSize: 12),
                      maxLines: 5,
                      overflow: TextOverflow.ellipsis),
                ],
              ),
            ),
            const SizedBox(height: 8),
          ],
          // 参数信息
          Row(
            children: [
              _paramChip('温度', agent.temperature.toStringAsFixed(1),
                  theme),
              const SizedBox(width: 8),
              _paramChip('Token', '${agent.maxTokens}', theme),
              if (agent.modelProvider.isNotEmpty) ...[
                const SizedBox(width: 8),
                _paramChip('模型', agent.modelProvider, theme),
              ],
            ],
          ),
          const SizedBox(height: 8),
          // 操作按钮
          Row(
            mainAxisAlignment: MainAxisAlignment.end,
            children: [
              OutlinedButton.icon(
                onPressed: () => _handleToggle(context),
                icon: Icon(
                    agent.isActive ? Icons.pause : Icons.play_arrow,
                    size: 18),
                label: Text(agent.isActive ? '停用' : '启用'),
                style: OutlinedButton.styleFrom(
                  visualDensity: VisualDensity.compact,
                  textStyle: const TextStyle(fontSize: 13),
                ),
              ),
              const SizedBox(width: 8),
              OutlinedButton.icon(
                onPressed: () => _showEditDialog(context),
                icon: const Icon(Icons.edit, size: 18),
                label: const Text('编辑'),
                style: OutlinedButton.styleFrom(
                  visualDensity: VisualDensity.compact,
                  textStyle: const TextStyle(fontSize: 13),
                ),
              ),
              const SizedBox(width: 8),
              OutlinedButton.icon(
                onPressed: () => _handleDelete(context),
                icon: const Icon(Icons.delete_outline, size: 18),
                label: const Text('删除'),
                style: OutlinedButton.styleFrom(
                  foregroundColor: Theme.of(context).colorScheme.error,
                  visualDensity: VisualDensity.compact,
                  textStyle: const TextStyle(fontSize: 13),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _typeIcon(String type) {
    switch (type) {
      case 'qa':
        return const CircleAvatar(
          radius: 18,
          backgroundColor: Color(0xFFE3F2FD),
          child: Icon(Icons.chat, size: 18, color: Color(0xFF1565C0)),
        );
      case 'policy':
        return const CircleAvatar(
          radius: 18,
          backgroundColor: Color(0xFFFFF3E0),
          child: Icon(Icons.gavel, size: 18, color: Color(0xFFE65100)),
        );
      case 'emotion':
        return const CircleAvatar(
          radius: 18,
          backgroundColor: Color(0xFFFCE4EC),
          child: Icon(Icons.favorite_border, size: 18, color: Color(0xFFC62828)),
        );
      default:
        return const CircleAvatar(
          radius: 18,
          backgroundColor: Color(0xFFF3E5F5),
          child: Icon(Icons.smart_toy, size: 18, color: Color(0xFF7B1FA2)),
        );
    }
  }

  Widget _paramChip(String label, String value, ThemeData theme) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text('$label: $value', style: const TextStyle(fontSize: 11)),
    );
  }

  Future<void> _handleToggle(BuildContext context) async {
    final provider = context.read<AgentProvider>();
    final ok = await provider.toggleStatus(agent);
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '操作成功' : '操作失败')),
      );
    }
  }

  void _showEditDialog(BuildContext context) {
    final pageState = context
        .findAncestorStateOfType<_AgentManagementPageState>();
    pageState?._openEditDialog(context, agent: agent);
  }

  Future<void> _handleDelete(BuildContext context) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('删除智能体'),
        content: Text('确定要删除「${agent.name}」吗？此操作不可撤销。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: TextButton.styleFrom(
              foregroundColor: Theme.of(context).colorScheme.error,
            ),
            child: const Text('删除'),
          ),
        ],
      ),
    );

    if (confirm == true && context.mounted) {
      final ok = await context.read<AgentProvider>().delete(agent.agentId);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(ok ? '删除成功' : '删除失败')),
        );
      }
    }
  }
}

/// 创建/编辑智能体对话框
class _AgentEditDialog extends StatefulWidget {
  final Agent? agent;
  const _AgentEditDialog({this.agent});

  @override
  State<_AgentEditDialog> createState() => _AgentEditDialogState();
}

class _AgentEditDialogState extends State<_AgentEditDialog> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _idCtrl;
  late final TextEditingController _nameCtrl;
  late final TextEditingController _descCtrl;
  late final TextEditingController _promptCtrl;
  late final TextEditingController _providerCtrl;
  String _type = 'qa';
  bool _saving = false;

  bool get _isEdit => widget.agent != null;

  @override
  void initState() {
    super.initState();
    final a = widget.agent;
    _idCtrl = TextEditingController(text: a?.agentId ?? '');
    _nameCtrl = TextEditingController(text: a?.name ?? '');
    _descCtrl = TextEditingController(text: a?.description ?? '');
    _promptCtrl = TextEditingController(text: a?.systemPrompt ?? '');
    _providerCtrl = TextEditingController(text: a?.modelProvider ?? '');
    if (a != null) _type = a.agentType;
  }

  @override
  void dispose() {
    _idCtrl.dispose();
    _nameCtrl.dispose();
    _descCtrl.dispose();
    _promptCtrl.dispose();
    _providerCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(_isEdit ? '编辑智能体' : '创建智能体'),
      content: SizedBox(
        width: 500,
        child: Form(
          key: _formKey,
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                TextFormField(
                  controller: _idCtrl,
                  decoration: const InputDecoration(
                    labelText: '智能体 ID *',
                    hintText: '如 qa-default',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  enabled: !_isEdit,
                  validator: (v) =>
                      (v == null || v.trim().isEmpty) ? '必填' : null,
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _nameCtrl,
                  decoration: const InputDecoration(
                    labelText: '名称 *',
                    hintText: '如 通用问答助手',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  validator: (v) =>
                      (v == null || v.trim().isEmpty) ? '必填' : null,
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _descCtrl,
                  decoration: const InputDecoration(
                    labelText: '描述',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  maxLines: 2,
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  value: _type,
                  decoration: const InputDecoration(
                    labelText: '类型',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  items: const [
                    DropdownMenuItem(value: 'qa', child: Text('通用问答')),
                    DropdownMenuItem(value: 'policy', child: Text('政策解读')),
                    DropdownMenuItem(value: 'emotion', child: Text('情感分析')),
                    DropdownMenuItem(value: 'custom', child: Text('自定义')),
                  ],
                  onChanged: (v) {
                    if (v != null) setState(() => _type = v);
                  },
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _promptCtrl,
                  decoration: const InputDecoration(
                    labelText: '系统提示词',
                    hintText: '定义智能体的行为和角色...',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  maxLines: 4,
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _providerCtrl,
                  decoration: const InputDecoration(
                    labelText: '模型提供商',
                    hintText: 'deepseek / zhipu（留空使用默认）',
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
              : Text(_isEdit ? '保存' : '创建'),
        ),
      ],
    );
  }

  Future<void> _handleSave() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _saving = true);
    final provider = context.read<AgentProvider>();

    bool ok;
    if (_isEdit) {
      ok = await provider.update(_idCtrl.text.trim(), {
        'name': _nameCtrl.text.trim(),
        'description': _descCtrl.text.trim(),
        'agent_type': _type,
        'system_prompt': _promptCtrl.text.trim(),
        'model_provider': _providerCtrl.text.trim(),
      });
    } else {
      ok = await provider.create(AgentSaveRequest(
        agentId: _idCtrl.text.trim(),
        name: _nameCtrl.text.trim(),
        description: _descCtrl.text.trim(),
        agentType: _type,
        systemPrompt: _promptCtrl.text.trim(),
        modelProvider: _providerCtrl.text.trim(),
      ));
    }

    if (context.mounted) {
      if (ok) {
        Navigator.pop(context);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(_isEdit ? '保存成功' : '创建成功')),
        );
      } else {
        setState(() => _saving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(provider.error)),
        );
      }
    }
  }
}
