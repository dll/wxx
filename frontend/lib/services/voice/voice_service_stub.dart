import 'dart:typed_data';

/// 非 Web / 非移动端语音服务兜底（暂不支持）
class VoiceService {
  bool get isSupported => false;
  bool get isRecording => false;
  bool get isPlaying => false;

  Future<void> startRecording() async {
    throw UnsupportedError('语音功能在当前平台不可用');
  }

  Future<Uint8List?> stopRecording() async {
    throw UnsupportedError('语音功能在当前平台不可用');
  }

  Future<String?> speechToText(Uint8List audioBytes) async {
    throw UnsupportedError('语音功能在当前平台不可用');
  }

  Future<Uint8List?> textToSpeech(String text) async {
    throw UnsupportedError('语音功能在当前平台不可用');
  }

  Future<void> playAudio(Uint8List audioData) async {
    throw UnsupportedError('语音功能在当前平台不可用');
  }

  void stopPlayback() {}
  void dispose() {}
}
