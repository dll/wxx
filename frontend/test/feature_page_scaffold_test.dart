import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:wxx_app/widgets/feature_page_scaffold.dart';

Widget _host(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('加载态显示转圈且不渲染内容', (tester) async {
    await tester.pumpWidget(_host(FeaturePageScaffold(
      title: '测试页',
      loading: true,
      onRefresh: () async {},
      contentBuilder: (_) => const Text('内容'),
    )));
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(find.text('内容'), findsNothing);
  });

  testWidgets('错误态显示 ErrorView 与重试', (tester) async {
    var retried = false;
    await tester.pumpWidget(_host(FeaturePageScaffold(
      title: '测试页',
      loading: false,
      error: '网络异常',
      onRefresh: () async => retried = true,
      contentBuilder: (_) => const Text('内容'),
    )));
    expect(find.text('网络异常'), findsOneWidget);
    expect(find.text('内容'), findsNothing);

    await tester.tap(find.text('重试'));
    await tester.pumpAndSettle();
    expect(retried, isTrue);
  });

  testWidgets('空白错误串视为无错误并渲染内容', (tester) async {
    await tester.pumpWidget(_host(FeaturePageScaffold(
      title: '测试页',
      loading: false,
      error: '   ',
      onRefresh: () async {},
      contentBuilder: (_) => const Text('内容'),
    )));
    expect(find.text('内容'), findsOneWidget);
    expect(find.text('重试'), findsNothing);
  });

  testWidgets('正常态渲染内容且支持下拉刷新', (tester) async {
    var refreshed = false;
    await tester.pumpWidget(_host(FeaturePageScaffold(
      title: '测试页',
      loading: false,
      onRefresh: () async => refreshed = true,
      contentBuilder: (_) => ListView(children: const [Text('内容')]),
    )));
    expect(find.text('内容'), findsOneWidget);

    await tester.fling(find.text('内容'), const Offset(0, 300), 1000);
    await tester.pumpAndSettle();
    expect(refreshed, isTrue);
  });
}
