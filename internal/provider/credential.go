package provider

import "os"

// ResolveCredential 按 zeroclaw 顺序解析凭证：显式 key → 厂商 env → PATHFINDER_API_KEY / API_KEY。
func ResolveCredential(name string, apiKey *string) string {
	if apiKey != nil && *apiKey != "" {
		return *apiKey
	}
	for _, envVar := range envVarsForProvider(name) {
		if v := os.Getenv(envVar); v != "" {
			return v
		}
	}
	for _, envVar := range []string{"PATHFINDER_API_KEY", "ZEROCLAW_API_KEY", "API_KEY"} {
		if v := os.Getenv(envVar); v != "" {
			return v
		}
	}
	return ""
}

func envVarsForProvider(name string) []string {
	switch name {
	case "deepseek":
		return []string{"DEEPSEEK_API_KEY"}
	case "openai":
		return []string{"OPENAI_API_KEY"}
	case "anthropic":
		return []string{"ANTHROPIC_OAUTH_TOKEN", "ANTHROPIC_API_KEY"}
	case "openrouter":
		return []string{"OPENROUTER_API_KEY"}
	case "ollama":
		return []string{"OLLAMA_API_KEY"}
	case "groq":
		return []string{"GROQ_API_KEY"}
	case "mistral":
		return []string{"MISTRAL_API_KEY"}
	case "xai", "grok":
		return []string{"XAI_API_KEY"}
	default:
		return nil
	}
}
