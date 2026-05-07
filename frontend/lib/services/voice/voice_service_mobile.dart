import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:audioplayers/audioplayers.dart';
import 'package:dio/dio.dart';
import 'package:path_provider/path_provider.dart';
import 'package:record/record.dart';

import '../../config/api_config.dart';
import '../../utils/storage.dart';
import '../api_service.dart';

/// Android/iOS 语音服务实现
/// 使用 record 插件录音（WAV PCM 16kHz）→ 后端 ASR / TTS → audioplayers 播放
class VoiceService {
  final AudioRecorder _recorder = AudioRecorder();
  AudioPlayer? _player;

  /// 录音完成回调，由 stopRecording() 触发
  Completer<Uint8List>? _recordCompleter;

  bool _isRecording = false;
  bool _isPlaying = false;
  String? _currentRecordingPath;

  /// 移动端始终支持语音（需用户授权麦克风）
  bool get isSupported => true;

  bool get isRecording => _isRecording;

  bool get isPlaying => _isPlaying;

  /// 开始录音
  /// 请求麦克风权限，配置 WAV PCM 16kHz 单声道（适配讯飞 ASR）
  Future<void> startRecording() async {
    if (_isRecording) return;

    // 请求麦克风权限
    if (!await _recorder.hasPermission()) {
      throw Exception('麦克风权限未授予');
    }

    // 临时文件路径
    final dir = await getTemporaryDirectory();
    _currentRecordingPath =
        '${dir.path}/voice_record_${DateTime.now().millisecondsSinceEpoch}.wav';

    const config = RecordConfig(
      encoder: AudioEncoder.wav,
      sampleRate: 16000,
      numChannels: 1,
      echoCancel: true,
    );

    _recordCompleter = Completer<Uint8List>();
    await _recorder.start(config, path: _currentRecordingPath!);
    _isRecording = true;
  }

  /// 停止录音，返回 WAV 音频字节
  Future<Uint8List?> stopRecording() async {
    if (!_isRecording) return null;

    _isRecording = false;

    try {
      final path = await _recorder.stop();
      if (path == null || path.isEmpty) {
        _recordCompleter?.completeError('录音文件为空');
        _recordCompleter = null;
        return null;
      }

      // 读取录音文件
      final file = File(path);
      if (!await file.exists()) {
        _recordCompleter?.completeError('录音文件不存在');
        _recordCompleter = null;
        return null;
      }

      final bytes = await file.readAsBytes();

      // 通知等待 startRecording() 返回值的调用方
      _recordCompleter?.complete(bytes);
      _recordCompleter = null;

      // 清理临时文件
      try {
        await file.delete();
      } catch (_) {}

      return bytes;
    } catch (e) {
      _recordCompleter?.completeError(e);
      _recordCompleter = null;
      return null;
    }
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
      final dio = Dio(BaseOptions(
        baseUrl: ApiConfig.baseUrl,
        connectTimeout: Duration(milliseconds: ApiConfig.connectTimeout),
        receiveTimeout: Duration(milliseconds: ApiConfig.receiveTimeout),
        responseType: ResponseType.bytes,
      ));

      final token = Storage.token;
      final response = await dio.post(
        ApiConfig.voiceTts,
        data: {'text': text, 'voice': 'x_xiaoyan'},
        options: Options(
          headers: token != null ? {'Authorization': 'Bearer $token'} : null,
        ),
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
    await stopPlayback();

    // 写入临时文件供 audioplayers 播放
    final dir = await getTemporaryDirectory();
    final path = '${dir.path}/tts_play_${DateTime.now().millisecondsSinceEpoch}.mp3';
    final file = File(path);
    await file.writeAsBytes(audioData);

    _player = AudioPlayer();
    _isPlaying = true;

    await _player!.play(DeviceFileSource(path));

    _player!.onPlayerComplete.listen((_) {
      _isPlaying = false;
      try {
        file.delete();
      } catch (_) {}
    });
  }

  /// 停止播放
  Future<void> stopPlayback() async {
    if (_player != null) {
      await _player!.stop();
      _player = null;
    }
    _isPlaying = false;
  }

  /// 释放资源
  void dispose() {
    _player?.dispose();
    _player = null;
    _recorder.dispose();
    _isRecording = false;
    _isPlaying = false;
  }
}
