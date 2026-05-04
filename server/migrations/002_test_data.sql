-- 插入测试知识数据

-- 1. 奖学金政策
INSERT INTO kb_resources (
    resource_id, resource_type, owner_scope, owner_id, role_scope,
    version, status, title, summary, content,
    source_link, effective_at, tags, updated_by
) VALUES (
    'policy-scholarship-2026', 'Policy', 'school', '', '["student","counselor","teacher"]',
    'v1.0', 'published', '2026年度国家奖学金评选办法',
    '国家奖学金用于奖励特别优秀的全日制本专科学生，奖励标准为每人每年8000元。',
    '一、申请条件
1. 热爱社会主义祖国，拥护中国共产党的领导
2. 遵守宪法和法律，遵守学校规章制度
3. 诚实守信，道德品质优良
4. 在校期间学习成绩优异，社会实践、创新能力、综合素质等方面特别突出
5. 上一学年平均学分绩点（GPA）排名在本专业前10%，且无不及格科目

二、评选时间
每年9月15日至10月15日

三、申请材料
1. 《国家奖学金申请表》
2. 成绩单（教务处盖章）
3. 获奖证书复印件
4. 个人事迹材料（2000字以内）

四、评选流程
学生申请 → 班级评议 → 学院审核 → 学校评审委员会审定 → 公示（5个工作日）→ 上报教育部

五、注意事项
- 同一学年内，获得国家奖学金的学生不能同时获得国家励志奖学金
- 材料必须真实，如有弄虚作假，取消评选资格并记入诚信档案',
    'http://jwc.example.edu.cn/scholarship/2026',
    '2026-09-01 00:00:00',
    '["奖学金","国家奖学金","评选"]',
    'admin'
);

-- 2. 转专业流程
INSERT INTO kb_resources (
    resource_id, resource_type, owner_scope, owner_id, role_scope,
    version, status, title, summary, content,
    source_link, effective_at, tags, updated_by
) VALUES (
    'process-major-transfer-2026', 'Process', 'school', '', '["student","counselor"]',
    'v1.0', 'published', '本科生转专业办理流程',
    '符合条件的学生可在第一学年结束后申请转专业，每年6月办理。',
    '一、申请条件
1. 在校全日制本科一年级学生
2. 第一学年所有课程成绩合格，无违纪处分
3. 平均学分绩点（GPA）≥3.0
4. 转入专业有接收名额

二、不得转专业的情况
1. 艺术类、体育类专业学生
2. 定向生、委培生
3. 已转过一次专业的学生
4. 休学、保留学籍期间的学生

三、办理时间
每年6月1日至6月30日

四、办理流程
第1步：在线申请（教务系统）
第2步：转出学院审核（3个工作日）
第3步：转入学院考核（笔试+面试，具体时间由学院通知）
第4步：教务处审批（5个工作日）
第5步：公示（3个工作日）
第6步：办理学籍异动手续

五、所需材料
1. 《转专业申请表》（系统打印）
2. 第一学年成绩单
3. 个人陈述（1000字，说明转专业理由）
4. 转入学院要求的其他材料

六、注意事项
- 转专业后，需补修转入专业已开设但本人未修的必修课程
- 转专业成功后，学费按转入专业标准收取
- 咨询电话：教务处 0571-88888888',
    'http://jwc.example.edu.cn/major-transfer',
    '2026-06-01 00:00:00',
    '["转专业","学籍异动","流程"]',
    'admin'
);

-- 3. 毕业生离校手续 FAQ
INSERT INTO kb_resources (
    resource_id, resource_type, owner_scope, owner_id, role_scope,
    version, status, title, summary, content,
    source_link, effective_at, tags, updated_by
) VALUES (
    'faq-graduation-2026', 'FAQ', 'school', '', '["student"]',
    'v1.0', 'published', '毕业生离校手续常见问题',
    '毕业生离校需办理图书馆、宿舍、财务等多个部门的手续，可通过"一表通"系统在线办理。',
    'Q1: 什么时候开始办理离校手续？
A: 每年6月初开放"一表通"离校系统，建议在6月20日前完成所有手续。

Q2: 离校手续包括哪些？
A: 包括以下8个环节：
1. 图书馆：归还所有借阅图书，结清欠款
2. 宿舍管理中心：退宿、验收宿舍
3. 财务处：结清学费、住宿费等费用
4. 学院：领取毕业证、学位证
5. 档案馆：确认档案转递地址
6. 校医院：结清医疗费用
7. 网络中心：注销校园网账号
8. 保卫处：上交校园卡

Q3: 可以代办吗？
A: 图书归还、宿舍退宿必须本人办理，其他环节可委托他人代办（需提供委托书和双方身份证复印件）。

Q4: 如果有图书逾期怎么办？
A: 需先缴纳逾期费用（0.1元/天），然后才能办理离校手续。

Q5: 档案寄往哪里？
A:
- 已签约单位：寄往单位人事部门
- 未就业：寄回生源地人才市场
- 升学：寄往录取学校
请在系统中准确填写档案接收地址和邮编。

Q6: 离校手续办理完成后多久能拿到毕业证？
A: 所有手续办理完成后，学院会在3个工作日内发放毕业证和学位证。

Q7: 遇到问题找谁？
A:
- 系统问题：信息中心 0571-88888881
- 手续问题：学工部 0571-88888882
- 证书问题：教务处 0571-88888883',
    'http://ybt.example.edu.cn/graduation-faq',
    '2026-06-01 00:00:00',
    '["毕业","离校手续","FAQ"]',
    'admin'
);

-- 插入转专业流程的步骤明细
INSERT INTO process_steps (resource_id, step_order, title, materials, entry_url, deadline, location, notes)
VALUES
('process-major-transfer-2026', 1, '在线申请', '[]', 'http://jwc.example.edu.cn/system', '6月10日前', '教务系统', '登录教务系统填写《转专业申请表》'),
('process-major-transfer-2026', 2, '转出学院审核', '["成绩单"]', '', '申请后3个工作日', '转出学院教务办', '学院审核学生资格'),
('process-major-transfer-2026', 3, '转入学院考核', '["个人陈述","其他材料"]', '', '6月15日-20日', '转入学院指定地点', '参加笔试和面试，具体时间由学院通知'),
('process-major-transfer-2026', 4, '教务处审批', '[]', '', '考核后5个工作日', '教务处', '教务处审批转专业申请'),
('process-major-transfer-2026', 5, '公示', '[]', '', '审批后3个工作日', '教务处网站', '公示转专业名单'),
('process-major-transfer-2026', 6, '办理学籍异动', '["转专业申请表","学生证"]', 'http://jwc.example.edu.cn/system', '公示结束后', '教务处学籍科', '携带材料到教务处办理学籍异动手续');
