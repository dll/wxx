-- 修正 048 种子数据中的错误坐标（会峰/琅琊写反、琅琊偏移 2km）
-- 数据来源：OpenStreetMap way 734826227（会峰）、734826234（琅琊）
-- 会峰校区中心 32.2743/118.3055，琅琊校区中心 32.2943/118.2978
-- 此前 048 种子用的旧坐标：会峰(32.2921,118.2988)实为琅琊位置，
-- 琅琊(32.3136,118.3098)偏到校区以北 2km，现按 OSM 权威数据纠正。

-- 会峰校区 6 个步骤（原坐标指向琅琊位置，现纠正到会峰校区内）
UPDATE campus_checkin_steps SET lat=32.2705, lng=118.3055, updated_at=datetime('now')
  WHERE campus_id='huifeng' AND step_order=1;
UPDATE campus_checkin_steps SET lat=32.2745, lng=118.3070, updated_at=datetime('now')
  WHERE campus_id='huifeng' AND step_order=2;
UPDATE campus_checkin_steps SET lat=32.2735, lng=118.3060, updated_at=datetime('now')
  WHERE campus_id='huifeng' AND step_order=3;
UPDATE campus_checkin_steps SET lat=32.2770, lng=118.3040, updated_at=datetime('now')
  WHERE campus_id='huifeng' AND step_order=4;
UPDATE campus_checkin_steps SET lat=32.2740, lng=118.3090, updated_at=datetime('now')
  WHERE campus_id='huifeng' AND step_order=5;
UPDATE campus_checkin_steps SET lat=32.2720, lng=118.3030, updated_at=datetime('now')
  WHERE campus_id='huifeng' AND step_order=6;

-- 琅琊校区 6 个步骤（原坐标偏移到校区以北 2km，现纠正到琅琊校区内）
UPDATE campus_checkin_steps SET lat=32.2921, lng=118.2988, updated_at=datetime('now')
  WHERE campus_id='langya' AND step_order=1;
UPDATE campus_checkin_steps SET lat=32.2932, lng=118.3002, updated_at=datetime('now')
  WHERE campus_id='langya' AND step_order=2;
UPDATE campus_checkin_steps SET lat=32.2928, lng=118.2995, updated_at=datetime('now')
  WHERE campus_id='langya' AND step_order=3;
UPDATE campus_checkin_steps SET lat=32.2940, lng=118.2976, updated_at=datetime('now')
  WHERE campus_id='langya' AND step_order=4;
UPDATE campus_checkin_steps SET lat=32.2926, lng=118.3000, updated_at=datetime('now')
  WHERE campus_id='langya' AND step_order=5;
UPDATE campus_checkin_steps SET lat=32.2917, lng=118.2992, updated_at=datetime('now')
  WHERE campus_id='langya' AND step_order=6;
