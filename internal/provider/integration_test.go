//go:build integration

package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"pathfinder/internal/config"
)

// 集成测试：从配置读取 default_provider=deepseek，创建 provider 并调用 SimpleChat。
// 需设置 DEEPSEEK_API_KEY：环境变量、项目根 .env 或 ~/.pathfinder/.env；执行：go test -tags=integration ./internal/provider/...
func TestCreateFromConfig_DeepSeek_SimpleChat(t *testing.T) {
	// 先加载 .env，否则 config.Load() 只读临时目录，读不到本机 key（godotenv 不覆盖已存在的 env）
	if wd, err := os.Getwd(); err == nil {
		_ = godotenv.Load(filepath.Join(wd, ".env"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		_ = godotenv.Load(filepath.Join(home, ".pathfinder", ".env"))
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
default_provider = "deepseek"
default_model = "deepseek-chat"
default_temperature = 0.7
provider_timeout_secs = 60
`), 0600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("PATHFINDER_WORKSPACE", dir)
	defer os.Unsetenv("PATHFINDER_WORKSPACE")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.DefaultProvider != "deepseek" {
		t.Fatalf("DefaultProvider = %q, want deepseek", cfg.DefaultProvider)
	}

	ctx := context.Background()
	p, err := CreateFromConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("CreateFromConfig: %v", err)
	}
	if p == nil {
		t.Fatal("provider is nil")
	}

	key := ResolveCredential("deepseek", cfg.APIKey)
	if key == "" {
		t.Skip("DEEPSEEK_API_KEY 未设置，跳过真实请求；测试内容 配置接入+创建 provider 成功")
		return
	}

	text, err := p.SimpleChat(ctx, "只回复：OK", cfg.DefaultModel, cfg.DefaultTemperature)
	if err != nil {
		t.Fatalf("SimpleChat: %v", err)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		t.Error("response text empty")
	}
	t.Logf("测试内容 配置读取 deepseek 并 SimpleChat 成功，回复: %s", text)
}
