package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prismbox/prismbox/internal/config"
	"github.com/prismbox/prismbox/internal/subscription"
)

// ==================== OUTBOUNDS ====================

// HandleGetOutbounds returns an HTTP handler that lists all outbounds.
func HandleGetOutbounds(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		outbounds := store.GetOutbounds()
		writeJSON(w, http.StatusOK, outbounds)
	}
}

// HandleCreateOutbound returns an HTTP handler that creates a new outbound.
func HandleCreateOutbound(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var outbound config.Outbound
		if err := json.NewDecoder(r.Body).Decode(&outbound); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if outbound.Type == "" {
			writeError(w, http.StatusBadRequest, "outbound type is required")
			return
		}
		if outbound.Tag == "" {
			outbound.Tag = fmt.Sprintf("%s-%s", outbound.Type, outbound.Server)
		}

		created, err := store.AddOutbound(outbound)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to add outbound: "+err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, created)
	}
}

// HandleUpdateOutbound returns an HTTP handler that updates an outbound by ID.
func HandleUpdateOutbound(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "outbound id is required")
			return
		}

		var outbound config.Outbound
		if err := json.NewDecoder(r.Body).Decode(&outbound); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if err := store.UpdateOutbound(id, outbound); err != nil {
			if errors.Is(err, config.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update outbound: "+err.Error())
			return
		}

		outbound.ID = id
		writeJSON(w, http.StatusOK, outbound)
	}
}

// HandleDeleteOutbound returns an HTTP handler that deletes an outbound by ID.
func HandleDeleteOutbound(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "outbound id is required")
			return
		}

		if err := store.DeleteOutbound(id); err != nil {
			if errors.Is(err, config.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to delete outbound: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"id":      id,
		})
	}
}

// ImportLinksRequest defines the payload for importing raw share links.
type ImportLinksRequest struct {
	Links string `json:"links"`
}

// HandleImportLinks parses multiple share links and saves the valid nodes to the store.
func HandleImportLinks(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ImportLinksRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		req.Links = strings.TrimSpace(req.Links)
		if req.Links == "" {
			writeError(w, http.StatusBadRequest, "links cannot be empty")
			return
		}

		var parsed []config.Outbound

		// If input contains URI schemes, parse line by line
		if strings.Contains(req.Links, "://") {
			scanner := bufio.NewScanner(strings.NewReader(req.Links))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
					continue
				}
				outbound, err := subscription.ParseLink(line)
				if err == nil {
					parsed = append(parsed, outbound)
				}
			}
		} else {
			// Try decoding base64 link list
			var err error
			parsed, err = subscription.ParseBase64([]byte(req.Links))
			if err != nil {
				writeError(w, http.StatusBadRequest, "failed to parse base64 links: "+err.Error())
				return
			}
		}

		if len(parsed) == 0 {
			writeError(w, http.StatusBadRequest, "no valid proxy links found in request")
			return
		}

		added := make([]config.Outbound, 0, len(parsed))
		for _, node := range parsed {
			created, err := store.AddOutbound(node)
			if err != nil {
				continue
			}
			added = append(added, created)
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":   true,
			"count":     len(added),
			"outbounds": added,
		})
	}
}

// SubscribeRequest defines the payload for subscription import.
type SubscribeRequest struct {
	URL string `json:"url"`
	Tag string `json:"tag,omitempty"`
}

// HandleSubscribe imports outbounds from a remote subscription URL.
func HandleSubscribe(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SubscribeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		req.URL = strings.TrimSpace(req.URL)
		if req.URL == "" {
			writeError(w, http.StatusBadRequest, "url is required")
			return
		}

		outbounds, err := subscription.ParseSubscription(req.URL)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to parse subscription: "+err.Error())
			return
		}

		added := make([]config.Outbound, 0, len(outbounds))
		for _, out := range outbounds {
			if req.Tag != "" {
				out.SubscriptionTag = req.Tag
			}
			out.SubscriptionURL = req.URL
			created, err := store.AddOutbound(out)
			if err != nil {
				continue
			}
			added = append(added, created)
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":   true,
			"count":     len(added),
			"outbounds": added,
		})
	}
}

// SpeedTestRequest defines options for speed testing outbounds.
type SpeedTestRequest struct {
	IDs     []string `json:"ids"`
	URL     string   `json:"url,omitempty"`
	Timeout int      `json:"timeout,omitempty"`
}

// SpeedTestResult represents latency measurement for an outbound.
type SpeedTestResult struct {
	ID      string `json:"id"`
	Tag     string `json:"tag"`
	Latency int    `json:"latency"` // ms, -1 if unreachable
}

// HandleSpeedTest performs a concurrent TCP latency test for specified outbounds.
func HandleSpeedTest(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SpeedTestRequest
		if r.ContentLength > 0 {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}

		timeoutSec := req.Timeout
		if timeoutSec <= 0 {
			timeoutSec = 5
		}
		dialTimeout := time.Duration(timeoutSec) * time.Second

		allOutbounds := store.GetOutbounds()
		targetMap := make(map[string]bool)
		for _, id := range req.IDs {
			targetMap[id] = true
		}

		var targets []config.Outbound
		for _, out := range allOutbounds {
			if len(req.IDs) == 0 || targetMap[out.ID] {
				if out.Server != "" && out.Port > 0 {
					targets = append(targets, out)
				}
			}
		}

		results := make([]SpeedTestResult, len(targets))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 16) // Max 16 concurrent TCP handshakes

		for i, out := range targets {
			wg.Add(1)
			go func(idx int, outbound config.Outbound) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				addr := net.JoinHostPort(outbound.Server, strconv.Itoa(outbound.Port))
				start := time.Now()
				conn, err := net.DialTimeout("tcp", addr, dialTimeout)

				latency := -1
				if err == nil {
					latency = int(time.Since(start).Milliseconds())
					_ = conn.Close()
				}

				_ = store.UpdateOutboundLatency(outbound.ID, latency)
				results[idx] = SpeedTestResult{
					ID:      outbound.ID,
					Tag:     outbound.Tag,
					Latency: latency,
				}
			}(i, out)
		}

		wg.Wait()

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"results": results,
		})
	}
}

// ==================== GROUPS ====================

// HandleGetGroups returns an HTTP handler that lists all groups.
func HandleGetGroups(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groups := store.GetGroups()
		writeJSON(w, http.StatusOK, groups)
	}
}

// HandleCreateGroup returns an HTTP handler that creates a new outbound group.
func HandleCreateGroup(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var group config.Group
		if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if group.Tag == "" {
			writeError(w, http.StatusBadRequest, "group tag is required")
			return
		}
		if group.Type == "" {
			group.Type = "selector"
		}

		created, err := store.AddGroup(group)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to add group: "+err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, created)
	}
}

// HandleUpdateGroup returns an HTTP handler that updates a group by ID.
func HandleUpdateGroup(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "group id is required")
			return
		}

		var group config.Group
		if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if err := store.UpdateGroup(id, group); err != nil {
			if errors.Is(err, config.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update group: "+err.Error())
			return
		}

		group.ID = id
		writeJSON(w, http.StatusOK, group)
	}
}

// HandleDeleteGroup returns an HTTP handler that deletes a group by ID.
func HandleDeleteGroup(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "group id is required")
			return
		}

		if err := store.DeleteGroup(id); err != nil {
			if errors.Is(err, config.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to delete group: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"id":      id,
		})
	}
}

// SelectGroupRequest defines the body for choosing an active outbound in a group.
type SelectGroupRequest struct {
	Selected string `json:"selected"`
}

// HandleSelectGroupOutbound selects the active outbound in a selector group.
func HandleSelectGroupOutbound(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "group id is required")
			return
		}

		var req SelectGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if req.Selected == "" {
			writeError(w, http.StatusBadRequest, "selected outbound tag is required")
			return
		}

		groups := store.GetGroups()
		found := false
		for i, grp := range groups {
			if grp.ID == id || grp.Tag == id {
				found = true
				idx := -1
				for j, tag := range grp.Outbounds {
					if tag == req.Selected {
						idx = j
						break
					}
				}
				if idx == -1 {
					writeError(w, http.StatusBadRequest, fmt.Sprintf("outbound %s not found in group %s", req.Selected, grp.Tag))
					return
				}
				if idx > 0 {
					outbounds := make([]string, 0, len(grp.Outbounds))
					outbounds = append(outbounds, req.Selected)
					for j, tag := range grp.Outbounds {
						if j != idx {
							outbounds = append(outbounds, tag)
						}
					}
					groups[i].Outbounds = outbounds
					if err := store.SetGroups(groups); err != nil {
						writeError(w, http.StatusInternalServerError, "failed to update group selection: "+err.Error())
						return
					}
				}
				break
			}
		}

		if !found {
			writeError(w, http.StatusNotFound, fmt.Sprintf("group %s not found", id))
			return
		}

		writeSuccess(w, map[string]interface{}{
			"group_id": id,
			"selected": req.Selected,
		})
	}
}
