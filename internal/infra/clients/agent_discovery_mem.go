package clients

import (
	"context"

	"pathfinder/internal/agent"
)

// AgentDiscoveryMem 内存实现的 Agent 发现，返回固定占位 Agent，供联调与测试。
type AgentDiscoveryMem struct {
	Agents []agent.Agent
}

// NewAgentDiscoveryMem 构造，默认带一个占位 Agent。
func NewAgentDiscoveryMem() *AgentDiscoveryMem {
	return &AgentDiscoveryMem{
		Agents: []agent.Agent{{
			Id:           "agent-1",
			Name:         "stub",
			Capabilities: []string{},
			Tags:         []string{},
		}},
	}
}

// ListAgents 返回内存中的 Agent 列表。
func (a *AgentDiscoveryMem) ListAgents(ctx context.Context, filter agent.AgentPoolFilter) ([]agent.Agent, error) {
	return a.Agents, nil
}

// GetAgent 按 Id 返回，无则 nil。
func (a *AgentDiscoveryMem) GetAgent(ctx context.Context, id agent.AgentId) (*agent.Agent, error) {
	for i := range a.Agents {
		if a.Agents[i].Id == id {
			cp := a.Agents[i]
			return &cp, nil
		}
	}
	return nil, nil
}
