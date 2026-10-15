package kernel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prismbox/prismbox/internal/config"
)

const (
	StatusRunning = "running"
	StatusStopped = "stopped"
	StatusError   = "error"
	maxLogLines   = 1000
)

// Manager coordinates the external child process lifecycle, logging, and health of sing-box.
type Manager struct {
	mu              sync.RWMutex
	logMu           sync.Mutex
	kernelPath      string
	dataDir         string
	configPath      string
	cmd             *exec.Cmd
	cancel          context.CancelFunc
	status          string
	lastError       string
	pid             int
	version         string
	logs            []string
	createdAt       time.Time
	crashCount      int
	store           *config.Store
	logListeners    []func(string)
	statusListeners []func(string)
	stopAutoRestart chan struct{}
}

// FindKernelBinary discovers the sing-box binary by inspecting the provided hint,
// checking ./sing-box (and ./sing-box.exe on Windows), the executable directory, and $PATH.
func FindKernelBinary(hint string) (string, error) {
	hint = strings.TrimSpace(hint)

	// 1. Explicit hint provided
	if hint != "" && hint != "auto" && hint != "embedded" {
		if path, err := verifyExecutable(hint); err == nil {
			return path, nil
		}
		if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(hint), ".exe") {
			if path, err := verifyExecutable(hint + ".exe"); err == nil {
				return path, nil
			}
		}
		if lp, err := exec.LookPath(hint); err == nil {
			if abs, err := filepath.Abs(lp); err == nil {
				return abs, nil
			}
			return lp, nil
		}
		return "", fmt.Errorf("kernel binary not found at %s", hint)
	}

	// 2. Check current working directory: ./sing-box or ./sing-box.exe
	candidates := []string{"sing-box"}
	if runtime.GOOS == "windows" {
		candidates = []string{"sing-box.exe", "sing-box"}
	}

	for _, name := range candidates {
		relPath := filepath.Join(".", name)
		if path, err := verifyExecutable(relPath); err == nil {
			return path, nil
		}
	}

	// 3. Check directory of current executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		for _, name := range candidates {
			target := filepath.Join(exeDir, name)
			if path, err := verifyExecutable(target); err == nil {
				return path, nil
			}
		}
	}

	// 4. Check system PATH
	for _, name := range candidates {
		if lp, err := exec.LookPath(name); err == nil {
			if abs, err := filepath.Abs(lp); err == nil {
				return abs, nil
			}
			return lp, nil
		}
	}

	return "", fmt.Errorf("sing-box kernel binary not found in ./sing-box or PATH")
}

func verifyExecutable(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return "", fmt.Errorf("%s is a directory", abs)
	}
	return abs, nil
}

// NewManager creates an external child-process sing-box Manager.
func NewManager(kernelPath, dataDir string) *Manager {
	configPath := filepath.Join(dataDir, "sing-box-config.json")

	resolvedPath := kernelPath
	if resolvedPath == "" || resolvedPath == "auto" || resolvedPath == "embedded" {
		if found, err := FindKernelBinary(""); err == nil {
			resolvedPath = found
		} else {
			resolvedPath = "sing-box"
		}
	}

	mgr := &Manager{
		kernelPath:      resolvedPath,
		dataDir:         dataDir,
		configPath:      configPath,
		status:          StatusStopped,
		logs:            make([]string, 0, maxLogLines),
		logListeners:    make([]func(string), 0),
		statusListeners: make([]func(string), 0),
	}

	mgr.appendLog("[kernel] sing-box child-process manager initialized")

	// Detect initial version in background/initially
	if v := mgr.DetectVersion(); v != "" {
		mgr.version = v
	}

	return mgr
}

// SetStore assigns the configuration store used to generate sing-box configs.
func (m *Manager) SetStore(store *config.Store) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = store
}

// DetectVersion runs `sing-box version` and parses the version string.
func (m *Manager) DetectVersion() string {
	m.mu.RLock()
	kernelPath := m.kernelPath
	m.mu.RUnlock()

	if kernelPath == "" {
		return "unknown"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, kernelPath, "version")
	output, err := cmd.Output()
	if err != nil {
		m.mu.RLock()
		ver := m.version
		m.mu.RUnlock()
		if ver != "" {
			return ver
		}
		return "unknown"
	}

	parsed := ParseVersionString(string(output))
	m.mu.Lock()
	m.version = parsed
	m.mu.Unlock()

	return parsed
}

// ParseVersionString parses version output from `sing-box version`.
func ParseVersionString(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "sing-box version") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2]
			}
		}
	}

	for _, line := range lines {
		fields := strings.Fields(line)
		for i, field := range fields {
			if strings.EqualFold(field, "version") && i+1 < len(fields) {
				return fields[i+1]
			}
		}
	}

	if len(lines) > 0 && lines[0] != "" {
		return lines[0]
	}

	return "unknown"
}

// CheckConfig runs `sing-box check -c <configPath>` and returns an error with stderr output if validation fails.
func (m *Manager) CheckConfig(configPath string) error {
	m.mu.RLock()
	kernelPath := m.kernelPath
	m.mu.RUnlock()

	if kernelPath == "" {
		return fmt.Errorf("kernel path is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, kernelPath, "check", "-c", configPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		if outStr != "" {
			return fmt.Errorf("%s", outStr)
		}
		return fmt.Errorf("validation error: %w", err)
	}

	return nil
}

// Start generates the config, validates it via CheckConfig, and spawns the sing-box process.
func (m *Manager) Start() error {
	m.mu.Lock()

	if m.status == StatusRunning && m.cmd != nil && m.cmd.Process != nil {
		m.mu.Unlock()
		return nil
	}

	// 1. Resolve binary if needed
	if _, err := verifyExecutable(m.kernelPath); err != nil {
		if found, findErr := FindKernelBinary(""); findErr == nil {
			m.kernelPath = found
		} else {
			m.status = StatusError
			m.lastError = fmt.Sprintf("cannot start sing-box: %v", err)
			m.mu.Unlock()
			m.appendLog(fmt.Sprintf("[kernel] error: %s", m.lastError))
			m.notifyStatus(StatusError)
			return fmt.Errorf("%s", m.lastError)
		}
	}

	// 2. Generate configuration from store
	var cfgData []byte
	var err error
	if m.store != nil {
		cfgData, err = config.GenerateSingBoxConfig(m.store)
		if err != nil {
			m.status = StatusError
			m.lastError = fmt.Sprintf("failed to generate sing-box config: %v", err)
			m.mu.Unlock()
			m.appendLog(fmt.Sprintf("[kernel] error: %s", m.lastError))
			m.notifyStatus(StatusError)
			return fmt.Errorf("%s", m.lastError)
		}

		if err := os.WriteFile(m.configPath, cfgData, 0644); err != nil {
			m.status = StatusError
			m.lastError = fmt.Sprintf("failed to write config file %s: %v", m.configPath, err)
			m.mu.Unlock()
			m.appendLog(fmt.Sprintf("[kernel] error: %s", m.lastError))
			m.notifyStatus(StatusError)
			return fmt.Errorf("%s", m.lastError)
		}
	} else if data, readErr := os.ReadFile(m.configPath); readErr == nil {
		cfgData = data
	} else {
		m.status = StatusError
		m.lastError = fmt.Sprintf("sing-box configuration not found at %s", m.configPath)
		m.mu.Unlock()
		m.appendLog(fmt.Sprintf("[kernel] error: %s", m.lastError))
		m.notifyStatus(StatusError)
		return fmt.Errorf("%s", m.lastError)
	}

	// 3. Validate configuration with CheckConfig before spawning process
	configPath := m.configPath
	m.mu.Unlock()

	if err := m.CheckConfig(configPath); err != nil {
		m.mu.Lock()
		m.status = StatusError
		m.lastError = fmt.Sprintf("config check failed: %v", err)
		m.mu.Unlock()
		m.appendLog(fmt.Sprintf("[kernel] check failed: %v", err))
		m.notifyStatus(StatusError)
		return fmt.Errorf("config check failed: %w", err)
	}

	m.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, m.kernelPath, "run", "-c", m.configPath)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		m.status = StatusError
		m.lastError = fmt.Sprintf("failed to create stdout pipe: %v", err)
		m.mu.Unlock()
		m.appendLog(m.lastError)
		m.notifyStatus(StatusError)
		return err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		m.status = StatusError
		m.lastError = fmt.Sprintf("failed to create stderr pipe: %v", err)
		m.mu.Unlock()
		m.appendLog(m.lastError)
		m.notifyStatus(StatusError)
		return err
	}

	if err := cmd.Start(); err != nil {
		cancel()
		m.status = StatusError
		m.lastError = fmt.Sprintf("failed to start sing-box process: %v", err)
		m.mu.Unlock()
		m.appendLog(m.lastError)
		m.notifyStatus(StatusError)
		return err
	}

	m.cmd = cmd
	m.cancel = cancel
	m.pid = cmd.Process.Pid
	m.status = StatusRunning
	m.lastError = ""
	m.createdAt = time.Now()
	m.crashCount = 0
	m.stopAutoRestart = make(chan struct{})
	pid := m.pid
	m.mu.Unlock()

	m.appendLog(fmt.Sprintf("[sing-box] started process (PID: %d)", pid))
	m.notifyStatus(StatusRunning)

	// Stream stdout & stderr into log ring buffer
	go m.streamLogs(stdoutPipe)
	go m.streamLogs(stderrPipe)

	// Supervise process lifecycle and handle auto-restart on unexpected crashes
	go m.supervise(cmd, cancel)

	return nil
}

func (m *Manager) streamLogs(pipe interface{ Read([]byte) (int, error) }) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			m.appendLog(line)
		}
	}
}

func (m *Manager) supervise(cmd *exec.Cmd, cancel context.CancelFunc) {
	waitErr := cmd.Wait()
	cancel()

	m.mu.Lock()
	// Ignore if this cmd is no longer the active one
	if m.cmd != cmd {
		m.mu.Unlock()
		return
	}

	wasRunning := (m.status == StatusRunning)
	if wasRunning && time.Since(m.createdAt) > 30*time.Second {
		m.crashCount = 0
	}
	m.cmd = nil
	m.pid = 0
	stopCh := m.stopAutoRestart

	if !wasRunning {
		// Normal stop was requested via Stop()
		m.status = StatusStopped
		m.mu.Unlock()
		m.notifyStatus(StatusStopped)
		return
	}

	// Unexpected exit (crash)
	m.crashCount++
	crashCount := m.crashCount
	m.status = StatusError
	if waitErr != nil {
		m.lastError = fmt.Sprintf("sing-box process exited unexpectedly: %v", waitErr)
	} else {
		m.lastError = "sing-box process exited unexpectedly"
	}
	m.mu.Unlock()

	m.appendLog(fmt.Sprintf("[kernel] %s (crash count: %d)", m.lastError, crashCount))
	m.notifyStatus(StatusError)

	if crashCount > 5 {
		errMsg := fmt.Sprintf("sing-box crashed %d consecutive times; auto-restart halted", crashCount)
		m.mu.Lock()
		m.lastError = errMsg
		m.mu.Unlock()
		m.appendLog(fmt.Sprintf("[kernel] %s", errMsg))
		return
	}

	// Exponential backoff: 1s, 2s, 4s, 8s, 16s (max 30s)
	backoffSec := 1 << (crashCount - 1)
	if backoffSec > 30 {
		backoffSec = 30
	}
	backoff := time.Duration(backoffSec) * time.Second
	m.appendLog(fmt.Sprintf("[kernel] restarting in %v (attempt %d/5)...", backoff, crashCount))

	select {
	case <-time.After(backoff):
		m.mu.Lock()
		shouldRestart := (m.crashCount > 0 && m.status == StatusError)
		m.mu.Unlock()
		if shouldRestart {
			if err := m.Start(); err != nil {
				m.appendLog(fmt.Sprintf("[kernel] auto-restart failed: %v", err))
			}
		}
	case <-stopCh:
		return
	}
}

// Stop sends SIGTERM (or interrupt) to process, waits up to 5s, then SIGKILL.
func (m *Manager) Stop() error {
	m.mu.Lock()
	if m.stopAutoRestart != nil {
		select {
		case <-m.stopAutoRestart:
		default:
			close(m.stopAutoRestart)
		}
	}
	m.crashCount = 0

	cmd := m.cmd
	cancel := m.cancel
	if cmd == nil || cmd.Process == nil {
		m.status = StatusStopped
		m.pid = 0
		m.mu.Unlock()
		m.notifyStatus(StatusStopped)
		return nil
	}

	// Mark status as stopped immediately so supervise knows this was intentional
	m.status = StatusStopped
	m.mu.Unlock()

	m.appendLog("[kernel] stopping sing-box process...")

	done := make(chan struct{})
	go func() {
		terminateProcess(cmd.Process)
		_ = cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		m.appendLog("[kernel] process did not terminate within 5s, sending SIGKILL...")
		_ = cmd.Process.Kill()
		<-done
	}

	if cancel != nil {
		cancel()
	}

	m.mu.Lock()
	m.cmd = nil
	m.pid = 0
	m.status = StatusStopped
	m.lastError = ""
	m.mu.Unlock()

	m.appendLog("[sing-box] process stopped cleanly")
	m.notifyStatus(StatusStopped)
	return nil
}

func terminateProcess(p *os.Process) {
	if p == nil {
		return
	}
	if runtime.GOOS == "windows" {
		// Windows does not support SIGTERM; kill process directly
		_ = p.Kill()
		return
	}
	// On Unix/Linux, send SIGTERM
	if err := p.Signal(syscall.SIGTERM); err != nil {
		_ = p.Kill()
	}
}

// ValidateConfigData writes config data to a temporary file and invokes `sing-box check -c <tempFile>`.
// If the sing-box binary cannot be found or executed, it performs basic JSON syntax validation.
func (m *Manager) ValidateConfigData(data []byte) error {
	m.mu.RLock()
	kernelPath := m.kernelPath
	dataDir := m.dataDir
	m.mu.RUnlock()

	// Generic JSON validation first
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid JSON syntax: %w", err)
	}

	// Resolve kernel executable if possible
	resolved := kernelPath
	if resolved == "" {
		if found, err := FindKernelBinary(""); err == nil {
			resolved = found
		}
	} else if _, err := verifyExecutable(resolved); err != nil {
		if found, findErr := FindKernelBinary(""); findErr == nil {
			resolved = found
		} else {
			resolved = ""
		}
	}

	if resolved == "" {
		// Kernel binary not present yet; basic JSON validation passed
		return nil
	}

	// Write to temporary check file
	tmpFile := filepath.Join(dataDir, fmt.Sprintf(".sing-box-check-%d.tmp.json", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp check file: %w", err)
	}
	defer os.Remove(tmpFile)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, resolved, "check", "-c", tmpFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		if outStr != "" {
			return fmt.Errorf("%s", outStr)
		}
		return fmt.Errorf("sing-box check failed: %w", err)
	}

	return nil
}

// Restart performs safe configuration pre-validation before stopping the running kernel.
// If the new configuration fails validation via `sing-box check -c`, restart is intercepted
// and the existing kernel continues to run undisturbed.
func (m *Manager) Restart() error {
	// 1. Generate new configuration from store
	var cfgData []byte
	var err error
	if m.store != nil {
		cfgData, err = config.GenerateSingBoxConfig(m.store)
		if err != nil {
			m.appendLog(fmt.Sprintf("[kernel] restart intercepted: config generation failed: %v", err))
			return fmt.Errorf("config generation failed: %w (running kernel kept intact)", err)
		}
	} else if data, readErr := os.ReadFile(m.configPath); readErr == nil {
		cfgData = data
	} else {
		return fmt.Errorf("sing-box configuration not found at %s (running kernel kept intact)", m.configPath)
	}

	// 2. Pre-check configuration using `sing-box check -c` before touching the running process!
	if err := m.ValidateConfigData(cfgData); err != nil {
		errMsg := fmt.Sprintf("sing-box check failed: %v", err)
		m.appendLog(fmt.Sprintf("[kernel] restart intercepted: %s", errMsg))
		return fmt.Errorf("configuration pre-check failed: %w (running kernel kept intact)", err)
	}

	// 3. Write verified config
	if err := os.WriteFile(m.configPath, cfgData, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", m.configPath, err)
	}

	// 4. Safe to stop existing process
	if err := m.Stop(); err != nil {
		m.appendLog(fmt.Sprintf("[kernel] warning during stop before restart: %v", err))
	}

	// 5. Start with new validated config
	return m.Start()
}

// Status returns current lifecycle status ("running", "stopped", "error").
func (m *Manager) Status() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// IsRunning reports whether the sing-box engine child process is active.
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status == StatusRunning && m.pid > 0
}

// GetPID returns the current process ID of sing-box or 0.
func (m *Manager) GetPID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pid
}

// GetVersion returns the detected sing-box version string.
func (m *Manager) GetVersion() string {
	m.mu.RLock()
	ver := m.version
	m.mu.RUnlock()
	if ver != "" && ver != "unknown" {
		return ver
	}
	return m.DetectVersion()
}

// GetLastError returns the most recent error message, if any.
func (m *Manager) GetLastError() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastError
}

// GetLogs returns up to limit recent log entries from the ring buffer.
func (m *Manager) GetLogs(limit int) []string {
	m.logMu.Lock()
	defer m.logMu.Unlock()

	if limit <= 0 || limit > len(m.logs) {
		limit = len(m.logs)
	}
	result := make([]string, limit)
	copy(result, m.logs[len(m.logs)-limit:])
	return result
}

// GetUptime returns the elapsed duration since the current process was started.
func (m *Manager) GetUptime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status != StatusRunning || m.createdAt.IsZero() {
		return 0
	}
	return time.Since(m.createdAt)
}

// GetCrashCount returns the current consecutive crash count.
func (m *Manager) GetCrashCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.crashCount
}

// OnLog registers a listener for real-time log output.
func (m *Manager) OnLog(listener func(string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logListeners = append(m.logListeners, listener)
}

// OnStatusChange registers a listener for kernel status changes.
func (m *Manager) OnStatusChange(listener func(string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusListeners = append(m.statusListeners, listener)
}

func (m *Manager) notifyStatus(status string) {
	m.mu.RLock()
	listeners := make([]func(string), len(m.statusListeners))
	copy(listeners, m.statusListeners)
	m.mu.RUnlock()

	for _, fn := range listeners {
		go fn(status)
	}
}

// WriteMessage provides compatibility for platform log interfaces.
func (m *Manager) WriteMessage(level interface{}, message ...string) {
	if len(message) > 0 {
		m.appendLog(fmt.Sprintf("[%v] %s", level, message[0]))
	} else if s, ok := level.(string); ok {
		m.appendLog(s)
	} else {
		m.appendLog(fmt.Sprintf("%v", level))
	}
}

func (m *Manager) appendLog(msg string) {
	m.logMu.Lock()
	ts := time.Now().Format("2006-01-02 15:04:05")
	entry := msg
	if !strings.HasPrefix(msg, "[20") {
		entry = fmt.Sprintf("[%s] %s", ts, msg)
	}

	if len(m.logs) >= maxLogLines {
		m.logs = m.logs[1:]
	}
	m.logs = append(m.logs, entry)

	m.mu.RLock()
	listeners := make([]func(string), len(m.logListeners))
	copy(listeners, m.logListeners)
	m.mu.RUnlock()
	m.logMu.Unlock()

	for _, fn := range listeners {
		go fn(entry)
	}
}
