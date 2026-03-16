package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"pathfinder/internal/agent"
	"pathfinder/internal/capabilitycatalog"
	"pathfinder/internal/config"
	"pathfinder/internal/infra/clients"
	"pathfinder/internal/infra/persistence"
	"pathfinder/internal/orchestration"
	"pathfinder/internal/planning"
	"pathfinder/internal/provider"
	"pathfinder/internal/runtime"
)

type server struct {
	svc             *orchestration.WorkflowOrchestrationApplicationService
	capabilitySvc   capabilitycatalog.CapabilityCatalogQueryService
	runtimeQuerySvc runtime.RuntimeQueryService
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
			ID:          string(a.Id),
			Name:        a.Name,
			Description: "",
			Version:     "",
			Tags:        append([]string(nil), a.Tags...),
			Groups:      nil,
			Capabilities: nil,
			DangerousOperations: nil,
			SupportedContexts:  nil,
			Requirements:       nil,
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

	s := &server{
		svc:             svc,
		capabilitySvc:   capSvc,
		runtimeQuerySvc: runtimeQuerySvc,
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

