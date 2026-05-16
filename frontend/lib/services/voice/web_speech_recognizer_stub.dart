/// 非 Web 平台兜底：不可用，调用方需自行判断 isListening 始终为 false。
class WebSpeechRecognizer {
  bool get isListening => false;

  void Function(String interim, String finalText, bool isFinal)? onTranscript;
  void Function(String error)? onError;
  void Function()? onEnd;

  void start({bool continuous = true, String lang = 'zh-CN'}) {
    onError?.call('当前平台不支持浏览器实时语音识别');
  }

  void stop() {}

  void abort() {}

  void dispose() {
    onTranscript = null;
    onError = null;
    onEnd = null;
  }
}
