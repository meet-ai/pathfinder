package progress

import (
	"context"

	"pathfinder/internal/planning"
	"pathfinder/internal/runtime"
)

// TaskProgressRepository 任务进度仓储端口：按 JobId+TaskId 持久化 TaskProgress，由 infra 实现。
type TaskProgressRepository interface {
	Save(ctx context.Context, runId runtime.JobId, t *TaskProgress) error
	Get(ctx context.Context, runId runtime.JobId, taskId planning.TaskId) (*TaskProgress, error)
	ListByRunId(ctx context.Context, runId runtime.JobId) ([]TaskProgress, error)
}
