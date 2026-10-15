package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prismbox/prismbox/internal/kernel"
	"github.com/prismbox/prismbox/internal/monitor"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS: allow any origin
	},
}

// Client represents a single connected WebSocket client.
type Client struct {
	hub      *WSHub
	conn     *websocket.Conn
	send     chan []byte
	closed   bool
	closedMu sync.Mutex
}

func (c *Client) closeSend() {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

// WSHub maintains active client connections and broadcasts real-time system stats,
// kernel status updates, and log entries.
type WSHub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	monitor    *monitor.Monitor
	manager    *kernel.Manager
	stopCh     chan struct{}
	running    bool
}

// NewWSHub instantiates a new WebSocket hub connected to the monitor and kernel manager.
func NewWSHub(mon *monitor.Monitor, mgr *kernel.Manager) *WSHub {
	hub := &WSHub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		monitor:    mon,
		manager:    mgr,
		stopCh:     make(chan struct{}),
	}

	if mgr != nil {
		mgr.OnLog(func(line string) {
			hub.BroadcastLog(line)
		})
		mgr.OnStatusChange(func(status string) {
			hub.BroadcastStatus(status)
		})
	}

	return hub
}

// Start launches the hub processing loops.
func (h *WSHub) Start() {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()

	go h.run()
	go h.statsBroadcaster()
}

// Stop cleanly terminates the hub and closes all client connections.
func (h *WSHub) Stop() {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return
	}
	h.running = false
	close(h.stopCh)

	for client := range h.clients {
		delete(h.clients, client)
		client.closeSend()
		_ = client.conn.Close()
	}
	h.mu.Unlock()
}

// BroadcastLog sends a realtime log entry to all connected WebSocket clients.
func (h *WSHub) BroadcastLog(line string) {
	h.mu.RLock()
	clientCount := len(h.clients)
	running := h.running
	h.mu.RUnlock()

	if !running || clientCount == 0 {
		return
	}

	payload := map[string]interface{}{
		"type":      "log",
		"message":   line,
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	select {
	case h.broadcast <- data:
	default:
	}
}

// BroadcastStatus sends a realtime kernel status change event to connected clients.
func (h *WSHub) BroadcastStatus(status string) {
	h.mu.RLock()
	clientCount := len(h.clients)
	running := h.running
	h.mu.RUnlock()

	if !running || clientCount == 0 {
		return
	}

	var pid int
	var ver string
	var uptime uint64
	if h.manager != nil {
		pid = h.manager.GetPID()
		ver = h.manager.GetVersion()
		uptime = uint64(h.manager.GetUptime().Seconds())
	}

	payload := map[string]interface{}{
		"type":   "kernel_status",
		"status": status,
		"kernel": map[string]interface{}{
			"status":  status,
			"pid":     pid,
			"version": ver,
			"uptime":  uptime,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	select {
	case h.broadcast <- data:
	default:
	}
}

func (h *WSHub) run() {
	for {
		select {
		case <-h.stopCh:
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.closeSend()
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// statsBroadcaster periodically gathers metrics and dispatches them via WebSocket.
func (h *WSHub) statsBroadcaster() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.mu.RLock()
			clientCount := len(h.clients)
			h.mu.RUnlock()

			if clientCount == 0 {
				continue
			}

			var stats monitor.Stats
			if h.monitor != nil {
				stats = h.monitor.GetStats()
			}

			kernelStatus := "stopped"
			var kernelPID int
			var kernelVersion string
			var kernelUptime uint64
			if h.manager != nil {
				kernelStatus = h.manager.Status()
				kernelPID = h.manager.GetPID()
				kernelVersion = h.manager.GetVersion()
				kernelUptime = uint64(h.manager.GetUptime().Seconds())
			}

			payload := map[string]interface{}{
				"type":        "stats",
				"cpu":         stats.CPUPercent,
				"cpu_percent": stats.CPUPercent,
				"memory": map[string]interface{}{
					"used":  stats.MemUsedMB * 1024 * 1024,
					"total": stats.MemTotalMB * 1024 * 1024,
				},
				"mem_used_mb":  stats.MemUsedMB,
				"mem_total_mb": stats.MemTotalMB,
				"speed": map[string]interface{}{
					"upload":   stats.UploadSpeed,
					"download": stats.DownloadSpeed,
				},
				"upload_speed":   stats.UploadSpeed,
				"download_speed": stats.DownloadSpeed,
				"traffic": map[string]interface{}{
					"upload":   stats.TotalUpload,
					"download": stats.TotalDownload,
				},
				"total_upload":   stats.TotalUpload,
				"total_download": stats.TotalDownload,
				"uptime":         stats.Uptime,
				"kernel_status":  kernelStatus,
				"kernel": map[string]interface{}{
					"status":  kernelStatus,
					"pid":     kernelPID,
					"version": kernelVersion,
					"uptime":  kernelUptime,
				},
			}

			data, err := json.Marshal(payload)
			if err != nil {
				continue
			}

			select {
			case h.broadcast <- data:
			default:
			}
		}
	}
}

// HandleWS upgrades incoming HTTP connections to WebSocket and registers the client.
func (h *WSHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket unexpected close: %v", err)
			}
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
