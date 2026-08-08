import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/health_provider.dart';

/// 学生「身体健康」页：身体基本信息 / 体检记录 / 病历记录
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
    _tabController = TabController(length: 3, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<HealthProvider>();
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
          tabs: const [
            Tab(text: '基本信息'),
            Tab(text: '体检记录'),
            Tab(text: '病历记录'),
          ],
          onTap: (index) {
            switch (index) {
              case 0:
                provider.fetchBasicInfo();
              case 1:
                provider.fetchCheckups();
              case 2:
                provider.fetchRecords();
            }
          },
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          _BasicInfoTab(provider: provider),
          _CheckupTab(provider: provider),
          _RecordTab(provider: provider),
        ],
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
