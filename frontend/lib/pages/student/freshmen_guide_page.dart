import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:go_router/go_router.dart';
import '../../config/api_config.dart';
import '../../services/api_service.dart';
import '../../widgets/error_view.dart';

class FreshmenGuidePage extends StatefulWidget {
  const FreshmenGuidePage({super.key});

  @override
  State<FreshmenGuidePage> createState() => _FreshmenGuidePageState();
}

class _FreshmenGuidePageState extends State<FreshmenGuidePage> {
  Map<String, dynamic>? _data;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final resp = await ApiService().get(ApiConfig.freshmenGuide);
      if (!mounted) return;
      if (resp.statusCode == 200 && resp.data is Map<String, dynamic>) {
        setState(() => _data = resp.data as Map<String, dynamic>);
      } else {
        setState(() => _error = '新生指南加载失败，请稍后重试');
      }
    } catch (_) {
      if (mounted) {
        setState(() => _error = '网络异常，请检查网络后重试');
      }
    } finally {
      if (mounted) {
        setState(() => _loading = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('新生指南')),
      body: RefreshIndicator(
        onRefresh: _load,
        child: _buildBody(theme),
      ),
    );
  }

  Widget _buildBody(ThemeData theme) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null) {
      return ErrorView.error(message: _error!, onRetry: _load);
    }
    final data = _data;
    if (data == null) {
      return ErrorView.empty(
        message: '暂无新生指南内容',
        icon: Icons.school_outlined,
      );
    }

    final guide = _map(data['guide']);
    final handbook = _map(data['handbook']);
    final zzsb = _map(data['zzsb']);
    final rawSteps = (data['steps'] as List?) ?? const [];

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _buildHeader(theme, guide),
        const SizedBox(height: 12),
        _buildActionRow(theme),
        if (guide != null) ...[
          const SizedBox(height: 16),
          _buildMarkdownSection(
            theme,
            title: guide['title'] ?? '新生入学指南',
            icon: Icons.menu_book_outlined,
            summary: guide['summary'] ?? '',
            content: guide['content'] ?? '',
          ),
        ],
        if (rawSteps.isNotEmpty) ...[
          const SizedBox(height: 16),
          _buildStepsSection(theme, rawSteps),
        ],
        if (zzsb != null) ...[
          const SizedBox(height: 16),
          _buildMarkdownSection(
            theme,
            title: zzsb['title'] ?? '专升本新生入学须知',
            icon: Icons.import_contacts_outlined,
            summary: zzsb['summary'] ?? '',
            content: zzsb['content'] ?? '',
          ),
        ],
        if (handbook != null) ...[
          const SizedBox(height: 16),
          _buildMarkdownSection(
            theme,
            title: handbook['title'] ?? '2025年学生手册要点',
            icon: Icons.policy_outlined,
            summary: handbook['summary'] ?? '',
            content: handbook['content'] ?? '',
          ),
        ],
        if ((data['source_files'] as List?)?.isNotEmpty ?? false) ...[
          const SizedBox(height: 16),
          _buildSourceFiles(theme, data['source_files'] as List),
        ],
        const SizedBox(height: 24),
      ],
    );
  }

  Map<String, dynamic>? _map(dynamic value) {
    if (value is Map<String, dynamic>) return value;
    if (value is Map) return value.cast<String, dynamic>();
    return null;
  }

  Widget _buildHeader(ThemeData theme, Map<String, dynamic>? guide) {
    final title = guide?['title'] ?? '2026级新生入学指南';
    final summary = guide?['summary'] ?? '报到、材料、档案、交通、校园生活与学生手册一站式整理';
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFF00695C), Color(0xFF2E7D32)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.16),
              borderRadius: BorderRadius.circular(10),
            ),
            child: const Icon(
              Icons.school_outlined,
              color: Colors.white,
              size: 30,
            ),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 20,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 6),
                Text(
                  summary,
                  style: TextStyle(
                    color: Colors.white.withOpacity(0.9),
                    fontSize: 13,
                    height: 1.5,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildActionRow(ThemeData theme) {
    return Row(
      children: [
        Expanded(
          child: FilledButton.icon(
            onPressed: () => context.go('/enrollment'),
            icon: const Icon(Icons.assignment_outlined, size: 18),
            label: const Text('报到流程'),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: OutlinedButton.icon(
            onPressed: () => context.go('/campus'),
            icon: const Icon(Icons.map_outlined, size: 18),
            label: const Text('校园导航'),
          ),
        ),
      ],
    );
  }

  Widget _buildMarkdownSection(
    ThemeData theme, {
    required String title,
    required IconData icon,
    required String summary,
    required String content,
  }) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: theme.colorScheme.outlineVariant.withOpacity(0.6),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 20, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  title,
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w700),
                ),
              ),
            ],
          ),
          if (summary.isNotEmpty) ...[
            const SizedBox(height: 6),
            Text(
              summary,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
          const Divider(height: 20),
          MarkdownBody(
            data: content,
            selectable: true,
            styleSheet: MarkdownStyleSheet(
              p: theme.textTheme.bodyMedium?.copyWith(height: 1.65),
              h1: theme.textTheme.titleLarge
                  ?.copyWith(fontWeight: FontWeight.w700),
              h2: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w700),
              h3: theme.textTheme.titleSmall
                  ?.copyWith(fontWeight: FontWeight.w700),
              listBullet: theme.textTheme.bodyMedium,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildStepsSection(ThemeData theme, List rawSteps) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: theme.colorScheme.outlineVariant.withOpacity(0.6),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.format_list_numbered, size: 20, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Text(
                '新生报到流程步骤',
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.w700),
              ),
            ],
          ),
          const SizedBox(height: 8),
          for (final raw in rawSteps)
            _buildStepTile(theme, _map(raw) ?? const {}),
        ],
      ),
    );
  }

  Widget _buildStepTile(ThemeData theme, Map<String, dynamic> step) {
    final title = step['title'] ?? '';
    final location = step['location'] ?? '';
    final deadline = step['deadline'] ?? '';
    final materials = _decodeList(step['materials']);
    final notes = step['notes'] ?? '';
    final contact = step['contact'] ?? '';
    final phone = step['phone'] ?? '';
    final officeHours = step['office_hours'] ?? '';
    final entryUrl = step['entry_url'] ?? '';

    return ExpansionTile(
      tilePadding: EdgeInsets.zero,
      childrenPadding: const EdgeInsets.only(bottom: 8),
      leading: CircleAvatar(
        radius: 16,
        backgroundColor: theme.colorScheme.primaryContainer,
        child: Text(
          '${step['step_order'] ?? step['step'] ?? ''}',
          style: TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w700,
            color: theme.colorScheme.onPrimaryContainer,
          ),
        ),
      ),
      title: Text(
        title,
        style: theme.textTheme.bodyMedium?.copyWith(
          fontWeight: FontWeight.w600,
        ),
      ),
      subtitle: Text(
        [if (deadline.isNotEmpty) deadline, if (location.isNotEmpty) location]
            .join(' · '),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.onSurfaceVariant,
        ),
      ),
      children: [
        if (notes.isNotEmpty) _buildStepRow(theme, Icons.info_outline, notes),
        if (materials.isNotEmpty)
          _buildStepRow(
            theme,
            Icons.description_outlined,
            '材料：${materials.join('、')}',
          ),
        if (location.isNotEmpty)
          _buildStepRow(theme, Icons.location_on_outlined, location),
        if (deadline.isNotEmpty)
          _buildStepRow(theme, Icons.schedule, '截止：$deadline'),
        if (contact.isNotEmpty)
          _buildStepRow(theme, Icons.person_outline, contact),
        if (phone.isNotEmpty) _buildStepRow(theme, Icons.phone_outlined, phone),
        if (officeHours.isNotEmpty)
          _buildStepRow(theme, Icons.access_time, officeHours),
        if (entryUrl.isNotEmpty)
          _buildStepRow(theme, Icons.open_in_new, entryUrl),
      ],
    );
  }

  Widget _buildStepRow(ThemeData theme, IconData icon, String text) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 14, color: theme.colorScheme.onSurfaceVariant),
          const SizedBox(width: 6),
          Expanded(
            child: Text(
              text,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
                height: 1.4,
              ),
            ),
          ),
        ],
      ),
    );
  }

  List<String> _decodeList(dynamic raw) {
    if (raw is List) {
      return raw.map((e) => '$e').toList();
    }
    if (raw is String && raw.isNotEmpty && raw != '[]') {
      try {
        final decoded = jsonDecode(raw);
        if (decoded is List) {
          return decoded.map((e) => '$e').toList();
        }
      } catch (_) {}
    }
    return const [];
  }

  Widget _buildSourceFiles(ThemeData theme, List files) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.4),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            '官方资料',
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: files.map((raw) {
              final file = _map(raw) ?? const {};
              final title = file['title'] ?? '';
              final note = file['note'] ?? '';
              final path = file['path'] ?? '';
              return Tooltip(
                message: path,
                child: Chip(
                  avatar: const Icon(Icons.description_outlined, size: 16),
                  label: Text(
                    [title, if (note.isNotEmpty) note].join(' · '),
                    style: const TextStyle(fontSize: 12),
                  ),
                ),
              );
            }).toList(),
          ),
        ],
      ),
    );
  }
}
