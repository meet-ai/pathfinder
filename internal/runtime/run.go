package runtime

import (
	"pathfinder/internal/planning"
	"time"
)

// JobId 单次运行的唯一标识。
type JobId string

// StreamHandle 流式输出句柄，供订阅进度与流式日志。
type StreamHandle string

// JobStatus 运行状态。
type JobStatus string

const (
	JobStatusCreated   JobStatus = "created"
	JobStatusPlanning JobStatus = "planning"
	JobStatusRunning  JobStatus = "running"
	JobStatusAborted  JobStatus = "aborted"
	JobStatusCompleted JobStatus = "completed"
)

// Job 运行时聚合根：job 生命周期、取消标志、deadline。
type Job struct {
	Id               JobId
	StreamHandle     StreamHandle
	Status           JobStatus
	CreatedAt        time.Time
	Deadline         *time.Time
	CancelRequested  bool
	PlanId           planning.PlanId
}

// Create 构造新 Job（由应用层/仓储调用，领域仅约定不变式）。
func Create(id JobId, streamHandle StreamHandle, planId planning.PlanId, deadline *time.Time) *Job {
	return &Job{
		Id:           id,
		StreamHandle: streamHandle,
		Status:       JobStatusCreated,
		CreatedAt:    time.Now().UTC(),
		Deadline:     deadline,
		PlanId:       planId,
	}
}

// Cancel 标记用户请求取消。
func (r *Job) Cancel() {
	r.CancelRequested = true
}

// MarkAborted 标记已中止。
func (r *Job) MarkAborted() {
	r.Status = JobStatusAborted
}

// MarkCompleted 标记已完成。
func (r *Job) MarkCompleted() {
	r.Status = JobStatusCompleted
}

// IsCancelRequested 是否已请求取消。
func (r *Job) IsCancelRequested() bool {
	return r.CancelRequested
}

// IsOverDeadline 是否已过截止时间。
func (r *Job) IsOverDeadline(now time.Time) bool {
	if r.Deadline == nil {
		return false
	}
	return now.After(*r.Deadline) || now.Equal(*r.Deadline)
}
