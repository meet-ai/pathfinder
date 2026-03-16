package clients

import (
	"context"

	"pathfinder/internal/capabilitycatalog"
)

// ProductMetricsStub 是占位实现：始终返回空指标，仅携带 ProductID，便于在早期打通链路。
type ProductMetricsStub struct{}

func NewProductMetricsStub() *ProductMetricsStub {
	return &ProductMetricsStub{}
}

func (s *ProductMetricsStub) CollectForAgent(ctx context.Context, productID string, agentID string) (capabilitycatalog.ProductSpecificMetricsDTO, error) {
	return capabilitycatalog.ProductSpecificMetricsDTO{
		ProductID: productID,
		Metrics:   nil,
	}, nil
}

