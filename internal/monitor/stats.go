package monitor

import (
	"bufio"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Stats encapsulates system performance metrics and network traffic rates.
type Stats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemUsedMB     uint64  `json:"mem_used_mb"`
	MemTotalMB    uint64  `json:"mem_total_mb"`
	UploadSpeed   uint64  `json:"upload_speed"`
	DownloadSpeed uint64  `json:"download_speed"`
	TotalUpload   uint64  `json:"total_upload"`
	TotalDownload uint64  `json:"total_download"`
	Uptime        uint64  `json:"uptime"`
}

// Monitor periodically gathers CPU, memory, and network statistics from /proc on Linux,
// and produces realistic simulated metrics on other operating systems (Windows, macOS).
type Monitor struct {
	mu           sync.RWMutex
	stats        Stats
	startTime    time.Time
	stopCh       chan struct{}
	running      bool
	prevCPUBusy  uint64
	prevCPUTotal uint64
	hasPrevCPU   bool
	prevNetRx    uint64
	prevNetTx    uint64
	hasPrevNet   bool
	cumUpload    uint64
	cumDownload  uint64
	mockRand     *rand.Rand
}

// NewMonitor instantiates a new background system monitor.
func NewMonitor() *Monitor {
	m := &Monitor{
		startTime: time.Now(),
		stopCh:    make(chan struct{}),
		mockRand:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Prime initial baseline
	m.sample()
	m.Start()
	return m
}

// Start begins periodic metric sampling every 1 second in a background goroutine.
func (m *Monitor) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	go m.run()
}

// Stop cleanly terminates the background sampling loop.
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return
	}
	m.running = false
	close(m.stopCh)
}

// GetStats returns a thread-safe snapshot of the current system stats.
func (m *Monitor) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

func (m *Monitor) run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.sample()
		}
	}
}

func (m *Monitor) sample() {
	m.mu.Lock()
	defer m.mu.Unlock()

	uptime := uint64(time.Since(m.startTime).Seconds())

	if runtime.GOOS == "linux" && procAvailable() {
		cpuPercent := m.readLinuxCPU()
		memUsed, memTotal := m.readLinuxMemory()
		upSpeed, downSpeed := m.readLinuxNetwork()

		m.stats = Stats{
			CPUPercent:    cpuPercent,
			MemUsedMB:     memUsed,
			MemTotalMB:    memTotal,
			UploadSpeed:   upSpeed,
			DownloadSpeed: downSpeed,
			TotalUpload:   m.cumUpload,
			TotalDownload: m.cumDownload,
			Uptime:        uptime,
		}
		return
	}

	// Non-Linux or container without /proc: generate realistic mock telemetry
	m.sampleMock(uptime)
}

func procAvailable() bool {
	_, err := os.Stat("/proc/stat")
	return err == nil
}

func (m *Monitor) sampleMock(uptime uint64) {
	// Simulated CPU usage around 1.0% - 4.5%
	cpu := 1.0 + m.mockRand.Float64()*3.5

	// Memory usage using runtime allocator as baseline + realistic cache simulation
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	allocMB := memStats.Alloc / (1024 * 1024)
	usedMB := 48 + allocMB + uint64(m.mockRand.Intn(16))
	totalMB := uint64(1024) // Standard reference 1GB container baseline

	// Simulated bandwidth fluctuations
	upSpeed := uint64(10240 + m.mockRand.Intn(40960))    // 10 - 50 KB/s
	downSpeed := uint64(25600 + m.mockRand.Intn(120000)) // 25 - 145 KB/s

	m.cumUpload += upSpeed
	m.cumDownload += downSpeed

	m.stats = Stats{
		CPUPercent:    float64(int(cpu*10)) / 10.0,
		MemUsedMB:     usedMB,
		MemTotalMB:    totalMB,
		UploadSpeed:   upSpeed,
		DownloadSpeed: downSpeed,
		TotalUpload:   m.cumUpload,
		TotalDownload: m.cumDownload,
		Uptime:        uptime,
	}
}

func (m *Monitor) readLinuxCPU() float64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0.0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return 0.0
			}

			var total uint64
			var idle uint64
			var iowait uint64

			for i := 1; i < len(fields); i++ {
				val, _ := strconv.ParseUint(fields[i], 10, 64)
				total += val
				if i == 4 {
					idle = val
				} else if i == 5 {
					iowait = val
				}
			}

			busy := total - idle - iowait

			if !m.hasPrevCPU {
				m.prevCPUBusy = busy
				m.prevCPUTotal = total
				m.hasPrevCPU = true
				return 0.0
			}

			deltaTotal := total - m.prevCPUTotal
			deltaBusy := busy - m.prevCPUBusy

			m.prevCPUBusy = busy
			m.prevCPUTotal = total

			if deltaTotal == 0 {
				return 0.0
			}

			percent := (float64(deltaBusy) / float64(deltaTotal)) * 100.0
			if percent < 0.0 {
				percent = 0.0
			} else if percent > 100.0 {
				percent = 100.0
			}
			return percent
		}
	}

	return 0.0
}

func (m *Monitor) readLinuxMemory() (usedMB uint64, totalMB uint64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var memTotalKB uint64
	var memAvailKB uint64
	var memFreeKB uint64
	var buffersKB uint64
	var cachedKB uint64
	hasAvail := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) == 0 {
			continue
		}

		val, _ := strconv.ParseUint(fields[0], 10, 64)

		switch key {
		case "MemTotal":
			memTotalKB = val
		case "MemAvailable":
			memAvailKB = val
			hasAvail = true
		case "MemFree":
			memFreeKB = val
		case "Buffers":
			buffersKB = val
		case "Cached":
			cachedKB = val
		}
	}

	totalMB = memTotalKB / 1024

	if hasAvail {
		if memTotalKB > memAvailKB {
			usedMB = (memTotalKB - memAvailKB) / 1024
		}
	} else {
		freeTotal := memFreeKB + buffersKB + cachedKB
		if memTotalKB > freeTotal {
			usedMB = (memTotalKB - freeTotal) / 1024
		}
	}

	return usedMB, totalMB
}

func (m *Monitor) readLinuxNetwork() (uploadSpeed uint64, downloadSpeed uint64) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var currentRx uint64
	var currentTx uint64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) >= 9 {
			rx, _ := strconv.ParseUint(fields[0], 10, 64)
			tx, _ := strconv.ParseUint(fields[8], 10, 64)
			currentRx += rx
			currentTx += tx
		}
	}

	if !m.hasPrevNet {
		m.prevNetRx = currentRx
		m.prevNetTx = currentTx
		m.hasPrevNet = true
		return 0, 0
	}

	var deltaRx uint64
	var deltaTx uint64

	if currentRx >= m.prevNetRx {
		deltaRx = currentRx - m.prevNetRx
	}
	if currentTx >= m.prevNetTx {
		deltaTx = currentTx - m.prevNetTx
	}

	m.prevNetRx = currentRx
	m.prevNetTx = currentTx

	m.cumDownload += deltaRx
	m.cumUpload += deltaTx

	return deltaTx, deltaRx
}
