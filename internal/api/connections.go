package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/prismbox/prismbox/internal/kernel"
)

// ConnectionsResponse wraps the Clash API connections snapshot.
type ConnectionsResponse struct {
	DownloadTotal int64            `json:"downloadTotal"`
	UploadTotal   int64            `json:"uploadTotal"`
	Connections   []ConnectionItem `json:"connections"`
}

// ConnectionItem represents an active connection tracked by sing-box.
type ConnectionItem struct {
	ID          string             `json:"id"`
	Metadata    ConnectionMetadata `json:"metadata"`
	Upload      int64              `json:"upload"`
	Download    int64              `json:"download"`
	Start       string             `json:"start"`
	Chains      []string           `json:"chains"`
	Rule        string             `json:"rule"`
	RulePayload string             `json:"rulePayload"`
}

// ConnectionMetadata holds network, source and destination details.
type ConnectionMetadata struct {
	Network         string `json:"network"`
	Type            string `json:"type"`
	SourceIP        string `json:"sourceIP"`
	SourcePort      string `json:"sourcePort"`
	DestinationIP   string `json:"destinationIP"`
	DestinationPort string `json:"destinationPort"`
	Host            string `json:"host"`
	DNSMode         string `json:"dnsMode"`
	ProcessPath     string `json:"processPath,omitempty"`
}

var connectionsClashEndpoint = "127.0.0.1:9091"

// SetConnectionsClashEndpoint allows overriding the Clash API endpoint for tests.
func SetConnectionsClashEndpoint(endpoint string) {
	connectionsClashEndpoint = endpoint
}

// HandleGetConnections returns the live connections snapshot from sing-box clash_api.
func HandleGetConnections(mgr *kernel.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emptyResp := ConnectionsResponse{
			DownloadTotal: 0,
			UploadTotal:   0,
			Connections:   make([]ConnectionItem, 0),
		}

		if mgr != nil && !mgr.IsRunning() {
			writeSuccess(w, emptyResp)
			return
		}

		endpoint := connectionsClashEndpoint
		if endpoint == "" {
			endpoint = "127.0.0.1:9091"
		}

		apiURL := fmt.Sprintf("http://%s/connections", endpoint)
		client := &http.Client{Timeout: 2 * time.Second}

		resp, err := client.Get(apiURL)
		if err != nil {
			// Kernel may be busy, reloading or clash_api not ready yet. Return graceful empty list.
			writeSuccess(w, emptyResp)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			writeSuccess(w, emptyResp)
			return
		}

		var data ConnectionsResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			writeSuccess(w, emptyResp)
			return
		}

		if data.Connections == nil {
			data.Connections = make([]ConnectionItem, 0)
		}

		writeSuccess(w, data)
	}
}

// HandleCloseConnection terminates a single specific connection by its ID.
func HandleCloseConnection(mgr *kernel.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "Connection ID is required")
			return
		}

		endpoint := connectionsClashEndpoint
		if endpoint == "" {
			endpoint = "127.0.0.1:9091"
		}

		apiURL := fmt.Sprintf("http://%s/connections/%s", endpoint, url.PathEscape(id))
		req, err := http.NewRequest(http.MethodDelete, apiURL, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			writeError(w, http.StatusBadGateway, fmt.Sprintf("Failed to contact sing-box controller: %v", err))
			return
		}
		defer resp.Body.Close()

		writeSuccess(w, map[string]interface{}{
			"closed": true,
			"id":     id,
		})
	}
}

// HandleCloseAllConnections terminates all active connections in sing-box.
func HandleCloseAllConnections(mgr *kernel.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		endpoint := connectionsClashEndpoint
		if endpoint == "" {
			endpoint = "127.0.0.1:9091"
		}

		apiURL := fmt.Sprintf("http://%s/connections", endpoint)
		req, err := http.NewRequest(http.MethodDelete, apiURL, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			writeError(w, http.StatusBadGateway, fmt.Sprintf("Failed to contact sing-box controller: %v", err))
			return
		}
		defer resp.Body.Close()

		writeSuccess(w, map[string]interface{}{
			"closedAll": true,
		})
	}
}
