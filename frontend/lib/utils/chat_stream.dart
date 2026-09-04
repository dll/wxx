/// 流式对话消费统一入口
///
/// 按平台选择实现：
/// - Web：使用 dart:html HttpRequest 消费 SSE（逐块回调 onDelta）
/// - Android/iOS：无实现（no-op），聊天走非流式 ask()
library;
export 'chat_stream_io.dart' if (dart.library.html) 'chat_stream_web.dart';
