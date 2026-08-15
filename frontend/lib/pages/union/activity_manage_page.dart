import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/health_provider.dart';

/// 学生会 - 活动报名管理中心（活动报名闭环）
///
/// 在既有 AI 策划/海报能力外，提供"活动管理"闭环：
/// 1. 列出活动（含报名人数/容量、报名状态、截止时间）
/// 2. 新建活动（标题/时间/地点/容量/报名截止/分类）
/// 3. 查看某活动报名热度，便于后续复盘
/// 数据复用后端 health/activities 接口（真实落库）。
class UnionActivityManagePage extends StatefulWidget {
  const UnionActivityManagePage({super.key});
  @override
  State<UnionActivityManagePage> createState() => _UnionActivityManagePageState();
}

class _UnionActivityManagePageState extends State<UnionActivityManagePage> {
  String _category = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<HealthProvider>().fetchActivities();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<HealthProvider>();
    final theme = Theme.of(context);
    final acts = provider.activities;
    return Scaffold(
      appBar: AppBar(title: const Text('活动报名管理')),
      floatingActionButton: Column(
        mainAxisAlignment: MainAxisAlignment.end,
        crossAxisAlignment: CrossAxisAlignment.end,
        mainAxisSize: MainAxisSize.min,
        children: [
          FloatingActionButton.extended(
            heroTag: 'import_ext',
            onPressed: () => _showImportDialog(context, provider),
            icon: const Icon(Icons.file_download_outlined),
            label: const Text('导入外部活动'),
          ),
          const SizedBox(height: 10),
          FloatingActionButton(
            heroTag: 'add_act',
            onPressed: () => _showCreateDialog(context, provider),
            child: const Icon(Icons.add),
          ),
        ],
      ),
      body: Column(
        children: [
          _buildSummary(provider, theme),
          _buildCategoryFilter(theme),
          Expanded(
            child: acts.isEmpty
                ? Center(
                    child: Text('暂无活动',
                        style: TextStyle(color: theme.colorScheme.onSurfaceVariant)))
                : RefreshIndicator(
                    onRefresh: () => provider.fetchActivities(),
                    child: ListView.builder(
                      padding: const EdgeInsets.fromLTRB(12, 4, 12, 80),
                      itemCount: acts.length,
                      itemBuilder: (_, i) => _buildCard(theme, provider, acts[i]),
                    ),
                  ),
          ),
        ],
      ),
    );
  }

  Widget _buildSummary(HealthProvider provider, ThemeData theme) {
    final acts = provider.activities;
    final active = acts.where((a) => a.status == 'active').length;
    final totalSignups = acts.fold<int>(0, (s, a) => s + a.signupCount);
    return Container(
      margin: const EdgeInsets.all(12),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer.withOpacity(0.25),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Row(
        children: [
          _sumItem(theme, '${acts.length}', '全部活动'),
          _sumItem(theme, '$active', '进行中'),
          _sumItem(theme, '$totalSignups', '累计报名'),
        ],
      ),
    );
  }

  Widget _sumItem(ThemeData theme, String v, String l) => Expanded(
        child: Column(children: [
          Text(v, style: theme.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800)),
          Text(l, style: theme.textTheme.labelSmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
        ]),
      );

  Widget _buildCategoryFilter(ThemeData theme) {
    final cats = [
      ('', '全部'),
      ('sports', '体育'),
      ('culture', '文体'),
      ('other', '其他'),
    ];
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Row(
        children: cats.map((c) {
          final sel = _category == c.$1;
          return Expanded(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 2),
              child: ChoiceChip(
                label: Text(c.$2),
                selected: sel,
                onSelected: (_) => setState(() => _category = c.$1),
              ),
            ),
          );
        }).toList(),
      ),
    );
  }

  Widget _buildCard(ThemeData theme, HealthProvider provider, HealthActivity a) {
    final isActive = a.status == 'active';
    final capText = a.capacity > 0 ? '/${a.capacity}' : '';
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(a.title,
                      style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700),
                      maxLines: 2, overflow: TextOverflow.ellipsis),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                  decoration: BoxDecoration(
                    color: (isActive ? Colors.green : Colors.grey).withOpacity(0.12),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(isActive ? '进行中' : (a.status == 'closed' ? '已截止' : '已结束'),
                      style: TextStyle(
                          fontSize: 12,
                          color: isActive ? Colors.green : Colors.grey,
                          fontWeight: FontWeight.w600)),
                ),
              ],
            ),
            if (a.description.isNotEmpty) ...[
              const SizedBox(height: 6),
              Text(a.description, maxLines: 2, overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
            ],
            const SizedBox(height: 10),
            Row(children: [
              Icon(Icons.location_on_outlined, size: 16, color: theme.colorScheme.primary),
              const SizedBox(width: 4),
              Expanded(child: Text(a.venue.isNotEmpty ? a.venue : '待定', style: theme.textTheme.bodySmall)),
            ]),
            const SizedBox(height: 4),
            Row(children: [
              Icon(Icons.groups_outlined, size: 16, color: theme.colorScheme.secondary),
              const SizedBox(width: 4),
              Expanded(
                child: Text('报名 ${a.signupCount}$capText',
                    style: theme.textTheme.bodySmall?.copyWith(fontWeight: FontWeight.w600)),
              ),
              Icon(Icons.schedule, size: 16, color: theme.colorScheme.outline),
              const SizedBox(width: 4),
              Text(a.startAt.isNotEmpty ? a.startAt : '时间待定', style: theme.textTheme.bodySmall),
            ]),
            if (isActive) ...[
              const SizedBox(height: 8),
              Row(children: [
                // 报名人数统计（参与/有效）
                Text('${a.signupCount} 人报名',
                    style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.primary, fontWeight: FontWeight.w700)),
                const Spacer(),
                TextButton(
                  onPressed: () => _confirmEndActivity(context, provider, a),
                  child: const Text('结束活动'),
                ),
              ]),
            ],
          ],
        ),
      ),
    );
  }

  /// 确认结束活动：active → closed（停止报名，活动结束）
  void _confirmEndActivity(BuildContext context, HealthProvider provider, HealthActivity a) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('结束活动？'),
        content: Text('「${a.title}」将停止报名，标记为已结束。报名人数 ${a.signupCount} 已保留，可继续查看统计。'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () {
              provider.updateActivityStatus(a.activityId, 'closed');
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('活动已结束')));
            },
            child: const Text('确认结束'),
          ),
        ],
      ),
    );
  }

  /// 导入外部活动（粘贴 QQ/社团文案）→ 结构化提取 → 预填新建表单
  void _showImportDialog(BuildContext context, HealthProvider provider) {
    final textCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('导入外部活动'),
        content: Column(mainAxisSize: MainAxisSize.min, children: [
          TextField(
            controller: textCtrl,
            maxLines: 5,
            decoration: const InputDecoration(
              hintText: '粘贴 QQ 群公告 / 社团网站的活动文案\n\n例如：\n“XX杯篮球赛报名开始：5月10日下午3点在东操场…”',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 8),
          Text('粘贴后点击“提取并导入”，将自动识别 标题/时间/地点 并预填活动表单。',
              style: Theme.of(ctx).textTheme.bodySmall?.copyWith(color: Theme.of(ctx).colorScheme.onSurfaceVariant)),
        ]),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () {
              final parsed = _parseExternalActivity(textCtrl.text);
              Navigator.pop(ctx);
              _showCreateDialog(context, provider, prefill: parsed);
            },
            child: const Text('提取并导入'),
          ),
        ],
      ),
    );
  }

  /// 简单结构化提取外部活动文案（正则：标题行/时间/地点）；缺字段留空由人工补
  Map<String, String> _parseExternalActivity(String raw) {
    final t = raw.trim();
    // 标题：第一行非空，去掉日期/报名等前缀
    var title = '';
    final lines = t.split(RegExp(r'[\r\n]')).map((e) => e.trim()).where((e) => e.isNotEmpty).toList();
    for (final l in lines) {
      if (l.length >= 3 && !l.contains('报名') && !RegExp(r'^[\d:年月日分时]+$').hasMatch(l)) {
        title = l;
        break;
      }
    }
    // 时间：识别 数字月日 或 下午X点/时分
    final timeRe = RegExp(r'(\d{1,2})月(\d{1,2})日[^，。]*?([0-9]{1,2})[:：点]([0-9]{0,2}分?)');
    var startAt = '';
    final mt = timeRe.firstMatch(t);
    if (mt != null) {
      startAt = '${mt.group(1)!.padLeft(2, '0')}-${mt.group(2)!.padLeft(2, '0')} ${mt.group(3)}:${(mt.group(4) ?? '').replaceAll('分', '').padLeft(2, '0')}';
    } else {
      final m2 = RegExp(r'(\d{1,2})月(\d{1,2})日').firstMatch(t);
      if (m2 != null) startAt = '${m2.group(1)!.padLeft(2, '0')}-${m2.group(2)!.padLeft(2, '0')}';
    }
    // 地点：找常见地点词，取其后方的 在/于/@ 分隔段
    var venue = '';
    for (final kw in ['操场', '体育馆', '教室', '报告厅', '会议室', '广场', '中心', '馆', '楼']) {
      final idx = t.indexOf(kw);
      if (idx <= 0) continue;
      final before = t.substring(0, idx);
      var start = before.lastIndexOf('在');
      final s2 = before.lastIndexOf('于');
      final s3 = before.lastIndexOf('@');
      if (s2 > start) start = s2;
      if (s3 > start) start = s3;
      final seg = t.substring(start + 1, idx + kw.length).trim();
      if (seg.length >= 2 && seg.length <= 24) { venue = seg; break; }
    }
    return {'title': title, 'description': t, 'start_at': startAt.trim(), 'venue': venue.trim()};
  }

  void _showCreateDialog(BuildContext context, HealthProvider provider, {Map<String, String>? prefill}) {
    final titleCtrl = TextEditingController(text: prefill?['title'] ?? '');
    final descCtrl = TextEditingController(text: prefill?['description'] ?? '');
    final venueCtrl = TextEditingController(text: prefill?['venue'] ?? '');
    final startAtCtrl = TextEditingController(text: prefill?['start_at'] ?? '');
    final capacityCtrl = TextEditingController();
    String category = 'sports';
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('新建活动'),
        content: SingleChildScrollView(
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            TextField(controller: titleCtrl, decoration: const InputDecoration(labelText: '活动名称 *')),
            const SizedBox(height: 12),
            TextField(controller: descCtrl, decoration: const InputDecoration(labelText: '活动介绍'), maxLines: 3),
            const SizedBox(height: 12),
            TextField(controller: venueCtrl, decoration: const InputDecoration(labelText: '地点')),
            const SizedBox(height: 12),
            TextField(controller: startAtCtrl, decoration: const InputDecoration(labelText: '开始时间（如 09-05 14:00）')),
            const SizedBox(height: 12),
            TextField(controller: capacityCtrl,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(labelText: '名额上限（0=不限）')),
            const SizedBox(height: 12),
            DropdownButtonFormField<String>(
              value: category,
              decoration: const InputDecoration(labelText: '分类'),
              items: const [
                DropdownMenuItem(value: 'sports', child: Text('体育')),
                DropdownMenuItem(value: 'culture', child: Text('文体')),
                DropdownMenuItem(value: 'other', child: Text('其他')),
              ],
              onChanged: (v) => category = v ?? 'sports',
            ),
          ]),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () {
              if (titleCtrl.text.trim().isEmpty) {
                ScaffoldMessenger.of(ctx).showSnackBar(const SnackBar(content: Text('请填写活动名称')));
                return;
              }
              provider.createActivity({
                'title': titleCtrl.text.trim(),
                'description': descCtrl.text.trim(),
                'venue': venueCtrl.text.trim(),
                'start_at': startAtCtrl.text.trim(),
                'category': category,
                'capacity': int.tryParse(capacityCtrl.text.trim()) ?? 0,
              });
              Navigator.pop(ctx);
            },
            child: const Text('发布'),
          ),
        ],
      ),
    );
  }
}
