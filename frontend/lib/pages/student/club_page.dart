import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_new_features_provider.dart';

/// 社团生活页面
class ClubPage extends StatefulWidget {
  const ClubPage({super.key});
  @override
  State<ClubPage> createState() => _ClubPageState();
}

class _ClubPageState extends State<ClubPage> with SingleTickerProviderStateMixin {
  late TabController _tabCtrl;

  @override
  void initState() {
    super.initState();
    _tabCtrl = TabController(length: 2, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<StudentNewFeaturesProvider>();
      p.fetchClubs();
      p.fetchMyClubs();
      p.fetchClubActivities();
    });
  }

  @override
  void dispose() {
    _tabCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('社团生活'),
        bottom: TabBar(
          controller: _tabCtrl,
          tabs: const [Tab(text: '社团列表'), Tab(text: '我的社团')],
        ),
      ),
      body: Consumer<StudentNewFeaturesProvider>(
        builder: (_, p, __) {
          if (p.loading) return const Center(child: CircularProgressIndicator());
          return TabBarView(
            controller: _tabCtrl,
            children: [_buildClubsTab(context, p), _buildMyClubsTab(context, p)],
          );
        },
      ),
    );
  }

  Widget _buildClubsTab(BuildContext context, StudentNewFeaturesProvider p) {
    return CustomScrollView(
      slivers: [
        if (p.clubs.isNotEmpty) SliverToBoxAdapter(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
            child: Text('社团列表', style: Theme.of(context).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
          ),
        ),
        if (p.clubs.isEmpty) const SliverFillRemaining(child: Center(child: Text('暂无社团'))),
        SliverList(
          delegate: SliverChildBuilderDelegate(
            (_, i) {
              final c = p.clubs[i];
              return Card(
                margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                child: ListTile(
                  leading: CircleAvatar(
                    child: Text(_categoryIcon(c.category)),
                  ),
                  title: Text(c.name),
                  subtitle: Text('${c.description}\n${c.memberCount} 人', maxLines: 2, overflow: TextOverflow.ellipsis),
                  isThreeLine: true,
                  trailing: ElevatedButton(
                    onPressed: () async {
                      final ok = await p.joinClub(c.id);
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(
                          SnackBar(content: Text(ok ? '加入成功' : p.error)),
                        );
                      }
                    },
                    style: ElevatedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                      minimumSize: Size.zero,
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    ),
                    child: const Text('加入', style: TextStyle(fontSize: 12)),
                  ),
                ),
              );
            },
            childCount: p.clubs.length,
          ),
        ),
        if (p.clubActivities.isNotEmpty) SliverToBoxAdapter(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
            child: Text('社团活动', style: Theme.of(context).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
          ),
        ),
        SliverList(
          delegate: SliverChildBuilderDelegate(
            (_, i) {
              final a = p.clubActivities[i];
              return Card(
                margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                child: ListTile(
                  leading: Icon(Icons.event_outlined, color: Theme.of(context).colorScheme.primary),
                  title: Text(a.title),
                  subtitle: Text('${a.clubName} | ${a.location}\n${a.startTime}',
                      maxLines: 2, overflow: TextOverflow.ellipsis),
                  isThreeLine: true,
                  trailing: ElevatedButton(
                    onPressed: () async {
                      final ok = await context.read<StudentNewFeaturesProvider>().registerClubActivity(a.id);
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(
                          SnackBar(content: Text(ok ? '报名成功' : '报名失败')),
                        );
                      }
                    },
                    style: ElevatedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                      minimumSize: Size.zero,
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    ),
                    child: const Text('报名', style: TextStyle(fontSize: 12)),
                  ),
                ),
              );
            },
            childCount: p.clubActivities.length,
          ),
        ),
      ],
    );
  }

  Widget _buildMyClubsTab(BuildContext context, StudentNewFeaturesProvider p) {
    if (p.myClubs.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.group_outlined, size: 64, color: Theme.of(context).colorScheme.outline),
            const SizedBox(height: 16),
            Text('尚未加入任何社团', style: Theme.of(context).textTheme.titleMedium),
          ],
        ),
      );
    }
    return ListView.builder(
      padding: const EdgeInsets.all(12),
      itemCount: p.myClubs.length,
      itemBuilder: (_, i) {
        final c = p.myClubs[i];
        return Card(
          margin: const EdgeInsets.only(bottom: 8),
          child: ListTile(
            leading: CircleAvatar(child: Text(_categoryIcon(c.category))),
            title: Text(c.name),
            subtitle: Text(c.description, maxLines: 1, overflow: TextOverflow.ellipsis),
            trailing: const Icon(Icons.check_circle, color: Colors.green),
          ),
        );
      },
    );
  }

  String _categoryIcon(String category) {
    switch (category) {
      case 'technology': return '科';
      case 'sports': return '体';
      case 'arts': return '艺';
      case 'academic': return '学';
      case 'public welfare': return '益';
      default: return '社';
    }
  }
}
