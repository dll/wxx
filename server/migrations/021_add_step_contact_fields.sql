-- 021_add_step_contact_fields.sql — 给 process_steps 表补充步骤级详细信息字段
-- 修复审核问题 §4.1：联系人/电话/办公时间/FAQ 四类信息恒为空
-- 前端 ProcessStepDetail 已定义并渲染这些字段，后端补齐表结构、实体与种子数据

-- 1. 新增列（SQLite 不支持 ADD COLUMN IF NOT EXISTS，用 ALTER TABLE ADD COLUMN）
ALTER TABLE process_steps ADD COLUMN contact      TEXT NOT NULL DEFAULT '';
ALTER TABLE process_steps ADD COLUMN phone        TEXT NOT NULL DEFAULT '';
ALTER TABLE process_steps ADD COLUMN office_hours TEXT NOT NULL DEFAULT '';
ALTER TABLE process_steps ADD COLUMN faq          TEXT NOT NULL DEFAULT '[]';  -- JSON 数组：[{"q":"…","a":"…"}]

-- 2. 补充新生入学报到流程联系人信息
UPDATE process_steps SET contact='招生办', phone='0550-3510011', office_hours='报到日 8:00-18:00', faq='[{"q":"录取通知书丢了怎么办？","a":"联系招生办 0550-3510011，凭身份证核实身份后可正常报到"}]' WHERE resource_id='process-registration-2026' AND step_order=1;
UPDATE process_steps SET contact='财务处', phone='0550-3510033', office_hours='报到日 8:00-18:00', faq='[{"q":"可以现场缴费吗？","a":"建议提前在线缴费避免排队，现场也设有缴费窗口"},{"q":"办理了助学贷款还需要缴费吗？","a":"贷款到账后自动抵扣学费住宿费，报到时出示贷款凭证即可"}]' WHERE resource_id='process-registration-2026' AND step_order=2;
UPDATE process_steps SET contact='各学院辅导员', phone='见班级群通知', office_hours='报到日 8:00-18:00', faq='[]' WHERE resource_id='process-registration-2026' AND step_order=3;
UPDATE process_steps SET contact='公寓值班室', phone='0550-3510088', office_hours='报到日 8:00-20:00', faq='[{"q":"床上用品需要自带吗？","a":"学校不统一配发，请自行携带或到校后购买"}]' WHERE resource_id='process-registration-2026' AND step_order=4;
UPDATE process_steps SET contact='校医院', phone='0550-3510120', office_hours='按学院安排批次，抽血须在上午完成', faq='[{"q":"体检需要空腹吗？","a":"抽血项目须空腹，建议上午尽早前往，可携带早餐抽血后食用"}]' WHERE resource_id='process-registration-2026' AND step_order=5;
UPDATE process_steps SET contact='教务处学籍科', phone='0550-3510015', office_hours='报到日 8:00-18:00', faq='[]' WHERE resource_id='process-registration-2026' AND step_order=6;
UPDATE process_steps SET contact='教材中心', phone='0550-3510081', office_hours='报到日 8:00-18:00', faq='[]' WHERE resource_id='process-registration-2026' AND step_order=7;

-- 3. 补充毕业生离校流程联系人信息
UPDATE process_steps SET contact='信息中心', phone='0550-3510999', office_hours='工作日 8:30-11:30 14:30-17:00', faq='[{"q":"一表通系统在哪里登录？","a":"访问 http://ybt.chzu.edu.cn/graduation，使用学号和密码登录"},{"q":"忘记一表通密码怎么办？","a":"联系信息中心 0550-3510999 重置"}]' WHERE resource_id='process-graduation-2026' AND step_order=1;
UPDATE process_steps SET contact='图书馆借还台', phone='0550-3510088', office_hours='工作日 8:00-17:30（中午不休）', faq='[{"q":"图书馆欠费怎么查？","a":"登录图书馆网站或到借还台查询，也可通过"今日校园"APP查询"}]' WHERE resource_id='process-graduation-2026' AND step_order=2;
UPDATE process_steps SET contact='财务处', phone='0550-3510033', office_hours='工作日 8:30-11:30 14:30-17:00', faq='[{"q":"学费欠费如何补缴？","a":"登录 http://cw.chzu.edu.cn 在线缴费，或到行政楼财务处现场缴费"}]' WHERE resource_id='process-graduation-2026' AND step_order=3;
UPDATE process_steps SET contact='公寓值班室', phone='0550-3510088', office_hours='工作日 8:00-17:30', faq='[{"q":"宿舍押金怎么退？","a":"验收合格后由后勤统一退还至学生银行卡，一般离校后1个月内到账"}]' WHERE resource_id='process-graduation-2026' AND step_order=4;
UPDATE process_steps SET contact='一卡通服务中心', phone='0550-3510090', office_hours='工作日 9:00-16:30', faq='[{"q":"校园卡余额退到哪？","a":"余额退还至学生本人银行卡，请携带银行卡和身份证办理"},{"q":"校园卡需要注销吗？","a":"毕业离校前须注销校园卡，未注销的卡将在离校后自动冻结"}]' WHERE resource_id='process-graduation-2026' AND step_order=5;
UPDATE process_steps SET contact='党委组织部', phone='0550-3510020', office_hours='工作日 8:30-11:30 14:30-17:00', faq='[{"q":"不是党员需要办这个吗？","a":"非党员跳过此步骤，但团员材料随档案转出，无需单独办理"},{"q":"组织关系转到哪里？","a":"已就业的转到工作单位党组织，未就业的转到户籍地社区党组织"}]' WHERE resource_id='process-graduation-2026' AND step_order=6;
UPDATE process_steps SET contact='学院教学秘书', phone='见班级群通知', office_hours='工作日 8:30-11:30 14:30-17:00', faq='[]' WHERE resource_id='process-graduation-2026' AND step_order=7;
UPDATE process_steps SET contact='学院党政办', phone='见班级群通知', office_hours='工作日 8:30-11:30 14:30-17:00', faq='[{"q":"可以代领证书吗？","a":"一般须本人携带身份证领取，特殊情况需凭委托书和双方身份证复印件代领"}]' WHERE resource_id='process-graduation-2026' AND step_order=8;
