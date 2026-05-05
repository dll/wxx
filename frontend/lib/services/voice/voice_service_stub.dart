import 'dart:typed_data';

/// 非 Web 平台语音服务（暂不支持）
/// 移动端（Android/iOS）录音播放在后续版本补充
class VoiceService {
  bool get isSupported => false;
  bool get isRecording => false;
  bool get isPlaying => false;

  Future<Uint8List> startRecording() async {
    throw UnsupportedError('语音功能仅在 Web 平台可用');
  }

  Future<Uint8List?> stopRecording() async {
    throw UnsupportedError('语音功能仅在 Web 平台可用');
  }

  Future<String?> speechToText(Uint8List audioBytes) async {
    throw UnsupportedError('语音功能仅在 Web 平台可用');
  }

  Future<Uint8List?> textToSpeech(String text) async {
    throw UnsupportedError('语音功能仅在 Web 平台可用');
  }

  Future<void> playAudio(Uint8List audioData) async {
    throw UnsupportedError('语音功能仅在 Web 平台可用');
  }

  void stopPlayback() {}
  void dispose() {}
}
