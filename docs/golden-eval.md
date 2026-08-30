# 黄金评测集（A5）

> 检索质量回归评测：`server/eval/`。CI 通过标准 = 全部用例通过。

## 组成

| 文件 | 职责 |
|------|------|
| `golden_cases.json` | 42 条黄金用例（目标扩至 100+），覆盖 11 类意图 + 口语变体 + 指代多轮 + 编造守卫 + 过期排除 + 权限拦截 |
| `eval_test.go` | 评测 harness：内存 SQLite + 生产迁移（跳过内容种子，受控语料）→ 种子资源 → 走生产 `Ask` 全链路 → 断言 |

## 用例分类

- **hit-\***：单跳命中，期望资源进 sources Top-K（含口语装饰词/同义/方言词变体）
- **multi-\***：多轮指代（"它需要哪些材料"），依赖 CE-A2 查询改写 + 会话历史
- **guard-\***：编造守卫——无关问题/寒暄必须兜底（Fallback=true）
- **perm-\***：权限拦截——student 不可见 counselor 资料，必须兜底而非编造
- **mix-\***：复合问题双命中

## 全局断言（每例）

1. 过期资源（`kb-expired-transfer-notice`）永不进入 sources
2. `expect_fallback` 用例必须 `card.Fallback == true`
3. 命中用例期望资源必须在 Top-K 且 `Fallback == false`

## 评测曾发现的真实缺陷（已修复）

1. 生产引擎路径缺失 CE-02 低置信兜底（垃圾问题匹配弱资源不兜底）→ 引擎内新增 `filterByRelevance`
2. `retrieveWithContextEngine` 从未传 `SessionID` → CE-10 历史/指代改写从未生效 → 已接线
3. 结构化 LIKE 弱命中（≥3 条）会抑制 FTS 高质量召回 → 改为始终执行 FTS 合并
4. FTS MATCH 构建未过滤标点（"3.2" 的 `.` 导致整轮 FTS 语法错误）→ `isFTSTokenRune` 白名单
5. 历史末条 user 消息即当前问题 → 指代补全永远取到自身 → 跳过与当前问题相同的消息

## 扩展方式

1. 在 `golden_cases.json` 追加用例（含 `note` 说明意图）
2. 如需新知识域，在 `eval_test.go` 的 `seedResources` 增加对应资源
3. 运行 `go test ./eval/ -v`；单例调试 `-run "TestGoldenRetrieval/<case-id>"`
4. 调参观测：`WXX_DEBUG_SCORES=1` 打印全部候选得分
