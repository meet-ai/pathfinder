package clients

import (
	"context"
	"fmt"

	"pathfinder/internal/agent"
	"pathfinder/internal/planning"
	"pathfinder/internal/provider"
	"pathfinder/internal/runtime"
)

// DispatcherLLM 单步 LLM 派发器：用 Provider.SimpleChat 执行子任务描述，返回模型输出作为结果。
type DispatcherLLM struct {
	Provider    provider.Provider
	Model        string
	Temperature  float64
}

// Dispatch 将任务描述发给 LLM，返回回复文本；失败返回错误。
func (d *DispatcherLLM) Dispatch(ctx context.Context, runId runtime.JobId, task planning.SubTask, agentId agent.AgentId) (string, error) {
	if d.Provider == nil {
		return "", fmt.Errorf("dispatcher_llm: provider is nil")
	}
	text, err := d.Provider.SimpleChat(ctx, task.Description, d.Model, d.Temperature)
	if err != nil {
		return "", err
	}
	return text, nil
}
