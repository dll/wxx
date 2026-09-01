import 'package:dio/dio.dart';
import '../config/api_config.dart';
import '../utils/storage.dart';

/// 全局会话重置回调注册表。
/// 各 Provider 在构造时注册自己的 reset()，令牌吊销/过期（401）时统一触发，
/// 防止跨账号内存态泄露（Q-08 / S-01 令牌吊销联动）。
final List<void Function()> sessionResetCallbacks = [];

/// 触发所有已注册的会话重置回调
void triggerSessionReset() {
  for (final cb in sessionResetCallbacks) {
    cb();
  }
}

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
        // 仅当本地确实持有令牌时才触发会话重置。
        // 未登录页面访问公开/可选接口收到 401，不应清空状态或反复跳转登录页。
        if (error.response?.statusCode == 401 &&
            error.requestOptions.path != ApiConfig.login &&
            Storage.isLoggedIn) {
          onUnauthorized?.call();
          handler.resolve(Response(
            requestOptions: error.requestOptions,
            statusCode: 401,
            data: {'code': 401, 'message': '未登录或登录已过期'},
          ));
          return;
        }
        handler.next(error);
      },
    ));
  }

  // ── 通用请求方法 ──

  Future<Response> get(String path,
      {Map<String, dynamic>? params, Options? options}) {
    return _dio.get(path, queryParameters: params, options: options);
  }

  /// 以字节流接收（用于私有文件鉴权下载等二进制接口）。
  Future<Response> getBytes(String path) {
    return _dio.get(path,
        options: Options(responseType: ResponseType.bytes));
  }

  Future<Response> post(String path, {dynamic data}) {
    return _dio.post(path, data: data);
  }

  Future<Response> put(String path, {dynamic data}) {
    return _dio.put(path, data: data);
  }

  Future<Response> patch(String path, {dynamic data}) {
    return _dio.patch(path, data: data);
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
  Future<Response> upload(String path,
      {required String filePath, required String fieldName}) async {
    final filename = filePath.split('/').last;
    final formData = FormData.fromMap({
      fieldName: await MultipartFile.fromFile(filePath, filename: filename),
    });
    return _dio.post(path, data: formData);
  }

  /// 上传文件（Web / bytes）
  Future<Response> uploadBytes(
    String path, {
    required List<int> bytes,
    required String filename,
    required String fieldName,
    Map<String, dynamic>? fields,
  }) async {
    final formData = FormData.fromMap({
      ...?fields,
      fieldName: MultipartFile.fromBytes(bytes, filename: filename),
    });
    return _dio.post(path, data: formData);
  }
}
