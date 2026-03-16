# orchestration

端到端工作流编排、总结。

## 职责

- SubmitGoal：创建 Run、规划、保存、执行计划、总结
- ExecutePlan：按依赖派发、更新进度、取消/超时中止
- SummarizeRun、CancelRun
- 依赖 planning、runtime、progress、agent 等端口

## 对外提供 (provides)

- `WorkflowOrchestrationApplicationService`：SubmitGoal、SubmitGoalViaChannel、ExecutePlan、SummarizeRun、CancelRun
- Command/DTO：SubmitGoalCommand、RunDTO、PlanDTO、SummaryDTO 等（见 commands.go、dtos.go）
- provides.go 注释说明

## 外部依赖 (needs)

- planning.Planner、planning.PlanRepository
- runtime.RunRepository
- progress.TaskProgressRepository
- agent.AgentDiscovery、agent.Dispatcher
- needs.go 说明

## 文件说明

| 文件 | 说明 |
|------|------|
| service.go | WorkflowOrchestrationApplicationService 及用例方法 |
| commands.go | SubmitGoalCommand、SummarizeRunCommand 等 |
| dtos.go | RunDTO、PlanDTO、SummaryDTO 等 |
| provides.go | 对外提供说明 |
| needs.go | 依赖各包端口说明 |
