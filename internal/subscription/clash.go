package subscription

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/prismbox/prismbox/internal/config"
	"gopkg.in/yaml.v3"
)

// ClashConfig represents top-level Clash/Meta subscription YAML structure.
type ClashConfig struct {
	Proxies []map[string]interface{} `yaml:"proxies"`
}

// ParseClash parses Clash/Meta YAML subscription content into Outbound items.
func ParseClash(data []byte) ([]config.Outbound, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty clash data")
	}

	// Strip UTF-8 BOM if present
	trimmed = bytes.TrimPrefix(trimmed, []byte("\xef\xbb\xbf"))

	var cfg ClashConfig
	if err := yaml.Unmarshal(trimmed, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse clash YAML: %w", err)
	}

	if len(cfg.Proxies) == 0 {
		return nil, fmt.Errorf("no proxies found in clash configuration")
	}

	var outbounds []config.Outbound
	for _, p := range cfg.Proxies {
		out, err := convertClashProxy(p)
		if err != nil {
			// Skip unsupported or malformed proxy entries
			continue
		}
		outbounds = append(outbounds, out)
	}

	if len(outbounds) == 0 {
		return nil, fmt.Errorf("no supported proxies found in clash configuration")
	}

	return outbounds, nil
}

// convertClashProxy converts a single Clash proxy dictionary into a config.Outbound struct.
func convertClashProxy(p map[string]interface{}) (config.Outbound, error) {
	getString := func(keys ...string) string {
		for _, key := range keys {
			if v, ok := p[key]; ok && v != nil {
				return strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
		return ""
	}

	getInt := func(keys ...string) int {
		for _, key := range keys {
			if v, ok := p[key]; ok && v != nil {
				switch val := v.(type) {
				case int:
					return val
				case int64:
					return int(val)
				case float64:
					return int(val)
				case string:
					res, _ := strconv.Atoi(strings.TrimSpace(val))
					return res
				}
			}
		}
		return 0
	}

	getBool := func(keys ...string) bool {
		for _, key := range keys {
			if v, ok := p[key]; ok && v != nil {
				switch val := v.(type) {
				case bool:
					return val
				case string:
					val = strings.ToLower(strings.TrimSpace(val))
					return val == "true" || val == "1"
				case int:
					return val == 1
				case float64:
					return val == 1
				}
			}
		}
		return false
	}

	getMap := func(key string) map[string]interface{} {
		if v, ok := p[key]; ok && v != nil {
			switch m := v.(type) {
			case map[string]interface{}:
				return m
			case map[interface{}]interface{}:
				res := make(map[string]interface{}, len(m))
				for mk, mv := range m {
					res[fmt.Sprintf("%v", mk)] = mv
				}
				return res
			}
		}
		return nil
	}

	parseBandwidth := func(keys ...string) int {
		for _, k := range keys {
			if v, ok := p[k]; ok && v != nil {
				switch val := v.(type) {
				case int:
					return val
				case int64:
					return int(val)
				case float64:
					return int(val)
				case string:
					s := strings.TrimSpace(strings.ToLower(val))
					s = strings.TrimSuffix(s, "mbps")
					s = strings.TrimSuffix(s, "m")
					n, _ := strconv.Atoi(strings.TrimSpace(s))
					return n
				}
			}
		}
		return 0
	}

	rawType := strings.ToLower(getString("type"))
	name := getString("name")
	server := getString("server")
	port := getInt("port")

	if server == "" || port == 0 {
		return config.Outbound{}, fmt.Errorf("missing server or port")
	}

	var mappedType string
	switch rawType {
	case "ss", "shadowsocks":
		mappedType = "shadowsocks"
	case "vmess":
		mappedType = "vmess"
	case "vless":
		mappedType = "vless"
	case "trojan":
		mappedType = "trojan"
	case "hysteria2", "hy2":
		mappedType = "hysteria2"
	case "tuic":
		mappedType = "tuic"
	default:
		return config.Outbound{}, fmt.Errorf("unsupported clash proxy type: %s", rawType)
	}

	if name == "" {
		name = fmt.Sprintf("%s-%s:%d", mappedType, server, port)
	}

	settings := make(map[string]interface{})

	// Common TLS parsing
	hasTLS := getBool("tls")
	sni := getString("servername", "sni")
	skipCertVerify := getBool("skip-cert-verify")
	clientFP := getString("client-fingerprint", "fingerprint")

	var alpnList []string
	if rawALPN, ok := p["alpn"]; ok && rawALPN != nil {
		switch val := rawALPN.(type) {
		case []interface{}:
			for _, item := range val {
				if s := strings.TrimSpace(fmt.Sprintf("%v", item)); s != "" {
					alpnList = append(alpnList, s)
				}
			}
		case []string:
			alpnList = val
		case string:
			alpnList = splitAndTrim(val, ",")
		}
	}

	var tlsMap map[string]interface{}
	if hasTLS || sni != "" || mappedType == "trojan" || mappedType == "hysteria2" || mappedType == "tuic" {
		tlsMap = map[string]interface{}{
			"enabled": true,
		}
		if sni != "" {
			tlsMap["server_name"] = sni
		}
		if skipCertVerify {
			tlsMap["insecure"] = true
		}
		if len(alpnList) > 0 {
			tlsMap["alpn"] = alpnList
		}
		if clientFP != "" {
			tlsMap["utls"] = map[string]interface{}{
				"enabled":     true,
				"fingerprint": clientFP,
			}
		}

		// Reality options
		if realityOpts := getMap("reality-opts"); realityOpts != nil {
			rMap := map[string]interface{}{
				"enabled": true,
			}
			if pbk, ok := realityOpts["public-key"]; ok && pbk != nil {
				rMap["public_key"] = fmt.Sprintf("%v", pbk)
			}
			if sid, ok := realityOpts["short-id"]; ok && sid != nil {
				rMap["short_id"] = fmt.Sprintf("%v", sid)
			}
			tlsMap["reality"] = rMap
		}
	}

	// Transport parsing (ws, grpc, h2, http)
	net := strings.ToLower(getString("network"))
	var transportMap map[string]interface{}
	if net != "" && net != "tcp" {
		transportMap = map[string]interface{}{
			"type": net,
		}
		if net == "ws" {
			wsOpts := getMap("ws-opts")
			if wsOpts != nil {
				if path, ok := wsOpts["path"]; ok && path != nil {
					transportMap["path"] = fmt.Sprintf("%v", path)
				}
				if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
					hdrMap := make(map[string]string)
					for k, v := range headers {
						hdrMap[k] = fmt.Sprintf("%v", v)
					}
					transportMap["headers"] = hdrMap
				}
			} else {
				if path := getString("ws-path"); path != "" {
					transportMap["path"] = path
				}
			}
		} else if net == "grpc" {
			grpcOpts := getMap("grpc-opts")
			if grpcOpts != nil {
				if svc, ok := grpcOpts["grpc-service-name"]; ok && svc != nil {
					transportMap["service_name"] = fmt.Sprintf("%v", svc)
				}
			}
		} else if net == "h2" || net == "http" {
			h2Opts := getMap("h2-opts")
			if h2Opts != nil {
				if path, ok := h2Opts["path"]; ok && path != nil {
					transportMap["path"] = fmt.Sprintf("%v", path)
				}
				if host, ok := h2Opts["host"].([]interface{}); ok {
					var hosts []string
					for _, h := range host {
						hosts = append(hosts, fmt.Sprintf("%v", h))
					}
					transportMap["host"] = hosts
				}
			}
		}
	}

	// Protocol specific fields
	switch mappedType {
	case "shadowsocks":
		cipher := getString("cipher", "method")
		settings["method"] = cipher
		settings["password"] = getString("password")
		if plugin := getString("plugin"); plugin != "" {
			settings["plugin"] = plugin
			if pOpts := p["plugin-opts"]; pOpts != nil {
				settings["plugin_opts"] = pOpts
			}
		}

	case "vmess":
		settings["uuid"] = getString("uuid")
		settings["alter_id"] = getInt("alterId", "aid")
		scy := getString("cipher")
		if scy == "" {
			scy = "auto"
		}
		settings["security"] = scy
		if tlsMap != nil {
			settings["tls"] = tlsMap
		}
		if transportMap != nil {
			settings["transport"] = transportMap
		}

	case "vless":
		settings["uuid"] = getString("uuid")
		if flow := getString("flow"); flow != "" {
			settings["flow"] = flow
		}
		if tlsMap != nil {
			settings["tls"] = tlsMap
		}
		if transportMap != nil {
			settings["transport"] = transportMap
		}

	case "trojan":
		settings["password"] = getString("password")
		if tlsMap != nil {
			settings["tls"] = tlsMap
		}
		if transportMap != nil {
			settings["transport"] = transportMap
		}

	case "hysteria2":
		auth := getString("auth", "password")
		settings["password"] = auth
		if tlsMap != nil {
			settings["tls"] = tlsMap
		}
		if obfs := getString("obfs"); obfs != "" && obfs != "none" {
			settings["obfs"] = map[string]interface{}{
				"type":     obfs,
				"password": getString("obfs-password", "obfs_password"),
			}
		}
		if up := parseBandwidth("up_mbps", "up"); up > 0 {
			settings["up_mbps"] = up
		}
		if down := parseBandwidth("down_mbps", "down"); down > 0 {
			settings["down_mbps"] = down
		}

	case "tuic":
		settings["uuid"] = getString("uuid")
		settings["password"] = getString("password")
		cc := getString("congestion-controller", "congestion_controller", "cc")
		if cc != "" {
			settings["congestion_control"] = cc
		}
		if getBool("reduce-rtt", "reduce_rtt") {
			settings["reduce_rtt"] = true
		}
		if tlsMap != nil {
			if getBool("disable-sni", "disable_sni") {
				tlsMap["disable_sni"] = true
			}
			settings["tls"] = tlsMap
		}
	}

	return config.Outbound{
		ID:       uuid.New().String(),
		Tag:      name,
		Type:     mappedType,
		Enabled:  true,
		Server:   server,
		Port:     port,
		Settings: settings,
	}, nil
}
