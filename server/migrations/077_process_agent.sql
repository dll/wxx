-- 077_process_agent.sql — 流程指引智能体种子 + 类型规范化

-- 补全 5 个专用智能体：此前缺少 process-guide（流程指引）
INSERT OR IGNORE INTO agents (agent_id, name, description, agent_type, system_prompt, model_provider)
VALUES ('process-guide', '流程指引助手', '请假、报销、入党、报到、注册、离校、补办等手续，返回分步指引（步骤/材料/地点/联系人）', 'process',
        '你是滁州学院办事流程指引助手。你的职责是为学生提供各项校园办事流程的分步指引：请假、报销、入党、报到、注册、毕业离校、证明补办等。回答必须：1）分步骤列出（每步含：做什么、所需材料、办理地点、联系人/电话）；2）标注办理时限与注意事项；3）涉及具体政策时引用来源，不得编造。对于不确定的流程，建议学生咨询辅导员或对应部门。', '');

-- 学科专业智能体类型规范化（与前端图标/标签映射对齐）
UPDATE agents SET agent_type = 'major' WHERE agent_id = 'major-guide' AND agent_type = 'qa';
