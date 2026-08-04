import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

/// Android / iOS 端：用 webview_flutter 内嵌显示目标站点（不新开浏览器）。
class InlineWebPage extends StatefulWidget {
  final String url;
  const InlineWebPage({super.key, required this.url});

  @override
  State<InlineWebPage> createState() => _InlineWebPageState();
}

class _InlineWebPageState extends State<InlineWebPage> {
  late final WebViewController _controller;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setNavigationDelegate(
        NavigationDelegate(onPageFinished: (_) {
          if (mounted) setState(() => _loading = false);
        }),
      )
      ..loadRequest(Uri.parse(widget.url));
  }

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        WebViewWidget(controller: _controller),
        if (_loading)
          Container(
            color: Theme.of(context).colorScheme.surface,
            alignment: Alignment.center,
            child: const CircularProgressIndicator(),
          ),
      ],
    );
  }
}
