-- 056_feature_switches.sql — 全局功能开关默认种子
-- 管理员可在 /admin/settings 修改 feature.* 键值（true/false）控制模块上架；
-- 登录用户经 /public/feature-switches 读取，前端据此显示/隐藏模块。
INSERT OR IGNORE INTO system_settings (key, value, description) VALUES
    ('feature.party', 'true', '入党教育/入党进度'),
    ('feature.competition', 'true', '学科竞赛'),
    ('feature.plan', 'true', '大学规划'),
    ('feature.graduation', 'true', '毕设选题/毕业离校'),
    ('feature.career', 'true', '就业指导'),
    ('feature.club', 'true', '社团生活'),
    ('feature.culture', 'true', '校园文化'),
    ('feature.project', 'true', '实践项目'),
    ('feature.research', 'true', '科研论文'),
    ('feature.innovation', 'true', '创新创业'),
    ('feature.digital_twin', 'true', '数字孪生'),
    ('feature.emotion', 'true', '情感预警'),
    ('feature.voice', 'true', '语音服务');
