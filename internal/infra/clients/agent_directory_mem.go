package clients

import (
	"context"

	"pathfinder/internal/capabilitycatalog"
)

// AgentRegistryMem 是基于内存的 AgentRegistry 简单实现，供联调与测试使用。
type AgentRegistryMem struct {
	Agents []capabilitycatalog.AgentRecord
}

func NewAgentRegistryMem(agents []capabilitycatalog.AgentRecord) *AgentRegistryMem {
	return &AgentRegistryMem{Agents: agents}
}

func (d *AgentRegistryMem) ListAll(ctx context.Context) ([]capabilitycatalog.AgentRecord, error) {
	return append([]capabilitycatalog.AgentRecord(nil), d.Agents...), nil
}

func (d *AgentRegistryMem) GetByID(ctx context.Context, id string) (*capabilitycatalog.AgentRecord, error) {
	for i := range d.Agents {
		if d.Agents[i].ID == id {
			cp := d.Agents[i]
			return &cp, nil
		}
	}
	return nil, nil
}

