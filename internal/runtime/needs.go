package runtime

import "context"

// JobRepository 运行仓储端口：持久化 Job，由 infra 实现。
type JobRepository interface {
	Save(ctx context.Context, r *Job) error
	Get(ctx context.Context, id JobId) (*Job, error)
}

