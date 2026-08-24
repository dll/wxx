-- vOPC v2.0 数据回填：存量 v1.0 S0–S9 阶段数据收敛为 G0–G4（audit 阻断项 A）。
-- 纯增量、幂等（仅 UPDATE，重复执行无副作用；由 _migrations 跟踪保证只跑一次）。
--
-- 映射依据 refactor-notes.md §3：
--   S0 → G0 想法(draft)
--   S1/S2/S3 → G1 目标与方案(pending_review)
--   S4 → G2 执行(developing)
--   S5/S7 → G3 验证与反馈(validating)
--   S6/S8/S9 → G4 复盘与闭环(closeable)
--
-- 注：不修改 097 的列默认值 'S0'→'G0'——SQLite 不支持 ALTER COLUMN SET DEFAULT，
-- 为保双方言兼容不动 schema；应用层所有 INSERT 均显式写入 'G0'，旧默认值不会再生效。

-- 项目主表阶段与周期状态同步映射
UPDATE vopc_projects SET stage='G0', status='draft' WHERE stage='S0';
UPDATE vopc_projects SET stage='G1', status='pending_review' WHERE stage IN ('S1','S2','S3');
UPDATE vopc_projects SET stage='G2', status='developing' WHERE stage='S4';
UPDATE vopc_projects SET stage='G3', status='validating' WHERE stage IN ('S5','S7');
UPDATE vopc_projects SET stage='G4', status='closeable' WHERE stage IN ('S6','S8','S9');

-- 里程碑表同步（v1.0 创建时播种的是 S 阶段里程碑行）
UPDATE vopc_milestones SET stage='G0' WHERE stage='S0';
UPDATE vopc_milestones SET stage='G1' WHERE stage IN ('S1','S2','S3');
UPDATE vopc_milestones SET stage='G2' WHERE stage='S4';
UPDATE vopc_milestones SET stage='G3' WHERE stage IN ('S5','S7');
UPDATE vopc_milestones SET stage='G4' WHERE stage IN ('S6','S8','S9');

-- 成果版次门禁字段同步（100 引入 intended_stage）
UPDATE vopc_artifact_versions SET intended_stage='G0' WHERE intended_stage='S0';
UPDATE vopc_artifact_versions SET intended_stage='G1' WHERE intended_stage IN ('S1','S2','S3');
UPDATE vopc_artifact_versions SET intended_stage='G2' WHERE intended_stage='S4';
UPDATE vopc_artifact_versions SET intended_stage='G3' WHERE intended_stage IN ('S5','S7');
UPDATE vopc_artifact_versions SET intended_stage='G4' WHERE intended_stage IN ('S6','S8','S9');
