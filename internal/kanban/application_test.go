package kanban

import (
	"context"
	"testing"
)

type boardRepoMem struct {
	board *Board
}

func (r *boardRepoMem) Get(ctx context.Context, boardId string) (*Board, error) {
	_ = ctx
	if r.board != nil && r.board.Id == boardId {
		return r.board, nil
	}
	return nil, nil
}

func (r *boardRepoMem) Save(ctx context.Context, board *Board) error {
	_ = ctx
	r.board = board
	return nil
}

type cardRepoMem struct {
	cards map[string]*Card
}

func (r *cardRepoMem) Create(ctx context.Context, card *Card) error {
	_ = ctx
	if r.cards == nil {
		r.cards = map[string]*Card{}
	}
	r.cards[card.Id] = card
	return nil
}

func (r *cardRepoMem) Save(ctx context.Context, card *Card) error {
	_ = ctx
	if r.cards == nil {
		r.cards = map[string]*Card{}
	}
	r.cards[card.Id] = card
	return nil
}

func (r *cardRepoMem) Get(ctx context.Context, cardId string) (*Card, error) {
	_ = ctx
	card := r.cards[cardId]
	if card == nil {
		return nil, nil
	}
	return card, nil
}

func (r *cardRepoMem) ListByBoard(ctx context.Context, boardId string) ([]Card, error) {
	_ = ctx
	out := make([]Card, 0, len(r.cards))
	for _, card := range r.cards {
		if card.BoardId == boardId {
			out = append(out, *card)
		}
	}
	return out, nil
}

func (r *cardRepoMem) CountByBoardAndStatus(ctx context.Context, boardId string, status CardStatus) (int, error) {
	_ = ctx
	count := 0
	for _, card := range r.cards {
		if card.BoardId == boardId && card.Status == status {
			count++
		}
	}
	return count, nil
}

type resolverStub struct{}

func (r *resolverStub) Resolve(ctx context.Context, boardId string, taskType string) (string, error) {
	_ = ctx
	_ = boardId
	_ = taskType
	return "agent-alpha", nil
}

type publisherMem struct {
	events []DomainEvent
}

func (p *publisherMem) Publish(ctx context.Context, event DomainEvent) error {
	_ = ctx
	p.events = append(p.events, event)
	return nil
}

func TestManageKanbanCardLifecycleHappyPathSuccess(t *testing.T) {
	svc := &ApplicationService{
		BoardRepo: &boardRepoMem{
			board: &Board{
				Id:              "board-1",
				InProgressLimit: 3,
				InReviewLimit:   2,
			},
		},
		CardRepo:         &cardRepoMem{cards: map[string]*Card{}},
		AssigneeResolver: &resolverStub{},
		EventPublisher:   &publisherMem{},
	}

	result, err := svc.ManageKanbanCardLifecycle(context.Background(), ManageKanbanCardLifecycleCommand{
		BoardId:        "board-1",
		Title:          "完善看板生命周期",
		Description:    "验证创建、阻塞、评审、完成主链路",
		Creator:        "planner",
		TaskType:       "feature",
		Reviewer:       "reviewer-1",
		NeedBlockCycle: true,
		BlockReason:    "等待上游依赖",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CurrentStatus != CardStatusDone {
		t.Fatalf("expected done, got %s", result.CurrentStatus)
	}
	if result.Assignee != "agent-alpha" {
		t.Fatalf("expected assignee agent-alpha, got %s", result.Assignee)
	}
	if result.CompletedAt == nil {
		t.Fatalf("expected completedAt not nil")
	}
	if len(result.LifecycleEvents) != 7 {
		t.Fatalf("expected 7 lifecycle events, got %d", len(result.LifecycleEvents))
	}

	t.Log("测试内容 Kanban 卡片生命周期 Happy Path 成功")
}
