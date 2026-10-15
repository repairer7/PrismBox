package config

import (
	"encoding/json"
	"fmt"
)

// SingBoxConfig represents the top-level sing-box 1.14.x JSON schema.
type SingBoxConfig struct {
	Schema       string                   `json:"$schema,omitempty"`
	Log          *SingBoxLog              `json:"log,omitempty"`
	DNS          *SingBoxDNS              `json:"dns,omitempty"`
	Inbounds     []map[string]interface{} `json:"inbounds"`
	Outbounds    []map[string]interface{} `json:"outbounds"`
	Route        *SingBoxRoute            `json:"route,omitempty"`
	Experimental *SingBoxExperimental     `json:"experimental,omitempty"`
}

// SingBoxLog defines log settings for sing-box.
type SingBoxLog struct {
	Level     string `json:"level"`
	Timestamp bool   `json:"timestamp"`
}

// SingBoxDNS defines DNS resolver configuration for sing-box.
type SingBoxDNS struct {
	Servers          []map[string]interface{} `json:"servers,omitempty"`
	Rules            []map[string]interface{} `json:"rules,omitempty"`
	Strategy         string                   `json:"strategy,omitempty"`
	DisableCache     bool                     `json:"disable_cache,omitempty"`
	DisableExpire    bool                     `json:"disable_expire,omitempty"`
	IndependentCache bool                     `json:"independent_cache,omitempty"`
}

// SingBoxRoute defines routing options and rule matching.
type SingBoxRoute struct {
	Rules                 []map[string]interface{} `json:"rules"`
	RuleSets              []map[string]interface{} `json:"rule_set,omitempty"`
	DefaultDomainResolver map[string]interface{}   `json:"default_domain_resolver,omitempty"`
	AutoDetectInterface   bool                     `json:"auto_detect_interface"`
}

// SingBoxExperimental defines experimental configurations such as Clash API.
type SingBoxExperimental struct {
	ClashAPI *SingBoxClashAPI `json:"clash_api,omitempty"`
}

// SingBoxClashAPI defines external controller settings for Clash API.
type SingBoxClashAPI struct {
	ExternalController string `json:"external_controller"`
}

// GenerateSingBoxConfig converts PrismBoxConfig stored in store into native sing-box 1.14.3 JSON.
func GenerateSingBoxConfig(store *Store) ([]byte, error) {
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}

	cfg := store.GetConfig()
	return GenerateSingBoxConfigFromModel(cfg)
}

// GenerateSingBoxConfigFromModel builds sing-box JSON from a PrismBoxConfig struct.
func GenerateSingBoxConfigFromModel(cfg PrismBoxConfig) ([]byte, error) {
	sb := SingBoxConfig{
		Schema: "https://sing-box.sagernet.org/schema.json",
		Log: &SingBoxLog{
			Level:     "info",
			Timestamp: true,
		},
		Inbounds:  make([]map[string]interface{}, 0),
		Outbounds: make([]map[string]interface{}, 0),
		Experimental: &SingBoxExperimental{
			ClashAPI: &SingBoxClashAPI{
				ExternalController: "127.0.0.1:9091",
			},
		},
	}

	// 1. DNS section - ONLY included if cfg.DNS.Enabled is true
	if cfg.DNS.Enabled {
		dns := &SingBoxDNS{
			Servers:          make([]map[string]interface{}, 0, len(cfg.DNS.Servers)),
			Rules:            make([]map[string]interface{}, 0, len(cfg.DNS.Rules)),
			Strategy:         cfg.DNS.Strategy,
			DisableCache:     cfg.DNS.DisableCache,
			DisableExpire:    cfg.DNS.DisableExpire,
			IndependentCache: cfg.DNS.IndependentCache,
		}

		for _, s := range cfg.DNS.Servers {
			srv := map[string]interface{}{
				"tag":     s.Tag,
				"address": s.Address,
			}
			if s.AddressResolver != "" {
				srv["address_resolver"] = s.AddressResolver
			}
			if s.Detour != "" {
				srv["detour"] = s.Detour
			}
			if s.Strategy != "" {
				srv["strategy"] = s.Strategy
			}
			dns.Servers = append(dns.Servers, srv)
		}

		for _, r := range cfg.DNS.Rules {
			rule := map[string]interface{}{
				"server": r.Server,
			}
			if len(r.Domain) > 0 {
				rule["domain"] = r.Domain
			}
			if len(r.DomainSuffix) > 0 {
				rule["domain_suffix"] = r.DomainSuffix
			}
			if len(r.DomainKeyword) > 0 {
				rule["domain_keyword"] = r.DomainKeyword
			}
			if len(r.GeoSite) > 0 {
				rule["geosite"] = r.GeoSite
			}
			if len(r.RuleSet) > 0 {
				rule["rule_set"] = r.RuleSet
			}
			if r.Outbound != "" {
				rule["outbound"] = r.Outbound
			}
			if r.ClashMode != "" {
				rule["clash_mode"] = r.ClashMode
			}
			dns.Rules = append(dns.Rules, rule)
		}

		sb.DNS = dns
	}

	// 2. Inbounds section
	for _, in := range cfg.Inbounds {
		if !in.Enabled {
			continue
		}

		inboundMap := map[string]interface{}{
			"type": in.Type,
			"tag":  in.Tag,
		}

		if in.Sniff {
			inboundMap["sniff"] = true
		}
		if in.SniffOverrideDestination {
			inboundMap["sniff_override_destination"] = true
		}

		switch in.Type {
		case "tproxy":
			listen := in.Listen
			if listen == "" {
				listen = "::"
			}
			inboundMap["listen"] = listen
			inboundMap["listen_port"] = in.Port
			network := in.Network
			if network == "" {
				network = "tcp,udp"
			}
			inboundMap["network"] = network

		case "tun":
			ifName := in.InterfaceName
			if ifName == "" {
				ifName = "tun0"
			}
			inboundMap["interface_name"] = ifName
			mtu := in.MTU
			if mtu <= 0 {
				mtu = 9000
			}
			inboundMap["mtu"] = mtu
			inboundMap["auto_route"] = in.AutoRoute
			inboundMap["strict_route"] = in.StrictRoute

			stack := in.Stack
			if stack == "" {
				stack = "system"
			}
			inboundMap["stack"] = stack

			if len(in.Inet4Address) > 0 {
				inboundMap["inet4_address"] = in.Inet4Address
			} else {
				inboundMap["inet4_address"] = []string{"172.19.0.1/30"}
			}

			if len(in.Inet6Address) > 0 {
				inboundMap["inet6_address"] = in.Inet6Address
			}

		case "mixed":
			listen := in.Listen
			if listen == "" {
				listen = "::"
			}
			inboundMap["listen"] = listen
			inboundMap["listen_port"] = in.Port
			if in.SetSystemProxy {
				inboundMap["set_system_proxy"] = true
			}

		default:
			if in.Listen != "" {
				inboundMap["listen"] = in.Listen
			}
			if in.Port > 0 {
				inboundMap["listen_port"] = in.Port
			}
		}

		sb.Inbounds = append(sb.Inbounds, inboundMap)
	}

	// 3. Outbounds section
	existingTags := make(map[string]bool)

	// Add user-defined outbounds: copy Settings map directly, add type/tag/server/server_port at top level
	for _, out := range cfg.Outbounds {
		if !out.Enabled {
			continue
		}

		outMap := make(map[string]interface{})
		for k, v := range out.Settings {
			outMap[k] = v
		}

		outMap["type"] = out.Type
		outMap["tag"] = out.Tag
		if out.Server != "" {
			outMap["server"] = out.Server
		}
		if out.Port > 0 {
			outMap["server_port"] = out.Port
		}

		existingTags[out.Tag] = true
		sb.Outbounds = append(sb.Outbounds, outMap)
	}

	// Add groups (selector, urltest)
	for _, grp := range cfg.Groups {
		grpMap := map[string]interface{}{
			"type":      grp.Type,
			"tag":       grp.Tag,
			"outbounds": grp.Outbounds,
		}

		if grp.Type == "selector" {
			if grp.Selected != "" {
				grpMap["default"] = grp.Selected
			} else if len(grp.Outbounds) > 0 {
				grpMap["default"] = grp.Outbounds[0]
			}
		}

		if grp.Type == "urltest" {
			url := grp.URL
			if url == "" {
				url = "https://www.gstatic.com/generate_204"
			}
			grpMap["url"] = url

			interval := grp.Interval
			if interval == "" {
				interval = "3m"
			}
			grpMap["interval"] = interval

			if grp.Tolerance > 0 {
				grpMap["tolerance"] = grp.Tolerance
			}
		}

		existingTags[grp.Tag] = true
		sb.Outbounds = append(sb.Outbounds, grpMap)
	}

	// Always ensure "direct", "block", and "dns-out" outbounds exist
	if !existingTags["direct"] {
		sb.Outbounds = append(sb.Outbounds, map[string]interface{}{
			"type": "direct",
			"tag":  "direct",
		})
	}
	if !existingTags["block"] {
		sb.Outbounds = append(sb.Outbounds, map[string]interface{}{
			"type": "block",
			"tag":  "block",
		})
	}
	if !existingTags["dns-out"] {
		sb.Outbounds = append(sb.Outbounds, map[string]interface{}{
			"type": "dns",
			"tag":  "dns-out",
		})
	}

	// 4. Route section
	route := &SingBoxRoute{
		Rules:               make([]map[string]interface{}, 0),
		RuleSets:            make([]map[string]interface{}, 0),
		AutoDetectInterface: true,
	}

	if cfg.DNS.Enabled && len(cfg.DNS.Servers) > 0 {
		strategy := cfg.Routing.DomainStrategy
		if strategy == "" {
			strategy = "prefer_ipv4"
		}
		route.DefaultDomainResolver = map[string]interface{}{
			"server":   cfg.DNS.Servers[0].Tag,
			"strategy": strategy,
		}
	}

	// Route rules (sing-box 1.14.x action-based format)
	// Rule 1: If sniff enabled globally, add {"action": "sniff"} as first rule
	if cfg.Routing.Sniff {
		route.Rules = append(route.Rules, map[string]interface{}{
			"action": "sniff",
		})
		// Rule 2: If sniff_override_destination, add {"action": "resolve"} after sniff
		if cfg.Routing.SniffOverrideDestination {
			route.Rules = append(route.Rules, map[string]interface{}{
				"action": "resolve",
			})
		}
	}

	// Rule 3: For DNS hijack, add {"protocol": "dns", "action": "hijack-dns"} when DNS is enabled
	if cfg.DNS.Enabled {
		route.Rules = append(route.Rules, map[string]interface{}{
			"protocol": "dns",
			"action":   "hijack-dns",
		})
	}

	// Rule 4: Regular routing rules
	for _, r := range cfg.Routing.Rules {
		if !r.Enabled {
			continue
		}

		ruleMap := make(map[string]interface{})
		if r.Outbound == "hijack-dns" {
			ruleMap["action"] = "hijack-dns"
		} else if r.Outbound != "" {
			ruleMap["outbound"] = r.Outbound
		}

		if len(r.Domain) > 0 {
			ruleMap["domain"] = r.Domain
		}
		if len(r.DomainSuffix) > 0 {
			ruleMap["domain_suffix"] = r.DomainSuffix
		}
		if len(r.DomainKeyword) > 0 {
			ruleMap["domain_keyword"] = r.DomainKeyword
		}
		if len(r.DomainRegex) > 0 {
			ruleMap["domain_regex"] = r.DomainRegex
		}
		if len(r.IPCidr) > 0 {
			ruleMap["ip_cidr"] = r.IPCidr
		}
		if len(r.GeoIP) > 0 {
			ruleMap["geoip"] = r.GeoIP
		}
		if len(r.GeoSite) > 0 {
			ruleMap["geosite"] = r.GeoSite
		}
		if len(r.RuleSet) > 0 {
			ruleMap["rule_set"] = r.RuleSet
		}
		if len(r.Protocol) > 0 {
			ruleMap["protocol"] = r.Protocol
		}
		if len(r.Port) > 0 {
			ruleMap["port"] = r.Port
		}
		if len(r.PortRange) > 0 {
			ruleMap["port_range"] = r.PortRange
		}
		if len(r.Network) > 0 {
			ruleMap["network"] = r.Network
		}
		if len(r.SourceIPCidr) > 0 {
			ruleMap["source_ip_cidr"] = r.SourceIPCidr
		}

		route.Rules = append(route.Rules, ruleMap)
	}

	// RuleSets formatting
	for _, rs := range cfg.Routing.RuleSets {
		rsType := rs.Type
		if rsType == "" {
			rsType = "remote"
		}
		rsFormat := rs.Format
		if rsFormat == "" {
			rsFormat = "binary"
		}
		detour := rs.DownloadDetour
		if detour == "" {
			detour = "direct"
		}
		interval := rs.UpdateInterval
		if interval == "" {
			interval = rs.Interval
		}
		if interval == "" {
			interval = "1d"
		}

		rsMap := map[string]interface{}{
			"type":            rsType,
			"tag":             rs.Tag,
			"format":          rsFormat,
			"url":             rs.URL,
			"download_detour": detour,
			"update_interval": interval,
		}

		route.RuleSets = append(route.RuleSets, rsMap)
	}

	sb.Route = route

	return json.MarshalIndent(sb, "", "  ")
}
