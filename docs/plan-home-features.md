# 任务方案 — 首页功能入口分组

## 背景与目标

5 个学生新功能（毕设选题/学科竞赛/大学规划/入党教育/社团生活）和 1 个管理功能（问题预案）代码已就绪，但首页无入口，用户需从"我的"页滚动到底才能找到。需在首页按角色分组添加快捷入口，遵循蔚小芯现有 UI 模式。

## 范围

### 做
- 首页新增「学生专区」入口卡片（5 个功能，仅 student/student_union 角色可见）
- 首页新增「管理专区」入口卡片（问题预案，仅 college_admin+ 角色可见）
- 复用首页 `_buildKnowledgeCard` 设计模式（图标+标签+颜色）
- 响应式：与知识卡片一致的四列网格

### 不做
- 不改底部导航栏
- 不改路由
- 不改后端
- 不改 profile_page 已有菜单

## 技术要点

- 栈：Flutter + Material Design 3（与首页一致）
- 风险：低，纯 UI 改动，无后端依赖
- 设计参考：`home_page.dart:267-310` 的 `_buildKnowledgeEntry` 四列卡片布局

## 步骤拆分

1. 首页 build 中 `_buildKnowledgeEntry` 下方添加角色条件判断，插入「学生专区」和「管理专区」
2. 新增 `_buildStudentFeatures` 方法，5 个卡片（毕设选题/学科竞赛/大学规划/入党教育/社团生活）
3. 新增 `_buildAdminFeatures` 方法，1 个卡片（问题预案）
4. 编译验证

## 验收标准

- student 角色：首页看到「学生专区」5 个入口卡片，点击跳转正确路由
- college_admin+ 角色：首页看到「管理专区」→ 问题预案入口
- 其他角色（counselor/teacher 等）：不显示这两个专区
- 布局与首页知识卡片一致

## 回滚与检查点

- 改动单文件：`frontend/lib/pages/home/home_page.dart`
- 提交前 `git diff` 确认仅新增 3 个方法 + build 中插入 2 个 `if` 块
