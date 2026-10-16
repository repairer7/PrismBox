package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestConnectionsAPI_MockClashAPI(t *testing.T) {
	// Create a mock Clash API server responding to /connections
	mockClash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path == "/connections" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"downloadTotal": 1048576,
					"uploadTotal":   524288,
					"connections": []map[string]interface{}{
						{
							"id": "conn-1234",
							"metadata": map[string]interface{}{
								"network":         "tcp",
								"type":            "tproxy",
								"sourceIP":        "192.168.1.108",
								"sourcePort":      "51234",
								"destinationIP":   "142.250.190.46",
								"destinationPort": "443",
								"host":            "www.google.com",
							},
							"upload":      12000,
							"download":    85000,
							"start":       "2026-10-16T10:00:00Z",
							"chains":      []string{"Proxy", "Hysteria2-HK"},
							"rule":        "geosite-google",
							"rulePayload": "geosite-google",
						},
						{
							"id": "conn-5678",
							"metadata": map[string]interface{}{
								"network":         "udp",
								"type":            "tun",
								"sourceIP":        "192.168.1.115",
								"sourcePort":      "49876",
								"destinationIP":   "1.1.1.1",
								"destinationPort": "53",
								"host":            "",
							},
							"upload":      500,
							"download":    1200,
							"start":       "2026-10-16T10:01:00Z",
							"chains":      []string{"direct"},
							"rule":        "default",
							"rulePayload": "",
						},
					},
				})
				return
			}
		case http.MethodDelete:
			if r.URL.Path == "/connections/conn-1234" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if r.URL.Path == "/connections" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer mockClash.Close()

	// Point test endpoint to mock server (strip http://)
	host := strings.TrimPrefix(mockClash.URL, "http://")
	SetConnectionsClashEndpoint(host)
	defer SetConnectionsClashEndpoint("127.0.0.1:9091")

	// 1. Test GET /connections
	req := httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil)
	rr := httptest.NewRecorder()
	handler := HandleGetConnections(nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Success bool                `json:"success"`
		Data    ConnectionsResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success: true")
	}
	if len(resp.Data.Connections) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(resp.Data.Connections))
	}
	if resp.Data.Connections[0].ID != "conn-1234" {
		t.Errorf("expected conn-1234, got %s", resp.Data.Connections[0].ID)
	}
	if resp.Data.Connections[0].Metadata.SourceIP != "192.168.1.108" {
		t.Errorf("expected sourceIP 192.168.1.108, got %s", resp.Data.Connections[0].Metadata.SourceIP)
	}

	// 2. Test DELETE /connections/{id}
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/connections/{id}", HandleCloseConnection(nil)).Methods(http.MethodDelete)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/connections/conn-1234", nil)
	delRR := httptest.NewRecorder()
	router.ServeHTTP(delRR, delReq)

	if delRR.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for delete, got %d: %s", delRR.Code, delRR.Body.String())
	}

	// 3. Test DELETE /connections (all)
	delAllReq := httptest.NewRequest(http.MethodDelete, "/api/v1/connections", nil)
	delAllRR := httptest.NewRecorder()
	delAllHandler := HandleCloseAllConnections(nil)
	delAllHandler.ServeHTTP(delAllRR, delAllReq)

	if delAllRR.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for delete all, got %d: %s", delAllRR.Code, delAllRR.Body.String())
	}
}
