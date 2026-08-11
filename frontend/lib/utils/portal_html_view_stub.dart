import 'package:flutter/widgets.dart';

/// 非 Web 平台：门户内嵌浏览暂不支持（移动端可在个人中心直接外链打开门户）
Widget buildPortalHtmlView(String html) {
  return const Center(child: Text('当前平台暂不支持内嵌门户浏览'));
}
