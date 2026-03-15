package app

import (
	"fmt"
	"log/slog"

	"pathfinder/internal/config"
)

// Run 接收任务描述并执行（当前为占位：打印任务；后续对接后端并启动 TUI）。
func Run(message string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	slog.Debug("config", "provider", cfg.DefaultProvider, "model", cfg.DefaultModel, "workspace", cfg.WorkspaceDir)
	fmt.Println("任务:", message)
	return nil
}
