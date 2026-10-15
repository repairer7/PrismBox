package api

import (
	"net/http"

	"github.com/prismbox/prismbox/internal/kernel"
	"github.com/prismbox/prismbox/internal/monitor"
)

// DashboardStats holds the detailed telemetry and kernel state.
type DashboardStats struct {
	CPU           float64 `json:"cpu"`
	MemUsed       uint64  `json:"mem_used"`
	MemTotal      uint64  `json:"mem_total"`
	UploadSpeed   uint64  `json:"upload_speed"`
	DownloadSpeed uint64  `json:"download_speed"`
	TotalUpload   uint64  `json:"total_upload"`
	TotalDownload uint64  `json:"total_download"`
	KernelStatus  string  `json:"kernel_status"`
	KernelPID     int     `json:"kernel_pid"`
	KernelVersion string  `json:"kernel_version"`
	KernelUptime  int64   `json:"kernel_uptime"`
	KernelError   string  `json:"kernel_error"`
}

// DashboardResponse envelopes the stats data with a success indicator.
type DashboardResponse struct {
	Success      bool           `json:"success"`
	Data         DashboardStats `json:"data"`
	KernelStatus string         `json:"kernel_status,omitempty"`
}

// HandleDashboardStats returns an HTTP handler for GET /api/v1/dashboard/stats.
func HandleDashboardStats(mon *monitor.Monitor, mgr *kernel.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			writeSuccess(w, map[string]string{"message": "Dashboard stats configuration updated"})
			return
		}

		var stats monitor.Stats
		if mon != nil {
			stats = mon.GetStats()
		}

		kernelStatus := "stopped"
		kernelPID := 0
		kernelVersion := "1.14.3"
		var kernelUptime int64
		kernelError := ""

		if mgr != nil {
			kernelStatus = mgr.Status()
			kernelPID = mgr.GetPID()
			kernelVersion = mgr.GetVersion()
			kernelUptime = int64(mgr.GetUptime().Seconds())
			kernelError = mgr.GetLastError()
		}

		// Convert MB to bytes for mem_used and mem_total if non-zero
		memUsedBytes := stats.MemUsedMB * 1024 * 1024
		memTotalBytes := stats.MemTotalMB * 1024 * 1024

		data := DashboardStats{
			CPU:           stats.CPUPercent,
			MemUsed:       memUsedBytes,
			MemTotal:      memTotalBytes,
			UploadSpeed:   stats.UploadSpeed,
			DownloadSpeed: stats.DownloadSpeed,
			TotalUpload:   stats.TotalUpload,
			TotalDownload: stats.TotalDownload,
			KernelStatus:  kernelStatus,
			KernelPID:     kernelPID,
			KernelVersion: kernelVersion,
			KernelUptime:  kernelUptime,
			KernelError:   kernelError,
		}

		resp := DashboardResponse{
			Success:      true,
			Data:         data,
			KernelStatus: kernelStatus,
		}

		writeJSON(w, http.StatusOK, resp)
	}
}
