package agent

import (
	"context"
	"pathfinder/internal/planning"
	"pathfinder/internal/progress"
	"pathfinder/internal/runtime"
)

// AgentDiscovery 执行体发现端口：列出/获取 Agent，由 infra 实现。
type AgentDiscovery interface {
	ListAgents(ctx context.Context, filter AgentPoolFilter) ([]Agent, error)
	GetAgent(ctx context.Context, id AgentId) (*Agent, error)
}

// AgentPoolFilter 查询 Agent 时的过滤条件。
type AgentPoolFilter struct {
	AgentPoolId    string
	CapabilityTags []string
}

// Dispatcher 派发端口：将子任务派发到指定 Agent 执行，由 infra 实现（如 openclaw/acpx）。
type Dispatcher interface {
	Dispatch(ctx context.Context, runId runtime.JobId, task planning.SubTask, agentId AgentId) (result string, err error)
}

// TaskProgressRepository 任务进度仓储端口，供 Loop 恢复/写回进度；由 infra 实现。
type TaskProgressRepository interface {
	Save(ctx context.Context, runId runtime.JobId, t *progress.TaskProgress) error
	ListByRunId(ctx context.Context, runId runtime.JobId) ([]progress.TaskProgress, error)
}

// AbortCheck 每轮前检查是否应中止（取消或超时）；由编排层注入，如基于 JobRepository 查询。
type AbortCheck func(ctx context.Context, runId runtime.JobId) bool
