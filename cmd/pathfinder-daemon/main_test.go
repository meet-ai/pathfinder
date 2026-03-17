package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"pathfinder/internal/agent"
	"pathfinder/internal/infra/clients"
	"pathfinder/internal/infra/persistence"
	"pathfinder/internal/orchestration"
)

// newTestService 创建仅用于测试的编排服务，使用内存仓储与 stub dispatcher，避免外部依赖。
func newTestService(t *testing.T) *server {
	t.Helper()

	runRepo := persistence.NewJobRepoMem()
	planRepo := persistence.NewPlanRepoMem()
	taskProgressRepo := persistence.NewTaskProgressRepoMem()
	planner := &persistence.PlannerStub{}
	agentDiscovery := clients.NewAgentDiscoveryMem()
	var dispatcher agent.Dispatcher = &clients.DispatcherStub{}

	svc := &orchestration.WorkflowOrchestrationApplicationService{
		Planner:        planner,
		RunRepo:        runRepo,
		PlanRepo:       planRepo,
		TaskProgress:   taskProgressRepo,
		AgentDiscovery: agentDiscovery,
		Dispatcher:     dispatcher,
	}
	return &server{
		svc:             svc,
		capabilitySvc:   nil,
		runtimeQuerySvc: nil,
	}
}

func TestHTTP_SubmitJob_Then_PollState(t *testing.T) {
	s := newTestService(t)

	// 提交一个 job
	body := map[string]any{
		"goal": "http e2e submit & poll",
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewReader(data))
	s.handleCreateJob(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /jobs status = %d, want 200", resp.StatusCode)
	}
	var created struct {
		JobID string `json:"jobId"`
		URL   string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.JobID == "" {
		t.Fatal("create response jobId 为空")
	}

	// 轮询状态直到 completed/aborted
	deadline := time.Now().Add(5 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("poll state timeout for jobId=%s", created.JobID)
		}

		// 构造带有 chi.RouteContext 的请求，供 handleGetJob 使用 chi.URLParam 解析 jobId。
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/jobs/"+created.JobID, nil)
		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("jobId", created.JobID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

		s.handleGetJob(rec, req)
		res := rec.Result()
		if res.StatusCode == http.StatusNotFound {
			_ = res.Body.Close()
			time.Sleep(20 * time.Millisecond)
			continue
		}
		if res.StatusCode != http.StatusOK {
			_ = res.Body.Close()
			t.Fatalf("GET job state status = %d, want 200", res.StatusCode)
		}
		var state struct {
			Status    string `json:"Status"`
			Completed int    `json:"Completed"`
			Total     int    `json:"Total"`
			Summary   string `json:"Summary"`
		}
		if err := json.NewDecoder(res.Body).Decode(&state); err != nil {
			_ = res.Body.Close()
			t.Fatalf("decode state: %v", err)
		}
		_ = res.Body.Close()

		if state.Status == "completed" || state.Status == "aborted" {
			if state.Completed < 1 || state.Total < 1 {
				t.Fatalf("进度 = %d/%d, want >= 1/1", state.Completed, state.Total)
			}
			if state.Summary == "" {
				t.Fatalf("Summary 为空")
			}
			t.Logf("测试内容 HTTP Submit+GetJobState 成功: jobId=%s status=%s 进度=%d/%d",
				created.JobID, state.Status, state.Completed, state.Total)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestHTTP_SubmitJob_Then_Cancel(t *testing.T) {
	s := newTestService(t)

	// 提交一个 job
	body := map[string]any{
		"goal": "http e2e cancel",
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewReader(data))
	s.handleCreateJob(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /jobs status = %d, want 200", resp.StatusCode)
	}
	var created struct {
		JobID string `json:"jobId"`
		URL   string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.JobID == "" {
		t.Fatal("create response jobId 为空")
	}

	// 发送取消请求
	cancelRec := httptest.NewRecorder()
	cancelReq := httptest.NewRequest(http.MethodPost, "/jobs/"+created.JobID+"/cancel", nil)
	cancelRouteCtx := chi.NewRouteContext()
	cancelRouteCtx.URLParams.Add("jobId", created.JobID)
	cancelReq = cancelReq.WithContext(context.WithValue(cancelReq.Context(), chi.RouteCtxKey, cancelRouteCtx))
	s.handleCancelJob(cancelRec, cancelReq)
	cancelResp := cancelRec.Result()
	cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusNoContent {
		t.Fatalf("cancel status = %d, want 204", cancelResp.StatusCode)
	}

	// 轮询直到 aborted
	deadline := time.Now().Add(5 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("poll state timeout for cancelled jobId=%s", created.JobID)
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/jobs/"+created.JobID, nil)
		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("jobId", created.JobID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

		s.handleGetJob(rec, req)
		res := rec.Result()
		if res.StatusCode == http.StatusNotFound {
			_ = res.Body.Close()
			time.Sleep(20 * time.Millisecond)
			continue
		}
		if res.StatusCode != http.StatusOK {
			_ = res.Body.Close()
			t.Fatalf("GET job state status = %d, want 200", res.StatusCode)
		}
		var state struct {
			Status string `json:"Status"`
		}
		if err := json.NewDecoder(res.Body).Decode(&state); err != nil {
			_ = res.Body.Close()
			t.Fatalf("decode state: %v", err)
		}
		_ = res.Body.Close()

		if state.Status == "aborted" {
			t.Logf("测试内容 HTTP CancelJob 成功: jobId=%s status=%s", created.JobID, state.Status)
			break
		}
		if state.Status == "completed" {
			t.Logf("取消请求发送成功，但 job 已在取消生效前完成: jobId=%s status=%s", created.JobID, state.Status)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
}
