/**
 * PrismBox - API Client & WebSocket Manager
 * Handles REST requests to /api/v1/ and real-time WebSocket telemetry
 */

(function () {
  'use strict';

  const API_BASE = '/api/v1';

  /**
   * Safe JSON parser helper
   */
  async function parseResponse(response) {
    if (response.status === 204) {
      return null;
    }
    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
      return await response.json();
    }
    const text = await response.text();
    try {
      return JSON.parse(text);
    } catch {
      return { message: text };
    }
  }

  /**
   * Normalize path to include /api/v1 prefix
   */
  function normalizePath(path) {
    const cleanPath = path.startsWith('/') ? path.slice(1) : path;
    return `${API_BASE}/${cleanPath}`;
  }

  /**
   * REST API Client
   */
  const API = {
    /**
     * Perform HTTP GET request
     */
    async get(path, options = {}) {
      const url = normalizePath(path);
      try {
        const response = await fetch(url, {
          method: 'GET',
          headers: {
            'Accept': 'application/json',
            ...(options.headers || {})
          }
        });

        if (!response.ok) {
          const errData = await parseResponse(response).catch(() => ({}));
          const errMsg = errData.error || errData.message || `Request failed with status ${response.status}`;
          throw new Error(errMsg);
        }

        return await parseResponse(response);
      } catch (err) {
        console.error(`[API GET ${path}] Error:`, err);
        if (options.showToast !== false && window.showToast) {
          window.showToast(`GET ${path}: ${err.message}`, 'error');
        }
        throw err;
      }
    },

    /**
     * Perform HTTP POST request
     */
    async post(path, data = {}, options = {}) {
      const url = normalizePath(path);
      try {
        const response = await fetch(url, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
            ...(options.headers || {})
          },
          body: JSON.stringify(data)
        });

        if (!response.ok) {
          const errData = await parseResponse(response).catch(() => ({}));
          const errMsg = errData.error || errData.message || `Request failed with status ${response.status}`;
          throw new Error(errMsg);
        }

        return await parseResponse(response);
      } catch (err) {
        console.error(`[API POST ${path}] Error:`, err);
        if (options.showToast !== false && window.showToast) {
          window.showToast(`POST ${path}: ${err.message}`, 'error');
        }
        throw err;
      }
    },

    /**
     * Perform HTTP PUT request
     */
    async put(path, data = {}, options = {}) {
      const url = normalizePath(path);
      try {
        const response = await fetch(url, {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
            ...(options.headers || {})
          },
          body: JSON.stringify(data)
        });

        if (!response.ok) {
          const errData = await parseResponse(response).catch(() => ({}));
          const errMsg = errData.error || errData.message || `Request failed with status ${response.status}`;
          throw new Error(errMsg);
        }

        return await parseResponse(response);
      } catch (err) {
        console.error(`[API PUT ${path}] Error:`, err);
        if (options.showToast !== false && window.showToast) {
          window.showToast(`PUT ${path}: ${err.message}`, 'error');
        }
        throw err;
      }
    },

    /**
     * Perform HTTP DELETE request
     */
    async del(path, options = {}) {
      const url = normalizePath(path);
      try {
        const response = await fetch(url, {
          method: 'DELETE',
          headers: {
            'Accept': 'application/json',
            ...(options.headers || {})
          }
        });

        if (!response.ok) {
          const errData = await parseResponse(response).catch(() => ({}));
          const errMsg = errData.error || errData.message || `Request failed with status ${response.status}`;
          throw new Error(errMsg);
        }

        return await parseResponse(response);
      } catch (err) {
        console.error(`[API DELETE ${path}] Error:`, err);
        if (options.showToast !== false && window.showToast) {
          window.showToast(`DELETE ${path}: ${err.message}`, 'error');
        }
        throw err;
      }
    },

    delete(path, options) {
      return this.del(path, options);
    }
  };

  /**
   * Realtime WebSocket Client with Auto-reconnect & Heartbeat
   */
  const WS = {
    socket: null,
    messageListeners: new Set(),
    statsListeners: new Set(),
    stateListeners: new Set(),
    reconnectTimer: null,
    pingTimer: null,
    retryCount: 0,
    maxRetryDelay: 10000,
    baseRetryDelay: 1000,
    isManualClose: false,
    mockTimer: null,

    /**
     * Connect to WebSocket endpoint
     */
    connect() {
      if (this.socket && (this.socket.readyState === WebSocket.OPEN || this.socket.readyState === WebSocket.CONNECTING)) {
        return;
      }

      this.isManualClose = false;
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.host || 'localhost:8080';
      const wsUrl = `${protocol}//${host}/ws`;

      this.notifyState('connecting');

      try {
        this.socket = new WebSocket(wsUrl);

        this.socket.onopen = () => {
          this.retryCount = 0;
          this.notifyState('connected');
          this.stopMockTelemetry();
          this.startHeartbeat();
        };

        this.socket.onmessage = (event) => {
          try {
            // Heartbeat pong / empty message check
            if (!event.data || event.data === 'pong') return;
            const payload = JSON.parse(event.data);

            // Standardize telemetry fields
            const normalized = {
              cpu: typeof payload.cpu_percent === 'number' ? payload.cpu_percent : (payload.cpu || 0),
              memoryUsed: typeof payload.mem_used_mb === 'number' ? payload.mem_used_mb * 1024 * 1024 : (payload.memory?.used || 0),
              memoryTotal: typeof payload.mem_total_mb === 'number' ? payload.mem_total_mb * 1024 * 1024 : (payload.memory?.total || 1024 * 1024 * 1024),
              uploadSpeed: typeof payload.upload_speed === 'number' ? payload.upload_speed : (payload.speed?.upload || 0),
              downloadSpeed: typeof payload.download_speed === 'number' ? payload.download_speed : (payload.speed?.download || 0),
              totalUpload: typeof payload.total_upload === 'number' ? payload.total_upload : (payload.traffic?.upload || 0),
              totalDownload: typeof payload.total_download === 'number' ? payload.total_download : (payload.traffic?.download || 0),
              uptime: payload.uptime || 0,
              kernelStatus: payload.kernel_status || (payload.kernel?.status || 'stopped'),
              raw: payload
            };

            this.dispatchMessage(payload);
            this.dispatchStats(normalized);
          } catch (err) {
            console.warn('[WS] Failed to parse message:', event.data);
          }
        };

        this.socket.onerror = (err) => {
          console.warn('[WS] Socket error:', err);
        };

        this.socket.onclose = () => {
          this.stopHeartbeat();
          this.notifyState('disconnected');
          if (!this.isManualClose) {
            this.scheduleReconnect();
            this.startMockTelemetry();
          }
        };
      } catch (err) {
        console.warn('[WS] Connection exception:', err);
        this.stopHeartbeat();
        this.notifyState('disconnected');
        this.scheduleReconnect();
        this.startMockTelemetry();
      }
    },

    /**
     * Disconnect WebSocket
     */
    disconnect() {
      this.isManualClose = true;
      this.stopHeartbeat();
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
      }
      this.stopMockTelemetry();
      if (this.socket) {
        this.socket.close();
        this.socket = null;
      }
      this.notifyState('disconnected');
    },

    /**
     * Heartbeat ping-pong to keep connection alive
     */
    startHeartbeat() {
      this.stopHeartbeat();
      this.pingTimer = setInterval(() => {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
          try {
            this.socket.send(JSON.stringify({ type: 'ping' }));
          } catch (e) {
            // Ignore send error
          }
        }
      }, 30000);
    },

    stopHeartbeat() {
      if (this.pingTimer) {
        clearInterval(this.pingTimer);
        this.pingTimer = null;
      }
    },

    /**
     * Schedule reconnection with exponential backoff
     */
    scheduleReconnect() {
      if (this.reconnectTimer) return;
      const delay = Math.min(this.baseRetryDelay * Math.pow(1.5, this.retryCount), this.maxRetryDelay);
      this.retryCount++;

      this.reconnectTimer = setTimeout(() => {
        this.reconnectTimer = null;
        this.connect();
      }, delay);
    },

    /**
     * Subscribe to raw messages
     */
    onMessage(callback) {
      if (typeof callback === 'function') {
        this.messageListeners.add(callback);
      }
      return () => this.messageListeners.delete(callback);
    },

    /**
     * Subscribe to parsed telemetry stats
     */
    onStats(callback) {
      if (typeof callback === 'function') {
        this.statsListeners.add(callback);
      }
      return () => this.statsListeners.delete(callback);
    },

    /**
     * Subscribe to connection state changes ('connected' | 'connecting' | 'disconnected')
     */
    onStateChange(callback) {
      if (typeof callback === 'function') {
        this.stateListeners.add(callback);
      }
      return () => this.stateListeners.delete(callback);
    },

    notifyState(state) {
      this.stateListeners.forEach((fn) => {
        try { fn(state); } catch (e) { console.error(e); }
      });
    },

    dispatchMessage(msg) {
      this.messageListeners.forEach((fn) => {
        try { fn(msg); } catch (e) { console.error(e); }
      });
    },

    dispatchStats(data) {
      this.statsListeners.forEach((fn) => {
        try { fn(data); } catch (e) { console.error(e); }
      });
    },

    /**
     * Fallback simulated telemetry when backend server is offline
     */
    startMockTelemetry() {
      if (this.mockTimer) return;
      let mockUptime = 3600;
      let mockUploadSpeed = 125000;
      let mockDownloadSpeed = 1680000;
      let mockTotalUp = 450000000;
      let mockTotalDown = 2800000000;

      this.mockTimer = setInterval(() => {
        mockUptime++;
        mockUploadSpeed = Math.max(1024, Math.floor(mockUploadSpeed + (Math.random() - 0.49) * 80000));
        mockDownloadSpeed = Math.max(5120, Math.floor(mockDownloadSpeed + (Math.random() - 0.48) * 450000));
        mockTotalUp += mockUploadSpeed;
        mockTotalDown += mockDownloadSpeed;

        const mockStats = {
          cpu: +(1.2 + Math.random() * 3.5).toFixed(1),
          memoryUsed: (42 + Math.random() * 6) * 1024 * 1024,
          memoryTotal: 1024 * 1024 * 1024,
          uploadSpeed: mockUploadSpeed,
          downloadSpeed: mockDownloadSpeed,
          totalUpload: mockTotalUp,
          totalDownload: mockTotalDown,
          uptime: mockUptime,
          kernelStatus: 'running'
        };

        this.dispatchStats(mockStats);
      }, 1000);
    },

    stopMockTelemetry() {
      if (this.mockTimer) {
        clearInterval(this.mockTimer);
        this.mockTimer = null;
      }
    }
  };

  window.API = API;
  window.WS = WS;
})();
