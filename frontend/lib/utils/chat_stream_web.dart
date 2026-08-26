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

  // XHR 的 responseText 在每次 progress 中都是“截至当前的完整累计响应”，
  // 不能反复把它整体追加到 buffer，否则 SSE 块会指数级重复并可能吞掉 done 事件。
  var buffer = '';
  var consumedLength = 0;
  var completed = false;

  void parse(String text) {
    buffer += text.replaceAll('\r\n', '\n');
    int idx;
    while ((idx = buffer.indexOf('\n\n')) != -1) {
      final block = buffer.substring(0, idx);
      buffer = buffer.substring(idx + 2);
      if (!block.startsWith('data:')) continue;
      final payload = block.substring(5).trim();
      if (payload.isEmpty) continue;
      try {
        final obj = jsonDecode(payload) as Map<String, dynamic>;
        final type = obj['type'];
        if (type == 'delta') {
          onDelta?.call(obj['delta']?.toString() ?? '');
        } else if (type == 'done' && !completed) {
          completed = true;
          onDone?.call(obj);
        } else if (type == 'error' && !completed) {
          completed = true;
          onError?.call(obj['message']?.toString() ?? '流式应答失败');
        }
      } catch (_) {
        // 忽略单个坏块，不影响后续事件
      }
    }
  }

  void consumeNewResponseText() {
    final text = req.responseText ?? '';
    // 极少数浏览器在请求重定向后可能重置 responseText。
    if (text.length < consumedLength) consumedLength = 0;
    if (text.length == consumedLength) return;
    parse(text.substring(consumedLength));
    consumedLength = text.length;
  }

  req.onProgress.listen((_) => consumeNewResponseText());
  req.onLoadEnd.listen((_) {
    consumeNewResponseText();
    if (!completed) {
      completed = true;
      final status = req.status ?? 0;
      if (status < 200 || status >= 300) {
        onError?.call('流式请求失败（HTTP $status）');
      } else {
        onError?.call('流式应答意外中断，请重试');
      }
    }
  });
  req.onError.listen((_) {
    if (!completed) {
      completed = true;
      onError?.call('网络错误');
    }
  });

  req.send(jsonEncode({
    'question': question,
    if (sessionId != null && sessionId.isNotEmpty) 'session_id': sessionId,
    if (agentId != null && agentId.isNotEmpty) 'agent_id': agentId,
  }));

  return req.abort;
}
