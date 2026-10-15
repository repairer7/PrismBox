package kernel

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/prismbox/prismbox/internal/config"
)

func TestFindKernelBinary(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-find-kernel-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	binaryName := "sing-box"
	if runtime.GOOS == "windows" {
		binaryName = "sing-box.exe"
	}
	mockBinaryPath := filepath.Join(tempDir, binaryName)

	// Create mock binary file
	if err := os.WriteFile(mockBinaryPath, []byte("#!/bin/sh\necho sing-box version 1.14.3\n"), 0755); err != nil {
		t.Fatalf("failed to write mock binary: %v", err)
	}

	// 1. Test explicit path to mock binary
	found, err := FindKernelBinary(mockBinaryPath)
	if err != nil {
		t.Fatalf("expected to find mock binary at %s, got err: %v", mockBinaryPath, err)
	}
	if filepath.Clean(found) != filepath.Clean(mockBinaryPath) {
		t.Errorf("expected %s, got %s", mockBinaryPath, found)
	}

	// 2. Test explicit non-existent path
	_, err = FindKernelBinary(filepath.Join(tempDir, "does-not-exist"))
	if err == nil {
		t.Error("expected error for non-existent binary path, got nil")
	}
}

func TestVersionParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "sing-box version 1.11.0",
			expected: "1.11.0",
		},
		{
			input:    "sing-box version 1.14.3 (go1.23.0 windows/amd64)\nBuilt with tags: with_quic",
			expected: "1.14.3",
		},
		{
			input:    "sing-box version 1.14.3-beta.1",
			expected: "1.14.3-beta.1",
		},
		{
			input:    "Custom engine Version 2.0.0-rc1",
			expected: "2.0.0-rc1",
		},
		{
			input:    "",
			expected: "unknown",
		},
	}

	for _, tc := range tests {
		actual := ParseVersionString(tc.input)
		if actual != tc.expected {
			t.Errorf("ParseVersionString(%q) = %q; expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestStatusTransitions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-status-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewManager("non-existent-sing-box-path", tempDir)
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}

	// Initial status should be StatusStopped
	if mgr.Status() != StatusStopped {
		t.Errorf("expected initial status %q, got %q", StatusStopped, mgr.Status())
	}
	if mgr.IsRunning() {
		t.Error("expected IsRunning to be false initially")
	}
	if mgr.GetPID() != 0 {
		t.Errorf("expected PID 0 initially, got %d", mgr.GetPID())
	}

	// Calling Stop when stopped
	if err := mgr.Stop(); err != nil {
		t.Errorf("unexpected error stopping stopped manager: %v", err)
	}
	if mgr.Status() != StatusStopped {
		t.Errorf("expected status %q, got %q", StatusStopped, mgr.Status())
	}

	// Trying to start non-existent binary should transition to StatusError
	err = mgr.Start()
	if err == nil {
		t.Error("expected error starting invalid binary, got nil")
	}
	if mgr.Status() != StatusError {
		t.Errorf("expected status %q after failed start, got %q", StatusError, mgr.Status())
	}
	if mgr.GetLastError() == "" {
		t.Error("expected lastError to be populated after failed start")
	}
}

func TestLogRingBuffer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-log-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewManager("", tempDir)

	var receivedMu sync.Mutex
	var received []string
	mgr.OnLog(func(line string) {
		receivedMu.Lock()
		defer receivedMu.Unlock()
		received = append(received, line)
	})

	// Add 1050 log lines
	for i := 0; i < 1050; i++ {
		mgr.appendLog(fmt.Sprintf("log line %04d", i))
	}

	logsAll := mgr.GetLogs(0)
	if len(logsAll) != maxLogLines {
		t.Fatalf("expected ring buffer capped at %d lines, got %d", maxLogLines, len(logsAll))
	}

	// Verify limit behavior
	logsRecent := mgr.GetLogs(5)
	if len(logsRecent) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(logsRecent))
	}

	// Verify listener received messages
	receivedMu.Lock()
	recCount := len(received)
	receivedMu.Unlock()
	if recCount < 1000 {
		t.Errorf("expected listener to receive at least 1000 lines, got %d", recCount)
	}

	// Verify WriteMessage compatibility
	mgr.WriteMessage("INFO", "test platform writer message")
	latest := mgr.GetLogs(1)
	if len(latest) == 0 {
		t.Fatal("expected at least 1 log line")
	}
}

func TestStoreIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-store-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := config.NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	mgr := NewManager("", tempDir)
	mgr.SetStore(store)

	expectedConfigPath := filepath.Join(tempDir, "sing-box-config.json")
	if mgr.configPath != expectedConfigPath {
		t.Errorf("expected configPath %q, got %q", expectedConfigPath, mgr.configPath)
	}
}

func TestValidateConfigData(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-val-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewManager("", tempDir)

	// Valid JSON
	validJSON := []byte(`{"inbounds":[{"type":"mixed","tag":"mixed-in","listen_port":1234}]}`)
	if err := mgr.ValidateConfigData(validJSON); err != nil {
		t.Errorf("expected valid JSON to pass validation, got: %v", err)
	}

	// Invalid JSON
	invalidJSON := []byte(`{"inbounds": [unclosed`)
	if err := mgr.ValidateConfigData(invalidJSON); err == nil {
		t.Error("expected invalid JSON to fail validation, got nil")
	}
}

func TestSafeRestartPreValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prismbox-restart-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := config.NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	mgr := NewManager("non-existent-sing-box-bin", tempDir)
	mgr.SetStore(store)

	// Set a mock running state
	mgr.mu.Lock()
	mgr.status = StatusRunning
	mgr.pid = 99999
	mgr.mu.Unlock()

	// Corrupt store config so generation produces something invalid if forced
	// (or test restart with missing kernel path when config is generated)
	// Default config is valid JSON, so ValidateConfigData passes generic JSON check.
	// But Start will fail since binary doesn't exist.
	// The important part is that if ValidateConfigData fails, running state is NOT stopped.
	mgr.mu.Lock()
	mgr.status = StatusRunning
	mgr.pid = 88888
	mgr.mu.Unlock()

	// If we write an invalid config file and store is nil
	mgr.SetStore(nil)
	invalidFile := filepath.Join(tempDir, "sing-box-config.json")
	_ = os.WriteFile(invalidFile, []byte("{invalid json"), 0644)

	err = mgr.Restart()
	if err == nil {
		t.Error("expected restart to fail with invalid config file, got nil")
	}

	// The running kernel must still have status Running because restart was intercepted!
	if mgr.Status() != StatusRunning {
		t.Errorf("expected running status to be preserved, got %q", mgr.Status())
	}
	if mgr.GetPID() != 88888 {
		t.Errorf("expected PID 88888 to be preserved, got %d", mgr.GetPID())
	}
}

// TestValidateWithOfficialSingBoxBinary validates that generated configs pass official sing-box 1.14.3 check.
func TestValidateWithOfficialSingBoxBinary(t *testing.T) {
	// Look for sing-box or sing-box.exe in workspace root
	binPath := filepath.Join("..", "..", "sing-box.exe")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		binPath = filepath.Join("..", "..", "sing-box")
		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			t.Skip("Official sing-box binary not present in workspace root, skipping live CLI check")
		}
	}

	tempDir, err := os.MkdirTemp("", "prismbox-live-val-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := config.NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	mgr := NewManager(binPath, tempDir)
	mgr.SetStore(store)

	// 1. Validate default generated config (no DNS, TProxy on 12345)
	defaultCfgData, err := config.GenerateSingBoxConfig(store)
	if err != nil {
		t.Fatalf("failed to generate default config: %v", err)
	}
	if err := mgr.ValidateConfigData(defaultCfgData); err != nil {
		t.Fatalf("default config failed official sing-box check: %v", err)
	}

	// 2. Validate with DNS enabled
	cfg := store.GetConfig()
	cfg.DNS = config.DNSConfig{
		Enabled: true,
		Servers: []config.DNSServer{
			{Tag: "google", Address: "https://dns.google/dns-query", Detour: "direct"},
			{Tag: "local", Address: "223.5.5.5", Detour: "direct"},
		},
		Rules: []config.DNSRule{
			{DomainSuffix: []string{".cn", "baidu.com"}, Server: "local"},
		},
		Strategy: "prefer_ipv4",
	}
	withDNSData, err := config.GenerateSingBoxConfigFromModel(cfg)
	if err != nil {
		t.Fatalf("failed to generate with-dns config: %v", err)
	}
	if err := mgr.ValidateConfigData(withDNSData); err != nil {
		t.Fatalf("with-dns config failed official sing-box check: %v", err)
	}
}

