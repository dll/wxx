import 'package:dio/dio.dart';

/// 将网络请求异常转换为面向用户的友好提示（含根因）。
///
/// 覆盖常见情况：
/// - 429 限流：请求过于频繁，提示等待后再试；
/// - 5xx：服务暂时不可用；
/// - 超时 / 断网：网络层问题；
/// - 后端返回的业务 message 透传；
/// - 其他：统一兜底文案，避免把 DioException 原始堆栈展示给用户。
String friendlyApiError(Object e, {String fallback = '请求失败，请稍后重试'}) {
  if (e is DioException) {
    final data = e.response?.data;
    final serverMsg = data is Map && data['message'] != null
        ? data['message'].toString()
        : '';
    // 限流优先：明确根因（请求过于频繁），服务端若有更具体文案则透传
    if (e.response?.statusCode == 429) {
      return serverMsg.isNotEmpty ? serverMsg : '操作过于频繁，已触发接口限流，请稍等片刻后再试';
    }
    switch (e.type) {
      case DioExceptionType.connectionTimeout:
      case DioExceptionType.sendTimeout:
      case DioExceptionType.receiveTimeout:
        return '网络请求超时，请稍后重试';
      case DioExceptionType.connectionError:
        return '暂时无法连接服务，请检查网络后重试';
      case DioExceptionType.badResponse:
        final status = e.response?.statusCode;
        if (status == 502 || status == 503 || status == 504) {
          return '服务暂时不可用，请稍后重试';
        }
        if (serverMsg.isNotEmpty) return serverMsg;
        if (status != null) return '请求失败（HTTP $status），请稍后重试';
        return fallback;
      default:
        if (serverMsg.isNotEmpty) return serverMsg;
        break;
    }
    return fallback;
  }

  final s = e.toString();
  if (s.contains('429') || s.contains('Too Many Requests')) {
    return '操作过于频繁，已触发接口限流，请稍等片刻后再试';
  }
  return fallback;
}
