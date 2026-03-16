package persistence

import (
	"context"
	"sync"

	"pathfinder/internal/runtime"
)

// JobRepoMem 内存实现的 Job 仓储，供测试与单机运行。
type JobRepoMem struct {
	mu   sync.RWMutex
	runs map[runtime.JobId]*runtime.Job
}

// NewJobRepoMem 构造内存 Job 仓储。
func NewJobRepoMem() *JobRepoMem {
	return &JobRepoMem{runs: make(map[runtime.JobId]*runtime.Job)}
}

// Save 保存 Job。
func (r *JobRepoMem) Save(ctx context.Context, run *runtime.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cpy := *run
	r.runs[run.Id] = &cpy
	return nil
}

// Get 按 Id 获取 Job。
func (r *JobRepoMem) Get(ctx context.Context, id runtime.JobId) (*runtime.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if run, ok := r.runs[id]; ok {
		cpy := *run
		return &cpy, nil
	}
	return nil, nil
}
