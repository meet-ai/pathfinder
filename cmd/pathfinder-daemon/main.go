package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"pathfinder/internal/agent"
	"pathfinder/internal/capabilitycatalog"
	"pathfinder/internal/config"
	"pathfinder/internal/infra/clients"
	"pathfinder/internal/infra/persistence"
	"pathfinder/internal/infra/openclaw"
	"pathfinder/internal/kanban"
	"pathfinder/internal/orchestration"
	"pathfinder/internal/planning"
	"pathfinder/internal/provider"
	"pathfinder/internal/runtime"
	"pathfinder/internal/sync"
)

type server struct {
	svc             *orchestration.WorkflowOrchestrationApplicationService
	capabilitySvc   capabilitycatalog.CapabilityCatalogQueryService
	runtimeQuerySvc runtime.RuntimeQueryService
	kanbanSvc       *kanban.ApplicationService
	syncSvc         *sync.ApplicationService
}

type agentDirectoryFromDiscovery struct {
	discovery agent.AgentDiscovery
}

func (d *agentDirectoryFromDiscovery) ListAll(ctx context.Context) ([]capabilitycatalog.AgentRecord, error) {
	agents, err := d.discovery.ListAgents(ctx, agent.AgentPoolFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]capabilitycatalog.AgentRecord, 0, len(agents))
	for _, a := range agents {
		out = append(out, capabilitycatalog.AgentRecord{
			ID:                  string(a.Id),
			Name:                a.Name,
			Description:         "",
			Version:             "",
			Tags:                append([]string(nil), a.Tags...),
			Groups:              nil,
			Capabilities:        nil,
			DangerousOperations: nil,
			SupportedContexts:   nil,
			Requirements:        nil,
		})
	}
	return out, nil
}

func (d *agentDirectoryFromDiscovery) GetByID(ctx context.Context, id string) (*capabilitycatalog.AgentRecord, error) {
	agents, err := d.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range agents {
		if agents[i].ID == id {
			cp := agents[i]
			return &cp, nil
		}
	}
	return nil, nil
}

type sseJobProgressConsumer struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

//go:embed web/kanban/index.html
var kanbanIndexHTML string

const defaultKanbanBoardID = "main-board"

func (c *sseJobProgressConsumer) Push(ctx context.Context, event runtime.JobProgressEvent) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.w, "data: %s\n\n", b); err != nil {
		return err
	}
	c.flusher.Flush()
	return nil
}

func (s *server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		Goal       string `json:"goal"`
		TimeoutSec int64  `json:"timeoutSec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	goal := strings.TrimSpace(req.Goal)
	if goal == "" {
		http.Error(w, "goal is required", http.StatusBadRequest)
		return
	}

	cmd := orchestration.SubmitGoalCommand{
		GoalDescription: planning.GoalDescription(goal),
	}
	if req.TimeoutSec > 0 {
		cmd.TimeoutSecs = uint64(req.TimeoutSec)
	}

	jobId, err := s.svc.StartJob(ctx, cmd)
	if err != nil {
		slog.Error("StartJob failed", "err", err)
		http.Error(w, "StartJob failed", http.StatusInternalServerError)
		return
	}
	go s.svc.ContinueJob(context.Background(), jobId)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "127.0.0.1:8080"
	}
	url := fmt.Sprintf("%s://%s/jobs/%s", scheme, host, jobId)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		JobID string `json:"jobId"`
		URL   string `json:"url"`
	}{
		JobID: jobId,
		URL:   url,
	})
}

type kanbanCardDTO struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Column      string `json:"column"`
	Assignee    string `json:"assignee"`
	Reviewer    string `json:"reviewer"`
	UpdatedAt   string `json:"updatedAt"`
}

func (s *server) handleKanbanIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(kanbanIndexHTML))
}

func (s *server) handleKanbanBoard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	boardId := r.URL.Query().Get("boardId")
	if boardId == "" {
		boardId = defaultKanbanBoardID
	}
	cardDTOs, err := s.kanbanSvc.ListCards(ctx, boardId)
	if err != nil {
		http.Error(w, "ListCards failed", http.StatusInternalServerError)
		return
	}
	cards := make([]kanbanCardDTO, 0, len(cardDTOs))
	for _, card := range cardDTOs {
		cards = append(cards, kanbanCardDTO{
			ID:          card.ID,
			Title:       card.Title,
			Description: card.Description,
			Column:      string(card.Status),
			Assignee:    card.Assignee,
			Reviewer:    card.Reviewer,
			UpdatedAt:   card.LastActivityAt.Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		BoardId string          `json:"boardId"`
		Cards   []kanbanCardDTO `json:"cards"`
	}{
		BoardId: "job-overview-board",
		Cards:   cards,
	})
}

func (s *server) handleGetKanbanCard(w http.ResponseWriter, r *http.Request) {
	cardId := chi.URLParam(r, "cardId")
	if cardId == "" {
		http.NotFound(w, r)
		return
	}
	card, err := s.kanbanSvc.GetCard(r.Context(), cardId)
	if err != nil {
		http.Error(w, "GetCard failed", http.StatusInternalServerError)
		return
	}
	if card == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(card)
}

func (s *server) handleCreateKanbanCard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BoardId     string `json:"boardId"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Creator     string `json:"creator"`
		Reviewer    string `json:"reviewer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.BoardId == "" {
		req.BoardId = defaultKanbanBoardID
	}
	card, err := s.kanbanSvc.CreateCard(
		r.Context(),
		req.BoardId,
		strings.TrimSpace(req.Title),
		strings.TrimSpace(req.Description),
		strings.TrimSpace(req.Creator),
		strings.TrimSpace(req.Reviewer),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(card)
}

func (s *server) handleAssignKanbanCard(w http.ResponseWriter, r *http.Request) {
	cardId := chi.URLParam(r, "cardId")
	var req struct {
		TaskType string `json:"taskType"`
		Operator string `json:"operator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Operator == "" {
		req.Operator = "system"
	}
	card, err := s.kanbanSvc.AssignCard(r.Context(), cardId, req.TaskType, req.Operator)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(card)
}

func (s *server) handleMoveKanbanCard(w http.ResponseWriter, r *http.Request) {
	cardId := chi.URLParam(r, "cardId")
	var req struct {
		To       string `json:"to"`
		Operator string `json:"operator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Operator == "" {
		req.Operator = "system"
	}
	card, err := s.kanbanSvc.MoveCard(r.Context(), cardId, kanban.CardStatus(req.To), req.Operator)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(card)
}

func (s *server) handleBlockKanbanCard(w http.ResponseWriter, r *http.Request) {
	cardId := chi.URLParam(r, "cardId")
	var req struct {
		Reason   string `json:"reason"`
		Operator string `json:"operator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Operator == "" {
		req.Operator = "system"
	}
	card, err := s.kanbanSvc.BlockCard(r.Context(), cardId, strings.TrimSpace(req.Reason), req.Operator)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(card)
}

func (s *server) handleUnblockKanbanCard(w http.ResponseWriter, r *http.Request) {
	cardId := chi.URLParam(r, "cardId")
	var req struct {
		Operator string `json:"operator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Operator == "" {
		req.Operator = "system"
	}
	card, err := s.kanbanSvc.UnblockCard(r.Context(), cardId, req.Operator)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(card)
}

func (s *server) handleReviewPassKanbanCard(w http.ResponseWriter, r *http.Request) {
	cardId := chi.URLParam(r, "cardId")
	var req struct {
		Reviewer string `json:"reviewer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	card, err := s.kanbanSvc.ReviewPassCard(r.Context(), cardId, req.Reviewer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(card)
}

func (s *server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobId := chi.URLParam(r, "jobId")
	if jobId == "" {
		http.NotFound(w, r)
		return
	}
	state, err := s.svc.GetJobState(ctx, jobId)
	if err != nil {
		slog.Error("GetJobState failed", "err", err, "jobId", jobId)
		http.Error(w, "GetJobState failed", http.StatusInternalServerError)
		return
	}
	if state == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}

func (s *server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobId := chi.URLParam(r, "jobId")
	if jobId == "" {
		http.NotFound(w, r)
		return
	}
	if err := s.svc.CancelJob(ctx, jobId); err != nil {
		slog.Error("CancelJob failed", "err", err, "jobId", jobId)
		http.Error(w, "CancelJob failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	query := capabilitycatalog.ListAgentsQuery{
		ProductID: q.Get("productId"),
		Keyword:   q.Get("keyword"),
	}
	if tags, ok := q["tag"]; ok {
		query.Tags = append(query.Tags, tags...)
	}
	if groups, ok := q["group"]; ok && len(groups) > 0 {
		query.Groups = append(query.Groups, groups...)
	}

	agents, err := s.capabilitySvc.ListAgents(ctx, query)
	if err != nil {
		slog.Error("ListAgents failed", "err", err)
		http.Error(w, "ListAgents failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(agents)
}

func (s *server) handleSyncConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	result, err := s.syncSvc.SyncConfig(ctx)
	if err != nil {
		slog.Error("SyncConfig failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// handleGetSyncedConfig 返回当前已同步到本地的拓扑与权限（GET 查看 synced 数据内容）。
func (s *server) handleGetSyncedConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	snap, err := s.syncSvc.GetSyncedSnapshot(ctx)
	if err != nil {
		slog.Error("GetSyncedSnapshot failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}

// handleSyncDoctor 返回配置一致性诊断结果（GET /api/sync/doctor）。
func (s *server) handleSyncDoctor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	report, err := s.syncSvc.RunDoctor(ctx)
	if err != nil {
		slog.Error("RunDoctor failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

func (s *server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	agentId := chi.URLParam(r, "agentId")
	if agentId == "" {
		http.NotFound(w, r)
		return
	}
	productId := r.URL.Query().Get("productId")
	detail, err := s.capabilitySvc.DescribeAgent(ctx, capabilitycatalog.DescribeAgentQuery{
		AgentID:   agentId,
		ProductID: productId,
	})
	if err != nil {
		slog.Error("DescribeAgent failed", "err", err, "agentId", agentId)
		http.Error(w, "DescribeAgent failed", http.StatusInternalServerError)
		return
	}
	if detail.ID == "" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

func newService() (*orchestration.WorkflowOrchestrationApplicationService, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	slog.Debug("config", "provider", cfg.DefaultProvider, "model", cfg.DefaultModel, "workspace", cfg.WorkspaceDir)

	runRepo := persistence.NewJobRepoMem()
	planRepo := persistence.NewPlanRepoMem()
	taskProgressRepo := persistence.NewTaskProgressRepoMem()
	planner := &persistence.PlannerStub{}
	agentDiscovery := clients.NewAgentDiscoveryMem()
	var dispatcher agent.Dispatcher
	if p, err := provider.CreateFromConfig(context.Background(), cfg); err == nil {
		dispatcher = &clients.DispatcherLLM{
			Provider:    p,
			Model:       cfg.DefaultModel,
			Temperature: cfg.DefaultTemperature,
		}
	} else {
		dispatcher = &clients.DispatcherStub{}
	}

	return &orchestration.WorkflowOrchestrationApplicationService{
		Planner:        planner,
		RunRepo:        runRepo,
		PlanRepo:       planRepo,
		TaskProgress:   taskProgressRepo,
		AgentDiscovery: agentDiscovery,
		Dispatcher:     dispatcher,
	}, nil
}

func main() {
	svc, err := newService()
	if err != nil {
		slog.Error("failed to init service", "err", err)
		os.Exit(1)
	}

	agentDiscovery := clients.NewAgentDiscoveryMem()
	dir := &agentDirectoryFromDiscovery{discovery: agentDiscovery}
	metrics := clients.NewProductMetricsStub()
	capSvc := capabilitycatalog.NewDefaultCapabilityCatalogQueryServiceWithMetrics(dir, metrics)

	// runtime query service 目前基于内存 JobRepo 构造最小实现（仅推送 Job 状态快照）。
	jobRepo := persistence.NewJobRepoMem()
	runtimeQuerySvc := runtime.NewDefaultRuntimeQueryService(jobRepo)
	kanbanBoardRepo := persistence.NewKanbanBoardRepoMem()
	kanbanCardRepo := persistence.NewKanbanCardRepoMem()
	_ = kanbanBoardRepo.Save(context.Background(), &kanban.Board{
		Id:              defaultKanbanBoardID,
		InProgressLimit: 10,
		InReviewLimit:   10,
	})

	configAdapter := &openclaw.ConfigAdapter{ConfigPath: os.Getenv("OPENCLAW_JSON_PATH")}
	canonicalStore := persistence.NewCanonicalStoreMem()
	syncSvc := &sync.ApplicationService{
		ConfigSource:   configAdapter,
		CanonicalStore: canonicalStore,
	}
	// 派单从同步结果读：AssigneeResolver 使用 CanonicalStore 的拓扑（agents/bindings）
	kanbanSvc := &kanban.ApplicationService{
		BoardRepo:        kanbanBoardRepo,
		CardRepo:         kanbanCardRepo,
		AssigneeResolver: clients.NewKanbanAssigneeResolverFromSync(canonicalStore),
		EventPublisher:   clients.NewKanbanEventPublisherMem(),
	}
	// 启动时全量同步配置一次，失败只打日志不阻塞启动
	if _, err := syncSvc.SyncConfig(context.Background()); err != nil {
		slog.Warn("startup SyncConfig failed (will use stub or empty store)", "err", err)
	} else {
		slog.Info("startup SyncConfig ok")
	}

	// 监听配置目录，变更时触发 SyncConfig（debounce 300ms）
	configPath := os.Getenv("OPENCLAW_JSON_PATH")
	if configPath == "" {
		configPath = openclaw.DefaultConfigPath()
	}
	watchDir := filepath.Dir(configPath)
	go openclaw.WatchConfigDir(context.Background(), watchDir, 300, func() {
		if _, err := syncSvc.SyncConfig(context.Background()); err != nil {
			slog.Warn("watch SyncConfig failed", "err", err)
		} else {
			slog.Info("watch SyncConfig ok")
		}
	})

	s := &server{
		svc:             svc,
		capabilitySvc:   capSvc,
		runtimeQuerySvc: runtimeQuerySvc,
		kanbanSvc:       kanbanSvc,
		syncSvc:         syncSvc,
	}

	r := chi.NewRouter()

	// jobs
	r.Route("/jobs", func(r chi.Router) {
		r.Post("/", s.handleCreateJob)
		r.Get("/{jobId}", s.handleGetJob)
		r.Post("/{jobId}/cancel", s.handleCancelJob)
		r.Get("/{jobId}/watch", func(w http.ResponseWriter, r *http.Request) {
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming not supported", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			ctx := r.Context()
			jobId := chi.URLParam(r, "jobId")
			if jobId == "" {
				http.NotFound(w, r)
				return
			}
			consumer := &sseJobProgressConsumer{w: w, flusher: flusher}
			if err := s.runtimeQuerySvc.WatchJobProgress(ctx, runtime.JobId(jobId), consumer); err != nil {
				slog.Error("WatchJobProgress failed", "err", err, "jobId", jobId)
			}
		})
	})

	// readonly kanban
	r.Get("/kanban", s.handleKanbanIndex)
	r.Get("/api/kanban/board", s.handleKanbanBoard)
	r.Get("/api/kanban/cards/{cardId}", s.handleGetKanbanCard)
	r.Post("/api/kanban/cards", s.handleCreateKanbanCard)
	r.Post("/api/kanban/cards/{cardId}/assign", s.handleAssignKanbanCard)
	r.Post("/api/kanban/cards/{cardId}/move", s.handleMoveKanbanCard)
	r.Post("/api/kanban/cards/{cardId}/block", s.handleBlockKanbanCard)
	r.Post("/api/kanban/cards/{cardId}/unblock", s.handleUnblockKanbanCard)
	r.Post("/api/kanban/cards/{cardId}/review-pass", s.handleReviewPassKanbanCard)

	// OpenClaw 同步：POST 触发同步，GET 查看已同步数据，GET doctor 诊断
	r.Post("/api/sync/config", s.handleSyncConfig)
	r.Get("/api/sync/config", s.handleGetSyncedConfig)
	r.Get("/api/sync/doctor", s.handleSyncDoctor)

	// capability catalog (agents)
	r.Get("/agents", s.handleListAgents)
	r.Get("/agents/{agentId}", s.handleGetAgent)

	// agents
	r.Get("/agents", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		list, err := s.capabilitySvc.ListAgents(ctx, capabilitycatalog.ListAgentsQuery{})
		if err != nil {
			slog.Error("ListAgents failed", "err", err)
			http.Error(w, "ListAgents failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	})

	addr := os.Getenv("PATHFINDER_DAEMON_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("listen failed", "addr", addr, "err", err)
		os.Exit(1)
	}
	slog.Info("pathfinder daemon listening", "addr", ln.Addr().String())
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		slog.Error("http server failed", "err", err)
		os.Exit(1)
	}
}
