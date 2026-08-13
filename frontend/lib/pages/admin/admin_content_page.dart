import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../widgets/error_view.dart';

/// 管理端内容管理页：毕设选题 / 就业指导 / 学科竞赛
/// 供学校/学院管理员对各类内容做增删改查。
class AdminContentPage extends StatefulWidget {
  const AdminContentPage({super.key});

  @override
  State<AdminContentPage> createState() => _AdminContentPageState();
}

enum _TabKey { thesis, career, competition }

class _AdminContentPageState extends State<AdminContentPage>
    with SingleTickerProviderStateMixin {
  late TabController _tabController;
  final ApiService _api = ApiService();
  List<dynamic> _thesisTopics = [];
  List<dynamic> _careerPolicies = [];
  List<dynamic> _competitions = [];
  bool _loading = false;

  static const _tabs = [
    _TabItem(_TabKey.thesis, '毕设选题'),
    _TabItem(_TabKey.career, '就业政策'),
    _TabItem(_TabKey.competition, '学科竞赛'),
  ];

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: _tabs.length, vsync: this);
    _loadAll();
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  Future<void> _loadAll() async {
    setState(() => _loading = true);
    await Future.wait([
      _loadThesisTopics(),
      _loadCareerPolicies(),
      _loadCompetitions(),
    ]);
    if (mounted) setState(() => _loading = false);
  }

  Future<void> _loadThesisTopics() async {
    try {
      final res = await _api.get(ApiConfig.graduationTopics);
      if (res.data is Map && (res.data as Map)['code'] == 0) {
        final data = (res.data as Map)['data'];
        if (data is List) {
          setState(() => _thesisTopics = data.whereType<Map>().toList());
        }
      }
    } catch (_) {}
  }

  Future<void> _loadCareerPolicies() async {
    try {
      final res = await _api.get(ApiConfig.adminCareerPolicies);
      if (res.data is Map && (res.data as Map)['code'] == 0) {
        final data = (res.data as Map)['data'];
        if (data is List) {
          setState(() => _careerPolicies = data.whereType<Map>().toList());
        }
      }
    } catch (_) {}
  }

  Future<void> _loadCompetitions() async {
    try {
      final res = await _api.get(ApiConfig.adminCompetitionList);
      if (res.data is Map && (res.data as Map)['code'] == 0) {
        final data = (res.data as Map)['data'];
        if (data is List) {
          setState(() => _competitions = data.whereType<Map>().toList());
        }
      }
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('内容管理'),
        bottom: TabBar(
          controller: _tabController,
          tabs: _tabs.map((t) => Tab(text: t.label)).toList(),
        ),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabController,
              children: [
                _buildThesisTab(context),
                _buildCareerTab(context),
                _buildCompetitionTab(context),
              ],
            ),
    );
  }

  // ── 毕设选题 ──
  Widget _buildThesisTab(BuildContext context) {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              Expanded(
                child: FilledButton.icon(
                  onPressed: () => _addThesisTopic(context),
                  icon: const Icon(Icons.add),
                  label: const Text('新增选题'),
                ),
              ),
            ],
          ),
        ),
        Expanded(
          child: _thesisTopics.isEmpty
              ? ErrorView.empty(message: '暂无选题', icon: Icons.menu_book)
              : RefreshIndicator(
                  onRefresh: _loadAll,
                  child: ListView.builder(
                    itemCount: _thesisTopics.length,
                    itemBuilder: (_, i) {
                      final t = _thesisTopics[i] as Map<String, dynamic>;
                      return Card(
                        margin: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 4),
                        child: ListTile(
                          title: Text(t['title']?.toString() ?? ''),
                          subtitle: Text(
                            '类型:${t['topic_type'] ?? ''} · 状态:${t['status'] ?? ''}',
                          ),
                          trailing: IconButton(
                            icon: const Icon(Icons.delete_outline),
                            onPressed: () => _deleteThesisTopic(
                                context, t['id']),
                          ),
                        ),
                      );
                    },
                  ),
                ),
        ),
      ],
    );
  }

  Future<void> _addThesisTopic(BuildContext context) async {
    final titleCtrl = TextEditingController();
    final advisorCtrl = TextEditingController();
    final collegeCtrl = TextEditingController();
    final descCtrl = TextEditingController();
    final batchCtrl = TextEditingController(text: '2026');
    await showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('新增毕设选题'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(controller: titleCtrl,
                  decoration: const InputDecoration(labelText: '选题标题')),
              TextField(controller: advisorCtrl,
                  decoration: const InputDecoration(labelText: '指导教师ID')),
              TextField(controller: collegeCtrl,
                  decoration: const InputDecoration(labelText: '学院')),
              TextField(controller: batchCtrl,
                  decoration: const InputDecoration(labelText: '届别')),
              TextField(controller: descCtrl, maxLines: 3,
                  decoration: const InputDecoration(labelText: '选题简介')),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('取消')),
          FilledButton(
            onPressed: () async {
              if (titleCtrl.text.trim().isEmpty) return;
              Navigator.pop(ctx);
              try {
                await _api.post(ApiConfig.graduationAdminTopics, data: {
                  'title': titleCtrl.text.trim(),
                  'advisor_id':
                      int.tryParse(advisorCtrl.text.trim()) ?? 1,
                  'college': collegeCtrl.text.trim(),
                  'batch': int.tryParse(batchCtrl.text.trim()) ?? 2026,
                  'description': descCtrl.text.trim(),
                  'status': 'active',
                });
                await _loadAll();
              } catch (_) {
                if (context.mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('新增失败')));
                }
              }
            },
            child: const Text('保存'),
          ),
        ],
      ),
    );
  }

  Future<void> _deleteThesisTopic(BuildContext context, dynamic id) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('删除选题'),
        content: const Text('确定删除该毕设选题吗？此操作不可恢复。'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('取消')),
          TextButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('删除')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.delete('${ApiConfig.graduationAdminTopics}/$id');
      await _loadAll();
    } catch (_) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('删除失败')));
      }
    }
  }

  // ── 就业政策 ──
  Widget _buildCareerTab(BuildContext context) {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              Expanded(
                child: FilledButton.icon(
                  onPressed: () => _addCareerPolicy(context),
                  icon: const Icon(Icons.add),
                  label: const Text('新增就业政策'),
                ),
              ),
            ],
          ),
        ),
        Expanded(
          child: _careerPolicies.isEmpty
              ? ErrorView.empty(message: '暂无就业政策', icon: Icons.work_outline)
              : RefreshIndicator(
                  onRefresh: _loadAll,
                  child: ListView.builder(
                    itemCount: _careerPolicies.length,
                    itemBuilder: (_, i) {
                      final p = _careerPolicies[i] as Map<String, dynamic>;
                      return Card(
                        margin: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 4),
                        child: ListTile(
                          title: Text(p['title']?.toString() ?? ''),
                          subtitle: Text(
                              '${p['category'] ?? ''} · ${p['status'] ?? ''}'),
                          trailing: IconButton(
                            icon: const Icon(Icons.delete_outline),
                            onPressed: () =>
                                _deleteCareerPolicy(context, p['id']),
                          ),
                        ),
                      );
                    },
                  ),
                ),
        ),
      ],
    );
  }

  Future<void> _addCareerPolicy(BuildContext context) async {
    final titleCtrl = TextEditingController();
    final contentCtrl = TextEditingController();
    final categoryCtrl =
        TextEditingController(text: 'employment_policy');
    await showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('新增就业政策'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(controller: titleCtrl,
                  decoration: const InputDecoration(labelText: '政策标题')),
              TextField(controller: categoryCtrl,
                  decoration: const InputDecoration(
                      labelText: '分类（employment_policy 等）')),
              TextField(controller: contentCtrl, maxLines: 5,
                  decoration: const InputDecoration(labelText: '政策内容')),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('取消')),
          FilledButton(
            onPressed: () async {
              if (titleCtrl.text.trim().isEmpty) return;
              Navigator.pop(ctx);
              try {
                await _api.post(ApiConfig.adminCareerPolicies, data: {
                  'title': titleCtrl.text.trim(),
                  'category': categoryCtrl.text.trim(),
                  'content': contentCtrl.text.trim(),
                  'status': 'active',
                });
                await _loadAll();
              } catch (_) {
                if (context.mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('新增失败')));
                }
              }
            },
            child: const Text('保存'),
          ),
        ],
      ),
    );
  }

  Future<void> _deleteCareerPolicy(BuildContext context, dynamic id) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('删除就业政策'),
        content: const Text('确定删除该就业政策吗？'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('取消')),
          TextButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('删除')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.delete('${ApiConfig.adminCareerPolicies}/$id');
      await _loadAll();
    } catch (_) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('删除失败')));
      }
    }
  }

  // ── 学科竞赛 ──
  Widget _buildCompetitionTab(BuildContext context) {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              Expanded(
                child: FilledButton.icon(
                  onPressed: () => _addCompetition(context),
                  icon: const Icon(Icons.add),
                  label: const Text('新增竞赛'),
                ),
              ),
            ],
          ),
        ),
        Expanded(
          child: _competitions.isEmpty
              ? ErrorView.empty(message: '暂无竞赛', icon: Icons.emoji_events)
              : RefreshIndicator(
                  onRefresh: _loadAll,
                  child: ListView.builder(
                    itemCount: _competitions.length,
                    itemBuilder: (_, i) {
                      final c = _competitions[i] as Map<String, dynamic>;
                      return Card(
                        margin: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 4),
                        child: ListTile(
                          title: Text(c['name']?.toString() ?? ''),
                          subtitle: Text(
                              '${c['level'] ?? ''} · ${c['status'] ?? ''}'),
                          trailing: IconButton(
                            icon: const Icon(Icons.delete_outline),
                            onPressed: () =>
                                _deleteCompetition(context, c['id']),
                          ),
                        ),
                      );
                    },
                  ),
                ),
        ),
      ],
    );
  }

  Future<void> _addCompetition(BuildContext context) async {
    final nameCtrl = TextEditingController();
    final levelCtrl = TextEditingController(text: 'school');
    final categoryCtrl = TextEditingController(text: 'other');
    final descCtrl = TextEditingController();
    await showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('新增竞赛'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(controller: nameCtrl,
                  decoration: const InputDecoration(labelText: '竞赛名称')),
              TextField(controller: levelCtrl,
                  decoration: const InputDecoration(
                      labelText: '级别（national/provincial/school）')),
              TextField(controller: categoryCtrl,
                  decoration: const InputDecoration(
                      labelText: '类别（programming/math/other）')),
              TextField(controller: descCtrl, maxLines: 3,
                  decoration: const InputDecoration(labelText: '竞赛简介')),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('取消')),
          FilledButton(
            onPressed: () async {
              if (nameCtrl.text.trim().isEmpty) return;
              Navigator.pop(ctx);
              try {
                await _api.post(ApiConfig.adminCompetition, data: {
                  'name': nameCtrl.text.trim(),
                  'level': levelCtrl.text.trim(),
                  'category': categoryCtrl.text.trim(),
                  'description': descCtrl.text.trim(),
                  'status': 'upcoming',
                });
                await _loadAll();
              } catch (_) {
                if (context.mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('新增失败')));
                }
              }
            },
            child: const Text('保存'),
          ),
        ],
      ),
    );
  }

  Future<void> _deleteCompetition(BuildContext context, dynamic id) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('删除竞赛'),
        content: const Text('确定删除该竞赛吗？'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('取消')),
          TextButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('删除')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.delete('${ApiConfig.adminCompetition}/$id');
      await _loadAll();
    } catch (_) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('删除失败')));
      }
    }
  }
}

class _TabItem {
  final _TabKey key;
  final String label;
  const _TabItem(this.key, this.label);
}
