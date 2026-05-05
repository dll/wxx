/// 语音服务统一入口
///
/// 根据平台自动选择实现：
/// - Web: 使用 dart:html MediaRecorder + Audio API
/// - 其他平台: 抛出 UnsupportedError（移动端后续补充）
library voice_service;

export 'voice_service_stub.dart'
    if (dart.library.html) 'voice_service_web.dart';
