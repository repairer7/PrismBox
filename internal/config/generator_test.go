package config

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestGenerateSingBoxConfig_DefaultConfigProducesValidJSON verifies default config generates valid sing-box JSON.
func TestGenerateSingBoxConfig_DefaultConfigProducesValidJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-gen-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	data, err := GenerateSingBoxConfig(store)
	if err != nil {
		t.Fatalf("GenerateSingBoxConfig failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("generated config is not valid JSON: %v", err)
	}

	// Verify required top-level sing-box fields
	if parsed["$schema"] != "https://sing-box.sagernet.org/schema.json" {
		t.Errorf("unexpected schema: %v", parsed["$schema"])
	}

	logSection, ok := parsed["log"].(map[string]interface{})
	if !ok || logSection["level"] != "info" {
		t.Errorf("expected log.level info, got %v", parsed["log"])
	}

	expSection, ok := parsed["experimental"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected experimental section")
	}
	clashAPI, ok := expSection["clash_api"].(map[string]interface{})
	if !ok || clashAPI["external_controller"] != "127.0.0.1:9091" {
		t.Errorf("expected clash_api external_controller 127.0.0.1:9091, got %v", expSection["clash_api"])
	}

	// Verify default outbounds include direct and block
	outbounds, ok := parsed["outbounds"].([]interface{})
	if !ok || len(outbounds) < 2 {
		t.Fatalf("expected at least 2 default outbounds, got %d", len(outbounds))
	}

	tags := make(map[string]bool)
	for _, out := range outbounds {
		outMap := out.(map[string]interface{})
		tags[outMap["tag"].(string)] = true
	}
	for _, expectedTag := range []string{"direct", "block"} {
		if !tags[expectedTag] {
			t.Errorf("missing expected default outbound %s", expectedTag)
		}
	}
}

// TestGenerateSingBoxConfig_DNSDisabledProducesNoDNSKey verifies dns key is omitted when DNS is disabled.
func TestGenerateSingBoxConfig_DNSDisabledProducesNoDNSKey(t *testing.T) {
	cfg := defaultConfig()
	cfg.DNS.Enabled = false

	data, err := GenerateSingBoxConfigFromModel(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if _, exists := parsed["dns"]; exists {
		t.Errorf("dns section should NOT exist when DNS is disabled")
	}

	// Verify route default_domain_resolver is also omitted
	route, ok := parsed["route"].(map[string]interface{})
	if !ok {
		t.Fatalf("route section missing")
	}
	if _, exists := route["default_domain_resolver"]; exists {
		t.Errorf("default_domain_resolver should NOT exist when DNS is disabled")
	}
}

// TestGenerateSingBoxConfig_DNSEnabledProducesProperServersAndRules tests DNS config generation.
func TestGenerateSingBoxConfig_DNSEnabledProducesProperServersAndRules(t *testing.T) {
	cfg := defaultConfig()
	cfg.DNS = DNSConfig{
		Enabled: true,
		Servers: []DNSServer{
			{
				Tag:             "google",
				Address:         "https://dns.google/dns-query",
				AddressResolver: "local",
				Detour:          "proxy",
				Strategy:        "prefer_ipv4",
			},
			{
				Tag:     "local",
				Address: "223.5.5.5",
				Detour:  "direct",
			},
		},
		Rules: []DNSRule{
			{
				RuleSet: []string{"geosite-cn"},
				Server:  "local",
			},
			{
				Domain:       []string{"example.com"},
				DomainSuffix: []string{".org"},
				Server:       "google",
				ClashMode:    "rule",
			},
		},
		Strategy:         "prefer_ipv4",
		DisableCache:     true,
		DisableExpire:    true,
		IndependentCache: true,
	}
	cfg.Routing.DomainStrategy = "prefer_ipv4"

	data, err := GenerateSingBoxConfigFromModel(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	dnsRaw, exists := parsed["dns"]
	if !exists {
		t.Fatalf("dns section should exist when DNS is enabled")
	}

	dnsMap := dnsRaw.(map[string]interface{})
	if dnsMap["strategy"] != "prefer_ipv4" {
		t.Errorf("expected dns strategy prefer_ipv4, got %v", dnsMap["strategy"])
	}
	if dnsMap["disable_cache"] != true {
		t.Errorf("expected disable_cache true")
	}
	if dnsMap["disable_expire"] != true {
		t.Errorf("expected disable_expire true")
	}
	if dnsMap["independent_cache"] != true {
		t.Errorf("expected independent_cache true")
	}

	// Verify DNS Servers
	servers := dnsMap["servers"].([]interface{})
	if len(servers) != 2 {
		t.Fatalf("expected 2 dns servers, got %d", len(servers))
	}
	s0 := servers[0].(map[string]interface{})
	if s0["tag"] != "google" || s0["type"] != "https" || s0["server"] != "dns.google" || s0["path"] != "/dns-query" || s0["detour"] != "proxy" {
		t.Errorf("unexpected dns server 0 content: %+v", s0)
	}
	if s0["domain_resolver"] != "local" {
		t.Errorf("expected domain_resolver 'local', got %v", s0["domain_resolver"])
	}
	if _, hasOld := s0["address_resolver"]; hasOld {
		t.Errorf("deprecated address_resolver should NOT be emitted in sing-box config: %+v", s0)
	}
	s1 := servers[1].(map[string]interface{})
	if _, hasDetour := s1["detour"]; hasDetour {
		t.Errorf("dns server with detour 'direct' should NOT emit detour field: %+v", s1)
	}

	// Verify DNS Rules
	rules := dnsMap["rules"].([]interface{})
	if len(rules) != 2 {
		t.Fatalf("expected 2 dns rules, got %d", len(rules))
	}
	r0 := rules[0].(map[string]interface{})
	if r0["server"] != "local" {
		t.Errorf("unexpected server in rule 0: %v", r0["server"])
	}

	// Verify route has default_domain_resolver and hijack-dns rule
	route := parsed["route"].(map[string]interface{})
	ddr, ok := route["default_domain_resolver"].(map[string]interface{})
	if !ok || ddr["server"] != "google" || ddr["strategy"] != "prefer_ipv4" {
		t.Errorf("unexpected default_domain_resolver: %+v", route["default_domain_resolver"])
	}

	routeRules := route["rules"].([]interface{})
	foundHijack := false
	for _, rr := range routeRules {
		rrMap := rr.(map[string]interface{})
		if rrMap["action"] == "hijack-dns" && rrMap["protocol"] == "dns" {
			foundHijack = true
			break
		}
	}
	if !foundHijack {
		t.Errorf("expected hijack-dns action rule when DNS is enabled")
	}
}

// TestGenerateSingBoxConfig_TProxyInboundGeneratesCorrectFormat tests TProxy inbound options.
func TestGenerateSingBoxConfig_TProxyInboundGeneratesCorrectFormat(t *testing.T) {
	cfg := PrismBoxConfig{
		Inbounds: []Inbound{
			{
				Tag:                      "tproxy-test",
				Type:                     "tproxy",
				Enabled:                  true,
				Listen:                   "::",
				Port:                     12345,
				Network:                  "tcp,udp",
				Sniff:                    true,
				SniffOverrideDestination: true,
			},
		},
		Outbounds: []Outbound{
			{Tag: "direct", Type: "direct", Enabled: true},
		},
	}

	data, err := GenerateSingBoxConfigFromModel(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	inbounds := parsed["inbounds"].([]interface{})
	if len(inbounds) != 1 {
		t.Fatalf("expected 1 inbound, got %d", len(inbounds))
	}

	inb := inbounds[0].(map[string]interface{})
	if inb["type"] != "tproxy" {
		t.Errorf("expected type tproxy, got %v", inb["type"])
	}
	if inb["tag"] != "tproxy-test" {
		t.Errorf("expected tag tproxy-test, got %v", inb["tag"])
	}
	if inb["listen"] != "::" {
		t.Errorf("expected listen ::, got %v", inb["listen"])
	}
	if inb["listen_port"] != float64(12345) {
		t.Errorf("expected listen_port 12345, got %v", inb["listen_port"])
	}
	if _, hasNet := inb["network"]; hasNet {
		t.Errorf("expected network to be omitted for tcp,udp in sing-box 1.14.3, got %v", inb["network"])
	}
}

// TestGenerateSingBoxConfig_OutboundSettingsProperlyMerged verifies outbound fields and settings merging.
func TestGenerateSingBoxConfig_OutboundSettingsProperlyMerged(t *testing.T) {
	cfg := PrismBoxConfig{
		Outbounds: []Outbound{
			{
				Tag:     "node-vless",
				Type:    "vless",
				Enabled: true,
				Server:  "node.example.com",
				Port:    443,
				Settings: map[string]interface{}{
					"uuid": "550e8400-e29b-41d4-a716-446655440000",
					"flow": "xtls-rprx-vision",
					"tls": map[string]interface{}{
						"enabled":     true,
						"server_name": "node.example.com",
					},
					"packet_encoding": "xudp",
				},
			},
		},
	}

	data, err := GenerateSingBoxConfigFromModel(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	outbounds := parsed["outbounds"].([]interface{})
	var vlessOut map[string]interface{}
	for _, o := range outbounds {
		oMap := o.(map[string]interface{})
		if oMap["tag"] == "node-vless" {
			vlessOut = oMap
			break
		}
	}

	if vlessOut == nil {
		t.Fatalf("node-vless outbound not found in generated config")
	}

	// Verify top-level standard fields
	if vlessOut["type"] != "vless" {
		t.Errorf("expected type vless, got %v", vlessOut["type"])
	}
	if vlessOut["server"] != "node.example.com" {
		t.Errorf("expected server node.example.com, got %v", vlessOut["server"])
	}
	if vlessOut["server_port"] != float64(443) {
		t.Errorf("expected server_port 443, got %v", vlessOut["server_port"])
	}

	// Verify protocol settings were merged to top level
	if vlessOut["uuid"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected uuid merged, got %v", vlessOut["uuid"])
	}
	if vlessOut["flow"] != "xtls-rprx-vision" {
		t.Errorf("expected flow xtls-rprx-vision, got %v", vlessOut["flow"])
	}
	if vlessOut["packet_encoding"] != "xudp" {
		t.Errorf("expected packet_encoding xudp, got %v", vlessOut["packet_encoding"])
	}
	tlsMap, ok := vlessOut["tls"].(map[string]interface{})
	if !ok || tlsMap["server_name"] != "node.example.com" {
		t.Errorf("expected tls settings merged, got %v", vlessOut["tls"])
	}
}

// TestGenerateSingBoxConfig_RuleSetsProperlyFormatted tests remote rule set formatting in route.
func TestGenerateSingBoxConfig_RuleSetsProperlyFormatted(t *testing.T) {
	cfg := PrismBoxConfig{
		Routing: Routing{
			RuleSets: []RuleSet{
				{
					Tag:            "geosite-cn",
					Type:           "remote",
					Format:         "binary",
					URL:            "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
					DownloadDetour: "direct",
					UpdateInterval: "1d",
				},
			},
			Rules: []Rule{
				{
					Enabled: true,
					RuleSet: []string{"geosite-cn"},
					Outbound: "proxy",
				},
				{
					Enabled:       true,
					DomainSuffix:  []string{".cn"},
					Outbound:      "direct",
				},
			},
		},
	}

	data, err := GenerateSingBoxConfigFromModel(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	route := parsed["route"].(map[string]interface{})
	ruleSets := route["rule_set"].([]interface{})
	if len(ruleSets) != 1 {
		t.Fatalf("expected 1 rule_set, got %d", len(ruleSets))
	}

	rs := ruleSets[0].(map[string]interface{})
	if rs["type"] != "remote" {
		t.Errorf("expected type remote, got %v", rs["type"])
	}
	if rs["tag"] != "geosite-cn" {
		t.Errorf("expected tag geosite-cn, got %v", rs["tag"])
	}
	if rs["format"] != "binary" {
		t.Errorf("expected format binary, got %v", rs["format"])
	}
	if rs["http_client"] != "http-client-direct" {
		t.Errorf("expected http_client http-client-direct, got %v", rs["http_client"])
	}
	if _, hasDetour := rs["download_detour"]; hasDetour {
		t.Errorf("deprecated download_detour should be omitted from rule_set")
	}
	if rs["update_interval"] != "1d" {
		t.Errorf("expected update_interval 1d, got %v", rs["update_interval"])
	}

	// Verify top-level http_clients definition
	httpClients, ok := parsed["http_clients"].([]interface{})
	if !ok || len(httpClients) == 0 {
		t.Fatalf("expected top-level http_clients array")
	}
	client0 := httpClients[0].(map[string]interface{})
	if client0["tag"] != "http-client-direct" {
		t.Errorf("expected http_client tag http-client-direct, got %v", client0["tag"])
	}
	if client0["detour"] != "direct" {
		t.Errorf("expected http_client detour direct, got %v", client0["detour"])
	}
	tlsObj, ok := client0["tls"].(map[string]interface{})
	if !ok || tlsObj["insecure"] != true {
		t.Errorf("expected http_client tls.insecure == true, got %v", client0["tls"])
	}

	// Verify route rules referencing rule_set
	rules := route["rules"].([]interface{})
	foundRuleSetMatch := false
	for _, r := range rules {
		rMap := r.(map[string]interface{})
		if rMap["outbound"] == "proxy" {
			rsList, ok := rMap["rule_set"].([]interface{})
			if ok && len(rsList) > 0 && rsList[0] == "geosite-cn" {
				foundRuleSetMatch = true
			}
		}
	}
	if !foundRuleSetMatch {
		t.Errorf("expected route rule referencing geosite-cn rule_set")
	}
}

// TestGenerateSingBoxConfig_SniffActionRulesGeneratedCorrectly tests action-based sniff/resolve order.
func TestGenerateSingBoxConfig_SniffActionRulesGeneratedCorrectly(t *testing.T) {
	// 1. Both sniff and resolve enabled
	cfg1 := PrismBoxConfig{
		Routing: Routing{
			Sniff:                    true,
			SniffOverrideDestination: true,
			Rules: []Rule{
				{Enabled: true, Outbound: "direct", Domain: []string{"example.com"}},
			},
		},
	}
	data1, err := GenerateSingBoxConfigFromModel(cfg1)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}
	var p1 map[string]interface{}
	_ = json.Unmarshal(data1, &p1)
	rules1 := p1["route"].(map[string]interface{})["rules"].([]interface{})
	if len(rules1) < 3 {
		t.Fatalf("expected at least 3 rules (sniff, resolve, custom), got %d", len(rules1))
	}
	if rules1[0].(map[string]interface{})["action"] != "sniff" {
		t.Errorf("rule 0 must be sniff action, got %v", rules1[0])
	}
	if rules1[1].(map[string]interface{})["action"] != "resolve" {
		t.Errorf("rule 1 must be resolve action, got %v", rules1[1])
	}

	// 2. Sniff enabled, but resolve disabled
	cfg2 := PrismBoxConfig{
		Routing: Routing{
			Sniff:                    true,
			SniffOverrideDestination: false,
		},
	}
	data2, _ := GenerateSingBoxConfigFromModel(cfg2)
	var p2 map[string]interface{}
	_ = json.Unmarshal(data2, &p2)
	rules2 := p2["route"].(map[string]interface{})["rules"].([]interface{})
	if len(rules2) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules2))
	}
	if rules2[0].(map[string]interface{})["action"] != "sniff" {
		t.Errorf("rule 0 must be sniff, got %v", rules2[0])
	}

	// 3. Sniff disabled
	cfg3 := PrismBoxConfig{
		Routing: Routing{
			Sniff:                    false,
			SniffOverrideDestination: false,
		},
	}
	data3, _ := GenerateSingBoxConfigFromModel(cfg3)
	var p3 map[string]interface{}
	_ = json.Unmarshal(data3, &p3)
	rules3 := p3["route"].(map[string]interface{})["rules"].([]interface{})
	if len(rules3) != 0 {
		t.Errorf("expected 0 rules when sniff disabled, got %d", len(rules3))
	}
}

// TestGenerateSingBoxConfig_TunAndMixedInbounds tests TUN and mixed inbound generation.
func TestGenerateSingBoxConfig_TunAndMixedInbounds(t *testing.T) {
	cfg := PrismBoxConfig{
		Inbounds: []Inbound{
			{
				Tag:           "tun-in",
				Type:          "tun",
				Enabled:       true,
				InterfaceName: "singbox-tun",
				MTU:           1500,
				AutoRoute:     true,
				StrictRoute:   true,
				Stack:         "system",
				Inet4Address:  []string{"172.19.0.1/30"},
				Inet6Address:  []string{"fdfe:dcba:9876::1/126"},
				Sniff:         true,
			},
			{
				Tag:            "mixed-in",
				Type:           "mixed",
				Enabled:        true,
				Listen:         "127.0.0.1",
				Port:           7890,
				SetSystemProxy: true,
			},
			{
				Tag:     "disabled-in",
				Type:    "mixed",
				Enabled: false,
			},
		},
	}

	data, err := GenerateSingBoxConfigFromModel(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	var parsed map[string]interface{}
	_ = json.Unmarshal(data, &parsed)
	inbounds := parsed["inbounds"].([]interface{})
	if len(inbounds) != 2 {
		t.Fatalf("expected 2 enabled inbounds, got %d", len(inbounds))
	}

	tunMap := inbounds[0].(map[string]interface{})
	if tunMap["interface_name"] != "singbox-tun" || tunMap["mtu"] != float64(1500) || tunMap["auto_route"] != true || tunMap["strict_route"] != true {
		t.Errorf("unexpected tun inbound: %+v", tunMap)
	}

	mixedMap := inbounds[1].(map[string]interface{})
	if mixedMap["listen"] != "127.0.0.1" || mixedMap["listen_port"] != float64(7890) || mixedMap["set_system_proxy"] != true {
		t.Errorf("unexpected mixed inbound: %+v", mixedMap)
	}
}

// TestGenerateSingBoxConfig_NoDeprecatedGeoFields verifies that sing-box 1.12+ deprecated
// "geosite" and "geoip" are never outputted and are cleanly converted.
func TestGenerateSingBoxConfig_NoDeprecatedGeoFields(t *testing.T) {
	cfg := defaultConfig()
	cfg.Routing.RuleSets = []RuleSet{
		{
			Tag: "geosite-google",
			URL: "https://example.com/geosite-google.srs",
		},
	}
	cfg.DNS.Enabled = true
	cfg.DNS.Servers = []DNSServer{
		{Tag: "local", Address: "223.5.5.5"},
		{Tag: "remote", Address: "https://1.1.1.1/dns-query"},
	}
	cfg.DNS.Rules = []DNSRule{
		{
			Server:  "local",
			GeoSite: []string{"cn"},
		},
		{
			Server:  "remote",
			GeoSite: []string{"google"},
		},
	}
	cfg.Routing.Rules = []Rule{
		{
			ID:       "r1",
			Enabled:  true,
			Outbound: "direct",
			GeoSite:  []string{"cn"},
			GeoIP:    []string{"private"},
		},
	}

	data, err := GenerateSingBoxConfigFromModel(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	rawJSON := string(data)
	if strings.Contains(rawJSON, `"geosite"`) {
		t.Errorf("generated config must NOT contain deprecated 'geosite' key: %s", rawJSON)
	}
	if strings.Contains(rawJSON, `"geoip"`) {
		t.Errorf("generated config must NOT contain deprecated 'geoip' key: %s", rawJSON)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Verify DNS rule 0 mapped cn -> domain_suffix: [".cn"]
	dnsObj := parsed["dns"].(map[string]interface{})
	dnsRules := dnsObj["rules"].([]interface{})
	rule0 := dnsRules[0].(map[string]interface{})
	suffixes, ok := rule0["domain_suffix"].([]interface{})
	if !ok || len(suffixes) == 0 || suffixes[0] != ".cn" {
		t.Errorf("expected domain_suffix [.cn] in dns rule 0, got %+v", rule0)
	}

	// Verify DNS rule 1 mapped google -> rule_set: ["geosite-google"]
	rule1 := dnsRules[1].(map[string]interface{})
	ruleSets, ok := rule1["rule_set"].([]interface{})
	if !ok || len(ruleSets) == 0 || ruleSets[0] != "geosite-google" {
		t.Errorf("expected rule_set [geosite-google] in dns rule 1, got %+v", rule1)
	}

	// Verify route rule mapped private -> ip_is_private: true
	routeObj := parsed["route"].(map[string]interface{})
	routeRules := routeObj["rules"].([]interface{})
	// Find r1
	var foundR1 map[string]interface{}
	for _, rr := range routeRules {
		rm := rr.(map[string]interface{})
		if rm["outbound"] == "direct" && rm["ip_is_private"] == true {
			foundR1 = rm
			break
		}
	}
	if foundR1 == nil {
		t.Fatalf("did not find route rule with ip_is_private: true, rules: %+v", routeRules)
	}
	r1Suffixes := foundR1["domain_suffix"].([]interface{})
	if len(r1Suffixes) == 0 || r1Suffixes[0] != ".cn" {
		t.Errorf("expected domain_suffix [.cn] in route rule, got %+v", foundR1)
	}
}

// TestGenerateSingBoxConfig_Hysteria2PortHopping verifies that server_ports is emitted
// and server_port is strictly omitted to prevent conflicts in sing-box.
func TestGenerateSingBoxConfig_Hysteria2PortHopping(t *testing.T) {
	cfg := PrismBoxConfig{
		Outbounds: []Outbound{
			{
				Tag:     "hy2-hopping",
				Type:    "hysteria2",
				Server:  "hy2.example.com",
				Port:    20000, // Should be omitted from generated JSON due to server_ports
				Enabled: true,
				Settings: map[string]interface{}{
					"password":         "secret123",
					"server_ports":     []string{"20000:50000"},
					"hop_interval":     "15s",
					"hop_interval_max": "45s",
				},
			},
			{Tag: "direct", Type: "direct", Enabled: true},
			{Tag: "block", Type: "block", Enabled: true},
		},
	}

	data, err := GenerateSingBoxConfigFromModel(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	outbounds := parsed["outbounds"].([]interface{})
	var hy2Out map[string]interface{}
	for _, o := range outbounds {
		om := o.(map[string]interface{})
		if om["tag"] == "hy2-hopping" {
			hy2Out = om
			break
		}
	}

	if hy2Out == nil {
		t.Fatalf("hy2-hopping outbound not found in generated config")
	}

	// 1. Verify server_ports is present
	sp, ok := hy2Out["server_ports"].([]interface{})
	if !ok || len(sp) == 0 || sp[0] != "20000:50000" {
		t.Errorf("expected server_ports [20000:50000], got %+v", hy2Out["server_ports"])
	}

	// 2. CRITICAL: Verify server_port is NOT present (conflicts in sing-box)
	if _, exists := hy2Out["server_port"]; exists {
		t.Errorf("server_port MUST be omitted when server_ports is present, but found %v", hy2Out["server_port"])
	}

	// 3. Verify hop_interval and hop_interval_max
	if hy2Out["hop_interval"] != "15s" {
		t.Errorf("expected hop_interval 15s, got %v", hy2Out["hop_interval"])
	}
	if hy2Out["hop_interval_max"] != "45s" {
		t.Errorf("expected hop_interval_max 45s, got %v", hy2Out["hop_interval_max"])
	}
}

// TestGenerateSingBoxConfig_Hysteria2PortHopping_ZeroPort verifies that when Port is 0 (omitted single port),
// sing-box config still generates successfully with server_ports and no server_port.
func TestGenerateSingBoxConfig_Hysteria2PortHopping_ZeroPort(t *testing.T) {
	cfg := PrismBoxConfig{
		Outbounds: []Outbound{
			{
				Tag:     "hy2-no-single-port",
				Type:    "hysteria2",
				Server:  "hy2.example.com",
				Port:    0, // Zero / omitted single port because port hopping is configured
				Enabled: true,
				Settings: map[string]interface{}{
					"password":     "secret123",
					"server_ports": []string{"20000:50000"},
				},
			},
			{Tag: "direct", Type: "direct", Enabled: true},
			{Tag: "block", Type: "block", Enabled: true},
		},
	}

	data, err := GenerateSingBoxConfigFromModel(cfg)
	if err != nil {
		t.Fatalf("failed to generate config: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	outbounds := parsed["outbounds"].([]interface{})
	var hy2Out map[string]interface{}
	for _, o := range outbounds {
		om := o.(map[string]interface{})
		if om["tag"] == "hy2-no-single-port" {
			hy2Out = om
			break
		}
	}

	if hy2Out == nil {
		t.Fatalf("hy2-no-single-port outbound not found in generated config")
	}

	sp, ok := hy2Out["server_ports"].([]interface{})
	if !ok || len(sp) == 0 || sp[0] != "20000:50000" {
		t.Errorf("expected server_ports [20000:50000], got %+v", hy2Out["server_ports"])
	}

	if _, exists := hy2Out["server_port"]; exists {
		t.Errorf("server_port must NOT exist when port is 0 and server_ports is present, but found %v", hy2Out["server_port"])
	}
}


