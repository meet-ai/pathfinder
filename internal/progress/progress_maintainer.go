package progress

import (
	"context"

	"pathfinder/internal/runtime"
)

// ProgressMaintainer 领域服务：批量更新进度、checkpoint、恢复；依赖 TaskProgressRepository。
type ProgressMaintainer struct {
	Repo TaskProgressRepository
}

// BatchUpdateProgress 批量写入任务进度。
func (m *ProgressMaintainer) BatchUpdateProgress(ctx context.Context, runId runtime.JobId, updates []TaskProgress) error {
	for i := range updates {
		if err := m.Repo.Save(ctx, runId, &updates[i]); err != nil {
			return err
		}
	}
	return nil
}

// Checkpoint 对当前 job 做进度快照（由 Repo 实现具体快照逻辑，此处仅调用保存）。
func (m *ProgressMaintainer) Checkpoint(ctx context.Context, runId runtime.JobId, tasks []TaskProgress) error {
	for i := range tasks {
		if err := m.Repo.Save(ctx, runId, &tasks[i]); err != nil {
			return err
		}
	}
	return nil
}

// Restore 按 JobId 恢复该 job 下所有 TaskProgress。
func (m *ProgressMaintainer) Restore(ctx context.Context, runId runtime.JobId) ([]TaskProgress, error) {
	return m.Repo.ListByRunId(ctx, runId)
}
