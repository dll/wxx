-- 079_clear_preloaded_data.sql — 清理预置/演示业务数据
--
-- 背景：
--   上线前清空所有迁移注入的演示业务数据，改由管理员经管理界面/导入重新
--   上传真实资源后提供服务（需求：管理员上传资源重建）。
--
-- 清理范围（按外键依赖顺序 DELETE）：
--   1. 知识库 kb_resources（60 条种子）+ 办事流程 process_steps/process_reminders
--   2. AI 简讯 ai_briefings（20+5 条内置）+ ai_briefing_favorites
--   3. 竞赛 competitions（5 条）+ 报名
--   4. 毕设 advisors/thesis_topics/milestones（样例导师/选题/里程碑）
--   5. 就业 career_policies（2 条）
--   6. 校历 academic_calendars/events、示例课表 course_schedules（user_id=1）
--   7. 测试站内通知 user_notifications（user_id=1）
--   8. 报到打卡点 campus_checkin_steps（12 条）
--   9. 健康活动 health_activities（6 条演示活动）
--
-- 保留（不动）：
--   - 系统必需项：admin 账号、system_settings 默认配置、agents 系统智能体、
--     app_versions 版本表、FTS 触发器与索引
--   - 无管理入口模块的种子：社团 clubs、心理 psych_*/crisis_hotlines/counselors/
--     psych_articles、入党 party_stages、学习资源 learning_resources、规划模板 plan_templates
--   - 全部用户账号（14 个种子账号按需求保留）
--   - 学生产生的真实业务数据（打卡/问答/积分/反馈/会话等）
--
-- 注意：kb_resources 的 DELETE 会经 kb_fts_delete 触发器同步清空 FTS 索引。

-- ── 1. 办事流程（先于 kb_resources，因 resource_id/process_id 外键引用） ──
DELETE FROM process_reminders;
DELETE FROM process_steps;

-- ── 2. 知识库 ──
DELETE FROM kb_resources;

-- ── 3. AI 简讯（先删收藏，再删资讯本体） ──
DELETE FROM ai_briefing_favorites;
DELETE FROM ai_briefings;

-- ── 4. 竞赛（先删报名，再删竞赛） ──
DELETE FROM competition_registrations;
DELETE FROM competitions;

-- ── 5. 毕设（先删选题记录/进度，再删选题/导师/里程碑） ──
DELETE FROM student_topic_selections;
DELETE FROM graduation_progress;
DELETE FROM thesis_topics;
DELETE FROM advisors;
DELETE FROM graduation_milestones;

-- ── 6. 就业 ──
DELETE FROM career_policies;

-- ── 7. 校历与示例课表 ──
DELETE FROM academic_calendar_events;
DELETE FROM academic_calendars;
DELETE FROM course_schedules;

-- ── 8. 测试站内通知 ──
DELETE FROM user_notifications;

-- ── 9. 报到打卡点 ──
DELETE FROM campus_checkin_steps;

-- ── 10. 健康活动（先删关注/报名，再删活动） ──
DELETE FROM health_activity_favorites;
DELETE FROM health_activity_signups;
DELETE FROM health_activities;