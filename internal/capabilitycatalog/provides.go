package capabilitycatalog

import "context"

type ListAgentsQuery struct {
	ProductID string
	Keyword   string
	Tags      []string
	Groups    []string
}

type AgentSummaryDTO struct {
	ID          string
	Name        string
	Description string
	Version     string
	Tags        []string
	Groups      []string
}

type DescribeAgentQuery struct {
	AgentID   string
	ProductID string
}

type AgentCapabilityDTO struct {
	Name          string
	Description   string
	InputSummary  string
	OutputSummary string
}

type MetricEntryDTO struct {
	Key         string
	Value       string
	Description string
}

type ProductSpecificMetricsDTO struct {
	ProductID string
	Metrics   []MetricEntryDTO
}

type AgentDetailDTO struct {
	ID                 string
	Name               string
	Description        string
	Version            string
	Capabilities       []AgentCapabilityDTO
	DangerousOperations []string
	SupportedContexts  []string
	Requirements       []string
	ProductMetrics     *ProductSpecificMetricsDTO
}

type CapabilityCatalogQueryService interface {
	ListAgents(ctx context.Context, query ListAgentsQuery) ([]AgentSummaryDTO, error)
	DescribeAgent(ctx context.Context, query DescribeAgentQuery) (AgentDetailDTO, error)
}

type DefaultCapabilityCatalogQueryService struct {
	registry AgentRegistry
	metrics ProductMetricsProvider
}

func NewDefaultCapabilityCatalogQueryService(reg AgentRegistry) *DefaultCapabilityCatalogQueryService {
	return &DefaultCapabilityCatalogQueryService{registry: reg}
}

func NewDefaultCapabilityCatalogQueryServiceWithMetrics(reg AgentRegistry, metrics ProductMetricsProvider) *DefaultCapabilityCatalogQueryService {
	return &DefaultCapabilityCatalogQueryService{
		registry: reg,
		metrics: metrics,
	}
}

func (s *DefaultCapabilityCatalogQueryService) ListAgents(ctx context.Context, query ListAgentsQuery) ([]AgentSummaryDTO, error) {
	records, err := s.registry.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]AgentSummaryDTO, 0, len(records))
	for _, r := range records {
		out = append(out, AgentSummaryDTO{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Version:     r.Version,
			Tags:        append([]string(nil), r.Tags...),
			Groups:      append([]string(nil), r.Groups...),
		})
	}

	// TODO: 按 query.Tags / query.Groups 做过滤与排序。
	return out, nil
}

func (s *DefaultCapabilityCatalogQueryService) DescribeAgent(ctx context.Context, query DescribeAgentQuery) (AgentDetailDTO, error) {
	var empty AgentDetailDTO

	if query.AgentID == "" {
		return empty, nil
	}

	record, err := s.registry.GetByID(ctx, query.AgentID)
	if err != nil {
		return empty, err
	}
	if record == nil {
		return empty, nil
	}

	dto := AgentDetailDTO{
		ID:                 record.ID,
		Name:               record.Name,
		Description:        record.Description,
		Version:            record.Version,
		Capabilities:       nil,
		DangerousOperations: append([]string(nil), record.DangerousOperations...),
		SupportedContexts:  append([]string(nil), record.SupportedContexts...),
		Requirements:       append([]string(nil), record.Requirements...),
	}

	for _, c := range record.Capabilities {
		dto.Capabilities = append(dto.Capabilities, AgentCapabilityDTO{
			Name:          c.Name,
			Description:   c.Description,
			InputSummary:  c.InputSummary,
			OutputSummary: c.OutputSummary,
		})
	}

	if s.metrics != nil && query.ProductID != "" {
		m, err := s.metrics.CollectForAgent(ctx, query.ProductID, query.AgentID)
		if err != nil {
			return dto, err
		}
		dto.ProductMetrics = &m
	}

	return dto, nil
}

