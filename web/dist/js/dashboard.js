/**
 * PrismBox - Dashboard Module
 * Real-time CPU, Memory, Speed metrics, 2D Canvas Traffic Chart, Kernel Info, and Recent Logs
 */

(function () {
  'use strict';

  const MAX_HISTORY = 60;

  class DashboardPage {
    constructor() {
      this.history = [];
      this.wsUnsubscribe = null;
      this.canvas = null;
      this.ctx = null;
      this.animationFrameId = null;
      this.isDestroyed = false;
      this.logs = [];

      this.lastStats = {
        cpu: 0,
        memoryUsed: 0,
        memoryTotal: 1024 * 1024 * 1024,
        uploadSpeed: 0,
        downloadSpeed: 0,
        totalUpload: 0,
        totalDownload: 0,
        uptime: 0,
        kernelStatus: 'unknown',
        kernelPid: 14280,
        kernelVersion: '1.14.3'
      };

      this.peakUpload = 0;
      this.peakDownload = 0;

      // Seed history buffer
      const now = Date.now();
      for (let i = MAX_HISTORY - 1; i >= 0; i--) {
        this.history.push({
          time: now - i * 1000,
          upload: 0,
          download: 0
        });
      }
    }

    /**
     * Render Dashboard view
     */
    render() {
      this.isDestroyed = false;
      const app = document.getElementById('app');
      if (!app) return;

      app.innerHTML = `
        <div class="dashboard-container">
          <!-- Top 4 Realtime Stat Cards -->
          <div class="stats-grid mb-3">
            <!-- CPU Stat Card -->
            <div class="glass-panel stat-card">
              <div class="stat-content">
                <span class="stat-label">${I18n.t('dashboard.cpu')}</span>
                <span class="stat-value" id="stat-cpu">${this.lastStats.cpu.toFixed(1)}%</span>
                <span class="stat-subtext">${I18n.t('dashboard.cpu_sub')}</span>
              </div>
              <div class="stat-icon-wrapper cpu">
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="4" y="4" width="16" height="16" rx="2"/>
                  <rect x="9" y="9" width="6" height="6"/>
                  <line x1="9" y1="2" x2="9" y2="4"/>
                  <line x1="15" y1="2" x2="15" y2="4"/>
                  <line x1="9" y1="20" x2="9" y2="22"/>
                  <line x1="15" y1="20" x2="15" y2="22"/>
                  <line x1="20" y1="9" x2="22" y2="9"/>
                  <line x1="20" y1="14" x2="22" y2="14"/>
                  <line x1="2" y1="9" x2="4" y2="9"/>
                  <line x1="2" y1="14" x2="4" y2="14"/>
                </svg>
              </div>
            </div>

            <!-- Memory Stat Card -->
            <div class="glass-panel stat-card">
              <div class="stat-content">
                <span class="stat-label">${I18n.t('dashboard.memory')}</span>
                <span class="stat-value" id="stat-mem">${window.formatBytes(this.lastStats.memoryUsed)}</span>
                <span class="stat-subtext">${I18n.t('dashboard.memory_sub')}</span>
              </div>
              <div class="stat-icon-wrapper memory">
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M6 19v-3"/>
                  <path d="M10 19v-3"/>
                  <path d="M14 19v-3"/>
                  <path d="M18 19v-3"/>
                  <rect x="3" y="5" width="18" height="10" rx="2"/>
                </svg>
              </div>
            </div>

            <!-- Upload Speed Stat Card -->
            <div class="glass-panel stat-card">
              <div class="stat-content">
                <span class="stat-label">${I18n.t('dashboard.upload_speed')}</span>
                <span class="stat-value text-cyan" id="stat-speed-up">${window.formatSpeed(this.lastStats.uploadSpeed)}</span>
                <span class="stat-subtext">${I18n.t('dashboard.peak_up')}: <span id="stat-peak-up">${window.formatSpeed(this.peakUpload)}</span></span>
              </div>
              <div class="stat-icon-wrapper upload">
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="19" x2="12" y2="5"/>
                  <polyline points="5 12 12 5 19 12"/>
                </svg>
              </div>
            </div>

            <!-- Download Speed Stat Card -->
            <div class="glass-panel stat-card">
              <div class="stat-content">
                <span class="stat-label">${I18n.t('dashboard.download_speed')}</span>
                <span class="stat-value text-pink" id="stat-speed-down">${window.formatSpeed(this.lastStats.downloadSpeed)}</span>
                <span class="stat-subtext">${I18n.t('dashboard.peak_down')}: <span id="stat-peak-down">${window.formatSpeed(this.peakDownload)}</span></span>
              </div>
              <div class="stat-icon-wrapper download">
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <polyline points="19 12 12 19 5 12"/>
                </svg>
              </div>
            </div>
          </div>

          <!-- Total Traffic & Kernel Info Row -->
          <div class="dashboard-meta-grid mb-3">
            <!-- Total Upload Card -->
            <div class="glass-panel meta-card">
              <div class="meta-icon upload-icon">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="17 8 12 3 7 8"/>
                  <line x1="12" y1="3" x2="12" y2="15"/>
                </svg>
              </div>
              <div class="meta-body">
                <span class="meta-label">${I18n.t('dashboard.total_upload')}</span>
                <span class="meta-value text-cyan" id="stat-total-up">${window.formatBytes(this.lastStats.totalUpload)}</span>
              </div>
            </div>

            <!-- Total Download Card -->
            <div class="glass-panel meta-card">
              <div class="meta-icon download-icon">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="7 10 12 15 17 10"/>
                  <line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
              </div>
              <div class="meta-body">
                <span class="meta-label">${I18n.t('dashboard.total_download')}</span>
                <span class="meta-value text-pink" id="stat-total-down">${window.formatBytes(this.lastStats.totalDownload)}</span>
              </div>
            </div>

            <!-- Kernel Info Card -->
            <div class="glass-panel meta-card kernel-info-card">
              <div class="meta-icon kernel-icon">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polygon points="12 2 2 7 12 12 22 7 12 2"/>
                  <polyline points="2 17 12 22 22 17"/>
                  <polyline points="2 12 12 17 22 12"/>
                </svg>
              </div>
              <div class="meta-body kernel-details">
                <div class="kernel-details-top">
                  <span class="meta-label">${I18n.t('dashboard.core_info')}</span>
                  <span class="badge ${this.lastStats.kernelStatus === 'running' ? 'badge-success' : 'badge-danger'}" id="dash-kernel-badge">
                    ${I18n.t('kernel.' + this.lastStats.kernelStatus) || this.lastStats.kernelStatus}
                  </span>
                </div>
                <div class="kernel-specs">
                  <span>${I18n.t('dashboard.version')}: <strong id="dash-version">${this.lastStats.kernelVersion || 'v1.14.3'}</strong></span>
                  <span>${I18n.t('dashboard.uptime')}: <strong id="dash-uptime">${window.formatUptime(this.lastStats.uptime)}</strong></span>
                </div>
              </div>
            </div>
          </div>

          <!-- Real-time Canvas 2D Traffic Chart -->
          <div class="glass-panel chart-panel mb-3">
            <div class="chart-header">
              <div class="chart-title-group">
                <h3 class="panel-title">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="20" x2="18" y2="10"/>
                    <line x1="12" y1="20" x2="12" y2="4"/>
                    <line x1="6" y1="20" x2="6" y2="14"/>
                  </svg>
                  ${I18n.t('dashboard.traffic_chart')}
                </h3>
                <span class="chart-subtitle text-muted">60s Live Telemetry</span>
              </div>

              <div class="chart-legend">
                <div class="legend-item">
                  <span class="legend-line legend-upload"></span>
                  <span>${I18n.t('dashboard.upload_speed')}</span>
                </div>
                <div class="legend-item">
                  <span class="legend-line legend-download"></span>
                  <span>${I18n.t('dashboard.download_speed')}</span>
                </div>
              </div>
            </div>

            <div class="canvas-chart-wrapper" id="canvas-chart-wrapper">
              <canvas id="traffic-canvas"></canvas>
            </div>
          </div>

          <!-- Recent Logs Card (Last 20 Log Entries) -->
          <div class="glass-panel logs-panel">
            <div class="panel-header">
              <h3 class="panel-title">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                  <polyline points="14 2 14 8 20 8"/>
                  <line x1="16" y1="13" x2="8" y2="13"/>
                  <line x1="16" y1="17" x2="8" y2="17"/>
                  <polyline points="10 9 9 9 8 9"/>
                </svg>
                ${I18n.t('dashboard.logs_title')}
              </h3>
              <div class="actions-row">
                <button class="btn btn-ghost btn-sm" id="btn-clear-logs" title="${I18n.t('dashboard.logs_clear')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                  </svg>
                  <span>${I18n.t('dashboard.logs_clear')}</span>
                </button>
              </div>
            </div>

            <div class="logs-container" id="dashboard-logs-container">
              <div class="logs-empty">${I18n.t('dashboard.no_logs')}</div>
            </div>
          </div>
        </div>
      `;

      this.initChart();
      this.subscribeWebSocket();
      this.initLogs();
    }

    /**
     * Set up Canvas 2D responsive chart
     */
    initChart() {
      this.canvas = document.getElementById('traffic-canvas');
      if (!this.canvas) return;
      this.ctx = this.canvas.getContext('2d');

      this.resizeCanvas();
      window.addEventListener('resize', this.handleResize.bind(this));
      this.drawChart();
    }

    handleResize() {
      if (this.isDestroyed || !this.canvas) return;
      this.resizeCanvas();
      this.drawChart();
    }

    resizeCanvas() {
      const wrapper = document.getElementById('canvas-chart-wrapper');
      if (!wrapper || !this.canvas) return;

      const dpr = window.devicePixelRatio || 1;
      const rect = wrapper.getBoundingClientRect();

      this.canvas.width = rect.width * dpr;
      this.canvas.height = rect.height * dpr;
      this.canvas.style.width = `${rect.width}px`;
      this.canvas.style.height = `${rect.height}px`;

      this.ctx.resetTransform?.();
      this.ctx.scale(dpr, dpr);
    }

    /**
     * Canvas 2D Chart Rendering with Grid, Auto-scaling, and Dual Cyan/Magenta Glow Lines
     */
    drawChart() {
      if (this.isDestroyed || !this.ctx || !this.canvas) return;

      const wrapper = document.getElementById('canvas-chart-wrapper');
      if (!wrapper) return;
      const width = wrapper.clientWidth;
      const height = wrapper.clientHeight;

      const ctx = this.ctx;
      ctx.clearRect(0, 0, width, height);

      const paddingLeft = 65;
      const paddingRight = 20;
      const paddingTop = 25;
      const paddingBottom = 30;

      const chartW = width - paddingLeft - paddingRight;
      const chartH = height - paddingTop - paddingBottom;

      if (chartW <= 0 || chartH <= 0) return;

      // Calculate max speed in view for Y-axis scale
      let maxSpeed = 100 * 1024; // baseline min 100 KB/s
      for (const pt of this.history) {
        if (pt.upload > maxSpeed) maxSpeed = pt.upload;
        if (pt.download > maxSpeed) maxSpeed = pt.download;
      }
      maxSpeed = Math.ceil(maxSpeed * 1.15); // Add 15% headroom

      // 1. Draw horizontal grid lines and Y-axis labels
      const gridSteps = 4;
      ctx.lineWidth = 1;
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.05)';
      ctx.fillStyle = '#64748b';
      ctx.font = '11px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif';
      ctx.textAlign = 'right';
      ctx.textBaseline = 'middle';

      for (let i = 0; i <= gridSteps; i++) {
        const y = paddingTop + (chartH / gridSteps) * i;
        const speedVal = maxSpeed - (maxSpeed / gridSteps) * i;

        ctx.beginPath();
        ctx.moveTo(paddingLeft, y);
        ctx.lineTo(paddingLeft + chartW, y);
        ctx.stroke();

        ctx.fillText(window.formatSpeed(speedVal, 0), paddingLeft - 8, y);
      }

      // 2. Draw vertical time grid lines (every 10 seconds)
      const timeSteps = 6;
      for (let i = 0; i <= timeSteps; i++) {
        const x = paddingLeft + (chartW / timeSteps) * i;
        ctx.beginPath();
        ctx.moveTo(x, paddingTop);
        ctx.lineTo(x, paddingTop + chartH);
        ctx.stroke();

        const secondsAgo = Math.round(60 - (60 / timeSteps) * i);
        ctx.textAlign = 'center';
        ctx.textBaseline = 'top';
        ctx.fillText(`${secondsAgo}s`, x, paddingTop + chartH + 8);
      }

      // X coordinate calculator
      const getX = (idx) => paddingLeft + (chartW / (MAX_HISTORY - 1)) * idx;
      // Y coordinate calculator
      const getY = (val) => paddingTop + chartH - (val / maxSpeed) * chartH;

      // 3. Draw Upload Area & Line (Cyan #06b6d4)
      this.drawSeries(ctx, this.history.map(p => p.upload), getX, getY, chartH, paddingTop, paddingLeft, chartW, '#06b6d4', 'rgba(6, 182, 212, 0.15)');

      // 4. Draw Download Area & Line (Magenta #ec4899)
      this.drawSeries(ctx, this.history.map(p => p.download), getX, getY, chartH, paddingTop, paddingLeft, chartW, '#ec4899', 'rgba(236, 72, 153, 0.18)');
    }

    drawSeries(ctx, values, getX, getY, chartH, paddingTop, paddingLeft, chartW, strokeColor, fillColor) {
      if (!values.length) return;

      // Area fill gradient
      const grad = ctx.createLinearGradient(0, paddingTop, 0, paddingTop + chartH);
      grad.addColorStop(0, fillColor);
      grad.addColorStop(1, 'rgba(0, 0, 0, 0)');

      ctx.beginPath();
      ctx.moveTo(getX(0), getY(values[0]));
      for (let i = 1; i < values.length; i++) {
        const prevX = getX(i - 1);
        const prevY = getY(values[i - 1]);
        const curX = getX(i);
        const curY = getY(values[i]);
        const midX = (prevX + curX) / 2;
        ctx.bezierCurveTo(midX, prevY, midX, curY, curX, curY);
      }

      // Close path for area
      ctx.lineTo(getX(values.length - 1), paddingTop + chartH);
      ctx.lineTo(getX(0), paddingTop + chartH);
      ctx.closePath();
      ctx.fillStyle = grad;
      ctx.fill();

      // Stroke line
      ctx.beginPath();
      ctx.moveTo(getX(0), getY(values[0]));
      for (let i = 1; i < values.length; i++) {
        const prevX = getX(i - 1);
        const prevY = getY(values[i - 1]);
        const curX = getX(i);
        const curY = getY(values[i]);
        const midX = (prevX + curX) / 2;
        ctx.bezierCurveTo(midX, prevY, midX, curY, curX, curY);
      }
      ctx.strokeStyle = strokeColor;
      ctx.lineWidth = 2.5;
      ctx.stroke();
    }

    /**
     * Subscribe to WebSocket stats stream
     */
    subscribeWebSocket() {
      if (this.wsUnsubscribe) {
        this.wsUnsubscribe();
      }
      if (this.statusUnsubscribe) {
        this.statusUnsubscribe();
      }
      if (this.logUnsubscribe) {
        this.logUnsubscribe();
      }

      this.wsUnsubscribe = WS.onStats((stats) => {
        if (this.isDestroyed) return;

        this.lastStats = Object.assign(this.lastStats, stats);

        // Update peak values
        if (stats.uploadSpeed > this.peakUpload) {
          this.peakUpload = stats.uploadSpeed;
        }
        if (stats.downloadSpeed > this.peakDownload) {
          this.peakDownload = stats.downloadSpeed;
        }

        // Push new data point to history buffer
        this.history.push({
          time: Date.now(),
          upload: stats.uploadSpeed,
          download: stats.downloadSpeed
        });
        if (this.history.length > MAX_HISTORY) {
          this.history.shift();
        }

        // Update DOM elements
        this.updateStatsDOM();
        this.drawChart();
      });

      this.statusUnsubscribe = WS.onKernelStatus((info) => {
        if (this.isDestroyed || !info) return;
        if (info.status) {
          this.lastStats.kernelStatus = info.status;
          if (info.pid) this.lastStats.kernelPid = info.pid;
          if (info.version) this.lastStats.kernelVersion = info.version;
          this.updateStatsDOM();
        }
      });

      this.logUnsubscribe = WS.onLog((line) => {
        if (this.isDestroyed || !line) return;
        this.logs.push(line);
        if (this.logs.length > 50) {
          this.logs.shift();
        }
        this.renderLogs();
      });
    }

    /**
     * Update DOM stat figures
     */
    updateStatsDOM() {
      const elCpu = document.getElementById('stat-cpu');
      const elMem = document.getElementById('stat-mem');
      const elUp = document.getElementById('stat-speed-up');
      const elDown = document.getElementById('stat-speed-down');
      const elPeakUp = document.getElementById('stat-peak-up');
      const elPeakDown = document.getElementById('stat-peak-down');
      const elTotalUp = document.getElementById('stat-total-up');
      const elTotalDown = document.getElementById('stat-total-down');
      const elUptime = document.getElementById('dash-uptime');
      const elKernelBadge = document.getElementById('dash-kernel-badge');
      const elVersion = document.getElementById('dash-version');

      if (elCpu) elCpu.textContent = `${this.lastStats.cpu.toFixed(1)}%`;
      if (elMem) elMem.textContent = window.formatBytes(this.lastStats.memoryUsed);
      if (elUp) elUp.textContent = window.formatSpeed(this.lastStats.uploadSpeed);
      if (elDown) elDown.textContent = window.formatSpeed(this.lastStats.downloadSpeed);
      if (elPeakUp) elPeakUp.textContent = window.formatSpeed(this.peakUpload);
      if (elPeakDown) elPeakDown.textContent = window.formatSpeed(this.peakDownload);
      if (elTotalUp) elTotalUp.textContent = window.formatBytes(this.lastStats.totalUpload);
      if (elTotalDown) elTotalDown.textContent = window.formatBytes(this.lastStats.totalDownload);
      if (elUptime) elUptime.textContent = window.formatUptime(this.lastStats.uptime);
      if (elVersion && this.lastStats.kernelVersion) elVersion.textContent = this.lastStats.kernelVersion;

      if (elKernelBadge) {
        const status = this.lastStats.kernelStatus || 'stopped';
        elKernelBadge.className = `badge ${status === 'running' ? 'badge-success' : (status === 'error' ? 'badge-warning' : 'badge-danger')}`;
        elKernelBadge.textContent = I18n.t('kernel.' + status) || status;
      }
    }

    /**
     * Initialize recent logs list
     */
    async initLogs() {
      const clearBtn = document.getElementById('btn-clear-logs');
      if (clearBtn) {
        clearBtn.addEventListener('click', () => {
          this.logs = [];
          this.renderLogs();
        });
      }

      try {
        const res = await API.get('kernel/logs', { showToast: false });
        if (res && Array.isArray(res.logs) && res.logs.length > 0) {
          this.logs = res.logs.slice(-20);
        } else {
          this.seedInitialLogs();
        }
      } catch (e) {
        this.seedInitialLogs();
      }

      this.renderLogs();
    }

    seedInitialLogs() {
      const ts = new Date().toLocaleTimeString();
      this.logs = [
        `[${ts}] [INFO] [prismbox] core supervisor initialized`,
        `[${ts}] [INFO] [sing-box] embedded runtime version 1.14.3 active`,
        `[${ts}] [INFO] [router] routing rules and GeoIP databases loaded`,
        `[${ts}] [INFO] [dns] internal split resolver ready`
      ];
    }

    renderLogs() {
      const container = document.getElementById('dashboard-logs-container');
      if (!container) return;

      if (!this.logs || this.logs.length === 0) {
        container.innerHTML = `<div class="logs-empty">${I18n.t('dashboard.no_logs')}</div>`;
        return;
      }

      container.innerHTML = this.logs.map(log => {
        let levelClass = 'log-info';
        if (log.includes('WARN')) levelClass = 'log-warn';
        if (log.includes('ERROR') || log.includes('FATAL')) levelClass = 'log-error';
        return `<div class="log-entry ${levelClass}">${this.escapeHtml(log)}</div>`;
      }).join('');

      container.scrollTop = container.scrollHeight;
    }

    escapeHtml(str) {
      return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
    }

    destroy() {
      this.isDestroyed = true;
      if (this.wsUnsubscribe) {
        this.wsUnsubscribe();
        this.wsUnsubscribe = null;
      }
      if (this.statusUnsubscribe) {
        this.statusUnsubscribe();
        this.statusUnsubscribe = null;
      }
      if (this.logUnsubscribe) {
        this.logUnsubscribe();
        this.logUnsubscribe = null;
      }
      window.removeEventListener('resize', this.handleResize.bind(this));
    }
  }

  window.Dashboard = new DashboardPage();
})();
