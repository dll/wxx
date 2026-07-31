-- 校园报到步骤动态管理表
-- 替代前端硬编码常量，支持管理员编辑 → 审核 → 发布工作流
CREATE TABLE IF NOT EXISTS campus_checkin_steps (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    campus_id   TEXT    NOT NULL DEFAULT 'huifeng', -- huifeng | langya
    step_order  INTEGER NOT NULL DEFAULT 0,
    title       TEXT    NOT NULL DEFAULT '',
    location    TEXT    NOT NULL DEFAULT '',
    lat         REAL    NOT NULL DEFAULT 0,
    lng         REAL    NOT NULL DEFAULT 0,
    duration    TEXT    NOT NULL DEFAULT '',
    task        TEXT    NOT NULL DEFAULT '',
    materials   TEXT    NOT NULL DEFAULT '',
    contact     TEXT    NOT NULL DEFAULT '',
    note        TEXT    NOT NULL DEFAULT '',
    icon_name   TEXT    NOT NULL DEFAULT 'place',
    status      TEXT    NOT NULL DEFAULT 'published', -- draft | pending_review | published
    created_by  INTEGER,
    reviewed_by INTEGER,
    published_at TEXT,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- 初始种子数据（与前端 campus_map_page.dart 硬编码常量保持一致）
INSERT OR IGNORE INTO campus_checkin_steps
    (campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status)
VALUES
('huifeng',1,'校门入校核验','会峰校区南门',32.2921,118.2988,'约 5 分钟','核验录取通知书、身份证，按学院引导进入校园。','录取通知书、身份证','迎新志愿者 / 保卫处 0550-3510110','建议提前准备证件，车辆服从现场引导。','login','published'),
('huifeng',2,'学院报到','计算机学院报到点',32.2932,118.3005,'约 15 分钟','领取班级信息、辅导员联系方式、报到流程单。','录取通知书、身份证、档案袋','学院辅导员，见班级群通知','这是后续宿舍、体检等流程的起点。','account_balance','published'),
('huifeng',3,'缴费与绿色通道','财务缴费点 / 绿色通道',32.2928,118.2995,'约 10-20 分钟','完成学杂费确认，助学贷款或缓缴学生办理绿色通道。','缴费凭证、贷款受理证明（如有）','财务处 0550-3510033','已线上缴费的学生现场只需核验状态。','payments','published'),
('huifeng',4,'宿舍入住','学生公寓楼值班室',32.2940,118.2976,'约 15 分钟','确认宿舍信息，领取钥匙，办理入住。','校园卡或身份证','公寓值班室 0550-3510088','入住后请检查床位、门锁、水电设施。','bed','published'),
('huifeng',5,'校园卡与网络','一卡通/信息服务点',32.2926,118.3000,'约 10 分钟','领取或激活校园卡，开通校园网账号。','身份证、学号信息','信息中心 0550-3510999','校园卡用于门禁、食堂、图书馆等场景。','credit_card','published'),
('huifeng',6,'入学体检与学籍核验','校医院 / 教务处学籍点',32.2917,118.2992,'约 30-45 分钟','按学院批次完成体检、照片采集和学籍信息核验。','身份证、体检表、录取通知书','校医院 0550-3510120 / 教务处 0550-3510015','抽血项目一般需空腹，请按学院通知批次办理。','health_and_safety','published'),
('langya',1,'校门入校核验','琅琊校区主入口',32.3136,118.3098,'约 5 分钟','核验录取通知书、身份证，确认学院迎新引导点。','录取通知书、身份证','迎新志愿者 / 保卫处 0550-3510110','老校区道路较集中，请按现场志愿者指引步行前往报到点。','login','published'),
('langya',2,'学院报到','琅琊校区学院集中报到点',32.3142,118.3107,'约 15 分钟','领取班级信息、辅导员联系方式、报到流程单。','录取通知书、身份证、档案袋','学院辅导员，见班级群通知','如专业报到点调整，以现场公告和蔚小芯通知为准。','account_balance','published'),
('langya',3,'缴费与绿色通道','琅琊校区综合服务点',32.3140,118.3094,'约 10-20 分钟','完成学杂费确认，助学贷款或缓缴学生办理绿色通道。','缴费凭证、贷款受理证明（如有）','财务处 0550-3510033','线上已缴费学生可快速核验，未缴费学生按现场窗口办理。','payments','published'),
('langya',4,'宿舍入住','琅琊校区学生公寓值班室',32.3150,118.3089,'约 15 分钟','确认宿舍信息，领取钥匙，办理入住。','校园卡或身份证','公寓值班室 0550-3510088','入住后请检查床位、门锁、水电设施，问题现场登记。','bed','published'),
('langya',5,'校园卡与网络','琅琊校区信息服务点',32.3138,118.3101,'约 10 分钟','领取或激活校园卡，开通校园网账号。','身份证、学号信息','信息中心 0550-3510999','校园卡用于门禁、食堂、图书馆等场景。','credit_card','published'),
('langya',6,'入学体检与学籍核验','琅琊校区医务/学籍核验点',32.3147,118.3102,'约 30-45 分钟','按学院批次完成体检、照片采集和学籍信息核验。','身份证、体检表、录取通知书','校医院 0550-3510120 / 教务处 0550-3510015','抽血项目一般需空腹，请按学院通知批次办理。','health_and_safety','published');
