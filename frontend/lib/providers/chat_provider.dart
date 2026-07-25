import 'package:flutter/foundation.dart';
import '../config/api_config.dart';
import '../models/models.dart';
import '../services/api_service.dart';
import '../services/voice/voice_service.dart';
import '../utils/capability_utils.dart';

/// 对话状态管理
class ChatProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  final List<Message> _messages = [];
  String? _sessionId;
  bool _sending = false;
  String? _error;

  // 智能体选择
  List<Agent> _agents = [];
  String? _selectedAgentId; // null = 默认智能体
  bool _agentsLoading = false;

  final VoiceService _voice = VoiceService();
  bool _isRecording = false;
  bool _isPlaying = false;
  int _playingIndex = -1;

  List<Message> get messages => List.unmodifiable(_messages);
  String? get sessionId => _sessionId;
  bool get sending => _sending;
  String? get error => _error;
  bool get isRecording => _isRecording;
  bool get isPlaying => _isPlaying;
  int get playingIndex => _playingIndex;
  VoiceService get voice => _voice;

  List<Agent> get agents => _agents;
  String? get selectedAgentId => _selectedAgentId;
  Agent? get selectedAgent {
    if (_selectedAgentId == null) return null;
    try {
      return _agents.firstWhere((a) => a.agentId == _selectedAgentId);
    } catch (_) {
      return null;
    }
  }
  bool get agentsLoading => _agentsLoading;

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
      final req = ChatRequest(question: question, sessionId: _sessionId, agentId: _selectedAgentId);
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

  /// 退出登录时重置全部内存态，防止跨账号泄露（Q-08）
  void reset() {
    _messages.clear();
    _sessionId = null;
    _error = null;
    _sending = false;
    _agents = [];
    _selectedAgentId = null;
    _agentsLoading = false;
    _isRecording = false;
    _isPlaying = false;
    _playingIndex = -1;
    notifyListeners();
  }

  /// 切换智能体
  void selectAgent(String? agentId) {
    _selectedAgentId = agentId;
    notifyListeners();
  }

  /// 加载激活的智能体列表（用于选择器）
  Future<void> loadAgents() async {
    if (_agentsLoading) return;
    // 仅 school_admin 及以上拥有 school.agent.write 才可访问 /agents 接口
    if (!CapabilityUtils.has(Capability.schoolAgentWrite)) return;
    _agentsLoading = true;
    notifyListeners();

    try {
      final resp = await _api.get(ApiConfig.agents);
      if (resp.data['code'] == 0) {
        final list = resp.data['data'] as List? ?? [];
        _agents = list
            .map((e) => Agent.fromJson(e as Map<String, dynamic>))
            .where((a) => a.isActive)
            .toList();
      }
    } catch (_) {
      // 加载失败不影响对话，静默处理
    } finally {
      _agentsLoading = false;
      notifyListeners();
    }
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

  // ── 语音相关方法 ──

  /// 开始语音录制
  Future<void> startRecording() async {
    if (_isRecording || _sending) return;
    try {
      await _voice.startRecording();
      _isRecording = true;
      notifyListeners();
    } catch (e) {
      _error = '无法访问麦克风：$e';
      notifyListeners();
    }
  }

  /// 仅停止录音（不触发 ASR/发送）
  void stopRecording() {
    if (!_isRecording) return;
    _isRecording = false;
    notifyListeners();
  }

  /// 停止录制并发送语音转文本
  Future<void> stopRecordingAndSend() async {
    if (!_isRecording) return;
    _isRecording = false;
    notifyListeners();

    final audioData = await _voice.stopRecording();
    if (audioData == null) return;

    final text = await _voice.speechToText(audioData);
    if (text == null || text.trim().isEmpty) {
      _error = '未识别到语音内容，请重试';
      notifyListeners();
      return;
    }

    await ask(text.trim());
  }

  /// 播放消息的 TTS 语音
  Future<void> playTTS(int messageIndex) async {
    if (messageIndex < 0 || messageIndex >= _messages.length) return;
    final msg = _messages[messageIndex];
    if (msg.isUser || msg.content.isEmpty) return;

    // 如果正在播放同一条消息，停止播放
    if (_isPlaying && _playingIndex == messageIndex) {
      _voice.stopPlayback();
      _isPlaying = false;
      _playingIndex = -1;
      notifyListeners();
      return;
    }

    _isPlaying = true;
    _playingIndex = messageIndex;
    notifyListeners();

    final audioData = await _voice.textToSpeech(msg.content);
    if (audioData == null) {
      _error = '语音合成失败：后端服务未响应或未配置 TTS 引擎，请检查网络连接';
      _isPlaying = false;
      _playingIndex = -1;
      notifyListeners();
      return;
    }

    await _voice.playAudio(audioData);
    _isPlaying = false;
    _playingIndex = -1;
    notifyListeners();
  }

  @override
  void dispose() {
    _voice.dispose();
    super.dispose();
  }
}
