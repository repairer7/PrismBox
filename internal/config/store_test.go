package config

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestStoreInitialization tests default config and initial file creation in data_dir/prismbox-config.json.
func TestStoreInitialization(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-init-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	configFile := filepath.Join(tempDir, "prismbox-config.json")
	if _, err := os.Stat(configFile); err != nil {
		t.Fatalf("expected prismbox-config.json to exist at %s: %v", configFile, err)
	}

	cfg := store.GetConfig()

	// Verify default TProxy inbound on port 12345, enabled, listen "::", network "tcp,udp", sniff enabled
	if len(cfg.Inbounds) != 1 {
		t.Fatalf("expected 1 default inbound, got %d", len(cfg.Inbounds))
	}
	inb := cfg.Inbounds[0]
	if inb.Type != "tproxy" || inb.Port != 12345 || !inb.Enabled || inb.Listen != "::" || inb.Network != "tcp,udp" || !inb.Sniff {
		t.Errorf("unexpected default inbound configuration: %+v", inb)
	}

	// Verify direct and block outbounds (sing-box 1.13+ removes dns outbound)
	if len(cfg.Outbounds) < 2 {
		t.Fatalf("expected at least 2 default outbounds, got %d", len(cfg.Outbounds))
	}
	outTags := make(map[string]string)
	for _, o := range cfg.Outbounds {
		outTags[o.Tag] = o.Type
	}
	if outTags["direct"] != "direct" {
		t.Errorf("missing default direct outbound")
	}
	if outTags["block"] != "block" {
		t.Errorf("missing default block outbound")
	}

	// Verify routing with sniff and sniff_override_destination enabled
	if !cfg.Routing.Sniff || !cfg.Routing.SniffOverrideDestination {
		t.Errorf("expected routing sniff and sniff_override_destination to be true, got sniff=%v, override=%v",
			cfg.Routing.Sniff, cfg.Routing.SniffOverrideDestination)
	}

	// Verify DNS disabled by default
	if cfg.DNS.Enabled {
		t.Errorf("expected DNS to be disabled by default")
	}
}

// TestStoreCRUD tests individual entity CRUD operations.
func TestStoreCRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-crud-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// 1. Inbound CRUD
	addedInb, err := store.AddInbound(Inbound{
		Tag:     "tun-test",
		Type:    "tun",
		Enabled: true,
		MTU:     1500,
	})
	if err != nil {
		t.Fatalf("failed to add inbound: %v", err)
	}
	if addedInb.ID == "" {
		t.Errorf("expected generated ID on inbound")
	}

	addedInb.MTU = 9000
	if err := store.UpdateInbound(addedInb.ID, addedInb); err != nil {
		t.Fatalf("failed to update inbound: %v", err)
	}

	inbounds := store.GetInbounds()
	foundInb := false
	for _, in := range inbounds {
		if in.ID == addedInb.ID {
			foundInb = true
			if in.MTU != 9000 {
				t.Errorf("expected updated MTU 9000, got %d", in.MTU)
			}
		}
	}
	if !foundInb {
		t.Errorf("added inbound not found in inbounds list")
	}

	if err := store.DeleteInbound(addedInb.ID); err != nil {
		t.Fatalf("failed to delete inbound: %v", err)
	}

	// 2. Outbound CRUD
	addedOut, err := store.AddOutbound(Outbound{
		Tag:     "node-test",
		Type:    "vless",
		Server:  "node.example.com",
		Port:    443,
		Enabled: true,
		Settings: map[string]interface{}{
			"uuid": "my-uuid",
		},
	})
	if err != nil {
		t.Fatalf("failed to add outbound: %v", err)
	}
	if addedOut.ID == "" {
		t.Errorf("expected generated ID on outbound")
	}

	if err := store.UpdateOutboundLatency(addedOut.ID, 88); err != nil {
		t.Fatalf("failed to update latency: %v", err)
	}

	outs := store.GetOutbounds()
	foundOut := false
	for _, o := range outs {
		if o.ID == addedOut.ID {
			foundOut = true
			if o.Latency != 88 {
				t.Errorf("expected latency 88, got %d", o.Latency)
			}
		}
	}
	if !foundOut {
		t.Errorf("added outbound not found")
	}

	addedOut.Server = "updated.example.com"
	if err := store.UpdateOutbound(addedOut.ID, addedOut); err != nil {
		t.Fatalf("failed to update outbound: %v", err)
	}

	if err := store.DeleteOutbound(addedOut.ID); err != nil {
		t.Fatalf("failed to delete outbound: %v", err)
	}

	// 3. Group CRUD
	addedGrp, err := store.AddGroup(Group{
		Tag:       "proxy-group",
		Type:      "selector",
		Outbounds: []string{"direct", "block"},
	})
	if err != nil {
		t.Fatalf("failed to add group: %v", err)
	}
	if addedGrp.ID == "" {
		t.Errorf("expected generated ID on group")
	}

	addedGrp.Selected = "direct"
	if err := store.UpdateGroup(addedGrp.ID, addedGrp); err != nil {
		t.Fatalf("failed to update group: %v", err)
	}

	grps := store.GetGroups()
	if len(grps) != 1 || grps[0].Selected != "direct" {
		t.Errorf("expected updated group selected direct, got %+v", grps)
	}

	if err := store.DeleteGroup(addedGrp.ID); err != nil {
		t.Fatalf("failed to delete group: %v", err)
	}

	// 4. RuleSet CRUD
	addedRS, err := store.AddRuleSet(RuleSet{
		Tag:            "geoip-cn",
		Type:           "remote",
		Format:         "binary",
		URL:            "https://example.com/geoip-cn.srs",
		UpdateInterval: "1d",
	})
	if err != nil {
		t.Fatalf("failed to add ruleset: %v", err)
	}
	if addedRS.ID == "" {
		t.Errorf("expected generated ID on ruleset")
	}

	addedRS.UpdateInterval = "7d"
	if err := store.UpdateRuleSet(addedRS.ID, addedRS); err != nil {
		t.Fatalf("failed to update ruleset: %v", err)
	}

	rss := store.GetRuleSets()
	if len(rss) != 1 || rss[0].UpdateInterval != "7d" {
		t.Errorf("expected ruleset update_interval 7d, got %+v", rss)
	}

	if err := store.DeleteRuleSet(addedRS.ID); err != nil {
		t.Fatalf("failed to delete ruleset: %v", err)
	}
}

// TestStoreBulkUpdatesAndPersistence tests Update* methods and persistence across restarts.
func TestStoreBulkUpdatesAndPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-persist-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// UpdateInbounds
	newInbounds := []Inbound{
		{Tag: "mixed-1", Type: "mixed", Listen: "127.0.0.1", Port: 2080, Enabled: true},
	}
	if err := store.UpdateInbounds(newInbounds); err != nil {
		t.Fatalf("UpdateInbounds failed: %v", err)
	}

	// UpdateOutbounds
	newOutbounds := []Outbound{
		{Tag: "proxy-1", Type: "shadowsocks", Server: "1.2.3.4", Port: 8388, Enabled: true},
	}
	if err := store.UpdateOutbounds(newOutbounds); err != nil {
		t.Fatalf("UpdateOutbounds failed: %v", err)
	}

	// UpdateGroups
	newGroups := []Group{
		{Tag: "auto-test", Type: "urltest", Outbounds: []string{"proxy-1"}, Interval: "5m"},
	}
	if err := store.UpdateGroups(newGroups); err != nil {
		t.Fatalf("UpdateGroups failed: %v", err)
	}

	// UpdateRouting
	newRouting := Routing{
		Sniff:                    true,
		SniffOverrideDestination: false,
		DomainStrategy:           "ipv4_only",
		Rules: []Rule{
			{Enabled: true, DomainSuffix: []string{".local"}, Outbound: "direct"},
		},
		RuleSets: []RuleSet{
			{Tag: "geosite-category-ads", Type: "remote", Format: "binary", URL: "https://example.com/ads.srs"},
		},
	}
	if err := store.UpdateRouting(newRouting); err != nil {
		t.Fatalf("UpdateRouting failed: %v", err)
	}

	// UpdateDNS
	newDNS := DNSConfig{
		Enabled: true,
		Servers: []DNSServer{
			{Tag: "alidns", Address: "223.5.5.5", Detour: "direct"},
		},
		Strategy:     "prefer_ipv4",
		DisableCache: true,
	}
	if err := store.UpdateDNS(newDNS); err != nil {
		t.Fatalf("UpdateDNS failed: %v", err)
	}

	// Reopen store from same directory and verify all data loaded
	store2, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to reload store: %v", err)
	}

	cfg2 := store2.GetConfig()
	if len(cfg2.Inbounds) != 1 || cfg2.Inbounds[0].Tag != "mixed-1" {
		t.Errorf("inbounds not persisted: %+v", cfg2.Inbounds)
	}
	if len(cfg2.Outbounds) != 1 || cfg2.Outbounds[0].Tag != "proxy-1" {
		t.Errorf("outbounds not persisted: %+v", cfg2.Outbounds)
	}
	if len(cfg2.Groups) != 1 || cfg2.Groups[0].Tag != "auto-test" {
		t.Errorf("groups not persisted: %+v", cfg2.Groups)
	}
	if cfg2.Routing.DomainStrategy != "ipv4_only" || len(cfg2.Routing.Rules) != 1 {
		t.Errorf("routing not persisted: %+v", cfg2.Routing)
	}
	if !cfg2.DNS.Enabled || len(cfg2.DNS.Servers) != 1 || cfg2.DNS.Servers[0].Tag != "alidns" {
		t.Errorf("dns not persisted: %+v", cfg2.DNS)
	}
}

// TestStoreNotFoundErrors tests ErrNotFound behavior for non-existent IDs.
func TestStoreNotFoundErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-notfound-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.UpdateInbound("non-existent", Inbound{Tag: "x"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound on UpdateInbound, got %v", err)
	}
	if err := store.DeleteInbound("non-existent"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound on DeleteInbound, got %v", err)
	}
	if err := store.UpdateOutbound("non-existent", Outbound{Tag: "x"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound on UpdateOutbound, got %v", err)
	}
	if err := store.DeleteOutbound("non-existent"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound on DeleteOutbound, got %v", err)
	}
	if err := store.UpdateGroup("non-existent", Group{Tag: "x"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound on UpdateGroup, got %v", err)
	}
	if err := store.DeleteGroup("non-existent"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound on DeleteGroup, got %v", err)
	}
	if err := store.UpdateRuleSet("non-existent", RuleSet{Tag: "x"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound on UpdateRuleSet, got %v", err)
	}
	if err := store.DeleteRuleSet("non-existent"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound on DeleteRuleSet, got %v", err)
	}
}

// TestStoreConcurrentAccess tests thread safety with concurrent readers and writers.
func TestStoreConcurrentAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-concurrent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	var wg sync.WaitGroup
	workers := 10

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 15; j++ {
				_ = store.GetConfig()
				_ = store.GetInbounds()
				_ = store.GetOutbounds()
				_ = store.GetDNS()

				out, err := store.AddOutbound(Outbound{
					Tag:     "concurrent-node",
					Type:    "direct",
					Enabled: true,
				})
				if err == nil {
					_ = store.UpdateOutboundLatency(out.ID, workerID*10+j)
					_ = store.DeleteOutbound(out.ID)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestStoreDeduplicationOnLoad verifies that duplicate Outbounds and Groups are deduplicated upon Load.
func TestStoreDeduplicationOnLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-dedupe-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	cfg := store.GetConfig()
	// Deliberately introduce duplicate direct and block outbounds, and duplicate groups
	cfg.Outbounds = append(cfg.Outbounds,
		Outbound{ID: "dup-1", Tag: "direct", Type: "direct", Enabled: true},
		Outbound{ID: "dup-2", Tag: "block", Type: "block", Enabled: true},
		Outbound{ID: "dup-3", Tag: "node-1", Type: "vless", Enabled: true},
		Outbound{ID: "dup-4", Tag: "node-1", Type: "vless", Enabled: true},
	)
	cfg.Groups = append(cfg.Groups,
		Group{ID: "grp-1", Tag: "AUTO", Type: "urltest"},
		Group{ID: "grp-2", Tag: "AUTO", Type: "selector"},
	)

	// Save raw JSON with duplicates
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Re-load and verify deduplication
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	loadedCfg := store.GetConfig()
	tagCounts := make(map[string]int)
	for _, o := range loadedCfg.Outbounds {
		tagCounts[o.Tag]++
	}
	for tag, count := range tagCounts {
		if count > 1 {
			t.Errorf("expected tag %q to be unique, got count %d", tag, count)
		}
	}

	grpCounts := make(map[string]int)
	for _, g := range loadedCfg.Groups {
		grpCounts[g.Tag]++
	}
	for tag, count := range grpCounts {
		if count > 1 {
			t.Errorf("expected group tag %q to be unique, got count %d", tag, count)
		}
	}
}
