// D1-1 书记 party/collab 专项可视化页 回归测试（纯新增，不改既有生产代码/断言语义）。
//
// 覆盖：
// 1. 两个专项页（PartyDashboardPage / CollabDashboardPage）渲染「空数据」态
//    （data 为空 → ErrorView.empty「数据待充实」，不伪造数字）。
// 2. 共享区块（PartyDashboardSection / CollabDashboardSection）渲染「数据」态
//    （传入 Map 数据 → 统计行 / 阶段 chip / DataSrcBadge），验证诚实数据来源标注。
//
// 通过测试专用 Noop subclass 覆盖 fetch，避免在单测中发起真实网络请求，
// 保证确定性；未改动任何生产文件。

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';

import 'package:wxx_app/pages/secretary/party_dashboard_page.dart';
import 'package:wxx_app/pages/secretary/collab_dashboard_page.dart';
import 'package:wxx_app/providers/secretary_provider.dart';
import 'package:wxx_app/widgets/secretary_dashboard_sections.dart';

/// 测试专用：覆盖 fetch，不做任何事，使页面停在「空数据」初始态。
class _NoopSecretaryProvider extends SecretaryProvider {
  @override
  Future<void> fetchPartyDashboard() async {}

  @override
  Future<void> fetchCollabDashboard() async {}

  @override
  Future<void> fetchDashboard({String college = ''}) async {}
}

void main() {
  testWidgets('PartyDashboardPage 空数据态：显示「数据待充实」，不伪造数值', (tester) async {
    final provider = _NoopSecretaryProvider();
    await tester.pumpWidget(
      ChangeNotifierProvider<SecretaryProvider>.value(
        value: provider,
        child: const MaterialApp(home: PartyDashboardPage()),
      ),
    );
    await tester.pump(); // 处理 addPostFrameCallback

    expect(find.byType(PartyDashboardPage), findsOneWidget);
    // AppBar 标题
    expect(find.text('党建育人专项'), findsWidgets);
    // 空态提示
    expect(find.text('暂无党建育人数据（数据待充实）'), findsOneWidget);
    // 不应出现任何伪造的统计数值（PartyDashboardSection 空态返回空容器）
    expect(find.text('党建育人（思想政治）'), findsNothing);
    expect(find.text('入党申请总人数'), findsNothing);
  });

  testWidgets('CollabDashboardPage 空数据态：显示「数据待充实」，不伪造数值', (tester) async {
    final provider = _NoopSecretaryProvider();
    await tester.pumpWidget(
      ChangeNotifierProvider<SecretaryProvider>.value(
        value: provider,
        child: const MaterialApp(home: CollabDashboardPage()),
      ),
    );
    await tester.pump();

    expect(find.byType(CollabDashboardPage), findsOneWidget);
    expect(find.text('协同育人总览'), findsWidgets);
    expect(find.text('暂无协同育人数据（数据待充实）'), findsOneWidget);
    // 空态下不应出现伪造的统计
    expect(find.text('协同育人总览（教师/教辅付出）'), findsNothing);
    expect(find.text('本院学生数'), findsNothing);
  });

  testWidgets('PartyDashboardSection 数据态：渲染统计行/阶段chip/真实数据徽章', (tester) async {
    final party = <String, dynamic>{
      'members': {'member': 5, 'probation': 3},
      'stage_distribution': {'activist': 2, 'member': 5},
      'stage_total': 10,
      'study_records': 4,
      'study_hours': 12,
      'study_by_type': {'党课学习': {'count': 6}},
      'data_source': 'real',
    };
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: PartyDashboardSection(party: party),
          ),
        ),
      ),
    );

    expect(find.text('党建育人（思想政治）'), findsOneWidget);
    expect(find.textContaining('入党申请总人数'), findsOneWidget);
    expect(find.textContaining('正式党员'), findsWidgets); // 同时出现在成员数统计与阶段chip，合法
    expect(find.textContaining('预备党员'), findsOneWidget);
    expect(find.textContaining('党课/学习记录'), findsOneWidget);
    expect(find.textContaining('学习时长'), findsOneWidget);
    // 阶段中文 chip 与 study_by_type chip
    expect(find.textContaining('入党积极分子'), findsWidgets);
    expect(find.textContaining('党课学习'), findsWidgets);
    // 真实数据徽章
    expect(find.text('真实数据'), findsOneWidget);
  });

  testWidgets('CollabDashboardSection 数据态：渲染统计/角色chip/数据来源', (tester) async {
    final collab = <String, dynamic>{
      'students_total': 120,
      'talk_records': 30,
      'facility_records': 8,
      'party_registrations': 5,
      'course_schedules': 40,
      'by_role': {'counselor': 20, 'teacher': 15},
      'data_source': 'reference',
    };
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: CollabDashboardSection(collab: collab),
          ),
        ),
      ),
    );

    expect(find.text('协同育人总览（教师/教辅付出）'), findsOneWidget);
    expect(find.textContaining('本院学生数'), findsOneWidget);
    expect(find.textContaining('谈心记录'), findsOneWidget);
    expect(find.textContaining('后勤服务'), findsOneWidget);
    expect(find.textContaining('党课/活动登记'), findsOneWidget);
    expect(find.textContaining('教学课表节次'), findsOneWidget);
    // 角色 chip（中文名）
    expect(find.textContaining('辅导员'), findsWidgets);
    expect(find.textContaining('教师'), findsWidgets);
    // 参考/AI 数据来源徽章（非真实，诚实标注）
    expect(find.text('参考/AI'), findsOneWidget);
  });
}
