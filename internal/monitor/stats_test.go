package monitor

import (
	"sync"
	"testing"
	"time"
)

func TestMonitorLifecycle(t *testing.T) {
	mon := NewMonitor()
	if mon == nil {
		t.Fatal("expected non-nil Monitor")
	}

	stats := mon.GetStats()
	if stats.Uptime > 10 {
		t.Errorf("unexpected initial uptime: %d", stats.Uptime)
	}

	time.Sleep(50 * time.Millisecond)
	mon.Stop()

	// Calling Stop multiple times should be safe
	mon.Stop()
	mon.Stop()

	// Restarting after stop
	mon.Start()
	stats2 := mon.GetStats()
	if stats2.MemTotalMB == 0 {
		t.Errorf("expected non-zero MemTotalMB, got %d", stats2.MemTotalMB)
	}
	mon.Stop()
}

func TestMonitorConcurrentStats(t *testing.T) {
	mon := NewMonitor()
	defer mon.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				s := mon.GetStats()
				if s.CPUPercent < 0 || s.CPUPercent > 100 {
					t.Errorf("unexpected CPUPercent: %f", s.CPUPercent)
				}
				time.Sleep(2 * time.Millisecond)
			}
		}()
	}
	wg.Wait()
}
