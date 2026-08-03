/// 移动端：无 SSE 消费实现（no-op），聊天走非流式 ask()
Function streamChat({
  required String question,
  String? sessionId,
  String? agentId,
  void Function(String delta)? onDelta,
  void Function(Map<String, dynamic> done)? onDone,
  void Function(String error)? onError,
}) {
  return () {};
}
