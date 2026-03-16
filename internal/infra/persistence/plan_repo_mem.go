package persistence

import (
	"context"
	"sync"

	"pathfinder/internal/planning"
)

// PlanRepoMem 内存实现的 Plan 仓储。
type PlanRepoMem struct {
	mu    sync.RWMutex
	plans map[planning.PlanId]*planning.Plan
}

// NewPlanRepoMem 构造内存 Plan 仓储。
func NewPlanRepoMem() *PlanRepoMem {
	return &PlanRepoMem{plans: make(map[planning.PlanId]*planning.Plan)}
}

// Save 保存 Plan。
func (p *PlanRepoMem) Save(ctx context.Context, plan *planning.Plan) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	cpy := *plan
	cpy.SubTasks = append([]planning.SubTask(nil), plan.SubTasks...)
	cpy.Dependencies = append([]planning.Dependency(nil), plan.Dependencies...)
	cpy.SuggestedAgents = append([]planning.SuggestedAgent(nil), plan.SuggestedAgents...)
	p.plans[plan.Id] = &cpy
	return nil
}

// Get 按 Id 获取 Plan。
func (p *PlanRepoMem) Get(ctx context.Context, id planning.PlanId) (*planning.Plan, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if plan, ok := p.plans[id]; ok {
		cpy := *plan
		cpy.SubTasks = append([]planning.SubTask(nil), plan.SubTasks...)
		cpy.Dependencies = append([]planning.Dependency(nil), plan.Dependencies...)
		cpy.SuggestedAgents = append([]planning.SuggestedAgent(nil), plan.SuggestedAgents...)
		return &cpy, nil
	}
	return nil, nil
}
