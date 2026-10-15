package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/prismbox/prismbox/internal/config"
	"github.com/prismbox/prismbox/internal/kernel"
	"github.com/prismbox/prismbox/internal/monitor"
)

func TestServerRoutes(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := config.NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	mgr := kernel.NewManager("sing-box", tempDir)
	mgr.SetStore(store)
	mon := monitor.NewMonitor()
	defer mon.Stop()

	srv := New(":0", store, mgr, mon)
	if srv == nil {
		t.Fatal("expected non-nil server")
	}

	// 1. Test GET /api/v1/dashboard/stats
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats", nil)
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var statsResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &statsResp); err != nil {
		t.Fatalf("failed to parse dashboard stats response: %v", err)
	}

	if _, ok := statsResp["kernel_status"]; !ok {
		t.Error("expected kernel_status field in dashboard stats response")
	}

	// 2. Test GET /api/v1/kernel/status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/kernel/status", nil)
	rrStatus := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rrStatus, reqStatus)

	if rrStatus.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rrStatus.Code)
	}

	// 3. Test SPA fallback on unknown route
	reqSPA := httptest.NewRequest(http.MethodGet, "/some/spa/route", nil)
	rrSPA := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rrSPA, reqSPA)

	if rrSPA.Code != http.StatusOK {
		t.Errorf("expected status 200 for SPA route, got %d", rrSPA.Code)
	}
}
