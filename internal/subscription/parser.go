package subscription

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/prismbox/prismbox/internal/config"
)

var defaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// FetchSubscription fetches subscription content from the given URL, sniffs its format
// (Clash YAML vs Base64/plain proxy links), and parses it into config.Outbound items.
func FetchSubscription(subURL string) ([]config.Outbound, error) {
	subURL = strings.TrimSpace(subURL)
	if subURL == "" {
		return nil, fmt.Errorf("subscription url cannot be empty")
	}

	req, err := http.NewRequest(http.MethodGet, subURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	// Use Clash-compatible User-Agent to ensure providers return the full node list
	req.Header.Set("User-Agent", "clash-verge/v1.7.5 ClashMeta/1.18.0 PrismBox/1.0")
	req.Header.Set("Accept", "*/*")

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("subscription request returned HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Limit body read to 10MB to avoid OOM on huge responses
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription response: %w", err)
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	outbounds, err := parseSubscriptionWithContentType(body, contentType)
	if err != nil {
		return nil, err
	}

	for i := range outbounds {
		outbounds[i].SubscriptionURL = subURL
	}

	return outbounds, nil
}

// ParseSubscription is an alias for FetchSubscription for backward compatibility.
func ParseSubscription(subURL string) ([]config.Outbound, error) {
	return FetchSubscription(subURL)
}

// ParseSubscriptionData parses raw subscription content by auto-detecting Clash vs Base64 format.
func ParseSubscriptionData(data []byte) ([]config.Outbound, error) {
	return parseSubscriptionWithContentType(data, "")
}

// parseSubscriptionWithContentType parses content taking optional HTTP Content-Type into account.
func parseSubscriptionWithContentType(data []byte, contentType string) ([]config.Outbound, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty subscription content")
	}

	// Strip UTF-8 BOM
	trimmed = bytes.TrimPrefix(trimmed, []byte("\xef\xbb\xbf"))

	// Check Content-Type hint
	isYAMLContentType := strings.Contains(contentType, "yaml") || strings.Contains(contentType, "yml")

	// Detect Clash YAML signatures
	lower := bytes.ToLower(trimmed)
	isClashCandidate := isYAMLContentType ||
		bytes.HasPrefix(lower, []byte("proxies:")) ||
		bytes.Contains(lower, []byte("\nproxies:")) ||
		bytes.Contains(lower, []byte("\r\nproxies:")) ||
		bytes.Contains(lower, []byte("proxy-groups:"))

	// If candidate looks like Clash, attempt Clash parse first
	if isClashCandidate {
		if outbounds, err := ParseClash(trimmed); err == nil && len(outbounds) > 0 {
			return outbounds, nil
		}
	}

	// Attempt Base64 / plain text proxy links
	if outbounds, err := ParseBase64(trimmed); err == nil && len(outbounds) > 0 {
		return outbounds, nil
	}

	// Fallback to Clash YAML if not tried yet or if base64 failed
	if !isClashCandidate {
		if outbounds, err := ParseClash(trimmed); err == nil && len(outbounds) > 0 {
			return outbounds, nil
		}
	}

	return nil, fmt.Errorf("failed to parse subscription: unrecognized format")
}
