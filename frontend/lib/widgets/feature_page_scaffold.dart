import 'package:flutter/material.dart';

import 'error_view.dart';

/// 学生功能页统一骨架（P4-b）。
///
/// 收敛 pages/student/ 约 20 个页面的重复三态结构：
/// Scaffold + AppBar + RefreshIndicator + loading/error/content 三态切换。
///
/// 用法：
/// ```dart
/// FeaturePageScaffold(
///   title: '热点关注',
///   loading: provider.loading,
///   error: provider.error.isEmpty ? null : provider.error,
///   onRefresh: provider.fetchHotTopics,
///   contentBuilder: (_) => ListView(...),
/// )
/// ```
class FeaturePageScaffold extends StatelessWidget {
  const FeaturePageScaffold({
    super.key,
    required this.title,
    required this.loading,
    required this.onRefresh,
    required this.contentBuilder,
    this.error,
    this.empty,
  });

  /// AppBar 标题
  final String title;

  /// 加载态（true 时显示居中转圈）
  final bool loading;

  /// 错误信息（非空显示 ErrorView + 重试；忽略空白串）
  final String? error;

  /// 下拉刷新回调（同时也是错误重试的默认回调）
  final Future<void> Function() onRefresh;

  /// 正常内容（loading/error 均不成立时展示）
  final WidgetBuilder contentBuilder;

  /// 空数据提示（可选；由调用方在 contentBuilder 内部判断更灵活，此处保留供简单页使用）
  final String? empty;

  @override
  Widget build(BuildContext context) {
    final hasError = error != null && error!.trim().isNotEmpty;
    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: RefreshIndicator(
        onRefresh: onRefresh,
        child: loading
            ? const Center(child: CircularProgressIndicator())
            : hasError
                ? ErrorView.error(message: error!, onRetry: onRefresh)
                : Builder(builder: contentBuilder),
      ),
    );
  }
}
