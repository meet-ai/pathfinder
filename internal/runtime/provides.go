package runtime

import (
	"context"
)

// JobProgressEvent 表示一次 Job 进度快照，用于 WatchJobProgress 流式推送。
type JobProgressEvent struct {
	JobId        JobId
	Status       JobStatus
	Phase        string // 规划 / 执行 / 总结 等，人类可读阶段名
	Completed    int    // 已完成子任务数
	Total        int    // 总子任务数
	CurrentTask  string // 当前子任务的简要描述（可选）
	CurrentAgent string // 当前子任务所属 agent（可选）
	Message      string // 额外提示信息，如「用户取消」「超时中止」（可选）
}

// JobProgressEventConsumer 由入口层实现，用于将进度事件写入具体连接（SSE/WebSocket 等）。
type JobProgressEventConsumer interface {
	Push(ctx context.Context, event JobProgressEvent) error
}

// RuntimeQueryService 提供 RuntimeContext 下的只读查询/订阅能力。
type RuntimeQueryService interface {
	WatchJobProgress(ctx context.Context, jobId JobId, consumer JobProgressEventConsumer) error
}

// DefaultRuntimeQueryService 是基于 JobRepository 的最小实现壳。
type DefaultRuntimeQueryService struct {
	jobs JobRepository
}

func NewDefaultRuntimeQueryService(jobs JobRepository) *DefaultRuntimeQueryService {
	return &DefaultRuntimeQueryService{jobs: jobs}
}

func (s *DefaultRuntimeQueryService) WatchJobProgress(ctx context.Context, jobId JobId, consumer JobProgressEventConsumer) error {
	job, err := s.jobs.Get(ctx, jobId)
	if err != nil {
		return err
	}
	if job == nil {
		return nil
	}

	event := JobProgressEvent{
		JobId:     jobId,
		Status:    job.Status,
		Phase:     "", // TODO: 后续根据执行阶段填充（规划/执行/总结）。
		Completed: 0,
		Total:     0,
	}
	if err := consumer.Push(ctx, event); err != nil {
		return err
	}
	return nil
}


