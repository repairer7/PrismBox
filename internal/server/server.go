package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prismbox/prismbox/internal/api"
	"github.com/prismbox/prismbox/internal/config"
	"github.com/prismbox/prismbox/internal/kernel"
	"github.com/prismbox/prismbox/internal/monitor"
	"github.com/prismbox/prismbox/web"
)

// Server encapsulates the HTTP server, routing, and background WebSocket hub.
type Server struct {
	httpServer *http.Server
	hub        *WSHub
	store      *config.Store
	manager    *kernel.Manager
	monitor    *monitor.Monitor
}

// New creates and configures a new Server instance.
func New(listenAddr string, store *config.Store, mgr *kernel.Manager, mon *monitor.Monitor) *Server {
	hub := NewWSHub(mon, mgr)

	s := &Server{
		hub:     hub,
		store:   store,
		manager: mgr,
		monitor: mon,
	}

	router := mux.NewRouter()

	// Global CORS middleware
	router.Use(corsMiddleware)

	// WebSocket endpoint
	router.HandleFunc("/ws", hub.HandleWS).Methods(http.MethodGet)

	// API subrouter: /api/v1/...
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	s.registerAPIRoutes(apiRouter)

	// Embedded WebUI static files and SPA fallback
	spaFS := web.GetFileSystem()
	router.PathPrefix("/").Handler(&spaHandler{fs: spaFS})

	s.httpServer = &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Start begins serving HTTP traffic and starts the WebSocket hub.
func (s *Server) Start() error {
	s.hub.Start()
	log.Printf("Starting HTTP server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully terminates the HTTP server and WebSocket hub.
func (s *Server) Shutdown(ctx context.Context) error {
	s.hub.Stop()
	return s.httpServer.Shutdown(ctx)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) registerAPIRoutes(r *mux.Router) {
	store := s.store
	mgr := s.manager
	mon := s.monitor

	// 1. Dashboard stats
	r.HandleFunc("/dashboard/stats", api.HandleDashboardStats(mon, mgr)).Methods(http.MethodGet, http.MethodPut, http.MethodOptions)

	// 2. Inbounds CRUD
	r.HandleFunc("/inbounds", api.HandleGetInbounds(store)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/inbounds", api.HandleCreateInbound(store)).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/inbounds/{id}", api.HandleUpdateInbound(store)).Methods(http.MethodPut, http.MethodOptions)
	r.HandleFunc("/inbounds/{id}", api.HandleDeleteInbound(store)).Methods(http.MethodDelete, http.MethodOptions)

	// 3. Outbounds CRUD, Subscription, SpeedTest & Groups
	r.HandleFunc("/outbounds", api.HandleGetOutbounds(store)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/outbounds", api.HandleCreateOutbound(store)).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/outbounds/import-links", api.HandleImportLinks(store)).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/outbounds/subscribe", api.HandleSubscribe(store)).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/outbounds/speedtest", api.HandleSpeedTest(store)).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/outbounds/{id}", api.HandleUpdateOutbound(store)).Methods(http.MethodPut, http.MethodOptions)
	r.HandleFunc("/outbounds/{id}", api.HandleDeleteOutbound(store)).Methods(http.MethodDelete, http.MethodOptions)

	r.HandleFunc("/groups", api.HandleGetGroups(store)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/groups", api.HandleCreateGroup(store)).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/groups/{id}", api.HandleUpdateGroup(store)).Methods(http.MethodPut, http.MethodOptions)
	r.HandleFunc("/groups/{id}", api.HandleDeleteGroup(store)).Methods(http.MethodDelete, http.MethodOptions)
	r.HandleFunc("/groups/{id}/select", api.HandleSelectGroupOutbound(store)).Methods(http.MethodPut, http.MethodPost, http.MethodOptions)

	// 4. Routing & RuleSets
	r.HandleFunc("/routing", api.HandleGetRouting(store)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/routing", api.HandleUpdateRouting(store)).Methods(http.MethodPut, http.MethodOptions)
	r.HandleFunc("/routing/rules", api.HandleGetRules(store)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/routing/rules", api.HandleCreateRule(store)).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/routing/rules/{id}", api.HandleUpdateRule(store)).Methods(http.MethodPut, http.MethodOptions)
	r.HandleFunc("/routing/rules/{id}", api.HandleDeleteRule(store)).Methods(http.MethodDelete, http.MethodOptions)
	r.HandleFunc("/routing/rulesets", api.HandleGetRuleSets(store)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/routing/rulesets", api.HandleCreateRuleSet(store)).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/routing/rulesets/{id}", api.HandleUpdateRuleSet(store)).Methods(http.MethodPut, http.MethodOptions)
	r.HandleFunc("/routing/rulesets/{id}", api.HandleDeleteRuleSet(store)).Methods(http.MethodDelete, http.MethodOptions)

	// 5. Live Connections (Inspection & termination via sing-box clash_api)
	r.HandleFunc("/connections", api.HandleGetConnections(mgr)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/connections/{id}", api.HandleCloseConnection(mgr)).Methods(http.MethodDelete, http.MethodOptions)
	r.HandleFunc("/connections", api.HandleCloseAllConnections(mgr)).Methods(http.MethodDelete, http.MethodOptions)

	// 6. DNS configuration
	r.HandleFunc("/dns", api.HandleGetDNS(store)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/dns", api.HandleUpdateDNS(store)).Methods(http.MethodPut, http.MethodOptions)

	// 7. Sing-Box Config export, validation, import & raw
	r.HandleFunc("/config", api.HandleGetRawConfig(store)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/config", api.HandleImportConfig(store)).Methods(http.MethodPut, http.MethodPost, http.MethodOptions)
	r.HandleFunc("/config/export", api.HandleExportConfig(store)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/config/validate", api.HandleValidateConfig(store, mgr)).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/config/import", api.HandleImportConfig(store)).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/config/raw", api.HandleGetRawConfig(store)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/config/raw", api.HandleSetRawConfig(store)).Methods(http.MethodPut, http.MethodOptions)

	// 7. Kernel control & status
	r.HandleFunc("/kernel/start", func(w http.ResponseWriter, r *http.Request) {
		if err := mgr.Start(); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to start sing-box: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"status":  mgr.Status(),
			"pid":     mgr.GetPID(),
			"message": "sing-box kernel started",
		})
	}).Methods(http.MethodPost, http.MethodOptions)

	r.HandleFunc("/kernel/stop", func(w http.ResponseWriter, r *http.Request) {
		if err := mgr.Stop(); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to stop sing-box: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"status":  mgr.Status(),
			"message": "sing-box kernel stopped",
		})
	}).Methods(http.MethodPost, http.MethodOptions)

	r.HandleFunc("/kernel/restart", func(w http.ResponseWriter, r *http.Request) {
		if err := mgr.Restart(); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to restart sing-box: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"status":  mgr.Status(),
			"pid":     mgr.GetPID(),
			"message": "sing-box kernel restarted",
		})
	}).Methods(http.MethodPost, http.MethodOptions)

	r.HandleFunc("/kernel/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":     true,
			"status":      mgr.Status(),
			"running":     mgr.IsRunning(),
			"pid":         mgr.GetPID(),
			"version":     mgr.GetVersion(),
			"uptime":      uint64(mgr.GetUptime().Seconds()),
			"last_error":  mgr.GetLastError(),
			"crash_count": mgr.GetCrashCount(),
		})
	}).Methods(http.MethodGet, http.MethodOptions)

	r.HandleFunc("/kernel/logs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"logs":    mgr.GetLogs(100),
		})
	}).Methods(http.MethodGet, http.MethodOptions)
}

func (s *Server) handleImportConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	if len(strings.TrimSpace(string(body))) == 0 {
		writeError(w, http.StatusBadRequest, "import payload cannot be empty")
		return
	}

	var newConfig config.PrismBoxConfig
	if err := json.Unmarshal(body, &newConfig); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON format: "+err.Error())
		return
	}

	// Validate transformation to sing-box config
	if _, err := config.GenerateSingBoxConfigFromModel(newConfig); err != nil {
		writeError(w, http.StatusBadRequest, "invalid sing-box configuration schema: "+err.Error())
		return
	}

	if err := s.store.SetConfig(newConfig); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save configuration: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Configuration successfully imported",
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// spaHandler serves embedded static files and falls back to index.html for SPA routes.
type spaHandler struct {
	fs http.FileSystem
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cleanPath := filepath.Clean(r.URL.Path)
	if cleanPath == "/" || cleanPath == "." {
		cleanPath = "/index.html"
	}

	trimmed := strings.TrimPrefix(cleanPath, "/")

	// Attempt to open the requested static asset
	f, err := h.fs.Open(trimmed)
	if err == nil {
		defer f.Close()
		stat, err := f.Stat()
		if err == nil && !stat.IsDir() {
			http.FileServer(h.fs).ServeHTTP(w, r)
			return
		}
	}

	// Fallback to index.html for single-page app routing
	indexFile, err := h.fs.Open("index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer indexFile.Close()

	content, err := io.ReadAll(indexFile)
	if err != nil {
		http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
