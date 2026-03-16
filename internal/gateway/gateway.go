package gateway

import (
	"context"
	"pathfinder/internal/runtime"
)

// StreamPublisher 流式推送端口：按 JobId 推送进度与流式日志；由 gateway 包实现或 infra 实现。
type StreamPublisher interface {
	Stream(ctx context.Context, runId runtime.JobId) (<-chan StreamEvent, error)
}

// StreamEvent 单条流式事件（进度/日志块）。
type StreamEvent struct {
	Kind string
	Body []byte
}

// CancelJob 取消 job 的端口；编排层提供，gateway 调用。
type CancelJob func(ctx context.Context, runId string) error

// Server HTTP/SSE/WS 入口：暴露 Stream(JobId)、Cancel(JobId)；needs TaskProgressRepository、orchestration.Cancel、config。
type Server struct {
	Stream   StreamPublisher
	CancelFn CancelJob
}
