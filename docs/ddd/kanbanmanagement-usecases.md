# KanbanManagementContext 用例说明

## 起步建议（按 ddd-starter）

- **起步上下文**：`KanbanManagementContext`
- **起步用例**：`ManageKanbanCardLifecycleUseCase`
- **一句话理由**：该用例把“创建任务卡片 -> 指派 -> 流转 -> 评审 -> 闭环”串成一条可观测链路，是看板管理最小闭环。

## ManageKanbanCardLifecycleUseCase — 主成功场景（Happy Path）

1. 调度系统或人工在看板中提交一个新任务，应用层创建 `Card` 并放入 `todo` 列，同时记录创建事件。
2. 系统根据任务类型、角色绑定和当前负载为卡片分配执行责任人（assignee），并写入指派事件。**数据源**：AssigneeResolver 从 OpenClaw 同步结果（CanonicalStore 的 agents/bindings）解析；binding 的 MatchType 与 taskType 匹配时优先选该 agent，否则取拓扑中第一个 agent；无拓扑时回退默认 id。
3. 执行责任人开始处理任务，系统将卡片从 `todo` 迁移到 `in_progress`，并更新 `lastActivityAt`。
4. 处理过程中若遇到外部依赖，执行责任人将卡片标记为 `blocked` 并填写阻塞原因，系统记录阻塞事件。
5. 外部依赖解除后，执行责任人解除阻塞，系统将卡片恢复到 `in_progress` 并记录解除阻塞事件。
6. 执行完成后，责任人提交结果，系统将卡片迁移到 `in_review`，等待审核。
7. 审核人通过后，系统将卡片迁移到 `done`，并记录完成事件，形成可复盘的生命周期轨迹。
8. 指标读模型订阅上述领域事件，更新看板统计（在制数、阻塞数、平均闭环时长）并展示在管理看板中。

## 失败与边界场景（本轮只占位）

- TODO: WIP 超限时的迁移拒绝规则与反馈文案。
- TODO: 非法状态迁移（如 `todo -> done`）的错误码与处理方式。
- TODO: 卡片长期 `blocked` 的自动升级触发阈值（SLA）与责任链策略。

**融合说明**：派单数据源与升级策略将对接「同步结果」与 opengoat 式上卷，具体规则在对应用例中再写。
