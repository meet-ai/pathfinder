package provider

import (
	"context"
	"errors"
	"strings"
)

var ErrUnknownProvider = errors.New("unknown provider")

// CreateProvider 按 name 创建 provider，凭证与 URL 可选。
func CreateProvider(ctx context.Context, name string, apiKey *string, apiURL *string) (Provider, error) {
	return CreateProviderWithOptions(ctx, name, apiKey, apiURL, nil)
}

// CreateProviderWithOptions 按 name + 选项创建 provider。
func CreateProviderWithOptions(ctx context.Context, name string, apiKey *string, apiURL *string, opts *RuntimeOptions) (Provider, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return nil, ErrUnknownProvider
	}
	key := ResolveCredential(name, apiKey)
	baseURL := baseURLForProvider(name, apiURL, opts)
	if baseURL == "" {
		return nil, ErrUnknownProvider
	}
	// deepseek：OpenAI 兼容，Bearer
	if name == "deepseek" {
		p := NewOpenAICompatible("DeepSeek", baseURL, key, AuthBearer)
		if opts != nil && opts.ProviderTimeoutSecs > 0 {
			p.WithTimeout(opts.ProviderTimeoutSecs)
		}
		return p, nil
	}
	return nil, ErrUnknownProvider
}

func baseURLForProvider(name string, apiURL *string, opts *RuntimeOptions) string {
	if apiURL != nil && strings.TrimSpace(*apiURL) != "" {
		return strings.TrimSuffix(strings.TrimSpace(*apiURL), "/")
	}
	if opts != nil && opts.ProviderAPIURL != "" {
		return strings.TrimSuffix(strings.TrimSpace(opts.ProviderAPIURL), "/")
	}
	switch name {
	case "deepseek":
		return "https://api.deepseek.com"
	default:
		return ""
	}
}
