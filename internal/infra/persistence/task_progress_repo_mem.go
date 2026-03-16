package persistence

import (
	"context"
	"sync"

	"pathfinder/internal/planning"
	"pathfinder/internal/progress"
	"pathfinder/internal/runtime"
)

// TaskProgressRepoMem 内存实现的任务进度仓储。
type TaskProgressRepoMem struct {
	mu     sync.RWMutex
	byRun  map[runtime.JobId][]*progress.TaskProgress
}

// NewTaskProgressRepoMem 构造内存 TaskProgress 仓储。
func NewTaskProgressRepoMem() *TaskProgressRepoMem {
	return &TaskProgressRepoMem{byRun: make(map[runtime.JobId][]*progress.TaskProgress)}
}

// Save 保存单条 TaskProgress（同 JobId+TaskId 则覆盖）；存堆上副本，避免悬空指针。
func (t *TaskProgressRepoMem) Save(ctx context.Context, runId runtime.JobId, tp *progress.TaskProgress) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	store := new(progress.TaskProgress)
	*store = *tp
	list := t.byRun[runId]
	for i, p := range list {
		if p.TaskId == tp.TaskId {
			list[i] = store
			return nil
		}
	}
	t.byRun[runId] = append(list, store)
	return nil
}

// Get 按 JobId+TaskId 获取。
func (t *TaskProgressRepoMem) Get(ctx context.Context, runId runtime.JobId, taskId planning.TaskId) (*progress.TaskProgress, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, p := range t.byRun[runId] {
		if p.TaskId == taskId {
			cpy := *p
			return &cpy, nil
		}
	}
	return nil, nil
}

// ListByRunId 返回该 job 下全部 TaskProgress。
func (t *TaskProgressRepoMem) ListByRunId(ctx context.Context, runId runtime.JobId) ([]progress.TaskProgress, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	list := t.byRun[runId]
	out := make([]progress.TaskProgress, len(list))
	for i, p := range list {
		out[i] = *p
	}
	return out, nil
}
