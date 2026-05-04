import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';

/// 对话状态管理
class ChatProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  final List<Message> _messages = [];
  String? _sessionId;
  bool _sending = false;
  String? _error;

  List<Message> get messages => List.unmodifiable(_messages);
  String? get sessionId => _sessionId;
  bool get sending => _sending;
  String? get error => _error;

  /// 发送问题
  Future<void> ask(String question) async {
    if (question.trim().isEmpty || _sending) return;

    // 添加用户消息到列表
    _messages.add(Message(
      id: DateTime.now().millisecondsSinceEpoch.toString(),
      role: 'user',
      content: question,
    ));
    _sending = true;
    _error = null;
    notifyListeners();

    try {
      final req = ChatRequest(question: question, sessionId: _sessionId);
      final resp = await _api.post(ApiConfig.chat, data: req.toJson());

      final chatResp = ChatResponse.fromJson(resp.data);

      if (chatResp.code != 0) {
        _error = chatResp.message;
        _sending = false;
        notifyListeners();
        return;
      }

      // 更新会话 ID
      if (chatResp.sessionId.isNotEmpty) {
        _sessionId = chatResp.sessionId;
      }

      // 添加 AI 回复
      _messages.add(Message(
        id: DateTime.now().millisecondsSinceEpoch.toString(),
        role: 'assistant',
        content: chatResp.data?.conclusion ?? '',
        answerCard: chatResp.data,
      ));

      _sending = false;
      notifyListeners();
    } catch (e) {
      // 标记最后一条用户消息为发送失败
      if (_messages.isNotEmpty && _messages.last.isUser) {
        _messages[_messages.length - 1] = _messages.last.copyWith(isFailed: true);
      }
      _error = '发送失败，请稍后重试';
      _sending = false;
      notifyListeners();
    }
  }

  /// 重发失败的消息
  Future<void> resendMessage(int index) async {
    if (index < 0 || index >= _messages.length) return;
    final msg = _messages[index];
    if (!msg.isUser || !msg.isFailed) return;

    // 移除失败消息及其后所有消息（通常是失败的 assistant 回复）
    _messages.removeRange(index, _messages.length);
    notifyListeners();

    await ask(msg.content);
  }

  /// 点击追问建议
  Future<void> askFollowUp(String question) => ask(question);

  /// 新建对话
  void newChat() {
    _messages.clear();
    _sessionId = null;
    _error = null;
    notifyListeners();
  }

  /// 加载历史会话的消息
  Future<void> loadSession(String sessionId) async {
    _messages.clear();
    _sessionId = sessionId;
    _error = null;
    notifyListeners();

    try {
      final resp = await _api.get(ApiConfig.sessionMessages(sessionId));
      final data = resp.data;
      final list = data['data'] as List? ?? [];

      for (final item in list) {
        _messages.add(Message.fromJson(item));
      }
      notifyListeners();
    } catch (e) {
      _error = '加载消息失败';
      notifyListeners();
    }
  }
}
