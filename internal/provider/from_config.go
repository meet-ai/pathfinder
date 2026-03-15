package provider

import (
	"context"

	"pathfinder/internal/config"
)

// CreateFromConfig 根据 config 创建 Provider（使用 DefaultProvider、APIKey、APIURL、ProviderTimeoutSecs）。
func CreateFromConfig(ctx context.Context, cfg *config.Config) (Provider, error) {
	if cfg == nil {
		return nil, ErrUnknownProvider
	}
	opts := &RuntimeOptions{
		ProviderTimeoutSecs: int(cfg.ProviderTimeoutSecs),
	}
	return CreateProviderWithOptions(ctx, cfg.DefaultProvider, cfg.APIKey, cfg.APIURL, opts)
}
