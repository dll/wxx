/// Maps server notification relation types to existing app routes.
/// Unknown types intentionally return null so callers can show a safe fallback.
String? notificationRouteForRelatedType(String relatedType) {
  switch (relatedType.toLowerCase()) {
    case 'feedback':
    case 'feedback_reply':
      return '/my-feedbacks';
    case 'knowledge':
    case 'resource':
    case 'kb_resource':
      return '/browse';
    case 'process':
    case 'process_record':
      return '/my-records';
    case 'activity':
      return '/services';
    case 'career':
    case 'job':
      return '/student/career';
    default:
      return null;
  }
}
