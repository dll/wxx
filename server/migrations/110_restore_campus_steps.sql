-- 110_restore_campus_steps.sql — 恢复报到打卡点种子数据
--
-- 背景：
--   079_clear_preloaded_data.sql 在上线前清空了 campus_checkin_steps 表全部 12 条
--   种子数据（会峰6+琅琊6），导致管理员进入校园报到地图后，admin 接口返回空列表，
--   前端回退到硬编码假节点，_remoteIds 保持为空，拖拽保存坐标永远失败（坐标无法持久化）。
--
-- 本迁移恢复这 12 条报到节点，坐标采用 050_fix_campus_step_coords.sql 纠正后的权威
--   WGS-84 值（与前端 campus_map_page.dart 硬编码常量一致），而非 048 的错误坐标。
--
-- 业务字段（title/location/duration/task/materials/contact/note/icon_name）从
--   048_campus_map_steps.sql 的种子数据精确复制；仅 lat/lng 用 050 纠正值。
--   status='published'，id 依赖自增（不显式指定）。
--
-- 【幂等性 & 审核流约束（reviewer 回修 H1/H2）】
--   原 110 曾用 `CREATE UNIQUE INDEX (campus_id, step_order)` + `INSERT OR IGNORE`
--   实现幂等去重，但存在两处高危缺陷，已废弃该方案：
--
--   H1：建唯一索引会中断迁移。079 清空后管理员可能已用重复 step_order 重建了部分
--       节点，此时建唯一索引会触发 "Duplicate entry" 报错，而 execSQL 的容错只识别
--       "索引已存在/duplicate key name"，不识别 "duplicate entry"，导致迁移失败、
--       服务无法启动。
--
--   H2：(campus_id, step_order) 不含 status，会阻碍审核流。审核流为
--       draft → pending_review → published，多状态行在同表共存，管理员新建同
--       step_order 的 draft、或为已发布节点再建 draft（多版本）都是合法操作；
--       唯一索引会把此类合法操作变成无意义的 500。
--
--   据此，step_order 并非唯一业务键（仅用于排序），campus_checkin_steps 表
--   自 048 起也只定义 id 主键、无任何 (campus_id, step_order) 唯一约束，
--   业务层 Create/Update/Submit/Publish 亦不依赖该唯一约束。
--
--   因此【不建任何唯一索引】，改为在 INSERT 前用 `WHERE NOT EXISTS` 判断同
--   (campus_id, step_order, status='published') 是否已存在，据此决定是否插入。
--   该写法：
--     - 幂等：重复执行时，已存在的 12 条命中 NOT EXISTS 判定 → 不再插入。
--     - 不丢数据：不 DELETE、不触碰管理员已有或已重建的同 step_order 节点。
--     - 不阻碍审核流：draft/pending_review 的同 step_order 节点不影响判定，
--       即便已存在，也会插入一条独立的 published 种子（与业务多状态共存一致）。
--     - 双方言通用：SQLite 与 MySQL 均支持 INSERT ... SELECT ... WHERE NOT EXISTS。

-- 会峰校区（6 步）
INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'huifeng',1,'校门入校核验','会峰校区南门',32.2705,118.3055,'约 5 分钟','核验录取通知书、身份证，按学院引导进入校园。','录取通知书、身份证','迎新志愿者 / 保卫处 0550-3510110','建议提前准备证件，车辆服从现场引导。','login','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='huifeng' AND step_order=1 AND status='published');

INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'huifeng',2,'学院报到','计算机学院报到点',32.2745,118.3070,'约 15 分钟','领取班级信息、辅导员联系方式、报到流程单。','录取通知书、身份证、档案袋','学院辅导员，见班级群通知','这是后续宿舍、体检等流程的起点。','account_balance','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='huifeng' AND step_order=2 AND status='published');

INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'huifeng',3,'缴费与绿色通道','财务缴费点 / 绿色通道',32.2735,118.3060,'约 10-20 分钟','完成学杂费确认，助学贷款或缓缴学生办理绿色通道。','缴费凭证、贷款受理证明（如有）','财务处 0550-3510033','已线上缴费的学生现场只需核验状态。','payments','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='huifeng' AND step_order=3 AND status='published');

INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'huifeng',4,'宿舍入住','学生公寓楼值班室',32.2770,118.3040,'约 15 分钟','确认宿舍信息，领取钥匙，办理入住。','校园卡或身份证','公寓值班室 0550-3510088','入住后请检查床位、门锁、水电设施。','bed','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='huifeng' AND step_order=4 AND status='published');

INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'huifeng',5,'校园卡与网络','一卡通/信息服务点',32.2740,118.3090,'约 10 分钟','领取或激活校园卡，开通校园网账号。','身份证、学号信息','信息中心 0550-3510999','校园卡用于门禁、食堂、图书馆等场景。','credit_card','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='huifeng' AND step_order=5 AND status='published');

INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'huifeng',6,'入学体检与学籍核验','校医院 / 教务处学籍点',32.2720,118.3030,'约 30-45 分钟','按学院批次完成体检、照片采集和学籍信息核验。','身份证、体检表、录取通知书','校医院 0550-3510120 / 教务处 0550-3510015','抽血项目一般需空腹，请按学院通知批次办理。','health_and_safety','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='huifeng' AND step_order=6 AND status='published');

-- 琅琊校区（6 步）
INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'langya',1,'校门入校核验','琅琊校区主入口',32.2921,118.2988,'约 5 分钟','核验录取通知书、身份证，确认学院迎新引导点。','录取通知书、身份证','迎新志愿者 / 保卫处 0550-3510110','老校区道路较集中，请按现场志愿者指引步行前往报到点。','login','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='langya' AND step_order=1 AND status='published');

INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'langya',2,'学院报到','琅琊校区学院集中报到点',32.2932,118.3002,'约 15 分钟','领取班级信息、辅导员联系方式、报到流程单。','录取通知书、身份证、档案袋','学院辅导员，见班级群通知','如专业报到点调整，以现场公告和蔚小芯通知为准。','account_balance','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='langya' AND step_order=2 AND status='published');

INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'langya',3,'缴费与绿色通道','琅琊校区综合服务点',32.2928,118.2995,'约 10-20 分钟','完成学杂费确认，助学贷款或缓缴学生办理绿色通道。','缴费凭证、贷款受理证明（如有）','财务处 0550-3510033','线上已缴费学生可快速核验，未缴费学生按现场窗口办理。','payments','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='langya' AND step_order=3 AND status='published');

INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'langya',4,'宿舍入住','琅琊校区学生公寓值班室',32.2940,118.2976,'约 15 分钟','确认宿舍信息，领取钥匙，办理入住。','校园卡或身份证','公寓值班室 0550-3510088','入住后请检查床位、门锁、水电设施，问题现场登记。','bed','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='langya' AND step_order=4 AND status='published');

INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'langya',5,'校园卡与网络','琅琊校区信息服务点',32.2926,118.3000,'约 10 分钟','领取或激活校园卡，开通校园网账号。','身份证、学号信息','信息中心 0550-3510999','校园卡用于门禁、食堂、图书馆等场景。','credit_card','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='langya' AND step_order=5 AND status='published');

INSERT INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
SELECT 'langya',6,'入学体检与学籍核验','琅琊校区医务/学籍核验点',32.2917,118.2992,'约 30-45 分钟','按学院批次完成体检、照片采集和学籍信息核验。','身份证、体检表、录取通知书','校医院 0550-3510120 / 教务处 0550-3510015','抽血项目一般需空腹，请按学院通知批次办理。','health_and_safety','published'
WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id='langya' AND step_order=6 AND status='published');
