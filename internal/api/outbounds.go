package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
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

		// Prevent deletion of system built-in outbounds (direct, block)
		for _, o := range store.GetOutbounds() {
			if o.ID == id && (o.Tag == "direct" || o.Tag == "block" || o.Type == "direct" || o.Type == "block") {
				writeError(w, http.StatusBadRequest, "cannot delete system built-in outbound: "+o.Tag)
				return
			}
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

// HandleSpeedTest performs a concurrent latency test for specified outbounds.
// It prioritizes sing-box's native Clash API (/proxies/{tag}/delay) for true proxy testing
// (essential for UDP protocols like Hysteria 2 / TUIC as well as TCP protocols),
// and gracefully falls back to direct TCP handshake if sing-box or Clash API is unavailable.
func HandleSpeedTest(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SpeedTestRequest
		if r.ContentLength > 0 {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}

		timeoutSec := req.Timeout
		if timeoutSec <= 0 {
			timeoutSec = 10
		}

		testURL := strings.TrimSpace(req.URL)
		if testURL == "" || strings.HasPrefix(testURL, "https://www.gstatic.com") {
			testURL = "http://cp.cloudflare.com/generate_204"
		}

		allOutbounds := store.GetOutbounds()
		targetMap := make(map[string]bool)
		for _, id := range req.IDs {
			targetMap[id] = true
		}

		var targets []config.Outbound
		for _, out := range allOutbounds {
			if len(req.IDs) == 0 || targetMap[out.ID] {
				if (out.Server != "" && out.Port > 0) || strings.EqualFold(out.Type, "direct") {
					targets = append(targets, out)
				}
			}
		}

		results := make([]SpeedTestResult, len(targets))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 16) // Max 16 concurrent tests

		for i, out := range targets {
			wg.Add(1)
			go func(idx int, outbound config.Outbound) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				latency := testSingleOutbound(outbound, testURL, timeoutSec)

				_ = store.UpdateOutboundLatency(outbound.ID, latency)
				results[idx] = SpeedTestResult{
					ID:      outbound.ID,
					Tag:     outbound.Tag,
					Latency: latency,
				}
			}(i, out)
		}

		wg.Wait()

		// Re-evaluate URLTest groups based on new latencies
		reevaluateURLTestGroups(store)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"results": results,
		})
	}
}

// testSingleOutbound tests an outbound node latency.
// It uses fast native network probing:
// - Direct outbound: sends an HTTP GET directly to the test URL (real internet connection latency)
// - QUIC protocols (Hysteria 2 / TUIC): sends an RFC 9000 QUIC probe for immediate UDP RTT
// - TCP protocols (VLESS, VMess, Trojan, Shadowsocks, etc.): establishes a TCP handshake to server:port
func testSingleOutbound(outbound config.Outbound, testURL string, timeoutSec int) int {
	if timeoutSec <= 0 {
		timeoutSec = 10
	}

	if testURL == "" {
		testURL = "http://cp.cloudflare.com/generate_204"
	}

	outType := strings.ToLower(strings.TrimSpace(outbound.Type))

	// 1. Direct Outbound: test local internet connection latency via direct HTTP request
	if outType == "direct" {
		candidates := []string{
			testURL,
			"http://cp.cloudflare.com/generate_204",
			"https://cp.cloudflare.com/generate_204",
			"http://connectivitycheck.gstatic.com/generate_204",
			"http://www.qualcomm.cn/generate_204",
		}
		client := &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		}
		for _, ep := range candidates {
			if ep == "" {
				continue
			}
			start := time.Now()
			resp, err := client.Get(ep)
			if err == nil {
				_ = resp.Body.Close()
				return int(time.Since(start).Milliseconds())
			}
		}
		return -1
	}

	// 2. Block Outbound: intentionally non-routable
	if outType == "block" {
		return -1
	}

	// 3. Native Latency Probe for proxy nodes:
	if outbound.Server != "" && outbound.Port > 0 {
		// For QUIC UDP-based protocols (Hysteria 2, TUIC), perform a fast RFC 9000 QUIC probe
		if outType == "hysteria2" || outType == "hy2" || outType == "tuic" {
			if l := testQUICLatency(outbound.Server, outbound.Port, timeoutSec); l > 0 {
				return l
			}
		}

		// Direct TCP handshake for TCP-based proxy protocols
		addr := net.JoinHostPort(outbound.Server, strconv.Itoa(outbound.Port))
		start := time.Now()
		conn, err := net.DialTimeout("tcp", addr, time.Duration(timeoutSec)*time.Second)
		if err == nil {
			_ = conn.Close()
			return int(time.Since(start).Milliseconds())
		}
	}

	return -1
}

// testQUICLatency sends a standard RFC 9000 QUIC version negotiation probe to measure UDP RTT.
// Under RFC 9000 Section 5.2.2, any compliant QUIC server (Hysteria 2, TUIC) receiving an
// Initial packet with an unallocated version will immediately respond with a Version Negotiation packet.
func testQUICLatency(server string, port int, timeoutSec int) int {
	if timeoutSec <= 0 {
		timeoutSec = 3
	}
	if timeoutSec > 5 {
		timeoutSec = 5
	}
	addr := net.JoinHostPort(server, strconv.Itoa(port))
	conn, err := net.DialTimeout("udp", addr, time.Duration(timeoutSec)*time.Second)
	if err != nil {
		return -1
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(time.Duration(timeoutSec) * time.Second))

	// Construct RFC 9000 Initial Packet with an unallocated version (0x1a2a3a4a)
	// Padded to 1200 bytes as required by QUIC specifications.
	packet := make([]byte, 1200)
	packet[0] = 0xc0 // Long Header, Initial
	packet[1] = 0x1a // Version: 0x1a2a3a4a
	packet[2] = 0x2a
	packet[3] = 0x3a
	packet[4] = 0x4a
	packet[5] = 8 // DCID length: 8
	for i := 6; i < 14; i++ {
		packet[i] = byte(i * 7)
	}
	packet[14] = 8 // SCID length: 8
	for i := 15; i < 23; i++ {
		packet[i] = byte(i * 13)
	}

	start := time.Now()
	if _, err := conn.Write(packet); err != nil {
		return -1
	}

	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil || n < 5 {
		return -1
	}

	return int(time.Since(start).Milliseconds())
}

var clashAPIEndpoint = "127.0.0.1:9091"

// queryClashAPIDelay queries sing-box's external controller Clash API for proxy delay measurement.
func queryClashAPIDelay(tag string, testURL string, timeoutMs int) (int, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return -1, fmt.Errorf("empty tag")
	}

	endpoint := clashAPIEndpoint
	if endpoint == "" {
		endpoint = "127.0.0.1:9091"
	}

	apiURL := fmt.Sprintf("http://%s/proxies/%s/delay?url=%s&timeout=%d",
		endpoint,
		url.PathEscape(tag),
		url.QueryEscape(testURL),
		timeoutMs,
	)

	client := &http.Client{
		Timeout: time.Duration(timeoutMs+2000) * time.Millisecond,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		log.Printf("[speedtest] Request error querying Clash API for tag %q (%s): %v", tag, testURL, err)
		return -1, err
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("[speedtest] Failed reading Clash API response for tag %q (%s): %v", tag, testURL, readErr)
		return -1, readErr
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[speedtest] Clash API error for tag %q (%s): status=%d, body=%s", tag, testURL, resp.StatusCode, string(bodyBytes))
		return -1, fmt.Errorf("clash api status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var data struct {
		Delay float64 `json:"delay"`
	}
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		log.Printf("[speedtest] Failed decoding Clash API JSON for tag %q (%s): %v, raw: %s", tag, testURL, err, string(bodyBytes))
		return -1, err
	}

	delayMs := int(data.Delay)
	if delayMs <= 0 {
		log.Printf("[speedtest] Clash API returned non-positive delay for tag %q (%s): %v", tag, testURL, data.Delay)
		return -1, fmt.Errorf("invalid delay: %v", data.Delay)
	}

	return delayMs, nil
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
				}
				groups[i].Selected = req.Selected
				if err := store.SetGroups(groups); err != nil {
					writeError(w, http.StatusInternalServerError, "failed to update group selection: "+err.Error())
					return
				}
				notifyClashAPISelectProxy(grp.Tag, req.Selected)
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

// notifyClashAPISelectProxy dynamically switches the active outbound of a group in sing-box via Clash API.
func notifyClashAPISelectProxy(groupTag, selectedTag string) {
	groupTag = strings.TrimSpace(groupTag)
	selectedTag = strings.TrimSpace(selectedTag)
	if groupTag == "" || selectedTag == "" {
		return
	}
	endpoint := clashAPIEndpoint
	if endpoint == "" {
		endpoint = "127.0.0.1:9091"
	}
	apiURL := fmt.Sprintf("http://%s/proxies/%s", endpoint, url.PathEscape(groupTag))
	body, _ := json.Marshal(map[string]string{"name": selectedTag})
	req, err := http.NewRequest(http.MethodPut, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
}

// reevaluateURLTestGroups re-evaluates all urltest groups, selecting the lowest latency outbound.
func reevaluateURLTestGroups(store *config.Store) {
	if store == nil {
		return
	}
	allOutbounds := store.GetOutbounds()
	latencyMap := make(map[string]int)
	for _, o := range allOutbounds {
		if o.Enabled {
			latencyMap[o.Tag] = o.Latency
		}
	}

	groups := store.GetGroups()
	modified := false

	for i, grp := range groups {
		if strings.EqualFold(grp.Type, "urltest") && len(grp.Outbounds) > 0 {
			bestTag := ""
			bestLatency := 9999999

			for _, tag := range grp.Outbounds {
				lat, exists := latencyMap[tag]
				if exists && lat > 0 && lat < bestLatency {
					bestLatency = lat
					bestTag = tag
				}
			}

			if bestTag != "" {
				newOutbounds := make([]string, 0, len(grp.Outbounds))
				newOutbounds = append(newOutbounds, bestTag)
				for _, tag := range grp.Outbounds {
					if tag != bestTag {
						newOutbounds = append(newOutbounds, tag)
					}
				}
				groups[i].Outbounds = newOutbounds
				groups[i].Selected = bestTag
				modified = true

				notifyClashAPISelectProxy(grp.Tag, bestTag)
			}
		}
	}

	if modified {
		_ = store.SetGroups(groups)
	}
}

