// 角色与权限常量

export const ROLES = {
  SYS_ADMIN: 'sys_admin',
  SCHOOL_ADMIN: 'school_admin',
  COLLEGE_ADMIN: 'college_admin',
  COUNSELOR: 'counselor',
  TEACHER: 'teacher',
  ASSISTANT: 'assistant',
  UNION: 'student_union',
  STUDENT: 'student',
  GUEST: 'guest',
};

// 角色继承关系（高级角色继承低级角色的所有能力）
export const ROLE_HIERARCHY = {
  sys_admin: ['school_admin', 'college_admin', 'counselor', 'teacher', 'assistant', 'student_union', 'student', 'guest'],
  school_admin: ['college_admin', 'counselor', 'teacher', 'assistant', 'student_union', 'student', 'guest'],
  college_admin: ['counselor', 'teacher', 'assistant', 'student_union', 'student', 'guest'],
  counselor: ['teacher', 'student_union', 'student', 'guest'],
  teacher: ['student', 'guest'],
  assistant: ['student', 'guest'],
  student_union: ['student', 'guest'],
  student: ['guest'],
  guest: [],
};

// 能力定义
export const CAPABILITIES = {
  SELF_CHAT: 'self.chat',
  SELF_SESSION_READ: 'self.session.read',
  SELF_SESSION_DELETE: 'self.session.delete',
  SELF_KNOWLEDGE_READ: 'self.knowledge.read',
  SELF_RECOMMEND_READ: 'self.recommend.read',
  SELF_PROFILE: 'self.profile',
  SELF_CONSENT: 'self.consent',
  SELF_EMOTION_STATS: 'self.emotion.stats',
  SELF_GRADUATION_READ: 'self.graduation.read',
  SELF_GRADUATION_WRITE: 'self.graduation.write',
  SELF_COMPETITION_READ: 'self.competition.read',
  SELF_COMPETITION_WRITE: 'self.competition.write',
  SELF_PLAN_READ: 'self.plan.read',
  SELF_PLAN_WRITE: 'self.plan.write',
  SELF_PARTY_READ: 'self.party.read',
  SELF_PARTY_WRITE: 'self.party.write',
  SELF_CLUB_READ: 'self.club.read',
  SELF_CLUB_WRITE: 'self.club.write',
  SELF_FEEDBACK_SUBMIT: 'self.feedback.submit',
  SELF_STUDY_READ: 'self.study.read',
  SELF_BRIEFING_READ: 'self.briefing.read',
  SELF_DIARY_READ: 'self.diary.read',
  SELF_CHECKIN_WRITE: 'self.checkin.write',
  SELF_TWIN_READ: 'self.twin.read',
  SELF_PERSONALITY_READ: 'self.personality.read',
  SELF_ACHIEVEMENTS: 'self.achievements',
  SELF_COURSE_MAP_READ: 'self.course_map.read',
  SELF_COURSE_ANALYTICS: 'self.course_analytics',
  SELF_WEEKLY_REPORT: 'self.weekly_report',
  SELF_GENERIC_AI: 'self.generic_ai',
  SELF_COMMUNITY_READ: 'self.community.read',
  SELF_PRIVATE_CHAT: 'self.private_chat',
  SELF_PROCESS_READ: 'self.process.read',
  SELF_CAREER_READ: 'self.career.read',
  SELF_EXPORT_SELF: 'self.export.self',
  SELF_VOICE: 'self.voice',

  COUNSELOR_KB_WRITE: 'counselor.kb.write',
  COUNSELOR_KB_REVIEW: 'counselor.kb.review',
  COUNSELOR_ALERT_ANALYZE: 'counselor.alert.analyze',
  COUNSELOR_ALERT_READ: 'counselor.alert.read',
  COUNSELOR_ALERT_HANDLE: 'counselor.alert.handle',
  COUNSELOR_EMOTION_TRENDS: 'counselor.emotion.trends',
  COUNSELOR_DAILY_FOCUS_READ: 'counselor.daily_focus.read',
  COUNSELOR_CLASS_REPORT: 'counselor.class_report',
  COUNSELOR_TWIN_BOARD: 'counselor.twin_board',
  COUNSELOR_PREDICTION_READ: 'counselor.prediction.read',
  COUNSELOR_INTERVENTION_WRITE: 'counselor.intervention.write',
  COUNSELOR_TALK_RECORD: 'counselor.talk_record',
  COUNSELOR_TALK_TIPS: 'counselor.talk_tips',
  COUNSELOR_IDEOLOGICAL: 'counselor.ideological',
  COUNSELOR_CLASS_PROFILE: 'counselor.class_profile',
  COUNSELOR_COMMUNITY_MANAGE: 'counselor.community.manage',
  COUNSELOR_HOT_TOPIC_SENSE: 'counselor.hot_topic_sense',
  COUNSELOR_PROCESS_EDIT: 'counselor.process_edit',
  COUNSELOR_STUDENT_LIST: 'counselor.student_list',
  COUNSELOR_TOKEN_SUBORDINATES: 'counselor.token_subordinates',
  COUNSELOR_NOTIFY: 'counselor.notify',
  COUNSELOR_IMPORT_STUDENT: 'counselor.import_student',
  COUNSELOR_INTEGRATION_READ: 'counselor.integration.read',
  COUNSELOR_REVIEW_PENDING: 'counselor.review.pending',
  COUNSELOR_METRICS_READ: 'counselor.metrics.read',
  COUNSELOR_AUDIT_READ: 'counselor.audit.read',
  COUNSELOR_USER_READ: 'counselor.user.read',
  COUNSELOR_FORECAST: 'counselor.forecast',

  UNION_KB_SUBMIT: 'union.kb.submit',
  UNION_FEEDBACK_LIST: 'union.feedback.list',
  UNION_FEEDBACK_READ: 'union.feedback.read',
  UNION_FEEDBACK_WRITE: 'union.feedback.write',
  UNION_EVENT_PLAN: 'union.event_plan',
  UNION_POSTER_GEN: 'union.poster_gen',

  COLLEGE_METRICS_READ: 'college.metrics.read',
  COLLEGE_USER_READ: 'college.user.read',
  COLLEGE_AUDIT_READ: 'college.audit.read',
  COLLEGE_FORECAST: 'college.forecast',
  COLLEGE_GRADUATION_READ: 'college.graduation.read',
  COLLEGE_GRADUATION_WRITE: 'college.graduation.write',
  COLLEGE_TWIN_SCREEN: 'college.twin_screen',
  COLLEGE_DATA_ANALYSIS: 'college.data_analysis',

  SCHOOL_KB_SYNC_EXPORT: 'school.kb.sync.export',
  SCHOOL_USER_UPDATE: 'school.user.update',
  SCHOOL_AGENT_WRITE: 'school.agent.write',

  SYSTEM_SETTINGS_WRITE: 'system.settings.write',
  SYSTEM_PASSWORD_RESET: 'system.password_reset',
};

// 角色 -> 直接能力映射
const ROLE_CAPABILITIES = {
  guest: [
    CAPABILITIES.SELF_CONSENT,
  ],
  student: [
    CAPABILITIES.SELF_CHAT,
    CAPABILITIES.SELF_SESSION_READ,
    CAPABILITIES.SELF_SESSION_DELETE,
    CAPABILITIES.SELF_KNOWLEDGE_READ,
    CAPABILITIES.SELF_RECOMMEND_READ,
    CAPABILITIES.SELF_PROFILE,
    CAPABILITIES.SELF_CONSENT,
    CAPABILITIES.SELF_EMOTION_STATS,
    CAPABILITIES.SELF_GRADUATION_READ,
    CAPABILITIES.SELF_GRADUATION_WRITE,
    CAPABILITIES.SELF_COMPETITION_READ,
    CAPABILITIES.SELF_COMPETITION_WRITE,
    CAPABILITIES.SELF_PLAN_READ,
    CAPABILITIES.SELF_PLAN_WRITE,
    CAPABILITIES.SELF_PARTY_READ,
    CAPABILITIES.SELF_PARTY_WRITE,
    CAPABILITIES.SELF_CLUB_READ,
    CAPABILITIES.SELF_CLUB_WRITE,
    CAPABILITIES.SELF_FEEDBACK_SUBMIT,
    CAPABILITIES.SELF_STUDY_READ,
    CAPABILITIES.SELF_BRIEFING_READ,
    CAPABILITIES.SELF_DIARY_READ,
    CAPABILITIES.SELF_CHECKIN_WRITE,
    CAPABILITIES.SELF_TWIN_READ,
    CAPABILITIES.SELF_PERSONALITY_READ,
    CAPABILITIES.SELF_ACHIEVEMENTS,
    CAPABILITIES.SELF_COURSE_MAP_READ,
    CAPABILITIES.SELF_COURSE_ANALYTICS,
    CAPABILITIES.SELF_WEEKLY_REPORT,
    CAPABILITIES.SELF_GENERIC_AI,
    CAPABILITIES.SELF_COMMUNITY_READ,
    CAPABILITIES.SELF_PRIVATE_CHAT,
    CAPABILITIES.SELF_PROCESS_READ,
    CAPABILITIES.SELF_CAREER_READ,
    CAPABILITIES.SELF_EXPORT_SELF,
    CAPABILITIES.SELF_VOICE,
  ],
  student_union: [
    CAPABILITIES.UNION_KB_SUBMIT,
    CAPABILITIES.UNION_FEEDBACK_LIST,
    CAPABILITIES.UNION_FEEDBACK_READ,
    CAPABILITIES.UNION_FEEDBACK_WRITE,
    CAPABILITIES.UNION_EVENT_PLAN,
    CAPABILITIES.UNION_POSTER_GEN,
  ],
  counselor: [
    CAPABILITIES.COUNSELOR_KB_WRITE,
    CAPABILITIES.COUNSELOR_KB_REVIEW,
    CAPABILITIES.COUNSELOR_ALERT_ANALYZE,
    CAPABILITIES.COUNSELOR_ALERT_READ,
    CAPABILITIES.COUNSELOR_ALERT_HANDLE,
    CAPABILITIES.COUNSELOR_EMOTION_TRENDS,
    CAPABILITIES.COUNSELOR_DAILY_FOCUS_READ,
    CAPABILITIES.COUNSELOR_CLASS_REPORT,
    CAPABILITIES.COUNSELOR_TWIN_BOARD,
    CAPABILITIES.COUNSELOR_PREDICTION_READ,
    CAPABILITIES.COUNSELOR_INTERVENTION_WRITE,
    CAPABILITIES.COUNSELOR_TALK_RECORD,
    CAPABILITIES.COUNSELOR_TALK_TIPS,
    CAPABILITIES.COUNSELOR_IDEOLOGICAL,
    CAPABILITIES.COUNSELOR_CLASS_PROFILE,
    CAPABILITIES.COUNSELOR_COMMUNITY_MANAGE,
    CAPABILITIES.COUNSELOR_HOT_TOPIC_SENSE,
    CAPABILITIES.COUNSELOR_PROCESS_EDIT,
    CAPABILITIES.COUNSELOR_STUDENT_LIST,
    CAPABILITIES.COUNSELOR_TOKEN_SUBORDINATES,
    CAPABILITIES.COUNSELOR_NOTIFY,
    CAPABILITIES.COUNSELOR_IMPORT_STUDENT,
    CAPABILITIES.COUNSELOR_INTEGRATION_READ,
    CAPABILITIES.COUNSELOR_REVIEW_PENDING,
    CAPABILITIES.COLLEGE_METRICS_READ,
    CAPABILITIES.COLLEGE_AUDIT_READ,
    CAPABILITIES.COLLEGE_USER_READ,
    CAPABILITIES.COLLEGE_FORECAST,
  ],
  college_admin: [
    CAPABILITIES.COLLEGE_GRADUATION_READ,
    CAPABILITIES.COLLEGE_GRADUATION_WRITE,
    CAPABILITIES.COLLEGE_TWIN_SCREEN,
    CAPABILITIES.COLLEGE_DATA_ANALYSIS,
  ],
  school_admin: [
    CAPABILITIES.SCHOOL_KB_SYNC_EXPORT,
    CAPABILITIES.SCHOOL_USER_UPDATE,
    CAPABILITIES.SCHOOL_AGENT_WRITE,
  ],
  sys_admin: [
    CAPABILITIES.SYSTEM_SETTINGS_WRITE,
    CAPABILITIES.SYSTEM_PASSWORD_RESET,
  ],
  teacher: [],
  assistant: [],
};

// 获取角色的所有能力（含继承）
export function getRoleCapabilities(role) {
  const capabilities = new Set();
  const visited = new Set();

  function addCapabilities(r) {
    if (visited.has(r)) return;
    visited.add(r);

    const direct = ROLE_CAPABILITIES[r] || [];
    direct.forEach(c => capabilities.add(c));

    const inherited = ROLE_HIERARCHY[r] || [];
    inherited.forEach(child => addCapabilities(child));
  }

  addCapabilities(role);
  return Array.from(capabilities);
}

// 检查角色是否具有某项能力
export function hasCapability(role, capability) {
  const caps = getRoleCapabilities(role);
  return caps.includes(capability);
}
