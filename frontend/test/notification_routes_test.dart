import 'package:flutter_test/flutter_test.dart';
import 'package:wxx_app/utils/notification_routes.dart';

void main() {
  test('maps supported notification relation types', () {
    expect(notificationRouteForRelatedType('feedback'), '/my-feedbacks');
    expect(notificationRouteForRelatedType('KB_RESOURCE'), '/browse');
    expect(notificationRouteForRelatedType('process_record'), '/my-records');
    expect(notificationRouteForRelatedType('job'), '/student/career');
  });

  test('returns null for unknown relation types', () {
    expect(notificationRouteForRelatedType('unknown'), isNull);
  });
}
