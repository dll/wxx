import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:wxx_app/utils/api_error.dart';

void main() {
  group('friendlyApiError 错误映射', () {
    test('429 限流：无服务端文案时给出明确限流提示', () {
      final msg = friendlyApiError(
        _dioException(statusCode: 429),
        fallback: '请求失败，请稍后重试',
      );
      expect(msg, contains('限流'));
    });

    test('502/503/504：服务暂时不可用', () {
      final msg = friendlyApiError(_dioException(statusCode: 503));
      expect(msg, contains('服务暂时不可用'));
    });

    test('服务端业务 message 优先透传', () {
      final msg = friendlyApiError(_dioException(statusCode: 400, message: '学号已存在'));
      expect(msg, '学号已存在');
    });

    test('非 DioException：含 429 关键字识别限流', () {
      final msg = friendlyApiError(Exception('HTTP 429 Too Many Requests'));
      expect(msg, contains('限流'));
    });

    test('其它异常走兜底文案', () {
      final msg = friendlyApiError(Exception('whatever'), fallback: '兜底文案');
      expect(msg, '兜底文案');
    });
  });
}

DioException _dioException({int? statusCode, String? message}) {
  return DioException(
    requestOptions: RequestOptions(path: '/test'),
    response: Response(
      requestOptions: RequestOptions(path: '/test'),
      statusCode: statusCode,
      data: message != null ? {'message': message} : null,
    ),
    type: statusCode != null ? DioExceptionType.badResponse : DioExceptionType.connectionError,
  );
}
