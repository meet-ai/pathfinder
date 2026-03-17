package clients

import (
	"context"

	"pathfinder/internal/kanban"
)

// KanbanAssigneeResolverMem 责任人分配占位实现。
type KanbanAssigneeResolverMem struct {
	DefaultAssignee string
}

func NewKanbanAssigneeResolverMem() *KanbanAssigneeResolverMem {
	return &KanbanAssigneeResolverMem{DefaultAssignee: "agent-1"}
}

func (r *KanbanAssigneeResolverMem) Resolve(ctx context.Context, boardId string, taskType string) (string, error) {
	_ = ctx
	_ = boardId
	_ = taskType
	return r.DefaultAssignee, nil
}

// KanbanEventPublisherMem 领域事件发布占位实现。
type KanbanEventPublisherMem struct{}

func NewKanbanEventPublisherMem() *KanbanEventPublisherMem {
	return &KanbanEventPublisherMem{}
}

func (p *KanbanEventPublisherMem) Publish(ctx context.Context, event kanban.DomainEvent) error {
	_ = ctx
	_ = event
	return nil
}
