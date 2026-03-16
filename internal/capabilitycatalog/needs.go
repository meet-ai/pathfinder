package capabilitycatalog

import "context"

// AgentRecord 是能力目录内部使用的原始 Agent 记录。
type AgentRecord struct {
	ID          string
	Name        string
	Description string
	Version     string
	Tags        []string
	Groups      []string

	Capabilities       []AgentCapabilityRecord
	DangerousOperations []string
	SupportedContexts  []string
	Requirements       []string
}

type AgentCapabilityRecord struct {
	Name          string
	Description   string
	InputSummary  string
	OutputSummary string
}

type AgentRegistry interface {
	ListAll(ctx context.Context) ([]AgentRecord, error)
	GetByID(ctx context.Context, id string) (*AgentRecord, error)
}

type ProductMetricsProvider interface {
	CollectForAgent(ctx context.Context, productID string, agentID string) (ProductSpecificMetricsDTO, error)
}

