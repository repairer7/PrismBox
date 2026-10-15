package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/prismbox/prismbox/internal/config"
	"github.com/prismbox/prismbox/internal/monitor"
)

type mockValidator struct {
	err error
}

func (m *mockValidator) ValidateConfigData(data []byte) error {
	return m.err
}

func setupTestStore(t *testing.T) (*config.Store, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "prismbox-api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	store, err := config.NewStore(tempDir)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		t.Fatalf("failed to create store: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}
	return store, cleanup
}

func TestResponseHelpers(t *testing.T) {
	// 1. writeJSON
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusAccepted, map[string]string{"foo": "bar"})
	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("expected application/json header, got %s", ct)
	}

	// 2. writeError
	rr = httptest.NewRecorder()
	writeError(rr, http.StatusBadRequest, "sample error")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var errResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp["success"] != false || errResp["error"] != "sample error" {
		t.Errorf("unexpected error payload: %+v", errResp)
	}

	// 3. writeSuccess
	rr = httptest.NewRecorder()
	writeSuccess(rr, map[string]string{"result": "ok"})
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var succResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &succResp); err != nil {
		t.Fatalf("failed to unmarshal success response: %v", err)
	}
	if succResp["success"] != true {
		t.Errorf("expected success: true, got %+v", succResp)
	}

	// 4. getIDParam
	req := httptest.NewRequest(http.MethodGet, "/items/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-id-1"})
	if id := getIDParam(req); id != "test-id-1" {
		t.Errorf("expected test-id-1, got %s", id)
	}

	reqFallback := httptest.NewRequest(http.MethodGet, "/items?id=query-id", nil)
	if id := getIDParam(reqFallback); id != "query-id" {
		t.Errorf("expected query-id, got %s", id)
	}
}

func TestDashboardStats(t *testing.T) {
	mon := monitor.NewMonitor()
	defer mon.Stop()

	handler := HandleDashboardStats(mon, nil)

	// GET stats
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp DashboardResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal dashboard response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success: true, got %v", resp.Success)
	}
	if resp.Data.KernelStatus == "" {
		t.Error("expected non-empty kernel_status")
	}

	// PUT stats (acknowledged)
	reqPut := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/stats", nil)
	rrPut := httptest.NewRecorder()
	handler(rrPut, reqPut)

	if rrPut.Code != http.StatusOK {
		t.Fatalf("expected 200 on PUT, got %d", rrPut.Code)
	}
}

func TestInboundsAPI(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// GET inbounds
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbounds", nil)
	rr := httptest.NewRecorder()
	HandleGetInbounds(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// POST inbound (bad request)
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/inbounds", bytes.NewBufferString(`invalid-json`))
	rrBad := httptest.NewRecorder()
	HandleCreateInbound(store)(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json, got %d", rrBad.Code)
	}

	// POST inbound (valid)
	body := `{"tag":"tun-test","type":"tun","enabled":true,"interface_name":"tun9","mtu":1500}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/inbounds", bytes.NewBufferString(body))
	rr = httptest.NewRecorder()
	HandleCreateInbound(store)(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var created config.Inbound
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to parse created inbound: %v", err)
	}
	if created.ID == "" || created.Tag != "tun-test" {
		t.Errorf("created inbound mismatch: %+v", created)
	}

	// PUT inbound
	updateBody := `{"tag":"tun-updated","type":"tun","enabled":true,"interface_name":"tun9","mtu":9000}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/inbounds/"+created.ID, bytes.NewBufferString(updateBody))
	req = mux.SetURLVars(req, map[string]string{"id": created.ID})
	rr = httptest.NewRecorder()
	HandleUpdateInbound(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d: %s", rr.Code, rr.Body.String())
	}

	// DELETE inbound
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/inbounds/"+created.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": created.ID})
	rr = httptest.NewRecorder()
	HandleDeleteInbound(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d: %s", rr.Code, rr.Body.String())
	}

	// DELETE inbound not found
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/inbounds/non-existent-id", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "non-existent-id"})
	rr = httptest.NewRecorder()
	HandleDeleteInbound(store)(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing inbound, got %d", rr.Code)
	}
}

func TestOutboundsAndGroupsAPI(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// POST outbound
	body := `{"tag":"vless-node","type":"vless","server":"127.0.0.1","server_port":443,"enabled":true,"settings":{"uuid":"test-uuid"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/outbounds", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	HandleCreateOutbound(store)(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var created config.Outbound
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to parse created outbound: %v", err)
	}

	// GET outbounds
	req = httptest.NewRequest(http.MethodGet, "/api/v1/outbounds", nil)
	rr = httptest.NewRecorder()
	HandleGetOutbounds(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// PUT outbound
	putBody := `{"tag":"vless-renamed","type":"vless","server":"127.0.0.1","server_port":443,"enabled":true}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/outbounds/"+created.ID, bytes.NewBufferString(putBody))
	req = mux.SetURLVars(req, map[string]string{"id": created.ID})
	rr = httptest.NewRecorder()
	HandleUpdateOutbound(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on outbound update, got %d: %s", rr.Code, rr.Body.String())
	}

	// POST import-links
	linksBody := `{"links":"vless://11111111-2222-3333-4444-555555555555@example.com:443?security=tls#ImportedNode\nhysteria2://secret@example.org:8443#Hy2Node"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/outbounds/import-links", bytes.NewBufferString(linksBody))
	rr = httptest.NewRecorder()
	HandleImportLinks(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for import-links, got %d: %s", rr.Code, rr.Body.String())
	}

	// Speed test
	speedReqBody := `{"ids":["` + created.ID + `"],"timeout":1}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/outbounds/speedtest", bytes.NewBufferString(speedReqBody))
	rr = httptest.NewRecorder()
	HandleSpeedTest(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for speedtest, got %d", rr.Code)
	}

	// POST group
	grpBody := `{"tag":"my-selector","type":"selector","outbounds":["vless-renamed","direct"]}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/groups", bytes.NewBufferString(grpBody))
	rr = httptest.NewRecorder()
	HandleCreateGroup(store)(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for group, got %d: %s", rr.Code, rr.Body.String())
	}

	var grpCreated config.Group
	if err := json.Unmarshal(rr.Body.Bytes(), &grpCreated); err != nil {
		t.Fatalf("failed to parse created group: %v", err)
	}

	// GET groups
	req = httptest.NewRequest(http.MethodGet, "/api/v1/groups", nil)
	rr = httptest.NewRecorder()
	HandleGetGroups(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for get groups, got %d", rr.Code)
	}

	// PUT group update
	grpUpdate := `{"tag":"my-selector-updated","type":"selector","outbounds":["vless-renamed","direct"]}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/groups/"+grpCreated.ID, bytes.NewBufferString(grpUpdate))
	req = mux.SetURLVars(req, map[string]string{"id": grpCreated.ID})
	rr = httptest.NewRecorder()
	HandleUpdateGroup(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on group update, got %d: %s", rr.Code, rr.Body.String())
	}

	// PUT /groups/{id}/select
	selectBody := `{"selected":"direct"}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/groups/"+grpCreated.ID+"/select", bytes.NewBufferString(selectBody))
	req = mux.SetURLVars(req, map[string]string{"id": grpCreated.ID})
	rr = httptest.NewRecorder()
	HandleSelectGroupOutbound(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on group select, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify that "direct" was selected and moved to index 0
	groups := store.GetGroups()
	if len(groups) == 0 || groups[0].Outbounds[0] != "direct" {
		t.Errorf("expected selected outbound 'direct' at index 0, got %+v", groups)
	}

	// DELETE group
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/groups/"+grpCreated.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": grpCreated.ID})
	rr = httptest.NewRecorder()
	HandleDeleteGroup(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete group, got %d", rr.Code)
	}

	// DELETE outbound
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/outbounds/"+created.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": created.ID})
	rr = httptest.NewRecorder()
	HandleDeleteOutbound(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete outbound, got %d", rr.Code)
	}
}

func TestRoutingAndRulesAPI(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// GET routing
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routing", nil)
	rr := httptest.NewRecorder()
	HandleGetRouting(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for routing, got %d", rr.Code)
	}

	// PUT routing
	routingBody := `{"sniff":true,"sniff_override_destination":true,"domain_strategy":"prefer_ipv4","rules":[],"rule_sets":[]}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/routing", bytes.NewBufferString(routingBody))
	rr = httptest.NewRecorder()
	HandleUpdateRouting(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for update routing, got %d", rr.Code)
	}

	// POST rule
	ruleBody := `{"enabled":true,"domain":["example.com"],"outbound":"direct"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/routing/rules", bytes.NewBufferString(ruleBody))
	rr = httptest.NewRecorder()
	HandleCreateRule(store)(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create rule, got %d: %s", rr.Code, rr.Body.String())
	}

	var createdRule config.Rule
	if err := json.Unmarshal(rr.Body.Bytes(), &createdRule); err != nil {
		t.Fatalf("failed to parse rule: %v", err)
	}

	// GET rules
	req = httptest.NewRequest(http.MethodGet, "/api/v1/routing/rules", nil)
	rr = httptest.NewRecorder()
	HandleGetRules(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for get rules, got %d", rr.Code)
	}

	// PUT rule
	ruleUpdateBody := `{"enabled":false,"domain":["example.com","test.org"],"outbound":"direct"}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/routing/rules/"+createdRule.ID, bytes.NewBufferString(ruleUpdateBody))
	req = mux.SetURLVars(req, map[string]string{"id": createdRule.ID})
	rr = httptest.NewRecorder()
	HandleUpdateRule(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on rule update, got %d: %s", rr.Code, rr.Body.String())
	}

	// DELETE rule
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/routing/rules/"+createdRule.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": createdRule.ID})
	rr = httptest.NewRecorder()
	HandleDeleteRule(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on rule delete, got %d: %s", rr.Code, rr.Body.String())
	}

	// POST rule set
	rsBody := `{"tag":"geosite-apple","type":"remote","format":"binary","url":"https://example.com/geosite-apple.srs"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/routing/rulesets", bytes.NewBufferString(rsBody))
	rr = httptest.NewRecorder()
	HandleCreateRuleSet(store)(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create rule set, got %d: %s", rr.Code, rr.Body.String())
	}

	var createdRS config.RuleSet
	if err := json.Unmarshal(rr.Body.Bytes(), &createdRS); err != nil {
		t.Fatalf("failed to parse rule set: %v", err)
	}

	// GET rule sets
	req = httptest.NewRequest(http.MethodGet, "/api/v1/routing/rulesets", nil)
	rr = httptest.NewRecorder()
	HandleGetRuleSets(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for get rule sets, got %d", rr.Code)
	}

	// DELETE rule set
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/routing/rulesets/"+createdRS.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": createdRS.ID})
	rr = httptest.NewRecorder()
	HandleDeleteRuleSet(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete rule set, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDNSAPI(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// GET DNS
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dns", nil)
	rr := httptest.NewRecorder()
	HandleGetDNS(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for DNS, got %d", rr.Code)
	}

	// PUT DNS (enabled=false)
	dnsBody := `{"enabled":false}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/dns", bytes.NewBufferString(dnsBody))
	rr = httptest.NewRecorder()
	HandleUpdateDNS(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on DNS update, got %d: %s", rr.Code, rr.Body.String())
	}

	var updatedDNS config.DNSConfig
	if err := json.Unmarshal(rr.Body.Bytes(), &updatedDNS); err != nil {
		t.Fatalf("failed to parse updated DNS: %v", err)
	}
	if updatedDNS.Enabled {
		t.Errorf("expected DNS enabled to be false")
	}
}

func TestConfigExportValidateImportRaw(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// 1. Export config
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/export", nil)
	rr := httptest.NewRecorder()
	HandleExportConfig(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on export, got %d: %s", rr.Code, rr.Body.String())
	}

	// 2. Validate config
	req = httptest.NewRequest(http.MethodPost, "/api/v1/config/validate", nil)
	rr = httptest.NewRecorder()
	HandleValidateConfig(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on validate, got %d: %s", rr.Code, rr.Body.String())
	}

	var valResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &valResp); err != nil {
		t.Fatalf("failed to parse validate response: %v", err)
	}
	if valResp["valid"] != true {
		t.Errorf("expected valid: true, got %+v", valResp)
	}

	// 2b. Validate with failing validator
	failValidator := &mockValidator{err: fmt.Errorf("sing-box check failed: bad inbound port")}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/config/validate", nil)
	rr = httptest.NewRecorder()
	HandleValidateConfig(store, failValidator)(rr, req)

	var failResp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &failResp)
	if failResp["valid"] != false {
		t.Errorf("expected valid: false on check error, got %+v", failResp)
	}

	// 3. GET /config/raw
	req = httptest.NewRequest(http.MethodGet, "/api/v1/config/raw", nil)
	rr = httptest.NewRecorder()
	HandleGetRawConfig(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on get raw config, got %d", rr.Code)
	}

	// 4. PUT /config/raw
	rawCfg := store.GetConfig()
	rawCfg.Routing.DomainStrategy = "ipv4_only"
	rawBytes, _ := json.Marshal(rawCfg)

	req = httptest.NewRequest(http.MethodPut, "/api/v1/config/raw", bytes.NewBuffer(rawBytes))
	rr = httptest.NewRecorder()
	HandleSetRawConfig(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on set raw config, got %d", rr.Code)
	}
	if store.GetConfig().Routing.DomainStrategy != "ipv4_only" {
		t.Errorf("raw config update failed to persist")
	}

	// 5. POST /config/import
	nativeSingBoxJSON := `{
		"inbounds": [
			{"type": "mixed", "tag": "mixed-in-test", "listen": "127.0.0.1", "listen_port": 1234}
		],
		"outbounds": [
			{"type": "direct", "tag": "direct-out"},
			{"type": "selector", "tag": "proxy-select", "outbounds": ["direct-out"]}
		]
	}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/config/import", bytes.NewBufferString(nativeSingBoxJSON))
	rr = httptest.NewRecorder()
	HandleImportConfig(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on import config, got %d: %s", rr.Code, rr.Body.String())
	}

	importedStoreCfg := store.GetConfig()
	if len(importedStoreCfg.Inbounds) == 0 || importedStoreCfg.Inbounds[0].Tag != "mixed-in-test" {
		t.Errorf("imported inbounds mismatch: %+v", importedStoreCfg.Inbounds)
	}
	if len(importedStoreCfg.Groups) == 0 || importedStoreCfg.Groups[0].Tag != "proxy-select" {
		t.Errorf("imported groups mismatch: %+v", importedStoreCfg.Groups)
	}
}
