// ignore_for_file: avoid_web_libraries_in_flutter

import 'dart:html' as html;

/// Web 浏览器实时语音识别（基于 SpeechRecognition API）
/// 提供实时文字流 + 一键停止能力，适用于对话输入和语音导航
class WebSpeechRecognizer {
  html.SpeechRecognition? _recognition;
  bool _isListening = false;
  bool _disposed = false;

  /// 实时文本回调：(interim 中间结果, final 最终结果, isFinal 是否最终)
  void Function(String interim, String finalText, bool isFinal)? onTranscript;

  /// 错误回调
  void Function(String error)? onError;

  /// 结束回调
  void Function()? onEnd;

  bool get isListening => _isListening;

  /// 开始持续聆听（continuous = true 表示不会自动停止）
  /// [continuous] 是否持续聆听（语音导航/AI助手用 true，对话输入用 false）
  void start({bool continuous = true, String lang = 'zh-CN'}) {
    if (_isListening || _disposed) return;

    try {
      _recognition = html.SpeechRecognition();
      _recognition!.lang = lang;
      _recognition!.continuous = continuous;
      _recognition!.interimResults = true;
      _recognition!.maxAlternatives = 1;

      _recognition!.onResult.listen(_onResult);
      _recognition!.onError.listen(_onError);
      _recognition!.onEnd.listen(_onRecognitionEnd);

      _recognition!.start();
      _isListening = true;
    } catch (e) {
      onError?.call('浏览器不支持语音识别，请使用 Chrome 或 Edge');
    }
  }

  void _onResult(html.Event event) {
    if (_disposed) return;
    try {
      final results = (event as dynamic).results;
      if (results == null) return;

      String interim = '';
      String finalText = '';
      bool hasFinal = false;

      for (int i = 0; i < results.length; i++) {
        final item = results.item(i);
        if (item == null) continue;
        final alt = item.item(0);
        if (alt == null) continue;
        final transcript = (alt.transcript ?? '').toString();
        if (item.isFinal == true) {
          finalText += transcript;
          hasFinal = true;
        } else {
          interim += transcript;
        }
      }

      onTranscript?.call(interim, finalText, hasFinal);
    } catch (_) {}
  }

  void _onError(html.Event event) {
    if (_disposed) return;
    final error = (event as dynamic).error as String?;
    if (error == 'no-speech' || error == 'aborted') {
      // 静默
      return;
    }
    if (error == 'not-allowed') {
      onError?.call('麦克风权限未授权');
      return;
    }
    onError?.call(error ?? '识别错误');
  }

  void _onRecognitionEnd(html.Event _) {
    _isListening = false;
    if (!_disposed) onEnd?.call();
  }

  /// 立即停止聆听
  void stop() {
    if (_recognition == null) return;
    try {
      _recognition!.stop();
    } catch (_) {}
    _isListening = false;
  }

  /// 中止（更彻底的停止）
  void abort() {
    if (_recognition == null) return;
    try {
      _recognition!.abort();
    } catch (_) {}
    _isListening = false;
  }

  void dispose() {
    _disposed = true;
    abort();
    _recognition = null;
    onTranscript = null;
    onError = null;
    onEnd = null;
  }
}
