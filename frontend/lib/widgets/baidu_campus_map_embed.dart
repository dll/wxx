// 平台条件导入：Web 用真实百度地图 HtmlElementView，其他平台用兜底卡片。
export 'baidu_campus_map_embed_stub.dart'
    if (dart.library.html) 'baidu_campus_map_embed_web.dart';
