import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:fl_chart/fl_chart.dart';
import '../../../providers/health_provider.dart';
import '../../../utils/storage.dart';

/// 学生「身体健康」页：日常记录 / 健身活动 / 基本信息 / 体检记录 / 病历记录
class HealthPage extends StatefulWidget {
  const HealthPage({super.key});

  @override
  State<HealthPage> createState() => _HealthPageState();
}

class _HealthPageState extends State<HealthPage>
    with SingleTickerProviderStateMixin {
  late final TabController _tabController;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 5, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<HealthProvider>();
      p.fetchDaily();
      p.fetchActivities();
      p.fetchBasicInfo();
      p.fetchCheckups();
      p.fetchRecords();
    });
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<HealthProvider>();
    return Scaffold(
      appBar: AppBar(
        title: const Text('身体健康'),
        bottom: TabBar(
          controller: _tabController,
          isScrollable: true,
          tabs: const [
            Tab(text: '日常记录'),
            Tab(text: '健身活动'),
            Tab(text: '基本信息'),
            Tab(text: '体检记录'),
            Tab(text: '病历记录'),
          ],
          onTap: (index) {
            switch (index) {
              case 0:
                provider.fetchDaily();
              case 1:
                provider.fetchActivities();
              case 2:
                provider.fetchBasicInfo();
              case 3:
                provider.fetchCheckups();
              case 4:
                provider.fetchRecords();
            }
          },
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          _DailyTab(provider: provider),
          _ActivitiesTab(provider: provider),
          _BasicInfoTab(provider: provider),
          _CheckupTab(provider: provider),
          _RecordTab(provider: provider),
        ],
      ),
    );
  }
}

// ── 日常记录 Tab（录入 + 趋势可视化）──

class _DailyTab extends StatefulWidget {
  final HealthProvider provider;
  const _DailyTab({required this.provider});

  @override
  State<_DailyTab> createState() => _DailyTabState();
}

class _DailyTabState extends State<_DailyTab> {
  // 当前趋势图指标：0=身高 1=体重 2=血压 3=心率
  int _metric = 1;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final daily = widget.provider.daily;

    return ListView(
      padding: const EdgeInsets.all(12),
      children: [
        // 录入入口
        Card(
          elevation: 0,
          color: theme.colorScheme.primaryContainer.withOpacity(0.3),
          shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12)),
          child: ListTile(
            leading: Icon(Icons.add_chart, color: theme.colorScheme.primary),
            title: const Text('记录今日健康数据'),
            subtitle: const Text('身高 / 体重 / 血压 / 心率'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => _openDailyEditor(context),
          ),
        ),
        const SizedBox(height: 12),

        // 指标切换
        SegmentedButton<int>(
          segments: const [
            ButtonSegment(value: 0, label: Text('身高')),
            ButtonSegment(value: 1, label: Text('体重')),
            ButtonSegment(value: 2, label: Text('血压')),
            ButtonSegment(value: 3, label: Text('心率')),
          ],
          selected: {_metric},
          onSelectionChanged: (v) => setState(() => _metric = v.first),
          showSelectedIcon: false,
        ),
        const SizedBox(height: 12),

        // 趋势图
        if (daily.isEmpty)
          _buildEmpty(theme)
        else
          _buildTrendChart(theme, daily),
        const SizedBox(height: 16),

        // 记录列表
        if (daily.isNotEmpty) ...[
          Padding(
            padding: const EdgeInsets.only(left: 4, bottom: 8),
            child: Text('历史记录',
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.bold)),
          ),
          for (final r in daily.reversed.take(20))
            _buildDailyRow(theme, r),
        ],
      ],
    );
  }

  Widget _buildEmpty(ThemeData theme) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 32),
      child: Center(
        child: Column(
          children: [
            Icon(Icons.monitor_heart_outlined,
                size: 48, color: theme.colorScheme.outline),
            const SizedBox(height: 8),
            Text('暂无日常记录，点击上方「记录今日健康数据」开始',
                style: theme.textTheme.bodyMedium
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ],
        ),
      ),
    );
  }

  Widget _buildTrendChart(ThemeData theme, List<HealthDailyRecord> daily) {
    final data = daily;
    if (data.isEmpty) return const SizedBox.shrink();

    final (spots, yMin, yMax, label) = switch (_metric) {
      0 => (
          data.map((e) => FlSpot(
              data.indexOf(e).toDouble(), e.heightCm)).toList(),
          0.0,
          data.map((e) => e.heightCm).reduce((a, b) => a > b ? a : b) * 1.1,
          '身高 (cm)',
        ),
      1 => (
          data.map((e) => FlSpot(
              data.indexOf(e).toDouble(), e.weightKg)).toList(),
          0.0,
          data.map((e) => e.weightKg).reduce((a, b) => a > b ? a : b) * 1.2,
          '体重 (kg)',
        ),
      2 => (
          data.map((e) => FlSpot(
              data.indexOf(e).toDouble(), e.systolic.toDouble())).toList(),
          0.0,
          (data.map((e) => e.systolic).reduce((a, b) => a > b ? a : b) + 20)
              .toDouble(),
          '收缩压 (mmHg)',
        ),
      _ => (
          data.map((e) => FlSpot(
              data.indexOf(e).toDouble(), e.heartRate.toDouble())).toList(),
          0.0,
          (data.map((e) => e.heartRate).reduce((a, b) => a > b ? a : b) + 20)
              .toDouble(),
          '心率 (bpm)',
        ),
    };

    final color = switch (_metric) {
      0 => Colors.orange,
      1 => Colors.blue,
      2 => Colors.red,
      _ => Colors.green,
    };

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(label,
                style: theme.textTheme.titleSmall
                    ?.copyWith(fontWeight: FontWeight.bold)),
            const SizedBox(height: 16),
            SizedBox(
              height: 200,
              child: LineChart(
                LineChartData(
                  gridData: FlGridData(show: true, drawVerticalLine: false),
                  titlesData: FlTitlesData(
                    leftTitles: AxisTitles(
                      sideTitles: SideTitles(
                        showTitles: true,
                        reservedSize: 44,
                        getTitlesWidget: (value, meta) => Text(
                          value.toStringAsFixed(0),
                          style: const TextStyle(fontSize: 10),
                        ),
                      ),
                    ),
                    bottomTitles: AxisTitles(
                      sideTitles: SideTitles(
                        showTitles: true,
                        interval: (data.length / 5).ceilToDouble(),
                        getTitlesWidget: (value, meta) {
                          final idx = value.toInt();
                          if (idx < 0 || idx >= data.length) {
                            return const SizedBox();
                          }
                          return Padding(
                            padding: const EdgeInsets.only(top: 8),
                            child: Text(
                              data[idx].recordDate.length >= 10
                                  ? data[idx].recordDate.substring(5)
                                  : data[idx].recordDate,
                              style: const TextStyle(fontSize: 9),
                            ),
                          );
                        },
                      ),
                    ),
                    rightTitles:
                        AxisTitles(sideTitles: SideTitles(showTitles: false)),
                    topTitles:
                        AxisTitles(sideTitles: SideTitles(showTitles: false)),
                  ),
                  borderData: FlBorderData(show: false),
                  lineBarsData: [
                    LineChartBarData(
                      spots: spots,
                      isCurved: true,
                      color: color,
                      barWidth: 2.5,
                      dotData: FlDotData(show: data.length <= 30),
                      belowBarData: BarAreaData(
                        show: true,
                        color: color.withOpacity(0.1),
                      ),
                    ),
                  ],
                  minY: yMin,
                  maxY: yMax > 0 ? yMax : 100,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildDailyRow(ThemeData theme, HealthDailyRecord r) {
    final parts = <String>[
      if (r.heightCm > 0) '身高 ${r.heightCm}cm',
      if (r.weightKg > 0) '体重 ${r.weightKg}kg',
      if (r.systolic > 0) '血压 ${r.systolic}/${r.diastolic}',
      if (r.heartRate > 0) '心率 ${r.heartRate}',
    ];
    return Card(
      elevation: 0,
      margin: const EdgeInsets.only(bottom: 6),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: ListTile(
        dense: true,
        leading: const Icon(Icons.calendar_today_outlined, size: 18),
        title: Text(r.recordDate, style: theme.textTheme.bodyMedium),
        subtitle: parts.isEmpty
            ? null
            : Text(parts.join(' · '), style: theme.textTheme.bodySmall),
        trailing: IconButton(
          icon: const Icon(Icons.delete_outline, size: 18),
          onPressed: () async {
            final ok = await showDialog<bool>(
              context: context,
              builder: (ctx) => AlertDialog(
                title: const Text('删除确认'),
                content: Text('确定删除 ${r.recordDate} 的记录吗？'),
                actions: [
                  TextButton(
                    onPressed: () => Navigator.pop(ctx, false),
                    child: const Text('取消'),
                  ),
                  FilledButton(
                    onPressed: () => Navigator.pop(ctx, true),
                    child: const Text('删除'),
                  ),
                ],
              ),
            );
            if (ok == true) {
              final d = await context
                  .read<HealthProvider>()
                  .deleteDaily(r.recordDate);
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(d ? '已删除' : '删除失败')),
                );
              }
            }
          },
        ),
      ),
    );
  }

  Future<void> _openDailyEditor(BuildContext context) async {
    final dateCtrl = TextEditingController(
        text: DateTime.now().toString().substring(0, 10));
    final heightCtrl = TextEditingController();
    final weightCtrl = TextEditingController();
    final systolicCtrl = TextEditingController();
    final diastolicCtrl = TextEditingController();
    final heartCtrl = TextEditingController();
    final noteCtrl = TextEditingController();

    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(
          left: 16,
          right: 16,
          top: 8,
          bottom: MediaQuery.of(ctx).viewInsets.bottom + 16,
        ),
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('记录今日健康数据',
                  style: Theme.of(ctx).textTheme.titleLarge
                      ?.copyWith(fontWeight: FontWeight.bold)),
              const SizedBox(height: 16),
              _dailyField(ctx, '日期 (yyyy-MM-dd)', dateCtrl),
              Row(
                children: [
                  Expanded(
                      child: _dailyField(ctx, '身高 (cm)', heightCtrl,
                          keyboardType: TextInputType.number)),
                  const SizedBox(width: 8),
                  Expanded(
                      child: _dailyField(ctx, '体重 (kg)', weightCtrl,
                          keyboardType: TextInputType.number)),
                ],
              ),
              Row(
                children: [
                  Expanded(
                      child: _dailyField(ctx, '收缩压', systolicCtrl,
                          keyboardType: TextInputType.number, hint: 'mmHg')),
                  const SizedBox(width: 8),
                  Expanded(
                      child: _dailyField(ctx, '舒张压', diastolicCtrl,
                          keyboardType: TextInputType.number, hint: 'mmHg')),
                ],
              ),
              _dailyField(ctx, '心率', heartCtrl,
                  keyboardType: TextInputType.number, hint: 'bpm'),
              _dailyField(ctx, '备注', noteCtrl, hint: '可选', maxLines: 2),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () async {
                    final ok =
                        await context.read<HealthProvider>().saveDaily({
                      'record_date': dateCtrl.text.trim(),
                      'height_cm': double.tryParse(heightCtrl.text) ?? 0,
                      'weight_kg': double.tryParse(weightCtrl.text) ?? 0,
                      'systolic': int.tryParse(systolicCtrl.text) ?? 0,
                      'diastolic': int.tryParse(diastolicCtrl.text) ?? 0,
                      'heart_rate': int.tryParse(heartCtrl.text) ?? 0,
                      'note': noteCtrl.text.trim(),
                    });
                    if (ctx.mounted) {
                      ScaffoldMessenger.of(ctx).showSnackBar(
                        SnackBar(content: Text(ok ? '已保存' : '保存失败')),
                      );
                      if (ok) Navigator.pop(ctx);
                    }
                  },
                  child: const Text('保存'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _dailyField(BuildContext ctx, String label, TextEditingController ctrl,
      {String? hint, int maxLines = 1, TextInputType? keyboardType}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: TextField(
        controller: ctrl,
        maxLines: maxLines,
        keyboardType: keyboardType,
        decoration: InputDecoration(
          labelText: label,
          hintText: hint,
          border: const OutlineInputBorder(),
          isDense: true,
        ),
      ),
    );
  }
}

// ── 健身活动 / 竞技比赛 Tab ──

class _ActivitiesTab extends StatelessWidget {
  final HealthProvider provider;
  const _ActivitiesTab({required this.provider});

  bool get _canCreate {
    final role = Storage.role;
    return role == 'student_union' ||
        role == 'sys_admin' ||
        role == 'school_admin' ||
        role == 'college_admin';
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final activities = provider.activities;

    return Stack(
      children: [
        if (activities.isEmpty)
          Padding(
            padding: const EdgeInsets.all(32),
            child: Center(
              child: Column(
                children: [
                  Icon(Icons.sports_soccer,
                      size: 48, color: theme.colorScheme.outline),
                  const SizedBox(height: 8),
                  Text('暂无健身/竞技活动',
                      style: theme.textTheme.bodyMedium
                          ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                ],
              ),
            ),
          )
        else
          ListView(
            padding: const EdgeInsets.all(12),
            children: [
              // 分类筛选
              SegmentedButton<String>(
                segments: const [
                  ButtonSegment(value: 'all', label: Text('全部')),
                  ButtonSegment(value: 'sports', label: Text('竞技')),
                  ButtonSegment(value: 'fitness', label: Text('健身')),
                ],
                selected: {'all'},
                onSelectionChanged: (_) {},
                showSelectedIcon: false,
              ),
              const SizedBox(height: 12),
              for (final a in activities)
                _buildActivityCard(context, theme, a),
            ],
          ),
        if (_canCreate)
          Positioned(
            right: 16,
            bottom: 16,
            child: FloatingActionButton.extended(
              heroTag: 'health_activity_fab',
              onPressed: () => _openCreateDialog(context),
              icon: const Icon(Icons.add),
              label: const Text('发起活动'),
            ),
          ),
      ],
    );
  }

  Widget _buildActivityCard(BuildContext context, ThemeData theme, HealthActivity a) {
    final catLabel = a.category == 'sports' ? '竞技' : '健身';
    final catColor =
        a.category == 'sports' ? Colors.orange : Colors.teal;
    return Card(
      elevation: 0,
      margin: const EdgeInsets.only(bottom: 10),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(14),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                  decoration: BoxDecoration(
                    color: catColor.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(catLabel,
                      style: TextStyle(
                          fontSize: 11, color: catColor,
                          fontWeight: FontWeight.w600)),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(a.title,
                      style: theme.textTheme.titleSmall
                          ?.copyWith(fontWeight: FontWeight.bold)),
                ),
                IconButton(
                  icon: Icon(
                    a.isFavorite ? Icons.favorite : Icons.favorite_border,
                    color: a.isFavorite ? Colors.red : null,
                    size: 20,
                  ),
                  tooltip: a.isFavorite ? '取消关注' : '关注',
                  onPressed: () => _toggleFavorite(context, a),
                ),
              ],
            ),
            if (a.description.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Text(a.description,
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
              ),
            const SizedBox(height: 10),
            _metaLine(theme, Icons.schedule, a.startAt),
            if (a.venue.isNotEmpty)
              _metaLine(theme, Icons.place_outlined, a.venue),
            _metaLine(theme, Icons.groups_outlined,
                '${a.organizer} · ${a.favoriteCount}人关注 · ${a.signupCount}人报名'),
            const SizedBox(height: 8),
            Row(
              children: [
                OutlinedButton.icon(
                  onPressed: () => _toggleSignup(context, a),
                  icon: Icon(a.isSignup ? Icons.check : Icons.how_to_reg,
                      size: 16),
                  label: Text(a.isSignup ? '已报名' : '报名参加'),
                  style: OutlinedButton.styleFrom(
                    visualDensity: VisualDensity.compact,
                    foregroundColor: a.isSignup ? Colors.green : null,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _metaLine(ThemeData theme, IconData icon, String text) {
    if (text.isEmpty) return const SizedBox.shrink();
    return Padding(
      padding: const EdgeInsets.only(top: 4),
      child: Row(
        children: [
          Icon(icon, size: 13, color: theme.colorScheme.onSurfaceVariant),
          const SizedBox(width: 4),
          Expanded(
            child: Text(text,
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ),
        ],
      ),
    );
  }

  Future<void> _toggleFavorite(BuildContext context, HealthActivity a) async {
    final ok = await context
        .read<HealthProvider>()
        .toggleFavorite(a.activityId, !a.isFavorite);
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '已更新' : '操作失败')),
      );
    }
  }

  Future<void> _toggleSignup(BuildContext context, HealthActivity a) async {
    final ok = await context
        .read<HealthProvider>()
        .toggleSignup(a.activityId, !a.isSignup);
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '已更新' : '操作失败')),
      );
    }
  }

  Future<void> _openCreateDialog(BuildContext context) async {
    final titleCtrl = TextEditingController();
    final descCtrl = TextEditingController();
    final venueCtrl = TextEditingController();
    final startCtrl = TextEditingController();
    final deadlineCtrl = TextEditingController();
    final capacityCtrl = TextEditingController();
    String category = 'sports';

    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(
          left: 16,
          right: 16,
          top: 8,
          bottom: MediaQuery.of(ctx).viewInsets.bottom + 16,
        ),
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('发起健身/竞技活动',
                  style: Theme.of(ctx).textTheme.titleLarge
                      ?.copyWith(fontWeight: FontWeight.bold)),
              const SizedBox(height: 16),
              _actField(ctx, '活动名称 *', titleCtrl),
              Row(
                children: [
                  Expanded(
                    child: _actField(ctx, '开始时间', startCtrl,
                        hint: '2026-09-15 14:00'),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: _actField(ctx, '报名截止', deadlineCtrl,
                        hint: '2026-09-12'),
                  ),
                ],
              ),
              _actField(ctx, '地点', venueCtrl),
              _actField(ctx, '名额上限(0不限)', capacityCtrl,
                  keyboardType: TextInputType.number),
              const SizedBox(height: 8),
              Text('分类', style: Theme.of(ctx).textTheme.labelLarge),
              const SizedBox(height: 4),
              SegmentedButton<String>(
                segments: const [
                  ButtonSegment(value: 'sports', label: Text('竞技')),
                  ButtonSegment(value: 'fitness', label: Text('健身')),
                ],
                selected: {category},
                onSelectionChanged: (v) => category = v.first,
                showSelectedIcon: false,
              ),
              const SizedBox(height: 8),
              _actField(ctx, '活动介绍', descCtrl, maxLines: 3),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () async {
                    if (titleCtrl.text.trim().isEmpty) {
                      ScaffoldMessenger.of(ctx).showSnackBar(
                        const SnackBar(content: Text('请输入活动名称')),
                      );
                      return;
                    }
                    final ok =
                        await context.read<HealthProvider>().createActivity({
                      'title': titleCtrl.text.trim(),
                      'category': category,
                      'description': descCtrl.text.trim(),
                      'start_at': startCtrl.text.trim(),
                      'signup_deadline': deadlineCtrl.text.trim(),
                      'venue': venueCtrl.text.trim(),
                      'capacity': int.tryParse(capacityCtrl.text) ?? 0,
                    });
                    if (ctx.mounted) {
                      ScaffoldMessenger.of(ctx).showSnackBar(
                        SnackBar(content: Text(ok ? '活动已发布' : '发布失败')),
                      );
                      if (ok) Navigator.pop(ctx);
                    }
                  },
                  child: const Text('发布'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _actField(BuildContext ctx, String label, TextEditingController ctrl,
      {String? hint, int maxLines = 1, TextInputType? keyboardType}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: TextField(
        controller: ctrl,
        maxLines: maxLines,
        keyboardType: keyboardType,
        decoration: InputDecoration(
          labelText: label,
          hintText: hint,
          border: const OutlineInputBorder(),
          isDense: true,
        ),
      ),
    );
  }
}

// ── 基本信息 Tab ──

class _BasicInfoTab extends StatelessWidget {
  final HealthProvider provider;
  const _BasicInfoTab({required this.provider});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final info = provider.basicInfo;

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(16),
            side: BorderSide(color: theme.colorScheme.outlineVariant),
          ),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(Icons.favorite_outline,
                        color: theme.colorScheme.primary),
                    const SizedBox(width: 8),
                    Text('身体基本信息',
                        style: theme.textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.bold)),
                    const Spacer(),
                    TextButton.icon(
                      onPressed: () => _openEditDialog(context, info),
                      icon: const Icon(Icons.edit_outlined, size: 18),
                      label: Text(info == null ? '完善信息' : '编辑'),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                if (info == null)
                  Text('尚未填写身体基本信息，点击右上角「完善信息」开始',
                      style: theme.textTheme.bodyMedium
                          ?.copyWith(color: theme.colorScheme.onSurfaceVariant))
                else ...[
                  _infoRow(theme, '身高', info.heightCm > 0 ? '${info.heightCm} cm' : '未填写'),
                  _infoRow(theme, '体重', info.weightKg > 0 ? '${info.weightKg} kg' : '未填写'),
                  _infoRow(theme, '血型', info.bloodType.isEmpty ? '未填写' : info.bloodType),
                  _infoRow(theme, '左眼视力', info.visionLeft.isEmpty ? '未填写' : info.visionLeft),
                  _infoRow(theme, '右眼视力', info.visionRight.isEmpty ? '未填写' : info.visionRight),
                  _infoRow(theme, '过敏史', info.allergies.isEmpty ? '无' : info.allergies),
                  _infoRow(theme, '既往病史', info.pastIllness.isEmpty ? '无' : info.pastIllness),
                  _infoRow(theme, '家族病史', info.familyHistory.isEmpty ? '无' : info.familyHistory),
                  _infoRow(theme, '紧急联系人', info.emergencyContact.isEmpty ? '未填写' : info.emergencyContact),
                  _infoRow(theme, '紧急联系电话', info.emergencyPhone.isEmpty ? '未填写' : info.emergencyPhone),
                ],
              ],
            ),
          ),
        ),
        const SizedBox(height: 12),
        Text('* 仅本人可见，用于健康关怀与紧急联系，请如实填写。',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: theme.colorScheme.outline)),
      ],
    );
  }

  Widget _infoRow(ThemeData theme, String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 90,
            child: Text(label,
                style: theme.textTheme.bodyMedium
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ),
          Expanded(
            child: Text(value, style: theme.textTheme.bodyMedium),
          ),
        ],
      ),
    );
  }

  Future<void> _openEditDialog(BuildContext context, dynamic info) async {
    final heightCtrl = TextEditingController(
        text: info != null && info.heightCm > 0 ? info.heightCm.toString() : '');
    final weightCtrl = TextEditingController(
        text: info != null && info.weightKg > 0 ? info.weightKg.toString() : '');
    final bloodCtrl = TextEditingController(
        text: info != null ? (info.bloodType ?? '') : '');
    final visionLCtrl = TextEditingController(
        text: info != null ? (info.visionLeft ?? '') : '');
    final visionRCtrl = TextEditingController(
        text: info != null ? (info.visionRight ?? '') : '');
    final allergyCtrl = TextEditingController(
        text: info != null ? (info.allergies ?? '') : '');
    final illnessCtrl = TextEditingController(
        text: info != null ? (info.pastIllness ?? '') : '');
    final familyCtrl = TextEditingController(
        text: info != null ? (info.familyHistory ?? '') : '');
    final contactCtrl = TextEditingController(
        text: info != null ? (info.emergencyContact ?? '') : '');
    final phoneCtrl = TextEditingController(
        text: info != null ? (info.emergencyPhone ?? '') : '');

    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(
          left: 16,
          right: 16,
          top: 8,
          bottom: MediaQuery.of(ctx).viewInsets.bottom + 16,
        ),
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('身体基本信息',
                  style: Theme.of(ctx).textTheme.titleLarge
                      ?.copyWith(fontWeight: FontWeight.bold)),
              const SizedBox(height: 16),
              _field('身高 (cm)', heightCtrl, keyboardType: TextInputType.number),
              _field('体重 (kg)', weightCtrl, keyboardType: TextInputType.number),
              _field('血型', bloodCtrl, hint: '如 A / B / AB / O'),
              _field('左眼视力', visionLCtrl, hint: '如 5.0'),
              _field('右眼视力', visionRCtrl, hint: '如 5.0'),
              _field('过敏史', allergyCtrl, hint: '无或填写具体过敏原'),
              _field('既往病史', illnessCtrl, hint: '无或填写既往疾病'),
              _field('家族病史', familyCtrl, hint: '无或填写家族病史'),
              _field('紧急联系人', contactCtrl),
              _field('紧急联系电话', phoneCtrl, keyboardType: TextInputType.phone),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () async {
                    final ok = await context.read<HealthProvider>().saveBasicInfo({
                      'height_cm': double.tryParse(heightCtrl.text) ?? 0,
                      'weight_kg': double.tryParse(weightCtrl.text) ?? 0,
                      'blood_type': bloodCtrl.text.trim(),
                      'vision_left': visionLCtrl.text.trim(),
                      'vision_right': visionRCtrl.text.trim(),
                      'allergies': allergyCtrl.text.trim(),
                      'past_illness': illnessCtrl.text.trim(),
                      'family_history': familyCtrl.text.trim(),
                      'emergency_contact': contactCtrl.text.trim(),
                      'emergency_phone': phoneCtrl.text.trim(),
                    });
                    if (ctx.mounted) {
                      ScaffoldMessenger.of(ctx).showSnackBar(
                        SnackBar(content: Text(ok ? '已保存' : '保存失败')),
                      );
                      if (ok) Navigator.pop(ctx);
                    }
                  },
                  child: const Text('保存'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _field(String label, TextEditingController ctrl,
      {String? hint, TextInputType? keyboardType}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: TextField(
        controller: ctrl,
        keyboardType: keyboardType,
        decoration: InputDecoration(
          labelText: label,
          hintText: hint,
          border: const OutlineInputBorder(),
          isDense: true,
        ),
      ),
    );
  }
}

// ── 通用记录列表（体检/病历共用）──

class _RecordListCard extends StatelessWidget {
  final List<Widget> cards;
  final String emptyText;
  const _RecordListCard({required this.cards, required this.emptyText});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (cards.isEmpty) {
      return Padding(
        padding: const EdgeInsets.all(32),
        child: Center(
          child: Column(
            children: [
              Icon(Icons.inbox_outlined,
                  size: 48, color: theme.colorScheme.outline),
              const SizedBox(height: 8),
              Text(emptyText,
                  style: theme.textTheme.bodyMedium
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
            ],
          ),
        ),
      );
    }
    return ListView(
      padding: const EdgeInsets.all(12),
      children: cards,
    );
  }
}

// ── 体检记录 Tab ──

class _CheckupTab extends StatelessWidget {
  final HealthProvider provider;
  const _CheckupTab({required this.provider});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cards = provider.checkups
        .map((c) => _buildCard(theme, c.checkupDate, '体检机构：${c.hospital}',
            '结论：${c.conclusion}', '详情：${c.details}',
            onEdit: () => _openEdit(context, c), onDelete: () => _delete(context, c)))
        .toList();

    return Stack(
      children: [
        _RecordListCard(
            cards: cards, emptyText: '暂无体检记录，点击右下角 + 新增'),
        Positioned(
          right: 16,
          bottom: 16,
          child: FloatingActionButton(
            heroTag: 'health_checkup_fab',
            onPressed: () => _openEdit(context, null),
            child: const Icon(Icons.add),
          ),
        ),
      ],
    );
  }

  Widget _buildCard(
    ThemeData theme,
    String title,
    String l1,
    String l2,
    String l3, {
    required VoidCallback onEdit,
    required VoidCallback onDelete,
  }) {
    return Card(
      elevation: 0,
      margin: const EdgeInsets.only(bottom: 8),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(title.isEmpty ? '未填写日期' : title,
                      style: theme.textTheme.titleSmall
                          ?.copyWith(fontWeight: FontWeight.bold)),
                ),
                IconButton(
                  icon: const Icon(Icons.edit_outlined, size: 18),
                  onPressed: onEdit,
                ),
                IconButton(
                  icon: const Icon(Icons.delete_outline, size: 18),
                  onPressed: onDelete,
                ),
              ],
            ),
            if (l1.isNotEmpty) Text(l1, style: theme.textTheme.bodySmall),
            if (l2.isNotEmpty) Text(l2, style: theme.textTheme.bodySmall),
            if (l3.isNotEmpty)
              Text(l3,
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ],
        ),
      ),
    );
  }

  void _openEdit(BuildContext context, dynamic item) {
    final dateCtrl = TextEditingController(text: item?.checkupDate ?? '');
    final hospitalCtrl = TextEditingController(text: item?.hospital ?? '');
    final conclusionCtrl = TextEditingController(text: item?.conclusion ?? '');
    final detailsCtrl = TextEditingController(text: item?.details ?? '');

    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(
          left: 16,
          right: 16,
          top: 8,
          bottom: MediaQuery.of(ctx).viewInsets.bottom + 16,
        ),
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(item == null ? '新增体检记录' : '编辑体检记录',
                  style: Theme.of(ctx).textTheme.titleLarge
                      ?.copyWith(fontWeight: FontWeight.bold)),
              const SizedBox(height: 16),
              _field(ctx, '体检日期 (yyyy-MM-dd)', dateCtrl),
              _field(ctx, '体检机构', hospitalCtrl),
              _field(ctx, '体检结论', conclusionCtrl, hint: '正常 / 异常等'),
              _field(ctx, '体检详情', detailsCtrl, maxLines: 3),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () async {
                    final body = {
                      'checkup_date': dateCtrl.text.trim(),
                      'hospital': hospitalCtrl.text.trim(),
                      'conclusion': conclusionCtrl.text.trim(),
                      'details': detailsCtrl.text.trim(),
                      'attachments': const <String>[],
                    };
                    final ok = item == null
                        ? await context
                            .read<HealthProvider>()
                            .createCheckup(body)
                        : await context
                            .read<HealthProvider>()
                            .updateCheckup(item.id, body);
                    if (ctx.mounted) {
                      ScaffoldMessenger.of(ctx).showSnackBar(
                        SnackBar(content: Text(ok ? '已保存' : '操作失败')),
                      );
                      if (ok) Navigator.pop(ctx);
                    }
                  },
                  child: const Text('保存'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _field(BuildContext ctx, String label, TextEditingController ctrl,
      {String? hint, int maxLines = 1}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: TextField(
        controller: ctrl,
        maxLines: maxLines,
        decoration: InputDecoration(
          labelText: label,
          hintText: hint,
          border: const OutlineInputBorder(),
          isDense: true,
        ),
      ),
    );
  }

  Future<void> _delete(BuildContext context, dynamic item) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('删除确认'),
        content: const Text('确定删除这条体检记录吗？'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('删除'),
          ),
        ],
      ),
    );
    if (ok == true) {
      final r = await context.read<HealthProvider>().deleteCheckup(item.id);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(r ? '已删除' : '删除失败')),
        );
      }
    }
  }
}

// ── 病历记录 Tab ──

class _RecordTab extends StatelessWidget {
  final HealthProvider provider;
  const _RecordTab({required this.provider});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cards = provider.records
        .map((r) => _buildCard(theme, r.recordDate, '医院：${r.hospital}',
            '科室：${r.department}', '诊断：${r.diagnosis}', '治疗：${r.treatment}',
            onEdit: () => _openEdit(context, r), onDelete: () => _delete(context, r)))
        .toList();

    return Stack(
      children: [
        _RecordListCard(cards: cards, emptyText: '暂无病历记录，点击右下角 + 新增'),
        Positioned(
          right: 16,
          bottom: 16,
          child: FloatingActionButton(
            heroTag: 'health_record_fab',
            onPressed: () => _openEdit(context, null),
            child: const Icon(Icons.add),
          ),
        ),
      ],
    );
  }

  Widget _buildCard(
    ThemeData theme,
    String title,
    String l1,
    String l2,
    String l3,
    String l4, {
    required VoidCallback onEdit,
    required VoidCallback onDelete,
  }) {
    return Card(
      elevation: 0,
      margin: const EdgeInsets.only(bottom: 8),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(title.isEmpty ? '未填写日期' : title,
                      style: theme.textTheme.titleSmall
                          ?.copyWith(fontWeight: FontWeight.bold)),
                ),
                IconButton(
                  icon: const Icon(Icons.edit_outlined, size: 18),
                  onPressed: onEdit,
                ),
                IconButton(
                  icon: const Icon(Icons.delete_outline, size: 18),
                  onPressed: onDelete,
                ),
              ],
            ),
            if (l1.isNotEmpty) Text(l1, style: theme.textTheme.bodySmall),
            if (l2.isNotEmpty) Text(l2, style: theme.textTheme.bodySmall),
            if (l3.isNotEmpty) Text(l3, style: theme.textTheme.bodySmall),
            if (l4.isNotEmpty)
              Text(l4,
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ],
        ),
      ),
    );
  }

  void _openEdit(BuildContext context, dynamic item) {
    final dateCtrl = TextEditingController(text: item?.recordDate ?? '');
    final hospitalCtrl = TextEditingController(text: item?.hospital ?? '');
    final deptCtrl = TextEditingController(text: item?.department ?? '');
    final diagCtrl = TextEditingController(text: item?.diagnosis ?? '');
    final treatCtrl = TextEditingController(text: item?.treatment ?? '');

    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(
          left: 16,
          right: 16,
          top: 8,
          bottom: MediaQuery.of(ctx).viewInsets.bottom + 16,
        ),
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(item == null ? '新增病历记录' : '编辑病历记录',
                  style: Theme.of(ctx).textTheme.titleLarge
                      ?.copyWith(fontWeight: FontWeight.bold)),
              const SizedBox(height: 16),
              _field(ctx, '就诊日期 (yyyy-MM-dd)', dateCtrl),
              _field(ctx, '就诊医院', hospitalCtrl),
              _field(ctx, '科室', deptCtrl),
              _field(ctx, '诊断', diagCtrl),
              _field(ctx, '治疗方案 / 用药', treatCtrl, maxLines: 3),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () async {
                    final body = {
                      'record_date': dateCtrl.text.trim(),
                      'hospital': hospitalCtrl.text.trim(),
                      'department': deptCtrl.text.trim(),
                      'diagnosis': diagCtrl.text.trim(),
                      'treatment': treatCtrl.text.trim(),
                      'attachments': const <String>[],
                    };
                    final ok = item == null
                        ? await context
                            .read<HealthProvider>()
                            .createRecord(body)
                        : await context
                            .read<HealthProvider>()
                            .updateRecord(item.id, body);
                    if (ctx.mounted) {
                      ScaffoldMessenger.of(ctx).showSnackBar(
                        SnackBar(content: Text(ok ? '已保存' : '操作失败')),
                      );
                      if (ok) Navigator.pop(ctx);
                    }
                  },
                  child: const Text('保存'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _field(BuildContext ctx, String label, TextEditingController ctrl,
      {String? hint, int maxLines = 1}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: TextField(
        controller: ctrl,
        maxLines: maxLines,
        decoration: InputDecoration(
          labelText: label,
          hintText: hint,
          border: const OutlineInputBorder(),
          isDense: true,
        ),
      ),
    );
  }

  Future<void> _delete(BuildContext context, dynamic item) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('删除确认'),
        content: const Text('确定删除这条病历记录吗？'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('删除'),
          ),
        ],
      ),
    );
    if (ok == true) {
      final r = await context.read<HealthProvider>().deleteRecord(item.id);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(r ? '已删除' : '删除失败')),
        );
      }
    }
  }
}
