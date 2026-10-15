package subscription

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/prismbox/prismbox/internal/config"
)

var (
	// ErrUnsupportedScheme indicates the link URI scheme is not recognized.
	ErrUnsupportedScheme = errors.New("unsupported proxy URI scheme")
	// ErrInvalidLink indicates the link URI is empty or malformed.
	ErrInvalidLink = errors.New("invalid proxy URI")
)

// ParseLink parses a single proxy URI into a config.Outbound struct.
// Supported schemes:
//   - hysteria2://, hy2://
//   - vless://
//   - vmess://
//   - trojan://
//   - ss://
//   - tuic://
func ParseLink(rawURI string) (config.Outbound, error) {
	rawURI = strings.TrimSpace(rawURI)
	if rawURI == "" {
		return config.Outbound{}, ErrInvalidLink
	}

	lowerURI := strings.ToLower(rawURI)
	switch {
	case strings.HasPrefix(lowerURI, "hysteria2://") || strings.HasPrefix(lowerURI, "hy2://"):
		return parseHysteria2Link(rawURI)
	case strings.HasPrefix(lowerURI, "vless://"):
		return parseVLESSLink(rawURI)
	case strings.HasPrefix(lowerURI, "vmess://"):
		return parseVMessLink(rawURI)
	case strings.HasPrefix(lowerURI, "trojan://"):
		return parseTrojanLink(rawURI)
	case strings.HasPrefix(lowerURI, "ss://"):
		return parseShadowsocksLink(rawURI)
	case strings.HasPrefix(lowerURI, "tuic://"):
		return parseTUICLink(rawURI)
	default:
		return config.Outbound{}, fmt.Errorf("%w: %s", ErrUnsupportedScheme, rawURI)
	}
}

// ParseMultipleLinks splits the input text by newline, skips comments/blank lines,
// parses each proxy share link, and returns valid outbounds and errors.
func ParseMultipleLinks(text string) ([]config.Outbound, []error) {
	var outbounds []config.Outbound
	var errs []error

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		outbound, err := ParseLink(line)
		if err != nil {
			errs = append(errs, fmt.Errorf("line %q: %w", line, err))
			continue
		}
		outbounds = append(outbounds, outbound)
	}

	return outbounds, errs
}

// decodeBase64Flexible decodes standard or URL-safe base64, with or without padding.
func decodeBase64Flexible(encoded string) ([]byte, error) {
	encoded = strings.TrimSpace(encoded)
	encoded = strings.ReplaceAll(encoded, "\r", "")
	encoded = strings.ReplaceAll(encoded, "\n", "")
	encoded = strings.ReplaceAll(encoded, " ", "")
	if encoded == "" {
		return nil, errors.New("empty base64 string")
	}

	// Try URL unescape first in case the base64 string has encoded padding (%3D)
	if unescaped, err := url.QueryUnescape(encoded); err == nil && unescaped != encoded {
		encoded = unescaped
	}

	padLen := (4 - (len(encoded) % 4)) % 4
	padded := encoded + strings.Repeat("=", padLen)

	// 1. Try standard base64 decoding (padded)
	if b, err := base64.StdEncoding.DecodeString(padded); err == nil {
		return b, nil
	}
	// 2. Try raw standard base64 decoding (unpadded)
	if b, err := base64.RawStdEncoding.DecodeString(encoded); err == nil {
		return b, nil
	}
	// 3. Try URL-safe base64 decoding (padded)
	if b, err := base64.URLEncoding.DecodeString(padded); err == nil {
		return b, nil
	}
	// 4. Try raw URL-safe base64 decoding (unpadded)
	if b, err := base64.RawURLEncoding.DecodeString(encoded); err == nil {
		return b, nil
	}

	return nil, fmt.Errorf("failed to decode flexible base64: %s", encoded)
}

// splitAndTrim splits string by separator, trimming whitespace and dropping empty items.
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}
	}
	return res
}

// parseHysteria2Link parses hysteria2://auth@server:port?query#Name or hy2://
func parseHysteria2Link(rawURI string) (config.Outbound, error) {
	uriToParse := rawURI
	if strings.HasPrefix(strings.ToLower(rawURI), "hy2://") {
		uriToParse = "hysteria2://" + rawURI[len("hy2://"):]
	}

	u, err := url.Parse(uriToParse)
	if err != nil {
		return config.Outbound{}, fmt.Errorf("invalid hysteria2 URI: %w", err)
	}

	server := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	// Extract and decode password from userinfo
	var password string
	if u.User != nil {
		if pass, ok := u.User.Password(); ok {
			if u.User.Username() != "" {
				password = pass
			} else {
				password = pass
			}
		} else {
			password = u.User.Username()
		}
	}
	// url.Parse unescapes standard userinfo, but ensure any remaining %-encoding is decoded
	if unescaped, err := url.PathUnescape(password); err == nil {
		password = unescaped
	}

	// Fragment -> tag (URL-decoded)
	tag := u.Fragment
	if tag != "" {
		if unescaped, err := url.QueryUnescape(tag); err == nil {
			tag = unescaped
		}
	}
	if tag == "" {
		tag = fmt.Sprintf("hy2-%s:%d", server, port)
	}

	q := u.Query()
	sni := q.Get("sni")
	if sni == "" {
		sni = q.Get("peer")
	}

	insecure := q.Get("insecure") == "1" ||
		strings.ToLower(q.Get("insecure")) == "true" ||
		q.Get("allowInsecure") == "1" ||
		strings.ToLower(q.Get("allowInsecure")) == "true"

	tlsMap := map[string]interface{}{
		"enabled": true,
	}
	if sni != "" {
		tlsMap["server_name"] = sni
	}
	if insecure {
		tlsMap["insecure"] = true
	}
	if alpn := q.Get("alpn"); alpn != "" {
		tlsMap["alpn"] = splitAndTrim(alpn, ",")
	}

	settings := map[string]interface{}{
		"password": password,
		"tls":      tlsMap,
	}

	if obfs := q.Get("obfs"); obfs != "" && obfs != "none" {
		obfsPass := q.Get("obfs-password")
		if obfsPass == "" {
			obfsPass = q.Get("obfs_password")
		}
		settings["obfs"] = map[string]interface{}{
			"type":     obfs,
			"password": obfsPass,
		}
	}

	if up := q.Get("up_mbps"); up != "" {
		if upInt, err := strconv.Atoi(up); err == nil {
			settings["up_mbps"] = upInt
		}
	} else if up := q.Get("up"); up != "" {
		upVal := strings.TrimSpace(strings.ToLower(up))
		upVal = strings.TrimSuffix(upVal, "mbps")
		upVal = strings.TrimSuffix(upVal, "m")
		if upInt, err := strconv.Atoi(strings.TrimSpace(upVal)); err == nil {
			settings["up_mbps"] = upInt
		}
	}

	if down := q.Get("down_mbps"); down != "" {
		if downInt, err := strconv.Atoi(down); err == nil {
			settings["down_mbps"] = downInt
		}
	} else if down := q.Get("down"); down != "" {
		downVal := strings.TrimSpace(strings.ToLower(down))
		downVal = strings.TrimSuffix(downVal, "mbps")
		downVal = strings.TrimSuffix(downVal, "m")
		if downInt, err := strconv.Atoi(strings.TrimSpace(downVal)); err == nil {
			settings["down_mbps"] = downInt
		}
	}

	return config.Outbound{
		ID:       uuid.New().String(),
		Tag:      tag,
		Type:     "hysteria2",
		Enabled:  true,
		Server:   server,
		Port:     port,
		Settings: settings,
	}, nil
}

// parseVLESSLink parses vless://uuid@server:port?query#name
func parseVLESSLink(rawURI string) (config.Outbound, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return config.Outbound{}, fmt.Errorf("invalid vless URI: %w", err)
	}

	server := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	uid := ""
	if u.User != nil {
		uid = u.User.Username()
	}
	if unescaped, err := url.PathUnescape(uid); err == nil {
		uid = unescaped
	}

	tag := u.Fragment
	if tag != "" {
		if unescaped, err := url.QueryUnescape(tag); err == nil {
			tag = unescaped
		}
	}
	if tag == "" {
		tag = fmt.Sprintf("vless-%s:%d", server, port)
	}

	q := u.Query()
	settings := map[string]interface{}{
		"uuid": uid,
	}

	if flow := q.Get("flow"); flow != "" {
		settings["flow"] = flow
	}

	sec := strings.ToLower(q.Get("security"))
	sni := q.Get("sni")
	fp := q.Get("fp")
	pbk := q.Get("pbk")
	sid := q.Get("sid")
	alpn := q.Get("alpn")
	insecure := q.Get("insecure") == "1" ||
		strings.ToLower(q.Get("insecure")) == "true" ||
		q.Get("allowInsecure") == "1" ||
		strings.ToLower(q.Get("allowInsecure")) == "true"

	if sec == "reality" {
		tlsMap := map[string]interface{}{
			"enabled": true,
			"reality": map[string]interface{}{
				"enabled":    true,
				"public_key": pbk,
				"short_id":   sid,
			},
		}
		if sni != "" {
			tlsMap["server_name"] = sni
		}
		if fp == "" {
			fp = "chrome"
		}
		tlsMap["utls"] = map[string]interface{}{
			"enabled":     true,
			"fingerprint": fp,
		}
		settings["tls"] = tlsMap
	} else if sec == "tls" || (sec == "" && sni != "") {
		tlsMap := map[string]interface{}{
			"enabled": true,
		}
		if sni != "" {
			tlsMap["server_name"] = sni
		}
		if alpn != "" {
			tlsMap["alpn"] = splitAndTrim(alpn, ",")
		}
		if insecure {
			tlsMap["insecure"] = true
		}
		if fp != "" {
			tlsMap["utls"] = map[string]interface{}{
				"enabled":     true,
				"fingerprint": fp,
			}
		}
		settings["tls"] = tlsMap
	}

	tp := strings.ToLower(q.Get("type"))
	if tp == "" {
		tp = strings.ToLower(q.Get("transport"))
	}
	if tp != "" && tp != "tcp" {
		transportMap := map[string]interface{}{
			"type": tp,
		}
		if tp == "ws" {
			if path := q.Get("path"); path != "" {
				transportMap["path"] = path
			}
			if host := q.Get("host"); host != "" {
				transportMap["headers"] = map[string]string{"Host": host}
			}
		} else if tp == "grpc" {
			svc := q.Get("serviceName")
			if svc == "" {
				svc = q.Get("service_name")
			}
			if svc == "" {
				svc = q.Get("path")
			}
			if svc != "" {
				transportMap["service_name"] = svc
			}
		} else if tp == "http" || tp == "h2" {
			if path := q.Get("path"); path != "" {
				transportMap["path"] = path
			}
			if host := q.Get("host"); host != "" {
				transportMap["host"] = splitAndTrim(host, ",")
			}
		}
		settings["transport"] = transportMap
	}

	return config.Outbound{
		ID:       uuid.New().String(),
		Tag:      tag,
		Type:     "vless",
		Enabled:  true,
		Server:   server,
		Port:     port,
		Settings: settings,
	}, nil
}

// parseVMessLink parses vmess://<base64-json>
func parseVMessLink(rawURI string) (config.Outbound, error) {
	b64Part := rawURI[len("vmess://"):]
	fragmentTag := ""
	if idx := strings.Index(b64Part, "#"); idx != -1 {
		fragmentTag = b64Part[idx+1:]
		b64Part = b64Part[:idx]
		if unescaped, err := url.QueryUnescape(fragmentTag); err == nil {
			fragmentTag = unescaped
		}
	}

	decoded, err := decodeBase64Flexible(b64Part)
	if err != nil {
		return config.Outbound{}, fmt.Errorf("failed to decode vmess base64: %w", err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(decoded, &rawMap); err != nil {
		return config.Outbound{}, fmt.Errorf("failed to parse vmess json: %w", err)
	}

	getString := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := rawMap[k]; ok && v != nil {
				return strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
		return ""
	}

	getInt := func(keys ...string) int {
		for _, k := range keys {
			if v, ok := rawMap[k]; ok && v != nil {
				switch val := v.(type) {
				case float64:
					return int(val)
				case int:
					return val
				case int64:
					return int(val)
				case string:
					val = strings.TrimSpace(val)
					if p, err := strconv.Atoi(val); err == nil {
						return p
					}
				}
			}
		}
		return 0
	}

	getBool := func(keys ...string) bool {
		for _, k := range keys {
			if v, ok := rawMap[k]; ok && v != nil {
				switch val := v.(type) {
				case bool:
					return val
				case string:
					val = strings.ToLower(strings.TrimSpace(val))
					return val == "true" || val == "1"
				case float64:
					return val == 1
				case int:
					return val == 1
				}
			}
		}
		return false
	}

	tag := getString("ps")
	if tag == "" && fragmentTag != "" {
		tag = fragmentTag
	}
	server := getString("add")
	port := getInt("port")
	uid := getString("id")
	aid := getInt("aid", "alterId")
	scy := getString("scy", "cipher", "security")
	if scy == "" {
		scy = "auto"
	}
	net := strings.ToLower(getString("net", "network"))
	host := getString("host")
	path := getString("path")
	tlsStr := strings.ToLower(getString("tls"))
	sni := getString("sni")
	alpn := getString("alpn")
	fp := getString("fp")
	insecure := getBool("allowInsecure", "insecure")

	if tag == "" {
		tag = fmt.Sprintf("vmess-%s:%d", server, port)
	}

	settings := map[string]interface{}{
		"uuid":     uid,
		"alter_id": aid,
		"security": scy,
	}

	if tlsStr == "tls" || tlsStr == "1" || tlsStr == "true" || sni != "" {
		tlsMap := map[string]interface{}{
			"enabled": true,
		}
		if sni != "" {
			tlsMap["server_name"] = sni
		} else if host != "" {
			tlsMap["server_name"] = host
		}
		if alpn != "" {
			tlsMap["alpn"] = splitAndTrim(alpn, ",")
		}
		if insecure {
			tlsMap["insecure"] = true
		}
		if fp != "" {
			tlsMap["utls"] = map[string]interface{}{
				"enabled":     true,
				"fingerprint": fp,
			}
		}
		settings["tls"] = tlsMap
	}

	if net != "" && net != "tcp" {
		transportMap := map[string]interface{}{
			"type": net,
		}
		if net == "ws" {
			if path != "" {
				transportMap["path"] = path
			}
			if host != "" {
				transportMap["headers"] = map[string]string{"Host": host}
			}
		} else if net == "grpc" {
			svc := getString("serviceName", "service_name")
			if svc == "" {
				svc = path
			}
			if svc != "" {
				transportMap["service_name"] = svc
			}
		} else if net == "http" || net == "h2" {
			if path != "" {
				transportMap["path"] = path
			}
			if host != "" {
				transportMap["host"] = splitAndTrim(host, ",")
			}
		}
		settings["transport"] = transportMap
	}

	return config.Outbound{
		ID:       uuid.New().String(),
		Tag:      tag,
		Type:     "vmess",
		Enabled:  true,
		Server:   server,
		Port:     port,
		Settings: settings,
	}, nil
}

// parseTrojanLink parses trojan://password@server:port?query#name
func parseTrojanLink(rawURI string) (config.Outbound, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return config.Outbound{}, fmt.Errorf("invalid trojan URI: %w", err)
	}

	server := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	var password string
	if u.User != nil {
		if pass, ok := u.User.Password(); ok && u.User.Username() == "" {
			password = pass
		} else {
			password = u.User.Username()
		}
	}
	if unescaped, err := url.PathUnescape(password); err == nil {
		password = unescaped
	}

	tag := u.Fragment
	if tag != "" {
		if unescaped, err := url.QueryUnescape(tag); err == nil {
			tag = unescaped
		}
	}
	if tag == "" {
		tag = fmt.Sprintf("trojan-%s:%d", server, port)
	}

	q := u.Query()
	sni := q.Get("sni")
	if sni == "" {
		sni = q.Get("peer")
	}
	if sni == "" {
		sni = server
	}

	insecure := q.Get("insecure") == "1" ||
		strings.ToLower(q.Get("insecure")) == "true" ||
		q.Get("allowInsecure") == "1" ||
		strings.ToLower(q.Get("allowInsecure")) == "true"

	tlsMap := map[string]interface{}{
		"enabled":     true,
		"server_name": sni,
	}
	if alpn := q.Get("alpn"); alpn != "" {
		tlsMap["alpn"] = splitAndTrim(alpn, ",")
	}
	if insecure {
		tlsMap["insecure"] = true
	}
	if fp := q.Get("fp"); fp != "" {
		tlsMap["utls"] = map[string]interface{}{
			"enabled":     true,
			"fingerprint": fp,
		}
	}

	settings := map[string]interface{}{
		"password": password,
		"tls":      tlsMap,
	}

	tp := strings.ToLower(q.Get("type"))
	if tp == "" {
		tp = strings.ToLower(q.Get("transport"))
	}
	if tp != "" && tp != "tcp" {
		transportMap := map[string]interface{}{
			"type": tp,
		}
		if tp == "ws" {
			if path := q.Get("path"); path != "" {
				transportMap["path"] = path
			}
			if host := q.Get("host"); host != "" {
				transportMap["headers"] = map[string]string{"Host": host}
			}
		} else if tp == "grpc" {
			svc := q.Get("serviceName")
			if svc == "" {
				svc = q.Get("service_name")
			}
			if svc == "" {
				svc = q.Get("path")
			}
			if svc != "" {
				transportMap["service_name"] = svc
			}
		} else if tp == "http" || tp == "h2" {
			if path := q.Get("path"); path != "" {
				transportMap["path"] = path
			}
			if host := q.Get("host"); host != "" {
				transportMap["host"] = splitAndTrim(host, ",")
			}
		}
		settings["transport"] = transportMap
	}

	return config.Outbound{
		ID:       uuid.New().String(),
		Tag:      tag,
		Type:     "trojan",
		Enabled:  true,
		Server:   server,
		Port:     port,
		Settings: settings,
	}, nil
}

// parseShadowsocksLink parses ss://<base64>#name or ss://<user>@server:port#name
func parseShadowsocksLink(rawURI string) (config.Outbound, error) {
	tag := ""
	if idx := strings.Index(rawURI, "#"); idx != -1 {
		tag = rawURI[idx+1:]
		rawURI = rawURI[:idx]
		if unescaped, err := url.QueryUnescape(tag); err == nil {
			tag = unescaped
		}
	}

	uriBody := rawURI[len("ss://"):]

	var method, password, server string
	var port int
	var plugin, pluginOpts string

	if atIdx := strings.LastIndex(uriBody, "@"); atIdx != -1 {
		// SIP002 format: ss://base64(method:password)@server:port/?plugin=...
		userPart := uriBody[:atIdx]
		hostPart := uriBody[atIdx+1:]

		// Decode userPart if base64 encoded
		decodedUser, err := decodeBase64Flexible(userPart)
		if err == nil && strings.Contains(string(decodedUser), ":") {
			parts := strings.SplitN(string(decodedUser), ":", 2)
			method = parts[0]
			password = parts[1]
		} else if strings.Contains(userPart, ":") {
			parts := strings.SplitN(userPart, ":", 2)
			method = parts[0]
			password = parts[1]
		}

		// Parse plugin from query string if present
		var queryStr string
		if qIdx := strings.Index(hostPart, "?"); qIdx != -1 {
			queryStr = hostPart[qIdx+1:]
			hostPart = hostPart[:qIdx]
		}
		if queryStr != "" {
			if values, err := url.ParseQuery(queryStr); err == nil {
				pluginStr := values.Get("plugin")
				if pluginStr != "" {
					pParts := strings.SplitN(pluginStr, ";", 2)
					plugin = pParts[0]
					if len(pParts) > 1 {
						pluginOpts = pParts[1]
					}
				}
			}
		}

		hostPart = strings.TrimSuffix(hostPart, "/")
		h, pStr, err := net.SplitHostPort(hostPart)
		if err == nil {
			server = h
			port, _ = strconv.Atoi(pStr)
		} else if idx := strings.LastIndex(hostPart, ":"); idx != -1 {
			server = hostPart[:idx]
			port, _ = strconv.Atoi(hostPart[idx+1:])
		} else {
			server = hostPart
		}
	} else {
		// Legacy format: ss://base64(method:password@server:port)
		decoded, err := decodeBase64Flexible(uriBody)
		if err != nil {
			return config.Outbound{}, fmt.Errorf("invalid shadowsocks URI base64: %w", err)
		}
		decodedStr := string(decoded)
		atIdx := strings.LastIndex(decodedStr, "@")
		if atIdx == -1 {
			return config.Outbound{}, fmt.Errorf("invalid decoded shadowsocks format: %s", decodedStr)
		}
		userPart := decodedStr[:atIdx]
		hostPart := decodedStr[atIdx+1:]

		uParts := strings.SplitN(userPart, ":", 2)
		if len(uParts) == 2 {
			method = uParts[0]
			password = uParts[1]
		}

		var queryStr string
		if qIdx := strings.Index(hostPart, "?"); qIdx != -1 {
			queryStr = hostPart[qIdx+1:]
			hostPart = hostPart[:qIdx]
		}
		if queryStr != "" {
			if values, err := url.ParseQuery(queryStr); err == nil {
				pluginStr := values.Get("plugin")
				if pluginStr != "" {
					pParts := strings.SplitN(pluginStr, ";", 2)
					plugin = pParts[0]
					if len(pParts) > 1 {
						pluginOpts = pParts[1]
					}
				}
			}
		}

		hostPart = strings.TrimSuffix(hostPart, "/")
		h, pStr, err := net.SplitHostPort(hostPart)
		if err == nil {
			server = h
			port, _ = strconv.Atoi(pStr)
		} else if idx := strings.LastIndex(hostPart, ":"); idx != -1 {
			server = hostPart[:idx]
			port, _ = strconv.Atoi(hostPart[idx+1:])
		} else {
			server = hostPart
		}
	}

	if server == "" || port == 0 {
		return config.Outbound{}, fmt.Errorf("failed to extract server/port from ss link")
	}

	if tag == "" {
		tag = fmt.Sprintf("ss-%s:%d", server, port)
	}

	settings := map[string]interface{}{
		"method":   method,
		"password": password,
	}
	if plugin != "" {
		settings["plugin"] = plugin
		if pluginOpts != "" {
			settings["plugin_opts"] = pluginOpts
		}
	}

	return config.Outbound{
		ID:       uuid.New().String(),
		Tag:      tag,
		Type:     "shadowsocks",
		Enabled:  true,
		Server:   server,
		Port:     port,
		Settings: settings,
	}, nil
}

// parseTUICLink parses tuic://uuid:password@server:port?query#name
func parseTUICLink(rawURI string) (config.Outbound, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return config.Outbound{}, fmt.Errorf("invalid tuic URI: %w", err)
	}

	server := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	var uid, pass string
	if u.User != nil {
		uid = u.User.Username()
		pass, _ = u.User.Password()
	}
	if unescaped, err := url.PathUnescape(uid); err == nil {
		uid = unescaped
	}
	if unescaped, err := url.PathUnescape(pass); err == nil {
		pass = unescaped
	}

	tag := u.Fragment
	if tag != "" {
		if unescaped, err := url.QueryUnescape(tag); err == nil {
			tag = unescaped
		}
	}
	if tag == "" {
		tag = fmt.Sprintf("tuic-%s:%d", server, port)
	}

	q := u.Query()
	sni := q.Get("sni")
	alpn := q.Get("alpn")
	cc := q.Get("congestion_controller")
	if cc == "" {
		cc = q.Get("congestion_control")
	}
	if cc == "" {
		cc = q.Get("cc")
	}

	insecure := q.Get("insecure") == "1" ||
		strings.ToLower(q.Get("insecure")) == "true" ||
		q.Get("allowInsecure") == "1" ||
		strings.ToLower(q.Get("allowInsecure")) == "true"

	tlsMap := map[string]interface{}{
		"enabled": true,
	}
	if sni != "" {
		tlsMap["server_name"] = sni
	}
	if alpn != "" {
		tlsMap["alpn"] = splitAndTrim(alpn, ",")
	}
	if insecure {
		tlsMap["insecure"] = true
	}
	if q.Get("disable_sni") == "1" || strings.ToLower(q.Get("disable_sni")) == "true" {
		tlsMap["disable_sni"] = true
	}

	settings := map[string]interface{}{
		"uuid":     uid,
		"password": pass,
		"tls":      tlsMap,
	}
	if cc != "" {
		settings["congestion_control"] = cc
	}
	if q.Get("reduce_rtt") == "1" || strings.ToLower(q.Get("reduce_rtt")) == "true" {
		settings["reduce_rtt"] = true
	}

	return config.Outbound{
		ID:       uuid.New().String(),
		Tag:      tag,
		Type:     "tuic",
		Enabled:  true,
		Server:   server,
		Port:     port,
		Settings: settings,
	}, nil
}
