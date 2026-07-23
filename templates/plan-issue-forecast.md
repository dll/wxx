# 任务方案 — 问题预案功能

## 背景与目标

**背景**：学院管理员需要全量分析汇总所有学生、教师、员工等可能存在的教育教学问题，提前发现问题趋势，及时干预。

**目标**：
1. 实现问题预案分析功能，支持 `sys_admin`、`college_admin` 角色访问
2. 全量分析情感预警、投诉建议、学业问题等多维度数据
3. 使用 AI 生成问题原因分析和解决预案
4. 实现问题趋势预测和风险等级评估
5. 支持简报导出和推送通知

## 范围（做 / 不做）

### 做
- [ ] 数据聚合层：汇总所有本地和外部系统数据
- [ ] AI 分析层：调用大模型生成问题原因分析和解决预案
- [ ] 趋势预测：基于历史数据分析问题趋势
- [ ] 风险评估：对问题进行风险等级分类（低/中/高/紧急）
- [ ] 问题分类：学业预警、情感问题、出勤异常、投诉建议、办事困难、纪律问题
- [ ] 简报导出：支持 PDF/Markdown 格式导出
- [ ] API 接口：`/api/v1/forecast/analysis`、`/api/v1/forecast/report`
- [ ] 前端页面：问题预案分析页面（仅管理端可见）

### 不做（后续迭代）
- [ ] 实时推送（WebSocket）- P2 期实现
- [ ] 邮件/短信通知 - 需对接外部服务
- [ ] 问题自动干预 - 需人工审核
- [ ] 跨学院数据聚合 - 学校管理员功能
- [ ] 历史对比分析 - 需要更长时间数据积累

## 技术要点（栈、接口、风险）

### 技术栈
| 层 | 技术 | 说明 |
|---|---|---|
| 数据聚合 | SQLite + FTS5 | 汇总多表数据 |
| AI 分析 | 智谱 GLM-4 | 生成原因分析和预案 |
| 接口层 | Go/Gin | REST API |
| 前端 | Flutter | 管理端页面 |

### 接口设计

#### 1. 问题分析接口
```
POST /api/v1/forecast/analysis
请求：
{
  "college_id": "计算机学院",      // 可选，学院筛选
  "time_range": "last_30_days", // 时间范围
  "analysis_type": "comprehensive",  // comprehensive|emotion|academic|attendance|complaint|discipline
  "data_sources": ["emotion", "feedback", "grades", "attendance", "leave", "discipline"]  // 可选，指定数据源
}
响应：
{
  "code": 0,
  "data": {
    "summary": {
      "total_issues": 156,
      "risk_distribution": {"low": 80, "medium": 45, "high": 25, "urgent": 6},
      "trend": "increasing",
      "key_findings": ["...", "..."]
    },
    "issues": [
      {
        "issue_id": "issue-001",
        "category": "学业预警",
        "subcategory": "挂科率偏高",
        "title": "某班级高数挂科率偏高",
        "risk_level": "high",
        "affected_count": 12,
        "affected_students": ["学号1", "学号2", "..."],
        "root_cause": "教学进度过快...",
        "suggested_actions": ["...", "..."],
        "sources": ["情感预警数据", "学业成绩数据"],
        "data_details": {
          "emotion_alerts": 5,
          "grade_failures": 12,
          "attendance_issues": 3
        },
        "created_at": "2026-06-18T10:00:00Z"
      }
    ],
    "report_url": "/api/v1/forecast/report/xxx"
  }
}
```

#### 2. 预案报告接口
```
GET /api/v1/forecast/report/:id
响应：Markdown 格式的详细报告
```

#### 3. 预案列表接口
```
GET /api/v1/forecast/issues
参数：?status=pending&risk_level=high&page=1&page_size=20
```

### RBAC 权限
| 角色 | 权限 | 说明 |
|------|------|------|
| `sys_admin` | 全部 | 查看所有学院数据 |
| `school_admin` | 只读 | 查看所有学院数据 |
| `college_admin` | 本学院 | 只查看本学院数据 |
| `counselor` | 无 | 不可见 |
| 其他 | 无 | 不可见 |

### 数据来源（全量）

#### 一、本地数据库
| 数据表 | 用途 | 权重 | 数据说明 |
|--------|------|------|----------|
| `emotion_alerts` | 情感预警 | 高 | 学生情感风险分析结果 |
| `feedback_screenshots` | 投诉建议 | 中 | 学生/教师反馈截图 |
| `process_records` | 办事流程 | 低 | 办事流程记录 |
| `sessions` + `messages` | 对话记录 | 中 | 用户提问历史（可挖掘潜在问题） |
| `audit_logs` | 操作日志 | 低 | 系统使用情况 |
| `token_usage` | 用量统计 | 低 | 各功能使用频率 |

#### 二、外部系统对接（学工系统）
| 接口路径 | 用途 | 权重 | 数据说明 |
|----------|------|------|----------|
| `/integration/xuegong/leave` | 请假记录 | 中 | 请假频率、请假类型分布 |
| `/integration/xuegong/attendance` | 出勤数据 | 高 | 缺勤率、迟到早退统计 |
| `/integration/xuegong/grades` | 学业成绩 | 高 | 挂科率、成绩分布 |
| `/integration/xuegong/scholarship` | 奖助学金 | 中 | 奖助覆盖率、贫困生分布 |
| `/integration/xuegong/discipline` | 纪律处分 | 高 | 违纪类型、频率 |

#### 三、外部系统对接（一表通）
| 接口路径 | 用途 | 权重 | 数据说明 |
|----------|------|------|----------|
| `/integration/ybt/graduation` | 毕业流程 | 低 | 离校手续完成情况 |
| `/integration/ybt/accommodation` | 住宿信息 | 中 | 宿舍分配、住宿问题 |

#### 四、知识库（参考数据）
| 数据表 | 用途 | 权重 | 数据说明 |
|--------|------|------|----------|
| `kb_resources` | 政策/FAQ | 参考 | 已有政策和常见问题 |
| `process_steps` | 办事流程 | 参考 | 流程步骤和所需材料 |

#### 五、用户数据
| 数据表 | 用途 | 权重 | 数据说明 |
|--------|------|------|----------|
| `users` | 用户信息 | 参考 | 角色、学院、班级分布 |

### 风险点
1. **数据量大**：全量分析可能超时 → 使用异步任务 + 缓存
2. **AI 调用成本**：大模型调用费用 → 缓存分析结果，定时刷新
3. **隐私问题**：涉及学生数据 → 脱敏处理，RBAC 严格控制

## 步骤拆分

### 阶段一：数据层（2-3天）
1. 创建问题预案数据表 `issue_forecasts`
2. 创建问题详情表 `issue_details`
3. 实现本地数据聚合 repository（情感预警、反馈、对话记录）
4. 实现外部数据对接 service（学工系统、一表通）
5. 实现统一数据聚合 service

### 阶段二：AI 分析层（2-3天）
1. 实现 AI 分析 prompt 模板（按问题分类）
2. 实现大模型调用 service
3. 实现风险评估逻辑
4. 实现趋势预测算法

### 阶段三：接口层（1-2天）
1. 实现分析接口 handler
2. 实现报告接口 handler
3. 实现列表接口 handler
4. 实现详情接口 handler
5. 更新 RBAC 矩阵
6. 更新 API 契约文档

### 阶段四：前端层（3-4天）
1. 实现问题预案分析页面
2. 实现问题列表页面
3. 实现报告查看页面
4. 实现数据可视化（图表）
5. 添加管理端菜单入口

### 阶段五：测试与文档（1-2天）
1. 单元测试
2. 接口测试
3. 更新 API 契约文档
4. 更新 RBAC 矩阵文档

## 验收标准

- [ ] `sys_admin`、`college_admin` 可访问问题预案功能
- [ ] 能正确汇总所有本地数据（情感预警、反馈、对话记录）
- [ ] 能正确对接外部系统（学工系统、一表通）
- [ ] AI 分析能生成合理的原因分析和解决预案
- [ ] 问题分类准确（学业预警、情感问题、出勤异常、投诉建议、办事困难、纪律问题）
- [ ] 风险等级分类准确（低/中/高/紧急）
- [ ] 简报可导出为 PDF/Markdown
- [ ] 接口响应时间 < 5s（缓存后）
- [ ] 代码符合分层规则（handler → service → repository）
- [ ] 更新 API 契约和 RBAC 矩阵文档

## 回滚与检查点（Git / 数据）

### 检查点
1. 数据表创建完成
2. 数据聚合层完成
3. AI 分析层完成
4. 接口层完成
5. 前端页面完成

### 回滚方案
1. 数据库：删除 `issue_forecasts` 表（不影响其他功能）
2. 代码：Git revert 对应提交
3. 前端：隐藏菜单入口

### Git 提交规范
```
feat(forecast): 添加问题预案数据表和聚合逻辑
feat(forecast): 添加 AI 分析服务
feat(forecast): 添加问题预案接口
feat(forecast): 添加管理端页面
docs: 更新 API 契约和 RBAC 矩阵
```
