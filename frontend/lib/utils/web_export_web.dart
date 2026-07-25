import 'dart:html' as html;

void openHtmlInNewTab(String htmlContent) {
  final blob = html.Blob([htmlContent], 'text/html');
  final url = html.Url.createObjectUrl(blob);
  html.window.open(url, '_blank');
  Future.delayed(const Duration(seconds: 10), () {
    html.Url.revokeObjectUrl(url);
  });
}
