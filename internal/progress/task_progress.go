package progress

import (
	"pathfinder/internal/planning"
	"pathfinder/internal/runtime"
	"time"
)

// TaskStatus 单任务进度状态。
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusSkipped   TaskStatus = "skipped"
)

// Checkpoint 进度快照，用于恢复。
type Checkpoint struct {
	RunId     runtime.JobId
	TaskId    planning.TaskId
	Status    TaskStatus
	AgentId   string
	StartedAt *time.Time
	Result    string
	UpdatedAt time.Time
}

// TaskProgress 单任务进度实体（按 JobId+TaskId 唯一）。
type TaskProgress struct {
	RunId     runtime.JobId
	TaskId    planning.TaskId
	Status    TaskStatus
	AgentId   string
	StartedAt *time.Time
	Result    string
	UpdatedAt time.Time
}

// Start 标记为进行中。
func (t *TaskProgress) Start(agentId string) {
	now := time.Now().UTC()
	t.Status = TaskStatusRunning
	t.AgentId = agentId
	t.StartedAt = &now
	t.UpdatedAt = now
}

// Complete 标记为已完成并写入结果。
func (t *TaskProgress) Complete(result string) {
	now := time.Now().UTC()
	t.Status = TaskStatusCompleted
	t.Result = result
	t.UpdatedAt = now
}

// Fail 标记为失败。
func (t *TaskProgress) Fail(result string) {
	now := time.Now().UTC()
	t.Status = TaskStatusFailed
	t.Result = result
	t.UpdatedAt = now
}

// WriteResult 仅更新结果字段（状态已为 completed/failed 时可用）。
func (t *TaskProgress) WriteResult(result string) {
	t.Result = result
	t.UpdatedAt = time.Now().UTC()
}
