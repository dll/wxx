import 'package:dio/dio.dart';
import '../config/api_config.dart';
import '../utils/storage.dart';

/// HTTP 请求服务，基于 Dio 封装
/// - 自动注入 JWT Token
/// - 401 时触发退出登录回调
class ApiService {
  static final ApiService _instance = ApiService._();
  factory ApiService() => _instance;

  late final Dio _dio;

  /// 401 回调：由 AuthProvider 设置，触发跳转登录页
  void Function()? onUnauthorized;

  ApiService._() {
    _dio = Dio(BaseOptions(
      baseUrl: ApiConfig.baseUrl,
      connectTimeout: Duration(milliseconds: ApiConfig.connectTimeout),
      receiveTimeout: Duration(milliseconds: ApiConfig.receiveTimeout),
      headers: {'Content-Type': 'application/json'},
    ));

    // 请求拦截器：注入 JWT
    _dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) {
        final token = Storage.token;
        if (token != null && token.isNotEmpty) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
      onError: (error, handler) {
        if (error.response?.statusCode == 401) {
          // Token 过期或无效，触发退出登录
          onUnauthorized?.call();
        }
        handler.next(error);
      },
    ));
  }

  // ── 通用请求方法 ──

  Future<Response> get(String path, {Map<String, dynamic>? params}) {
    return _dio.get(path, queryParameters: params);
  }

  Future<Response> post(String path, {dynamic data}) {
    return _dio.post(path, data: data);
  }

  Future<Response> put(String path, {dynamic data}) {
    return _dio.put(path, data: data);
  }

  Future<Response> delete(String path) {
    return _dio.delete(path);
  }

  /// 发送 POST 请求并以字节流接收响应（用于 TTS 等返回二进制数据的接口）
  Future<Response> postBytes(String path, {dynamic data}) {
    return _dio.post(
      path,
      data: data,
      options: Options(responseType: ResponseType.bytes),
    );
  }

  /// 上传文件（原生平台）
  Future<Response> upload(String path, {required String filePath, required String fieldName}) async {
    final filename = filePath.split('/').last;
    final formData = FormData.fromMap({
      fieldName: await MultipartFile.fromFile(filePath, filename: filename),
    });
    return _dio.post(path, data: formData);
  }

  /// 上传文件（Web / bytes）
  Future<Response> uploadBytes(String path, {required List<int> bytes, required String filename, required String fieldName}) async {
    final formData = FormData.fromMap({
      fieldName: MultipartFile.fromBytes(bytes, filename: filename),
    });
    return _dio.post(path, data: formData);
  }
}
