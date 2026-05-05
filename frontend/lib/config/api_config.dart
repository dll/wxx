/// API 配置常量
class ApiConfig {
  // 后端基础地址（开发环境）
  // Web 端与后端同源时可用相对路径；跨域时需完整 URL
  static const String baseUrl = 'http://localhost:9091';

  // API 版本前缀
  static const String apiPrefix = '/api/v1';

  // 超时设置（毫秒）
  static const int connectTimeout = 10000;
  static const int receiveTimeout = 30000; // LLM 响应较慢，给足时间

  // ── 接口路径 ──
  static const String login = '$apiPrefix/auth/login';
  static const String profile = '$apiPrefix/user/profile';
  static const String chat = '$apiPrefix/chat';
  static const String sessions = '$apiPrefix/sessions';
  static String sessionMessages(String id) => '$apiPrefix/sessions/$id/messages';
  static String sessionDelete(String id) => '$apiPrefix/sessions/$id';

  // ── 语音接口 ──
  static const String voiceAsr = '$apiPrefix/voice/asr';
  static const String voiceTts = '$apiPrefix/voice/tts';
}
