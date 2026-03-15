package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.DefaultProvider != "openrouter" {
		t.Errorf("DefaultProvider = %q, want openrouter", c.DefaultProvider)
	}
	if c.DefaultModel == "" {
		t.Error("DefaultModel empty")
	}
	if c.DefaultTemperature <= 0 {
		t.Errorf("DefaultTemperature = %v, want > 0", c.DefaultTemperature)
	}
	t.Log("测试内容 Default 成功")
}

func TestLoad_noFile_usesDefault(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("PATHFINDER_CONFIG_DIR", dir)
	defer os.Unsetenv("PATHFINDER_CONFIG_DIR")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.ConfigPath != filepath.Join(dir, configFileName) {
		t.Errorf("ConfigPath = %q", c.ConfigPath)
	}
	if c.DefaultProvider != "openrouter" {
		t.Errorf("DefaultProvider = %q", c.DefaultProvider)
	}
	t.Log("测试内容 Load 无文件使用默认 成功")
}

func TestLoad_withFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, []byte(`
default_provider = "ollama"
default_model = "llama3.2"
default_temperature = 0.5
`), 0600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("PATHFINDER_CONFIG_DIR", dir)
	defer os.Unsetenv("PATHFINDER_CONFIG_DIR")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.DefaultProvider != "ollama" {
		t.Errorf("DefaultProvider = %q, want ollama", c.DefaultProvider)
	}
	if c.DefaultModel != "llama3.2" {
		t.Errorf("DefaultModel = %q, want llama3.2", c.DefaultModel)
	}
	if c.DefaultTemperature != 0.5 {
		t.Errorf("DefaultTemperature = %v, want 0.5", c.DefaultTemperature)
	}
	t.Log("测试内容 Load 从文件 成功")
}

func TestResolveConfigDirForWorkspace(t *testing.T) {
	dir := t.TempDir()
	configInWorkspace := filepath.Join(dir, configFileName)
	if err := os.WriteFile(configInWorkspace, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	cfgDir, wrkDir := resolveConfigDirForWorkspace(dir)
	if cfgDir != dir {
		t.Errorf("configDir = %q, want %q", cfgDir, dir)
	}
	if wrkDir != filepath.Join(dir, workspaceSubdir) {
		t.Errorf("workspaceDir = %q", wrkDir)
	}
	t.Log("测试内容 resolveConfigDirForWorkspace 成功")
}

func TestLoad_envVarsFromDotenv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("PATHFINDER_PROVIDER=ollama\nSECRET_KEY=from_env\n"), 0600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("PATHFINDER_CONFIG_DIR", dir)
	defer os.Unsetenv("PATHFINDER_CONFIG_DIR")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.DefaultProvider != "ollama" {
		t.Errorf("DefaultProvider = %q, want ollama (from .env)", c.DefaultProvider)
	}
	if v := os.Getenv("SECRET_KEY"); v != "from_env" {
		t.Errorf("SECRET_KEY = %q, want from_env", v)
	}
	t.Log("测试内容 .env 隐私变量 成功")
}
