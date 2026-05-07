// 语音服务统一入口
// 根据平台自动选择实现：
// - Web: 使用 dart:html MediaRecorder + Audio API
// - Android/iOS: 使用 record + audioplayers 插件

export 'voice_service_stub.dart'
    if (dart.library.html) 'voice_service_web.dart'
    if (dart.library.io) 'voice_service_mobile.dart';
