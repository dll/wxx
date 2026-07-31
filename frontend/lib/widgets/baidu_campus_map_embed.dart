// 平台条件导入：
//   Web       → baidu_campus_map_embed_web.dart  (HtmlElementView + postMessage)
//   Android/iOS/Desktop → baidu_campus_map_embed_android.dart (webview_flutter)
//   其他       → baidu_campus_map_embed_stub.dart  (占位)
export 'baidu_campus_map_embed_stub.dart'
    if (dart.library.html) 'baidu_campus_map_embed_web.dart'
    if (dart.library.io) 'baidu_campus_map_embed_android.dart';
