package clients

import (
	"context"
	"fmt"

	"pathfinder/internal/agent"
	"pathfinder/internal/planning"
	"pathfinder/internal/runtime"
)

// DispatcherStub 占位派发器：直接返回任务描述作为结果，供联调与测试。
type DispatcherStub struct{}

// Dispatch 返回占位结果。
func (d *DispatcherStub) Dispatch(ctx context.Context, runId runtime.JobId, task planning.SubTask, agentId agent.AgentId) (string, error) {
	return fmt.Sprintf("done: %s", task.Description), nil
}
