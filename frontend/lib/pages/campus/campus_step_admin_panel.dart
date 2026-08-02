// 校园报到步骤管理面板（管理员 CRUD）
// 对接后端 /admin/campus/steps 全套接口：
//   GET    /admin/campus/steps?campus=xx      列表（含草稿）
//   POST   /admin/campus/steps                新建（draft）
//   PUT    /admin/campus/steps/:id            更新（仅 draft）
//   PATCH  /admin/campus/steps/:id/coords     更新坐标（任意状态）
//   POST   /admin/campus/steps/:id/submit     draft → pending_review
//   POST   /admin/campus/steps/:id/publish    pending_review → published
//   DELETE /admin/campus/steps/:id            删除（仅 draft）
// 非 admin 角色不应进入此面板（由调用方控制可见性）。
import 'package:flutter/material.dart';
import '../../config/api_config.dart';
import '../../services/api_service.dart';

/// 单个报到步骤的完整数据（与后端 campus_checkin_steps 表对应）。
class CampusStepRecord {
  final int? id;
  final String campusId;
  final int stepOrder;
  final String title;
  final String location;
  final double lat;
  final double lng;
  final String duration;
  final String task;
  final String materials;
  final String contact;
  final String note;
  final String iconName;
  final String status; // draft | pending_review | published

  const CampusStepRecord({
    this.id,
    required this.campusId,
    required this.stepOrder,
    required this.title,
    required this.location,
    required this.lat,
    required this.lng,
    this.duration = '',
    this.task = '',
    this.materials = '',
    this.contact = '',
    this.note = '',
    this.iconName = 'place',
    this.status = 'draft',
  });

  factory CampusStepRecord.fromJson(Map<String, dynamic> j) => CampusStepRecord(
        id: (j['id'] as num?)?.toInt(),
        campusId: j['campus_id'] ?? 'huifeng',
        stepOrder: (j['step_order'] as num?)?.toInt() ?? 0,
        title: j['title'] ?? '',
        location: j['location'] ?? '',
        lat: (j['lat'] as num?)?.toDouble() ?? 0.0,
        lng: (j['lng'] as num?)?.toDouble() ?? 0.0,
        duration: j['duration'] ?? '',
        task: j['task'] ?? '',
        materials: j['materials'] ?? '',
        contact: j['contact'] ?? '',
        note: j['note'] ?? '',
        iconName: j['icon_name'] ?? 'place',
        status: j['status'] ?? 'draft',
      );

  Map<String, dynamic> toRequestJson() => {
        'campus_id': campusId,
        'step_order': stepOrder,
        'title': title,
        'location': location,
        'lat': lat,
        'lng': lng,
        'duration': duration,
        'task': task,
        'materials': materials,
        'contact': contact,
        'note': note,
        'icon_name': iconName,
      };
}

/// 校园报到步骤管理面板。
/// 以 BottomSheet 形式弹出，管理员可新建/编辑/删除/提交审核/发布步骤。
class CampusStepAdminPanel extends StatefulWidget {
  final String campusId;
  final String campusName;

  const CampusStepAdminPanel({
    super.key,
    required this.campusId,
    required this.campusName,
  });

  /// 以 BottomSheet 形式弹出管理面板。
  static Future<void> show(
    BuildContext context, {
    required String campusId,
    required String campusName,
  }) {
    return showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (_) => DraggableScrollableSheet(
        initialChildSize: 0.85,
        minChildSize: 0.5,
        maxChildSize: 0.95,
        expand: false,
        builder: (context, __) => CampusStepAdminPanel(
          campusId: campusId,
          campusName: campusName,
        ),
      ),
    );
  }

  @override
  State<CampusStepAdminPanel> createState() => _CampusStepAdminPanelState();
}

class _CampusStepAdminPanelState extends State<CampusStepAdminPanel> {
  final ApiService _api = ApiService();
  List<CampusStepRecord> _steps = [];
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _loadSteps();
  }

  Future<void> _loadSteps() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final resp = await _api.get(
        '${ApiConfig.adminCampusSteps}?campus=${widget.campusId}',
      );
      if (resp.data['code'] == 0) {
        final list = resp.data['data'] as List? ?? [];
        _steps = list
            .map((e) => CampusStepRecord.fromJson(e as Map<String, dynamic>))
            .toList();
      } else {
        _error = resp.data['message'] ?? '加载失败';
      }
    } catch (e) {
      _error = '网络错误：$e';
    }
    setState(() => _loading = false);
  }

  Future<void> _deleteStep(CampusStepRecord step) async {
    if (step.id == null) return;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('删除步骤'),
        content: Text('确定删除「${step.title}」？仅草稿状态可删除。'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('删除'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    try {
      final resp = await _api.delete(ApiConfig.adminCampusStep(step.id.toString()));
      if (resp.data['code'] == 0) {
        _toast('已删除');
        _loadSteps();
      } else {
        _toast(resp.data['message'] ?? '删除失败');
      }
    } catch (e) {
      _toast('网络错误：$e');
    }
  }

  Future<void> _submitStep(CampusStepRecord step) async {
    if (step.id == null) return;
    try {
      final resp =
          await _api.post(ApiConfig.adminCampusStepSubmit(step.id.toString()));
      if (resp.data['code'] == 0) {
        _toast('已提交审核');
        _loadSteps();
      } else {
        _toast(resp.data['message'] ?? '提交失败');
      }
    } catch (e) {
      _toast('网络错误：$e');
    }
  }

  Future<void> _publishStep(CampusStepRecord step) async {
    if (step.id == null) return;
    try {
      final resp =
          await _api.post(ApiConfig.adminCampusStepPublish(step.id.toString()));
      if (resp.data['code'] == 0) {
        _toast('发布成功');
        _loadSteps();
      } else {
        _toast(resp.data['message'] ?? '发布失败');
      }
    } catch (e) {
      _toast('网络错误：$e');
    }
  }

  void _toast(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(msg), duration: const Duration(seconds: 2)),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        // 顶部标题栏
        Container(
          padding: const EdgeInsets.fromLTRB(20, 12, 12, 12),
          decoration: BoxDecoration(
            border: Border(
                bottom: BorderSide(color: theme.colorScheme.outlineVariant)),
          ),
          child: Row(
            children: [
              Icon(Icons.admin_panel_settings, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text('报到流程管理',
                        style: theme.textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.bold)),
                    Text('${widget.campusName} · ${_steps.length} 个节点',
                        style: theme.textTheme.bodySmall),
                  ],
                ),
              ),
              IconButton(
                icon: const Icon(Icons.refresh),
                onPressed: _loadSteps,
                tooltip: '刷新',
              ),
              IconButton(
                icon: const Icon(Icons.close),
                onPressed: () => Navigator.pop(context),
              ),
            ],
          ),
        ),
        // 内容区
        Expanded(
          child: _loading
              ? const Center(child: CircularProgressIndicator())
              : _error != null
                  ? Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(Icons.error_outline,
                              size: 40, color: theme.colorScheme.error),
                          const SizedBox(height: 8),
                          Text(_error!),
                          const SizedBox(height: 12),
                          FilledButton(onPressed: _loadSteps, child: const Text('重试')),
                        ],
                      ),
                    )
                  : _steps.isEmpty
                      ? Center(
                          child: Text('暂无步骤，点击右下角 + 新建',
                              style: theme.textTheme.bodyMedium),
                        )
                      : ListView.separated(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 12, vertical: 8),
                          itemCount: _steps.length,
                          separatorBuilder: (_, __) => const SizedBox(height: 8),
                          itemBuilder: (ctx, i) => _buildStepCard(_steps[i]),
                        ),
        ),
        // 底部新建按钮
        SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: FilledButton.icon(
              onPressed: () async {
                final created = await _editStepDialog(null);
                if (created == true) _loadSteps();
              },
              icon: const Icon(Icons.add),
              label: const Text('新建步骤'),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildStepCard(CampusStepRecord step) {
    final theme = Theme.of(context);
    final statusColor = switch (step.status) {
      'published' => Colors.green,
      'pending_review' => Colors.orange,
      _ => Colors.grey,
    };
    final statusLabel = switch (step.status) {
      'published' => '已发布',
      'pending_review' => '待审核',
      _ => '草稿',
    };
    return Card(
      elevation: 0,
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
                Container(
                  width: 28,
                  height: 28,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: theme.colorScheme.primaryContainer,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text('${step.stepOrder}',
                      style: TextStyle(
                          fontWeight: FontWeight.bold,
                          color: theme.colorScheme.onPrimaryContainer)),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(step.title,
                      style: theme.textTheme.titleSmall
                          ?.copyWith(fontWeight: FontWeight.bold)),
                ),
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                  decoration: BoxDecoration(
                    color: statusColor.withOpacity(0.15),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Text(statusLabel,
                      style: TextStyle(
                          fontSize: 11,
                          color: statusColor,
                          fontWeight: FontWeight.w600)),
                ),
              ],
            ),
            const SizedBox(height: 6),
            Text(step.location,
                style: TextStyle(
                    fontSize: 12, color: theme.colorScheme.onSurfaceVariant)),
            Text(
                '${step.lat.toStringAsFixed(5)}, ${step.lng.toStringAsFixed(5)}',
                style: TextStyle(
                    fontSize: 11, color: theme.colorScheme.onSurfaceVariant)),
            if (step.duration.isNotEmpty || step.task.isNotEmpty) ...[
              const SizedBox(height: 6),
              if (step.duration.isNotEmpty)
                Text('时长：${step.duration}',
                    style: const TextStyle(fontSize: 12)),
              if (step.task.isNotEmpty)
                Text('任务：${step.task}',
                    style: const TextStyle(fontSize: 12),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis),
            ],
            const SizedBox(height: 8),
            Wrap(
              spacing: 6,
              children: [
                if (step.status == 'draft')
                  TextButton.icon(
                    icon: const Icon(Icons.edit, size: 16),
                    label: const Text('编辑'),
                    onPressed: () async {
                      final ok = await _editStepDialog(step);
                      if (ok == true) _loadSteps();
                    },
                  ),
                if (step.status == 'draft')
                  TextButton.icon(
                    icon: const Icon(Icons.send, size: 16),
                    label: const Text('提交审核'),
                    onPressed: () => _submitStep(step),
                  ),
                if (step.status == 'pending_review')
                  TextButton.icon(
                    icon: const Icon(Icons.publish, size: 16),
                    label: const Text('发布'),
                    onPressed: () => _publishStep(step),
                  ),
                if (step.status == 'draft')
                  TextButton.icon(
                    icon: const Icon(Icons.delete_outline, size: 16),
                    label: const Text('删除'),
                    style: TextButton.styleFrom(foregroundColor: Colors.red),
                    onPressed: () => _deleteStep(step),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  /// 编辑/新建步骤对话框。step 为 null 时新建，否则编辑。
  /// 返回 true 表示有变更需要刷新。
  Future<bool?> _editStepDialog(CampusStepRecord? step) async {
    final isCreate = step == null;
    final titleCtrl = TextEditingController(text: step?.title ?? '');
    final locationCtrl = TextEditingController(text: step?.location ?? '');
    final latCtrl = TextEditingController(
        text: step?.lat.toStringAsFixed(6) ?? '0');
    final lngCtrl = TextEditingController(
        text: step?.lng.toStringAsFixed(6) ?? '0');
    final orderCtrl = TextEditingController(
        text: (step?.stepOrder ?? _steps.length + 1).toString());
    final durationCtrl = TextEditingController(text: step?.duration ?? '');
    final taskCtrl = TextEditingController(text: step?.task ?? '');
    final materialsCtrl = TextEditingController(text: step?.materials ?? '');
    final contactCtrl = TextEditingController(text: step?.contact ?? '');
    final noteCtrl = TextEditingController(text: step?.note ?? '');
    String iconName = step?.iconName ?? 'place';

    final result = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setSt) => AlertDialog(
          title: Text(isCreate ? '新建步骤' : '编辑步骤'),
          content: SizedBox(
            width: double.maxFinite,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  _field('步骤序号', orderCtrl, keyboard: TextInputType.number),
                  _field('标题', titleCtrl),
                  _field('位置名称', locationCtrl),
                  Row(children: [
                    Expanded(child: _field('纬度 lat', latCtrl, keyboard: const TextInputType.numberWithOptions(decimal: true, signed: true))),
                    const SizedBox(width: 8),
                    Expanded(child: _field('经度 lng', lngCtrl, keyboard: const TextInputType.numberWithOptions(decimal: true, signed: true))),
                  ]),
                  _field('时长（如：约 15 分钟）', durationCtrl),
                  _field('任务说明', taskCtrl, maxLines: 2),
                  _field('所需材料', materialsCtrl),
                  _field('联系方式', contactCtrl),
                  _field('备注', noteCtrl, maxLines: 2),
                  DropdownButtonFormField<String>(
                    value: iconName,
                    decoration: const InputDecoration(labelText: '图标'),
                    items: const [
                      DropdownMenuItem(value: 'login', child: Text('入校')),
                      DropdownMenuItem(value: 'account_balance', child: Text('学院')),
                      DropdownMenuItem(value: 'payments', child: Text('缴费')),
                      DropdownMenuItem(value: 'bed', child: Text('宿舍')),
                      DropdownMenuItem(value: 'credit_card', child: Text('校园卡')),
                      DropdownMenuItem(value: 'health_and_safety', child: Text('体检')),
                      DropdownMenuItem(value: 'place', child: Text('默认')),
                    ],
                    onChanged: (v) => setSt(() => iconName = v ?? 'place'),
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
            FilledButton(
              onPressed: () async {
                final record = CampusStepRecord(
                  id: step?.id,
                  campusId: widget.campusId,
                  stepOrder: int.tryParse(orderCtrl.text) ?? 1,
                  title: titleCtrl.text.trim(),
                  location: locationCtrl.text.trim(),
                  lat: double.tryParse(latCtrl.text) ?? 0,
                  lng: double.tryParse(lngCtrl.text) ?? 0,
                  duration: durationCtrl.text.trim(),
                  task: taskCtrl.text.trim(),
                  materials: materialsCtrl.text.trim(),
                  contact: contactCtrl.text.trim(),
                  note: noteCtrl.text.trim(),
                  iconName: iconName,
                );
                if (record.title.isEmpty || record.location.isEmpty) {
                  _toast('标题和位置名称不能为空');
                  return;
                }
                try {
                  if (isCreate) {
                    final resp = await _api.post(ApiConfig.adminCampusSteps,
                        data: record.toRequestJson());
                    if (resp.data['code'] != 0) {
                      _toast(resp.data['message'] ?? '创建失败');
                      return;
                    }
                  } else {
                    final resp = await _api.put(
                        ApiConfig.adminCampusStep(record.id.toString()),
                        data: record.toRequestJson());
                    if (resp.data['code'] != 0) {
                      _toast(resp.data['message'] ?? '更新失败');
                      return;
                    }
                  }
                  if (ctx.mounted) Navigator.pop(ctx, true);
                } catch (e) {
                  _toast('网络错误：$e');
                }
              },
              child: Text(isCreate ? '创建' : '保存'),
            ),
          ],
        ),
      ),
    );
    return result;
  }

  Widget _field(String label, TextEditingController ctrl,
      {TextInputType? keyboard, int maxLines = 1}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: TextField(
        controller: ctrl,
        decoration: InputDecoration(
          labelText: label,
          isDense: true,
          border: const OutlineInputBorder(),
        ),
        keyboardType: keyboard,
        maxLines: maxLines,
      ),
    );
  }
}
