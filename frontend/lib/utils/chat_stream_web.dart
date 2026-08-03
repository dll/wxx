import 'dart:convert';
import 'dart:html';
import '../config/api_config.dart';
import '../utils/storage.dart';

/// 发起 Web 端流式对话（SSE via XHR）。
/// 返回取消函数（abort）。
Function streamChat({
  required String question,
  String? sessionId,
  String? agentId,
  void Function(String delta)? onDelta,
  void Function(Map<String, dynamic> done)? onDone,
  void Function(String error)? onError,
}) {
  final req = HttpRequest();
  req.open('POST', ApiConfig.chatStream);
  req.setRequestHeader('Content-Type', 'application/json');
  final token = Storage.token;
  if (token != null && token.isNotEmpty) {
    req.setRequestHeader('Authorization', 'Bearer $token');
  }

  // 增量解析 SSE 块（data: {...}\n\n）
  var buffer = '';
  void parse(String text) {
    buffer += text;
    int idx;
    while ((idx = buffer.indexOf('\n\n')) != -1) {
      final block = buffer.substring(0, idx);
      buffer = buffer.substring(idx + 2);
      if (!block.startsWith('data: ')) continue;
      final payload = block.substring(6).trim();
      if (payload.isEmpty) continue;
      try {
        final obj = jsonDecode(payload) as Map<String, dynamic>;
        final type = obj['type'];
        if (type == 'delta') {
          onDelta?.call(obj['delta']?.toString() ?? '');
        } else if (type == 'done') {
          onDone?.call(obj);
        } else if (type == 'error') {
          onError?.call(obj['message']?.toString() ?? '流式应答失败');
        }
      } catch (_) {
        // 忽略单个坏块
      }
    }
  }

  req.onProgress.listen((_) => parse(req.responseText ?? ''));
  req.onLoadEnd.listen((_) => parse(req.responseText ?? ''));
  req.onError.listen((_) => onError?.call('网络错误'));

  req.send(jsonEncode({
    'question': question,
    if (sessionId != null && sessionId.isNotEmpty) 'session_id': sessionId,
    if (agentId != null && agentId.isNotEmpty) 'agent_id': agentId,
  }));

  return req.abort;
}
