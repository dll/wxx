import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/api_error.dart';
import '../../widgets/data_src_badge.dart';

/// 后勤服务台（并入教辅角色）
///
/// 覆盖实验室开门/关门、教室保洁、热水供应、宿舍晚归查岗、校园环卫、
/// 图书馆借阅等后勤保障工作。所有记录均为操作人手动登记的真实数据
/// （data_source=real），无数据时诚实展示为空，不提供示例/编造记录。
class FacilityWorkbenchPage extends StatefulWidget {
  const FacilityWorkbenchPage({super.key});
  @override
  State<FacilityWorkbenchPage> createState() => _FacilityWorkbenchPageState();
}

class _FacilityWorkbenchPageState extends State<FacilityWorkbenchPage>
    with SingleTickerProviderStateMixin {
  final ApiService _api = ApiService();

  late final TabController _tab;
  bool _loading = true;
  String _error = '';

  // 岗位类型（key -> 可读名），来自后端 /facility/roles
  final Map<String, String> _roles = {};

  // 看板数据
  Map<String, dynamic> _dashboard = {};

  // 登记表单
  String? _formRole;
  final TextEditingController _titleCtrl = TextEditingController();
  final TextEditingController _locCtrl = TextEditingController();
  final TextEditingController _detailCtrl = TextEditingController();
  final TextEditingController _stuCtrl = TextEditingController();
  bool _submitting = false;
  String _submitMsg = '';

  // 记录列表
  List<Map<String, dynamic>> _records = [];
  String _listSrc = '';

  static const Map<String, String> _roleFallback = {
    'lab': '实验室开门/关门',
    'clean': '教室保洁卫生',
    'hotwater': '热水供应',
    'dorm': '宿舍晚归查岗',
    'envir': '校园环卫学习环境',
    'library': '图书馆借阅管理',
  };

  @override
  void initState() {
    super.initState();
    _tab = TabController(length: 3, vsync: this);
    _loadAll();
  }

  Future<void> _loadAll() async {
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      await Future.wait([_loadRoles(), _loadDashboard(), _loadRecords()]);
    } catch (e) {
      if (mounted) setState(() => _error = friendlyApiError(e));
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _loadRoles() async {
    try {
      final res = await _api.get(ApiConfig.facilityRoles);
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map<String, dynamic> ? res.data : {};
        final roles = (data['roles'] as Map?)?.cast<String, dynamic>() ?? {};
        setState(() {
          _roles.clear();
          roles.forEach((k, v) => _roles[k] = v.toString());
        });
      }
    } catch (_) {
      // 拉取失败用前端兜底常量，不阻断看板
    }
  }

  Future<void> _loadDashboard() async {
    final res = await _api.get(ApiConfig.facilityDashboard);
    if (res.statusCode == 200 && res.data != null) {
      final data = res.data is Map<String, dynamic> ? res.data : {};
      if (mounted) setState(() => _dashboard = data);
    }
  }

  Future<void> _loadRecords() async {
    final res = await _api.get(ApiConfig.facilityRecords);
    if (res.statusCode == 200 && res.data != null) {
      final data = res.data is Map<String, dynamic> ? res.data : {};
      if (mounted) {
        setState(() {
          _records = (data['records'] as List?)?.cast<Map<String, dynamic>>() ??
              [];
          _listSrc = data['data_source']?.toString() ?? '';
        });
      }
    }
  }

  String _roleName(String key) {
    if (_roles.containsKey(key)) return _roles[key]!;
    return _roleFallback[key] ?? key;
  }

  Future<void> _submit() async {
    final role = _formRole;
    final title = _titleCtrl.text.trim();
    if (role == null || title.isEmpty) {
      setState(() => _submitMsg = '请选择岗位类型并填写事项简述');
      return;
    }
    setState(() {
      _submitting = true;
      _submitMsg = '';
    });
    try {
      final res = await _api.post(
        ApiConfig.facilityRecord,
        data: {
          'role': role,
          'title': title,
          'location': _locCtrl.text.trim(),
          'detail': _detailCtrl.text.trim(),
          // 关联学生：支持「姓名」或「学号」的简单录入，后端存姓名
          'student_name': _stuCtrl.text.trim(),
        },
      );
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map<String, dynamic> ? res.data : {};
        setState(() {
          _submitMsg = (data['message'] ?? '登记成功').toString();
          _titleCtrl.clear();
          _locCtrl.clear();
          _detailCtrl.clear();
          _stuCtrl.clear();
          _formRole = null;
        });
        // 刷新看板与列表
        await Future.wait([_loadDashboard(), _loadRecords()]);
      } else {
        setState(() => _submitMsg = '登记失败（HTTP ${res.statusCode}）');
      }
    } catch (e) {
      setState(() => _submitMsg = friendlyApiError(e));
    } finally {
      setState(() => _submitting = false);
    }
  }

  @override
  void dispose() {
    _tab.dispose();
    _titleCtrl.dispose();
    _locCtrl.dispose();
    _detailCtrl.dispose();
    _stuCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('后勤服务台'),
        bottom: TabBar(
          controller: _tab,
          tabs: const [
            Tab(text: '登记'),
            Tab(text: '今日看板'),
            Tab(text: '服务记录'),
          ],
        ),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error.isNotEmpty
              ? Center(
                  child: Text(_error,
                      style: TextStyle(color: theme.colorScheme.error)))
              : TabBarView(controller: _tab, children: [
                  _buildRegister(theme),
                  _buildDashboard(theme),
                  _buildRecords(theme),
                ]),
    );
  }

  // ── 登记表单 ──
  Widget _buildRegister(ThemeData theme) {
    final roleKeys = _roles.isNotEmpty ? _roles.keys.toList() : _roleFallback.keys.toList();
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        const DataSrcBadge(src: 'real'),
        const SizedBox(height: 4),
        Text('手动登记一次后勤服务，数据为真实记录（非示例）。',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: theme.colorScheme.outline)),
        const SizedBox(height: 16),
        DropdownButtonFormField<String>(
          value: _formRole,
          decoration: const InputDecoration(
              labelText: '岗位类型', border: OutlineInputBorder()),
          items: roleKeys
              .map((k) => DropdownMenuItem(value: k, child: Text(_roleName(k))))
              .toList(),
          onChanged: (v) => setState(() => _formRole = v),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _titleCtrl,
          decoration: const InputDecoration(
              labelText: '事项简述 *', hintText: '如：实验楼A 301 开门',
              border: OutlineInputBorder()),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _locCtrl,
          decoration: const InputDecoration(
              labelText: '地点', hintText: '如：实验楼A / 3号宿舍楼 / 图书馆2楼',
              border: OutlineInputBorder()),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _detailCtrl,
          maxLines: 3,
          decoration: const InputDecoration(
              labelText: '详情 / 备注', hintText: '数量、巡检结果、查岗情况等',
              border: OutlineInputBorder()),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _stuCtrl,
          decoration: const InputDecoration(
              labelText: '关联学生（可选）', hintText: '如：张同学借阅 / 305 查岗',
              border: OutlineInputBorder()),
        ),
        const SizedBox(height: 16),
        FilledButton.icon(
          onPressed: _submitting ? null : _submit,
          icon: _submitting
              ? const SizedBox(
                  width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
              : const Icon(Icons.add_task),
          label: const Text('登记后勤服务'),
        ),
        if (_submitMsg.isNotEmpty) ...[
          const SizedBox(height: 12),
          Text(_submitMsg,
              style: TextStyle(
                  color: _submitMsg.contains('成功') ? Colors.green : theme.colorScheme.error)),
        ],
      ],
    );
  }

  // ── 今日看板 ──
  Widget _buildDashboard(ThemeData theme) {
    final byRole = (_dashboard['by_role'] as Map?)?.cast<String, dynamic>() ?? {};
    final read = (_dashboard['role_readable'] as Map?)?.cast<String, dynamic>() ?? {};
    final total = (_dashboard['total'] ?? 0) as int;
    final studentServed = (_dashboard['student_served'] ?? 0) as int;
    final keys = byRole.keys.toList().isEmpty
        ? _roleFallback.keys.toList()
        : byRole.keys.toList();

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Row(children: [
          _statCard(theme, '$total', '服务总次数', Icons.build),
          const SizedBox(width: 12),
          _statCard(theme, '$studentServed', '关联学生数', Icons.people),
        ]),
        const SizedBox(height: 16),
        const DataSrcBadge(src: 'real'),
        const SizedBox(height: 8),
        Text('各岗位今日服务（无记录岗位显示 0，诚实呈现）',
            style: theme.textTheme.bodySmall),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: keys.map((k) {
            final cnt = byRole[k] as int? ?? 0;
            final name = read[k]?.toString() ?? _roleName(k);
            return Chip(
              avatar: Icon(Icons.check_circle,
                  size: 18,
                  color: cnt > 0 ? Colors.green : theme.colorScheme.outline),
              label: Text('$name · $cnt'),
              visualDensity: VisualDensity.compact,
            );
          }).toList(),
        ),
      ],
    );
  }

  Widget _statCard(ThemeData theme, String value, String label, IconData icon) {
    return Expanded(
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(children: [
            Icon(icon, color: theme.colorScheme.primary),
            const SizedBox(height: 8),
            Text(value,
                style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold)),
            Text(label, style: theme.textTheme.bodySmall),
          ]),
        ),
      ),
    );
  }

  // ── 服务记录列表 ──
  Widget _buildRecords(ThemeData theme) {
    if (_records.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            Icon(Icons.inventory_2_outlined, size: 48,
                color: theme.colorScheme.outline),
            const SizedBox(height: 12),
            const Text('暂无后勤服务记录'),
            const SizedBox(height: 4),
            Text('登记后勤服务后此处显示真实记录',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.outline)),
          ]),
        ),
      );
    }
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        DataSrcBadge(src: _listSrc),
        const SizedBox(height: 8),
        ..._records.map((r) => Card(
              margin: const EdgeInsets.only(bottom: 8),
              child: ListTile(
                leading: CircleAvatar(
                  child: Text(_roleName(r['role']?.toString() ?? '?')
                      .characters
                      .first),
                ),
                title: Text(r['title']?.toString() ?? ''),
                subtitle: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(_roleName(r['role']?.toString() ?? '') +
                        ((r['location']?.toString() ?? '').isEmpty
                            ? ''
                            : ' · ${r['location']}')),
                    if ((r['detail'] ?? '').toString().isNotEmpty)
                      Text(r['detail'].toString()),
                    if ((r['student_name'] ?? '').toString().isNotEmpty)
                      Text('关联学生：${r['student_name']}'),
                    Text('${r['operator_name']} · ${r['occurred_at']}',
                        style: theme.textTheme.bodySmall),
                  ],
                ),
              ),
            )),
      ],
    );
  }
}
