-- 034_fix_process_flow_seed.sql — 幂等修复办事流程为空/映射缺失

INSERT OR IGNORE INTO kb_resources (resource_id, resource_type, owner_scope, owner_id, role_scope, version, status, title, summary, content, source_link, expired_at, tags, updated_by)
VALUES
('process-registration-2026', 'Process', 'school', '', '["student","student_union","counselor","college_admin"]', 'v1.0', 'published', '新生入学报到流程', '新生报到需依次完成线上预报到、缴费、学院报到、宿舍入住、体检、学籍核验等步骤。', '新生入学报到流程：线上预报到后，按录取通知书规定时间到校，依次完成缴费、学院报到、宿舍入住、体检、学籍核验、教材领取。', 'https://www.chzu.edu.cn', '2026-12-31 00:00:00', '["新生","入学","报到","流程"]', 'system'),
('process-graduation-2026', 'Process', 'school', '', '["student","student_union","counselor","college_admin"]', 'v1.0', 'published', '毕业生离校流程', '毕业生通过一表通发起离校申请，并完成图书馆、财务、宿舍、校园卡、组织关系、学院证书领取等手续。', '毕业生离校流程：一表通在线申请，完成图书馆清账、财务结清、宿舍退宿、校园卡清退、组织关系转出、档案确认和证书领取。', 'http://ybt.chzu.edu.cn/graduation', '2026-12-31 00:00:00', '["毕业","离校","流程"]', 'system'),
('process-major-change-2026', 'Process', 'school', '', '["student","student_union","counselor","college_admin"]', 'v1.0', 'published', '转专业流程', '学生在规定时间提交转专业申请，经所在学院、拟转入学院和教务处审核后办理学籍变更。', '转专业流程：了解接收条件，填写申请表，所在学院审核，拟转入学院审核，教务处审批，学校公示，办理学籍变更。', 'http://jwc.chzu.edu.cn', '2026-12-31 00:00:00', '["转专业","学籍","流程"]', 'system'),
('process-student-loan-2026', 'Process', 'school', '', '["student","student_union","counselor","college_admin"]', 'v1.0', 'published', '助学贷款申请流程', '家庭经济困难学生可在暑期通过国家开发银行学生在线服务系统申请生源地信用助学贷款。', '助学贷款申请流程：网上申请，打印申请表，家庭经济困难认定，县区资助中心现场办理，学校回执录入，银行审核发放。', 'https://sls.cdb.com.cn', '2026-12-31 00:00:00', '["助学贷款","资助","流程"]', 'system');

DELETE FROM process_steps WHERE resource_id IN ('process-registration-2026','process-graduation-2026','process-major-change-2026','process-student-loan-2026');

INSERT INTO process_steps (resource_id, step_order, title, materials, entry_url, deadline, location, notes)
VALUES
('process-registration-2026', 1, '线上预报到', '["录取通知书","身份证"]', 'https://yx.chzu.edu.cn', '报到前完成', '迎新系统', '完成个人信息确认和到校信息登记'),
('process-registration-2026', 2, '缴纳学杂费', '["银行卡","缴费凭证"]', 'http://cw.chzu.edu.cn', '报到前或报到日', '财务系统/现场缴费点', '助学贷款学生携带贷款回执'),
('process-registration-2026', 3, '学院报到', '["录取通知书","身份证","档案"]', '', '报到日', '计算机学院报到点', '领取班级、辅导员和校园卡信息'),
('process-registration-2026', 4, '宿舍入住', '["校园卡","身份证"]', '', '报到日', '学生公寓', '按分配宿舍领取钥匙并入住'),
('process-registration-2026', 5, '入学体检与学籍核验', '["身份证","体检表"]', '', '入学后两周内', '校医院/教务处', '按学院通知分批完成'),
('process-graduation-2026', 1, '一表通在线申请', '["学生证"]', 'http://ybt.chzu.edu.cn/graduation', '6月初开放', '一表通线上系统', '提交离校申请'),
('process-graduation-2026', 2, '图书馆与财务清账', '["校园卡","缴费凭证"]', '', '6月20日前', '图书馆/财务处', '归还图书并结清欠费'),
('process-graduation-2026', 3, '宿舍退宿与校园卡清退', '["宿舍钥匙","校园卡"]', '', '6月25日前', '学生公寓/一卡通中心', '完成宿舍验收和余额清退'),
('process-graduation-2026', 4, '组织关系与档案确认', '["党员证","档案确认单"]', '', '6月25日前', '学院/党委组织部', '党员需转出组织关系'),
('process-graduation-2026', 5, '领取毕业证书', '["身份证","学生证"]', '', '毕业典礼后', '学院党政办', '领取毕业证、学位证和成绩单'),
('process-major-change-2026', 1, '了解接收条件', '[]', 'http://jwc.chzu.edu.cn', '每年5月/11月', '教务处/学院官网', '查看转入专业条件和名额'),
('process-major-change-2026', 2, '提交申请材料', '["转专业申请表","成绩单","个人陈述"]', '', '第12-14周', '所在学院教学办公室', '填写并提交申请表'),
('process-major-change-2026', 3, '学院与教务处审核', '["完整申请表","成绩单"]', '', '学期末', '所在学院/拟转入学院/教务处', '完成多级审批和公示'),
('process-major-change-2026', 4, '办理学籍变更', '[]', 'http://jwc.chzu.edu.cn', '公示期满后', '教务处学籍科', '完成学籍信息变更'),
('process-student-loan-2026', 1, '网上申请', '[]', 'https://sls.cdb.com.cn', '7月-9月', '国家开发银行学生在线系统', '注册并填写贷款申请'),
('process-student-loan-2026', 2, '打印并认定申请表', '["申请表"]', '', '7月-9月', '户籍地村居/乡镇', '完成家庭经济困难认定'),
('process-student-loan-2026', 3, '现场签订合同', '["身份证","录取通知书/学生证","户口簿"]', '', '7月-9月', '县区学生资助中心', '学生和共同借款人到场办理'),
('process-student-loan-2026', 4, '学校回执录入', '["受理证明"]', '', '开学后一周内', '学校学生资助中心', '提交回执并等待贷款发放');
