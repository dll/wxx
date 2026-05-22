-- 020_seed_graduation_process.sql — 滁州学院毕业生离校流程种子数据
-- 提供"离校流程"对应的 KB 资源（resource_id='process-graduation-2026'）和步骤明细，
-- 修复 /student/process-enhanced?type=graduation 之前因数据缺失被 mock 掉的问题。

-- 离校流程 KB 资源
INSERT OR IGNORE INTO kb_resources (
    resource_id, resource_type, owner_scope, owner_id, role_scope,
    version, status, title, summary, content,
    source_link, expired_at, tags, updated_by
)
VALUES (
    'process-graduation-2026', 'Process', 'school', '', '["student","counselor"]',
    'v1.0', 'published', '滁州学院毕业生离校流程',
    '毕业生须在毕业离校前一周内完成所有离校手续，可通过"一表通"系统在线办理大部分手续。',
    '滁州学院毕业生离校流程：

毕业前两周内通过"一表通"线上系统办理，剩余线下事项在离校当天集中办理。

一、办理时间
每年6月初开放"一表通"离校系统，建议在6月20日前完成所有手续，最晚不得超过毕业证发放日。

二、办理项目
1. 图书馆：归还所有借阅图书，无欠费
2. 财务处：结清学费、住宿费、违约金等
3. 后勤：归还宿舍钥匙、清空个人物品、宿舍验收
4. 校园卡：清退余额、注销卡片
5. 信息中心：解绑校园网账号
6. 学院：领取毕业证、学位证、就业报到证、档案
7. 党委组织部：组织关系转出
8. 教务处：领取成绩单原件

三、咨询渠道
学生处：0550-3510022
学院辅导员：见班级群通知',
    'http://ybt.chzu.edu.cn/graduation',
    '2026-12-31 00:00:00',
    '["毕业","离校","流程","一表通"]',
    'admin'
);

-- 离校流程步骤明细
INSERT OR IGNORE INTO process_steps (resource_id, step_order, title, materials, entry_url, deadline, location, notes)
VALUES
('process-graduation-2026', 1, '一表通在线申请', '["学生证"]', 'http://ybt.chzu.edu.cn/graduation', '6月初开放', '一表通线上系统', '登录"一表通"提交离校申请，自动同步到下级各部门'),
('process-graduation-2026', 2, '图书馆清账', '["校园卡"]', '', '6月20日前', '图书馆借还台', '归还所有借阅图书并结清欠费，欠费 0.1 元/天'),
('process-graduation-2026', 3, '财务结清', '["缴费凭证"]', 'http://cw.chzu.edu.cn', '6月20日前', '行政楼财务处', '结清学费、住宿费、违约金等所有费用'),
('process-graduation-2026', 4, '宿舍退宿', '["宿舍钥匙","清扫物资"]', '', '6月25日前', '各学生公寓楼值班室', '清空个人物品并通过宿舍验收，归还钥匙'),
('process-graduation-2026', 5, '校园卡清退', '[]', '', '6月25日前', '一卡通服务中心', '清退余额并注销校园卡'),
('process-graduation-2026', 6, '组织关系转出', '["党员证"]', '', '6月25日前', '党委组织部', '党员需办理组织关系转出，团员材料随档案转出'),
('process-graduation-2026', 7, '档案密封', '[]', '', '6月25日前', '学院教学秘书处', '签字确认档案内容，由学院密封'),
('process-graduation-2026', 8, '领取证书', '["学生证","身份证"]', '', '毕业典礼后', '各学院党政办', '领取毕业证、学位证、成绩单原件、就业报到证');
