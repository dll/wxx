import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wxx_app/config/api_config.dart';
import 'package:wxx_app/providers/vopc_provider.dart';

void main() {
  group('VopcProvider task workbench', () {
    test('loads project detail and its real task list', () async {
      final api = _FakeVopcApi()
        ..getResponses[ApiConfig.vopcProject(7)] = _response({
          'data': {
            'id': 7,
            'name': '真实项目',
            'summary': '摘要',
            'stage': 'S1',
            'status': 'pending_review',
            'project_type': '自由探索项目',
            'risk_level': 'R0',
          }
        })
        ..getResponses[ApiConfig.vopcProjectTasks(7)] = _response({
          'data': [
            {
              'id': 31,
              'title': '完成原型',
              'description': '',
              'assignee_user_id': 2,
              'acceptance_criteria': '主流程可运行',
              'priority': 'high',
              'status': 'todo',
            }
          ]
        })
        ..getResponses[ApiConfig.vopcProjectDecisions(7)] =
            _response({'data': <Object>[]})
        ..getResponses[ApiConfig.vopcProjectMembers(7)] =
            _response({'data': <Object>[]})
        ..getResponses[ApiConfig.vopcProjectArtifacts(7)] =
            _response({'data': <Object>[]})
        ..getResponses[ApiConfig.vopcMilestoneSubmissions(7)] =
            _response({'data': <Object>[]});
      final provider = VopcProvider(api);

      await provider.loadDetail(7);

      expect(provider.detail?.name, '真实项目');
      expect(provider.tasks, hasLength(1));
      expect(provider.tasks.single.assigneeUserId, 2);
      expect(provider.tasks.single.acceptanceCriteria, '主流程可运行');
      expect(provider.error, isNull);
    });

    test('create only succeeds with server id and refreshes list', () async {
      final path = ApiConfig.vopcProjectTasks(7);
      final api = _FakeVopcApi()
        ..postResponses[path] = _response({
          'data': {'id': 32, 'status': 'todo'}
        }, statusCode: 201)
        ..getResponses[path] = _response({'data': <Object>[]});
      final provider = VopcProvider(api);

      final ok = await provider.createTask(7, {
        'title': '任务',
        'acceptance_criteria': '可验收',
        'priority': 'normal',
      });

      expect(ok, isTrue);
      expect(api.getCalls, contains(path));
      expect(provider.error, isNull);
    });

    test('decision resolve refreshes only after validated success', () async {
      final path = ApiConfig.vopcProjectDecision(7, 9);
      final listPath = ApiConfig.vopcProjectDecisions(7);
      final api = _FakeVopcApi()
        ..putResponses[path] = _response({
          'data': {'id': 9, 'status': 'resolved'}
        })
        ..getResponses[listPath] = _response({'data': <Object>[]});
      final provider = VopcProvider(api);

      expect(
          await provider.actDecision(7, 9, 'resolve',
              decision: '采用 A', rationale: '可验证'),
          isTrue);
      expect(api.getCalls, contains(listPath));
    });

    test('artifact creation refuses a false success response', () async {
      final path = ApiConfig.vopcProjectArtifacts(7);
      final api = _FakeVopcApi()..postResponses[path] = _response({'data': {}});
      final provider = VopcProvider(api);

      expect(await provider.createArtifact(7, {'name': '成果'}), isFalse);
      expect(api.getCalls, isEmpty);
    });

    test('invitation acceptance refreshes invitation state', () async {
      final path = ApiConfig.vopcInvitationRespond(12);
      final api = _FakeVopcApi()
        ..postResponses[path] = _response({
          'data': {'id': 12, 'status': 'accepted'}
        })
        ..getResponses[ApiConfig.vopcInvitations] =
            _response({'data': <Object>[]});
      final provider = VopcProvider(api);

      expect(await provider.respondInvitation(12, 'accept'), isTrue);
      expect(api.getCalls, contains(ApiConfig.vopcInvitations));
    });

    test('milestone submission uses selected real artifact versions', () async {
      final versionsPath = ApiConfig.vopcArtifactVersions(7, 5);
      final submitPath = ApiConfig.vopcMilestoneSubmissions(7);
      final api = _FakeVopcApi()
        ..getResponses[versionsPath] = _response({
          'data': [
            {'id': 11, 'version': 'v1', 'release_notes': '首版'}
          ]
        })
        ..postResponses[submitPath] = _response({
          'data': {'id': 21, 'status': 'pending'}
        }, statusCode: 201)
        ..getResponses[submitPath] = _response({'data': <Object>[]});
      final provider = VopcProvider(api);

      final versions = await provider.loadArtifactVersions(7, 5);
      expect(versions.single['id'], 11);
      expect(
          await provider.submitMilestone(7, {
            'stage': 'S2',
            'evidence': '真实版本证据',
            'artifact_version_ids': [11]
          }),
          isTrue);
      expect(api.postData[submitPath]?['artifact_version_ids'], [11]);
    });

    test('surfaces 409 and does not report a false status success', () async {
      final path = ApiConfig.vopcProjectTask(7, 31);
      final api = _FakeVopcApi()
        ..putErrors[path] = DioException(
          requestOptions: RequestOptions(path: path),
          response: _response(
            {'message': '任务状态流转不合法'},
            statusCode: 409,
            path: path,
          ),
          type: DioExceptionType.badResponse,
        );
      final provider = VopcProvider(api);

      final ok = await provider.updateTaskStatus(7, 31, 'done');

      expect(ok, isFalse);
      expect(provider.statusCode, 409);
      expect(provider.error, '任务状态流转不合法');
      expect(api.getCalls, isEmpty);
    });
  });
}

Response<dynamic> _response(dynamic data,
        {int statusCode = 200, String path = '/test'}) =>
    Response<dynamic>(
      requestOptions: RequestOptions(path: path),
      statusCode: statusCode,
      data: data,
    );

class _FakeVopcApi implements VopcApiClient {
  final getResponses = <String, Response<dynamic>>{};
  final postResponses = <String, Response<dynamic>>{};
  final putResponses = <String, Response<dynamic>>{};
  final putErrors = <String, DioException>{};
  final postData = <String, dynamic>{};
  final getCalls = <String>[];

  @override
  Future<Response<dynamic>> get(String path, {Map<String, dynamic>? params}) async {
    getCalls.add(path);
    return getResponses[path]!;
  }

  @override
  Future<Response<dynamic>> delete(String path) async {
    return getResponses[path]!;
  }

  @override
  Future<Response<dynamic>> post(String path, {dynamic data}) async {
    postData[path] = data;
    return postResponses[path]!;
  }

  @override
  Future<Response<dynamic>> put(String path, {dynamic data}) async {
    final error = putErrors[path];
    if (error != null) throw error;
    return putResponses[path]!;
  }

  @override
  Future<Response<dynamic>> getBytes(String path) async {
    getCalls.add(path);
    return getResponses[path]!;
  }

  @override
  Future<Response<dynamic>> uploadFile(String path,
      {required List<int> bytes, required String filename}) async {
    return postResponses[path]!;
  }
}
