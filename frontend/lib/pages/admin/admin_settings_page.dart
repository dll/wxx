import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/admin_provider.dart';
import '../../models/models.dart';
import '../../widgets/error_view.dart';

/// 系统配置页面（sys_admin 独占）
class AdminSettingsPage extends StatefulWidget {
  const AdminSettingsPage({super.key});

  @override
  State<AdminSettingsPage> createState() => _AdminSettingsPageState();
}

class _AdminSettingsPageState extends State<AdminSettingsPage> {
  final Map<String, String> _edits = {};

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AdminProvider>().fetchSettings();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('系统配置'),
        actions: [
          IconButton(
            icon: const Icon(Icons.save_outlined),
            tooltip: '保存全部',
            onPressed: () => _saveAll(),
          ),
        ],
      ),
      body: Consumer<AdminProvider>(
        builder: (_, provider, __) {
          if (provider.settingsLoading && provider.settings.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.settings.isEmpty) {
            return ErrorView.empty(
                message: '暂无配置项', icon: Icons.settings_outlined);
          }
          return ListView.builder(
            padding: const EdgeInsets.all(12),
            itemCount: provider.settings.length,
            itemBuilder: (context, index) {
              final setting = provider.settings[index];
              return _SettingTile(
                setting: setting,
                edits: _edits,
              );
            },
          );
        },
      ),
    );
  }

  void _saveAll() {
    final provider = context.read<AdminProvider>();
    final map = <String, String>{};
    for (final s in provider.settings) {
      map[s.key] = _edits[s.key] ?? s.value;
    }
    provider.updateSettings(map).then((ok) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(ok ? '保存成功' : '保存失败')),
        );
        if (ok) _edits.clear();
      }
    });
  }
}

class _SettingTile extends StatefulWidget {
  final SystemSetting setting;
  final Map<String, String> edits;
  const _SettingTile({required this.setting, required this.edits});

  @override
  State<_SettingTile> createState() => _SettingTileState();
}

class _SettingTileState extends State<_SettingTile> {
  late TextEditingController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(
        text: widget.edits[widget.setting.key] ?? widget.setting.value);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(widget.setting.key, style: theme.textTheme.titleSmall),
            if (widget.setting.description.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(widget.setting.description,
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
              ),
            const SizedBox(height: 8),
            TextField(
              controller: _ctrl,
              decoration: const InputDecoration(
                border: OutlineInputBorder(),
                isDense: true,
              ),
              onChanged: (v) {
                widget.edits[widget.setting.key] = v;
              },
            ),
          ],
        ),
      ),
    );
  }
}
