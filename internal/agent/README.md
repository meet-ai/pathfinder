# agent

执行体发现、派发、循环；与 zeroclaw agent 对齐。

## 职责

- Agent 能力目录类型（Agent、AgentId、SkillId、ToolId）
- 派发端口 Dispatcher、发现端口 AgentDiscovery（由 infra 实现）
- Loop 按 Plan 派发一轮（可选）

## 对外提供 (provides)

- `Agent`、`AgentId`、`SkillId`、`ToolId`
- `Loop` 结构（见 loop.go）
- provides.go 注释说明

## 外部依赖 (needs)

- `AgentDiscovery`：ListAgents、GetAgent
- `AgentPoolFilter`
- `Dispatcher`：Dispatch(RunId, Task, AgentId)
- needs.go 定义端口

## 文件说明

| 文件 | 说明 |
|------|------|
| agent.go | Agent、AgentId、SkillId、ToolId |
| dispatcher.go | 派发说明（接口在 needs.go） |
| loop.go | Loop、Run |
| provides.go | 对外提供说明 |
| needs.go | AgentDiscovery、Dispatcher 端口 |
