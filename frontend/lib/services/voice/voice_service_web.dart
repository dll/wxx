// ignore_for_file: deprecated_member_use, avoid_web_libraries_in_flutter

import 'dart:async';
import 'dart:html' as html;
import 'dart:typed_data';
import 'package:dio/dio.dart';
import '../../config/api_config.dart';
import '../api_service.dart';

/// Web 平台语音服务实现
/// 使用浏览器原生 MediaRecorder API 录音，Audio 元素播放
class VoiceService {
  html.MediaRecorder? _recorder;
  final List<html.Blob> _chunks = [];
  html.AudioElement? _audio;

  bool _isRecording = false;
  bool _isPlaying = false;

  /// 是否支持语音功能（Web 平台始终返回 true，但需用户授权麦克风）
  bool get isSupported => true;

  /// 当前是否正在录音
  bool get isRecording => _isRecording;

  /// 是否正在播放
  bool get isPlaying => _isPlaying;

  /// 开始录音
  Future<void> startRecording() async {
    if (_isRecording) return;

    _chunks.clear();

    // 请求麦克风权限
    final stream = await html.window.navigator.mediaDevices!
        .getUserMedia({'audio': true});

    // MediaRecorder 自动处理音频编码
    _recorder = html.MediaRecorder(stream);

    // 使用 EventStreamProvider 处理 dataavailable 事件（兼容 dart:html API 变更）
    const dataAvailEvent = html.EventStreamProvider<html.Event>('dataavailable');
    dataAvailEvent.forTarget(_recorder!).listen((event) {
      final blobEvent = event as html.BlobEvent;
      if (blobEvent.data != null) {
        _chunks.add(blobEvent.data!);
      }
    });

    const stopEvent = html.EventStreamProvider<html.Event>('stop');
    stopEvent.forTarget(_recorder!).listen((_) {
      // 停止所有轨道
      stream.getTracks().forEach((track) => track.stop());
    });

    _recorder!.start();
    _isRecording = true;
  }

  /// 停止录音，返回音频字节
  Future<Uint8List?> stopRecording() async {
    if (_recorder == null || _recorder!.state != 'recording') {
      _isRecording = false;
      return null;
    }

    final completer = Completer<Uint8List?>();
    _isRecording = false;

    _recorder!.stop();

    // 轮询等待 blob 收集完成（MediaRecorder 异步交付 dataavailable）
    Timer.periodic(const Duration(milliseconds: 50), (timer) {
      if (_chunks.isEmpty && timer.tick < 20) return;
      timer.cancel();

      if (_chunks.isEmpty) {
        completer.complete(null);
        return;
      }

      final blob = html.Blob(_chunks);
      final reader = html.FileReader();

      reader.onLoadEnd.listen((_) {
        if (reader.result is ByteBuffer) {
          completer.complete(Uint8List.view(reader.result as ByteBuffer));
        } else {
          completer.complete(null);
        }
      });

      reader.onError.listen((_) {
        completer.complete(null);
      });

      reader.readAsArrayBuffer(blob);
      _chunks.clear();
    });

    final result = await completer.future;
    _recorder = null;
    return result;
  }

  /// 将音频数据发送到后端 ASR，返回识别文本
  Future<String?> speechToText(Uint8List audioBytes) async {
    try {
      final formData = FormData.fromMap({
        'audio': MultipartFile.fromBytes(
          audioBytes,
          filename: 'recording.wav',
        ),
      });

      final response = await ApiService().post(
        ApiConfig.voiceAsr,
        data: formData,
      );

      if (response.statusCode == 200 && response.data['code'] == 0) {
        return response.data['data']['text'] as String?;
      }
      return null;
    } on DioException {
      return null;
    }
  }

  /// 调用后端 TTS 将文本转为语音，返回 MP3 音频字节
  Future<Uint8List?> textToSpeech(String text) async {
    try {
      final response = await ApiService().postBytes(
        ApiConfig.voiceTts,
        data: {'text': text, 'voice': 'x_xiaoyan'},
      );

      if (response.statusCode == 200 && response.data != null) {
        if (response.data is List<int>) {
          return Uint8List.fromList(response.data as List<int>);
        }
      }
      return null;
    } on DioException {
      return null;
    }
  }

  /// 播放 MP3 音频
  Future<void> playAudio(Uint8List audioData) async {
    _audio?.remove();
    _audio = null;

    final blob = html.Blob([audioData], 'audio/mpeg');
    final url = html.Url.createObjectUrl(blob);

    _audio = html.AudioElement(url);
    _isPlaying = true;
    _audio!.play();

    _audio!.onEnded.listen((_) {
      html.Url.revokeObjectUrl(url);
      _audio = null;
      _isPlaying = false;
    });
  }

  /// 停止播放
  void stopPlayback() {
    if (_audio != null) {
      _audio!.pause();
      _audio!.currentTime = 0;
      _audio = null;
    }
    _isPlaying = false;
  }

  /// 释放资源
  void dispose() {
    stopPlayback();
    _recorder = null;
    _chunks.clear();
  }
}
