-- 035_seed_common_processes.sql — 补齐常用办事流程：请假 / 奖学金

INSERT OR IGNORE INTO kb_resources (resource_id, resource_type, owner_scope, owner_id, role_scope, version, status, title, summary, content, source_link, expired_at, tags, updated_by)
VALUES
('process-leave-2026', 'Process', 'school', '', '["student","student_union","counselor","college_admin"]', 'v1.0', 'published', '学生请假办理流程', '学生因病、因事需要离校或缺课时，应提前提交请假申请，经辅导员和学院按权限审批后执行，返校后及时销假。', '学生请假办理流程：提交请假申请，说明请假原因、时间、去向和联系方式；辅导员审核；超过规定天数或重要事项由学院审批；请假结束后返校销假。病假应提供医院证明，事假应提供必要说明或证明材料。', '', '2026-12-31 00:00:00', '["请假","审批","销假","流程"]', 'system'),
('process-scholarship-2026', 'Process', 'school', '', '["student","student_union","counselor","college_admin"]', 'v1.0', 'published', '奖学金申请流程', '奖学金申请按学校和学院评选通知执行，学生准备申请表、成绩和荣誉材料，经班级评议、学院审核、公示和学校审定后发放。', '奖学金申请流程：查看评选通知，确认奖项类别和条件；准备申请表、成绩单、荣誉证明和综测材料；班级评议；学院审核排序；学院和学校公示；学校审定后发放并归档。', '', '2026-12-31 00:00:00', '["奖学金","评奖评优","资助","流程"]', 'system');

DELETE FROM process_steps WHERE resource_id IN ('process-leave-2026','process-scholarship-2026');

INSERT INTO process_steps (resource_id, step_order, title, materials, entry_url, deadline, location, notes)
VALUES
('process-leave-2026', 1, '提交请假申请', '["请假事由说明","证明材料（病假需医院证明）"]', '', '离校或缺课前提交', '辅导员/学院线上表单', '说明请假起止时间、去向、联系方式和安全承诺'),
('process-leave-2026', 2, '辅导员审核', '[]', '', '提交后1个工作日内', '辅导员办公室', '辅导员核实请假原因、去向和联系人'),
('process-leave-2026', 3, '学院审批', '["请假申请表","证明材料"]', '', '按学院要求', '学院学生工作办公室', '请假天数较长或特殊情况需学院审批'),
('process-leave-2026', 4, '离校/缺课期间保持联系', '[]', '', '请假期间', '线上联系', '保持电话畅通，情况变化及时报告辅导员'),
('process-leave-2026', 5, '返校销假', '[]', '', '返校当日', '辅导员/班级群', '返校后及时销假并更新在校状态'),
('process-scholarship-2026', 1, '查看评选通知', '[]', '', '每学年评选期', '学院官网/班级群', '确认奖项类别、名额、申请条件和时间安排'),
('process-scholarship-2026', 2, '准备申请材料', '["申请表","成绩单","荣誉证明","综测材料"]', '', '通知规定时间内', '所在学院', '按奖项要求准备纸质或电子材料'),
('process-scholarship-2026', 3, '班级评议', '["完整申请材料"]', '', '学院评审期', '班级/辅导员', '完成民主评议和材料初核'),
('process-scholarship-2026', 4, '学院审核与公示', '["完整申请材料"]', '', '学院公示期', '学院学生工作办公室', '学院审核排序并公示，接受异议反馈'),
('process-scholarship-2026', 5, '学校审定', '[]', '', '学校评审期', '学生处/相关职能部门', '学校复核审定评选结果'),
('process-scholarship-2026', 6, '发放与归档', '["银行卡信息"]', '', '审定后', '财务处/学院', '奖助资金发放并完成材料归档');
