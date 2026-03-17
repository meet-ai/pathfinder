package kanban

import (
	"context"
	"time"
)

// CardStatus 卡片状态。
type CardStatus string

const (
	CardStatusTodo       CardStatus = "todo"
	CardStatusInProgress CardStatus = "in_progress"
	CardStatusInReview   CardStatus = "in_review"
	CardStatusBlocked    CardStatus = "blocked"
	CardStatusDone       CardStatus = "done"
)

// Board 看板聚合（最小占位）。
type Board struct {
	Id              string
	InProgressLimit int
	InReviewLimit   int
}

// Card 卡片聚合（最小占位）。
type Card struct {
	Id             string
	BoardId        string
	Title          string
	Description    string
	Creator        string
	Assignee       string
	Reviewer       string
	Status         CardStatus
	BlockReason    string
	CreatedAt      time.Time
	LastActivityAt time.Time
	CompletedAt    *time.Time
}

// DomainEvent 领域事件（最小占位）。
type DomainEvent struct {
	Name       string
	BoardId    string
	CardId     string
	Operator   string
	OccurredAt time.Time
}

// BoardRepository 看板仓储端口，由 infra 实现。
type BoardRepository interface {
	Get(ctx context.Context, boardId string) (*Board, error)
	Save(ctx context.Context, board *Board) error
}

// CardRepository 卡片仓储端口，由 infra 实现。
type CardRepository interface {
	Create(ctx context.Context, card *Card) error
	Save(ctx context.Context, card *Card) error
	Get(ctx context.Context, cardId string) (*Card, error)
	ListByBoard(ctx context.Context, boardId string) ([]Card, error)
	CountByBoardAndStatus(ctx context.Context, boardId string, status CardStatus) (int, error)
}

// AssigneeResolver 责任人解析端口，由上游调度/目录能力实现。
type AssigneeResolver interface {
	Resolve(ctx context.Context, boardId string, taskType string) (string, error)
}

// EventPublisher 事件发布端口，供读模型/审计消费。
type EventPublisher interface {
	Publish(ctx context.Context, event DomainEvent) error
}
