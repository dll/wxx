// Web 浏览器实时语音识别统一入口
// - Web: 使用 dart:html SpeechRecognition API
// - 非 Web: 走 stub，调用即返回不支持
export 'web_speech_recognizer_stub.dart'
    if (dart.library.html) 'web_speech_recognizer_web.dart';
