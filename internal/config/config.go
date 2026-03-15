// Package config 提供与 zeroclaw 对齐的配置解析与路径解析。
// 配置目录解析：PATHFINDER_WORKSPACE（若设）按工作区规则解析，否则默认 ~/.pathfinder。
// 隐私变量仅通过 .env 配置：先加载 config_dir/.env，再加载 workspace_dir/.env；config.toml 不存 api_key/api_url。

package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pelletier/go-toml/v2"
)

const (
	configFileName  = "config.toml"
	envFileName     = ".env"
	workspaceSubdir = "workspace"
	pathfinderDir   = ".pathfinder"
)

// Config 顶层配置，仅保留工作流所需项；隐私变量用 .env，不读 config.toml 中的 api_key/api_url。
type Config struct {
	WorkspaceDir         string  `toml:"-"` // 运行时解析，不序列化
	ConfigPath          string  `toml:"-"` // 运行时解析，不序列化
	APIKey              *string `toml:"-"` // 仅从环境变量（含 .env）注入，不读 TOML
	APIURL              *string `toml:"-"` // 仅从环境变量注入，不读 TOML
	DefaultProvider     string  `toml:"default_provider,omitempty"`
	DefaultModel        string  `toml:"default_model,omitempty"`
	DefaultTemperature  float64 `toml:"default_temperature,omitempty"`
	ProviderTimeoutSecs uint64  `toml:"provider_timeout_secs,omitempty"`
}

// Default 返回默认配置（与 zeroclaw 默认值一致）。
func Default() Config {
	return Config{
		DefaultProvider:     "openrouter",
		DefaultModel:        "anthropic/claude-sonnet-4-6",
		DefaultTemperature:  0.7,
		ProviderTimeoutSecs: 120,
	}
}

// Load 解析配置路径、读取 TOML、应用环境变量覆盖，返回 *Config。
// 配置目录：PATHFINDER_WORKSPACE 指向含 config.toml 的目录时用该目录，否则默认 ~/.pathfinder。
func Load() (*Config, error) {
	configDir, workspaceDir, err := resolveRuntimeDirs()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(configDir, configFileName)

	c := Default()
	c.ConfigPath = configPath
	c.WorkspaceDir = workspaceDir

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(workspaceDir, 0700); err != nil {
		return nil, err
	}

	loadEnvFiles(configDir, workspaceDir)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("Config loaded", "path", configPath, "workspace", workspaceDir, "source", "default")
			applyEnvOverrides(&c)
			applyEnvSecrets(&c)
			return &c, nil
		}
		return nil, err
	}

	var decoded struct {
		DefaultProvider     string   `toml:"default_provider,omitempty"`
		DefaultModel       string   `toml:"default_model,omitempty"`
		DefaultTemperature *float64 `toml:"default_temperature,omitempty"`
		ProviderTimeoutSecs uint64 `toml:"provider_timeout_secs,omitempty"`
	}
	if err := toml.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	if decoded.DefaultProvider != "" {
		c.DefaultProvider = decoded.DefaultProvider
	}
	if decoded.DefaultModel != "" {
		c.DefaultModel = decoded.DefaultModel
	}
	if decoded.DefaultTemperature != nil {
		c.DefaultTemperature = *decoded.DefaultTemperature
	}
	if decoded.ProviderTimeoutSecs > 0 {
		c.ProviderTimeoutSecs = decoded.ProviderTimeoutSecs
	}

	slog.Info("Config loaded", "path", configPath, "workspace", workspaceDir, "source", "file")
	applyEnvOverrides(&c)
	applyEnvSecrets(&c)
	return &c, nil
}

// loadEnvFiles 加载 .env：先 config_dir 再 workspace_dir；已存在的环境变量不覆盖（便于用 .env 存隐私变量）。
func loadEnvFiles(configDir, workspaceDir string) {
	for _, dir := range []string{configDir, workspaceDir} {
		path := filepath.Join(dir, envFileName)
		if err := godotenv.Load(path); err != nil {
			if !os.IsNotExist(err) {
				slog.Debug("load .env", "path", path, "err", err)
			}
			continue
		}
		slog.Debug("loaded .env", "path", path)
	}
}

// applyEnvOverrides 应用环境变量覆盖 default_provider。
func applyEnvOverrides(c *Config) {
	if v := os.Getenv("PATHFINDER_PROVIDER"); v != "" {
		c.DefaultProvider = v
		return
	}
	if v := os.Getenv("ZEROCLAW_PROVIDER"); v != "" {
		c.DefaultProvider = v
		return
	}
	if c.DefaultProvider == "" || c.DefaultProvider == "openrouter" {
		if v := os.Getenv("PROVIDER"); v != "" {
			c.DefaultProvider = v
		}
	}
}

// applyEnvSecrets 从环境变量（含 .env）注入 APIKey/APIURL，不读 config.toml。
func applyEnvSecrets(c *Config) {
	if c.APIKey == nil {
		for _, key := range []string{"PATHFINDER_API_KEY", "ZEROCLAW_API_KEY", "API_KEY"} {
			if v := os.Getenv(key); v != "" {
				c.APIKey = &v
				break
			}
		}
	}
	if c.APIURL == nil {
		if v := os.Getenv("PATHFINDER_API_URL"); v != "" {
			c.APIURL = &v
		}
	}
}

func defaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, pathfinderDir), nil
}

// resolveConfigDirForWorkspace 给定工作区目录，解析配置目录与工作区目录（与 zeroclaw resolve_config_dir_for_workspace 对齐）。
func resolveConfigDirForWorkspace(workspaceDir string) (configDir, resolvedWorkspace string) {
	if info, err := os.Stat(filepath.Join(workspaceDir, configFileName)); err == nil && !info.IsDir() {
		return workspaceDir, filepath.Join(workspaceDir, workspaceSubdir)
	}
	parent := filepath.Dir(workspaceDir)
	legacy := filepath.Join(parent, pathfinderDir)
	if info, err := os.Stat(filepath.Join(legacy, configFileName)); err == nil && !info.IsDir() {
		return legacy, workspaceDir
	}
	if filepath.Base(workspaceDir) == workspaceSubdir {
		return legacy, workspaceDir
	}
	return workspaceDir, filepath.Join(workspaceDir, workspaceSubdir)
}

func expandTilde(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	if len(path) == 1 || path[1] == '/' || path[1] == filepath.Separator {
		home, _ := os.UserHomeDir()
		if home == "" {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// resolveRuntimeDirs 解析运行时配置目录与工作区目录。
// 默认 ~/.pathfinder；若设 PATHFINDER_WORKSPACE 则按工作区规则解析（含 config.toml 的目录为 configDir）。
func resolveRuntimeDirs() (configDir, workspaceDir string, err error) {
	if v := os.Getenv("PATHFINDER_WORKSPACE"); strings.TrimSpace(v) != "" {
		ws := expandTilde(strings.TrimSpace(v))
		cfg, wrk := resolveConfigDirForWorkspace(ws)
		return cfg, wrk, nil
	}
	defaultDir, err := defaultConfigDir()
	if err != nil {
		return "", "", err
	}
	return defaultDir, filepath.Join(defaultDir, workspaceSubdir), nil
}
