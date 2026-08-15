# Agent Review 索引 — 新生视角审核报告

> 由 **wxx-agent** 维护。本目录存放以**大一新生视角**对蔚小芯（WXX）智能体的审核、改进与补齐记录。

## 报告清单

| 日期 | 文件 | 主题 | 状态 |
|------|------|------|------|
| 2026-08-14 | [review-2026-08-14.md](review-2026-08-14.md) | 新生视角初审：痛点分级 + 补齐清单 + 优化建议 | 已产出；【高】3 项 + 【中4】已实现（见“实现记录”） |
| 2026-08-14 | [multi-role-stickiness-2026-08-14.md](multi-role-stickiness-2026-08-14.md) | 分角色粘性增强方案：辅导员/学生会/教辅/教师/管理员 | 方案已出；辅导员谈心记录统计(交流次数/内容/效果)已实现 |
| 2026-08-15 | [school-data-integration-2026-08-15.md](school-data-integration-2026-08-15.md) | 学校系统课表数据接入方案（阶段1/2设计与合规边界） | 阶段0课表提醒已实现；路径C限制等待办 |
| 2026-08-15 | [role-management-2026-08-15.md](role-management-2026-08-15.md) | 角色管理增强：角色分配入口 + 职务字段 + 越权防护 | 已实现，后端/前端验证通过 |
| 2026-08-15 | [counselor-assistant-role-audit-2026-08-15.md](counselor-assistant-role-audit-2026-08-15.md) | 辅导员/教辅角色功能审核：入口+数据真实性 | 审核完成；教辅数据来源标注已落地 |
| 2026-08-15 | [staff-performance-twin-2026-08-15.md](staff-performance-twin-2026-08-15.md) | 教辅/教师绩效画像：绩效→数字孪生画像→三方绑定（方案A第一增量） | 已实现，后端/前端编译验证通过 |
| 2026-08-15 | [assistant-frontend-real-data-2026-08-15.md](assistant-frontend-real-data-2026-08-15.md) | 强关联教辅前端(教学日历/学生信息/通知) + 弱关联藏起 + 辅导员不瞎编整改 | 已实现并部署，验证通过 |
| 2026-08-15 | [cloud-deploy-verified-2026-08-15.md](cloud-deploy-verified-2026-08-15.md) | 云端部署验证：含全功能后端 + 首/批未部署改动上线 | 已部署，health/登录/绩效画像实测通过 |
| 2026-08-15 | [facility-workbench-2026-08-15.md](facility-workbench-2026-08-15.md) | 后勤服务台落地：实验/保洁/热水/查岗/环卫/借阅真实登记，并入教辅角色 + 绩效画像维度 | 已实现，后端编译+前端 analyze+本地运行时联调通过 |
| 2026-08-15 | [facility-workbench-plan-2026-08-15.md](facility-workbench-plan-2026-08-15.md) | 后勤服务台方案（合并方案定稿） | 方案已审 |
| 2026-08-15 | [triple-linkage-audit-2026-08-15.md](triple-linkage-audit-2026-08-15.md) | 教师/教辅×学生×蔚小芯三者关联强度审核 + 教师三方绑定真实化 | 三类联调通过，教师造假数据移除 |
| 2026-08-15 | [secretary-education-performance-2026-08-15.md](secretary-education-performance-2026-08-15.md) | 书记视角×蔚小芯育人绩效报告能力分析 | 已梳理差距，未实施 |
| 2026-08-15 | [secretary-party-closed-loop-2026-08-15.md](secretary-party-closed-loop-2026-08-15.md) | 书记×蔚小芯党建育人闭环蓝图（数据模型已对上，缺口+设计） | 方案已审，待确认口径 |

## 说明

- 审核视角：大一新生（"希望用 / 喜欢用"的用户体验），侧重学生教育侧功能。
- 审核方法见项目根 `WXX-AGENT.md`。
- 每次新增报告请在此登记，并在对应报告中记录"实现内容"。
