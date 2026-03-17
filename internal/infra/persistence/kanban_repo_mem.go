package persistence

import (
	"context"
	"sync"

	"pathfinder/internal/kanban"
)

// KanbanBoardRepoMem 看板内存仓储。
type KanbanBoardRepoMem struct {
	mu     sync.RWMutex
	boards map[string]*kanban.Board
}

func NewKanbanBoardRepoMem() *KanbanBoardRepoMem {
	return &KanbanBoardRepoMem{
		boards: map[string]*kanban.Board{},
	}
}

func (r *KanbanBoardRepoMem) Get(ctx context.Context, boardId string) (*kanban.Board, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	board := r.boards[boardId]
	if board == nil {
		return nil, nil
	}
	cp := *board
	return &cp, nil
}

func (r *KanbanBoardRepoMem) Save(ctx context.Context, board *kanban.Board) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *board
	r.boards[board.Id] = &cp
	return nil
}

// KanbanCardRepoMem 卡片内存仓储。
type KanbanCardRepoMem struct {
	mu    sync.RWMutex
	cards map[string]*kanban.Card
}

func NewKanbanCardRepoMem() *KanbanCardRepoMem {
	return &KanbanCardRepoMem{
		cards: map[string]*kanban.Card{},
	}
}

func (r *KanbanCardRepoMem) Create(ctx context.Context, card *kanban.Card) error {
	return r.Save(ctx, card)
}

func (r *KanbanCardRepoMem) Save(ctx context.Context, card *kanban.Card) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *card
	r.cards[card.Id] = &cp
	return nil
}

func (r *KanbanCardRepoMem) Get(ctx context.Context, cardId string) (*kanban.Card, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	card := r.cards[cardId]
	if card == nil {
		return nil, nil
	}
	cp := *card
	return &cp, nil
}

func (r *KanbanCardRepoMem) ListByBoard(ctx context.Context, boardId string) ([]kanban.Card, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]kanban.Card, 0, len(r.cards))
	for _, card := range r.cards {
		if card.BoardId == boardId {
			cp := *card
			out = append(out, cp)
		}
	}
	return out, nil
}

func (r *KanbanCardRepoMem) CountByBoardAndStatus(ctx context.Context, boardId string, status kanban.CardStatus) (int, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, card := range r.cards {
		if card.BoardId == boardId && card.Status == status {
			count++
		}
	}
	return count, nil
}
