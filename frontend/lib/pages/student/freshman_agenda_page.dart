import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'dart:convert';

/// 新生开学待办清单（本地持久化）
///
/// 按入学时间线整理的"今天该做什么"，覆盖报到→体检→军训→选课→领教材。
/// 每完成一项自动打勾 + 进度激励，形成新生"每天打开的理由"。
class FreshmanAgendaPage extends StatefulWidget {
  const FreshmanAgendaPage({super.key});

  @override
  State<FreshmanAgendaPage> createState() => _FreshmanAgendaPageState();
}

class _FreshmanAgendaPageState extends State<FreshmanAgendaPage> {
  static const _agendaKey = 'freshman_agenda_state_v1';

  // 报到节点（可来自后端，后端不可用时用内置清单兜底）
  static const _defaultAgenda = [
    _AgendaItem('报到注册', '带齐身份证、录取通知书，到学院报到点', Icons.app_registration, '报到'),
    _AgendaItem('领取校园卡', '报到后凭有效证件领取校园一卡通', Icons.credit_card, '报到'),
    _AgendaItem('入住宿舍', '按分配床位入住，办理寝室入住登记', Icons.single_bed, '报到'),
    _AgendaItem('新生体检', '根据学院通知到校医院完成入学体检', Icons.favorite, '体检'),
    _AgendaItem('领取军训服', '按通知时间地点领取军训服装', Icons.military_tech, '军训'),
    _AgendaItem('军训', '参加入学军训，注意防晒与补水', Icons.directions_run, '军训'),
    _AgendaItem('新生选课', '登录选课系统完成新生课程初选', Icons.edit_calendar, '选课'),
    _AgendaItem('领取教材', '按班级通知领取本学期教材', Icons.menu_book, '学习'),
  ];

  List<_AgendaItem> get _defaults => List.of(_defaultAgenda);

  // 从后端读取的动态节点（可空，优先使用；否则回落内置）
  List<Map<String, dynamic>>? _serverSteps;
  bool _loadingServer = true;

  @override
  void initState() {
    super.initState();
    _loadServerSteps();
  }

  Future<void> _loadServerSteps() async {
    try {
      final resp = await ApiService().get(ApiConfig.freshmenGuide);
      if (mounted &&
          resp.statusCode == 200 &&
          resp.data is Map<String, dynamic>) {
        final data = resp.data as Map<String, dynamic>;
        final steps = (data['steps'] as List?) ?? const [];
        setState(() {
          _serverSteps = steps.isNotEmpty
              ? steps.cast<Map<String, dynamic>>()
              : null;
          _loadingServer = false;
        });
        return;
      }
    } catch (_) {
      // 后端不可用时静默回落内置清单
    }
    if (mounted) {
      setState(() => _loadingServer = false);
    }
  }

  Future<Map<String, bool>> _loadState() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(_agendaKey);
    if (raw == null || raw.isEmpty) return {};
    try {
      final map = (jsonDecode(raw) as Map).cast<String, dynamic>();
      return map.map((k, v) => MapEntry(k, v == true));
    } catch (_) {
      return {};
    }
  }

  Future<void> _saveState(Map<String, bool> state) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_agendaKey, jsonEncode(state));
  }

  Future<void> _toggle(int index) async {
    final state = await _loadState();
    final items = _combinedItems();
    final key = items[index].key;
    state[key] = !(state[key] ?? false);
    await _saveState(state);
    if (mounted) setState(() {});
  }

  List<_AgendaItem> _combinedItems() {
    final server = _serverSteps;
    if (server != null && server.isNotEmpty) {
      // 优先使用后端流程节点，映射为待办项
      return [
        for (final s in server)
          _AgendaItem(
            (s['title'] ?? s['name'] ?? '未命名').toString(),
            (s['summary'] ?? s['description'] ?? s['notes'] ?? '').toString(),
            Icons.edit,
            (s['category'] ?? '报到').toString(),
          )
      ];
    }
    return _defaults;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('我的开学待办')),
      body: FutureBuilder<Map<String, bool>>(
        future: _loadState(),
        builder: (context, snap) {
          final state = snap.data ?? {};
          final items = _combinedItems();
          final done = items.where((i) => state[i.key] ?? false).length;
          final progress = items.isEmpty ? 0.0 : done / items.length;

          if (_loadingServer) {
            return const Center(child: CircularProgressIndicator());
          }

          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              // 进度激励卡
              Container(
                padding: const EdgeInsets.all(20),
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    colors: [
                      theme.colorScheme.primary.withOpacity(0.14),
                      theme.colorScheme.tertiary.withOpacity(0.10),
                    ],
                  ),
                  borderRadius: BorderRadius.circular(18),
                  border: Border.all(
                    color: theme.colorScheme.primary.withOpacity(0.18),
                  ),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Icon(Icons.celebration,
                            color: theme.colorScheme.primary, size: 24),
                        const SizedBox(width: 10),
                        Expanded(
                          child: Text(
                            '入学进度',
                            style: theme.textTheme.titleMedium
                                ?.copyWith(fontWeight: FontWeight.w800),
                          ),
                        ),
                        Text(
                          '$done / ${items.length}',
                          style: theme.textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.w800,
                            color: theme.colorScheme.primary,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    ClipRRect(
                      borderRadius: BorderRadius.circular(8),
                      child: LinearProgressIndicator(
                        value: progress,
                        minHeight: 10,
                        backgroundColor:
                            theme.colorScheme.surface.withOpacity(0.6),
                      ),
                    ),
                    const SizedBox(height: 10),
                    Text(
                      progress >= 1.0
                          ? '🎉 全部完成！你已经是正式的大学生啦'
                          : progress >= 0.5
                              ? '进度过半，继续保持！'
                              : '每完成一步，离大学新生活更近一步',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 16),
              Text('待办事项', style: theme.textTheme.titleMedium),
              const SizedBox(height: 8),
              if (items.isEmpty)
                Center(
                  child: Padding(
                    padding: const EdgeInsets.all(32),
                    child: Text('暂无待办',
                        style: TextStyle(
                            color: theme.colorScheme.outline)),
                  ),
                )
              else
                ...items.asMap().entries.map((entry) {
                  final i = entry.key;
                  final item = entry.value;
                  final isDone = state[item.key] ?? false;
                  return Card(
                    elevation: 0,
                    margin: const EdgeInsets.only(bottom: 10),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(14),
                      side: BorderSide(
                        color: isDone
                            ? theme.colorScheme.primary.withOpacity(0.4)
                            : theme.colorScheme.outlineVariant,
                      ),
                    ),
                    child: InkWell(
                      borderRadius: BorderRadius.circular(14),
                      onTap: () => _toggle(i),
                      child: Padding(
                        padding: const EdgeInsets.all(14),
                        child: Row(
                          children: [
                            AnimatedContainer(
                              duration: const Duration(milliseconds: 200),
                              width: 28,
                              height: 28,
                              decoration: BoxDecoration(
                                shape: BoxShape.circle,
                                color: isDone
                                    ? theme.colorScheme.primary
                                    : Colors.transparent,
                                border: Border.all(
                                  color: isDone
                                      ? theme.colorScheme.primary
                                      : theme.colorScheme.outline,
                                  width: 2,
                                ),
                              ),
                              child: isDone
                                  ? const Icon(Icons.check,
                                      color: Colors.white, size: 18)
                                  : null,
                            ),
                            const SizedBox(width: 14),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    item.title,
                                    style: theme.textTheme.bodyLarge?.copyWith(
                                      fontWeight: FontWeight.w600,
                                      decoration: isDone
                                          ? TextDecoration.lineThrough
                                          : null,
                                      color: isDone
                                          ? theme.colorScheme.outline
                                          : theme.colorScheme.onSurface,
                                    ),
                                  ),
                                  if (item.desc.isNotEmpty) ...[
                                    const SizedBox(height: 4),
                                    Text(
                                      item.desc,
                                      maxLines: 2,
                                      overflow: TextOverflow.ellipsis,
                                      style: theme.textTheme.bodySmall
                                          ?.copyWith(
                                        color:
                                            theme.colorScheme.onSurfaceVariant,
                                      ),
                                    ),
                                  ],
                                ],
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  );
                }),
              const SizedBox(height: 16),
              // 关联入口：报到流程 + 校园导航
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: () => context.go('/student/freshmen-guide'),
                      icon: const Icon(Icons.app_registration, size: 18),
                      label: const Text('报到指南'),
                    ),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: () => context.go('/campus?v=map'),
                      icon: const Icon(Icons.map_outlined, size: 18),
                      label: const Text('校园导航'),
                    ),
                  ),
                ],
              ),
            ],
          );
        },
      ),
    );
  }
}

class _AgendaItem {
  final String title;
  final String desc;
  final IconData icon;
  final String category;
  const _AgendaItem(this.title, this.desc, this.icon, this.category);

  String get key => 'agenda_$title';
}
