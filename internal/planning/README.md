# planning

计划结构、校验、state 约定。

## 职责

- 计划聚合（Plan、SubTask、Dependency、SuggestedAgent）
- 计划校验 Validate、SuggestedAgentFor
- 对外提供计划类型与标识；规划产出由 Planner 端口实现

## 对外提供 (provides)

- `PlanId`、`TaskId`、`GoalDescription`、`Dependency`、`SuggestedAgent`、`SubTask`、`Plan`
- `Plan.Validate()`、`Plan.SuggestedAgentFor()`
- provides.go 注释说明

## 外部依赖 (needs)

- `Planner`：根据目标产出 Plan，由 infra 实现
- `PlanRepository`：Save/Get Plan，由 infra 实现
- needs.go 定义端口

## 文件说明

| 文件 | 说明 |
|------|------|
| plan.go | 值类型、Plan、Validate、SuggestedAgentFor |
| errors.go | ErrPlanNoSubTasks、ErrPlanInvalidDependency |
| provides.go | 对外提供说明 |
| needs.go | Planner、PlanRepository 端口 |
