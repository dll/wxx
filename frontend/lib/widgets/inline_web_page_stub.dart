import 'package:flutter/material.dart';

/// 其它平台占位：不支持内嵌网页时给出提示。
class InlineWebPage extends StatelessWidget {
  final String url;
  const InlineWebPage({super.key, required this.url});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Text('当前平台不支持内嵌网页：$url',
          style: Theme.of(context).textTheme.bodySmall),
    );
  }
}
