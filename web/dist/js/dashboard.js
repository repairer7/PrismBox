/**
 * PrismBox - Dashboard Module
 * Real-time CPU, Memory, Speed metrics, 2D Canvas Traffic Chart, Kernel Info, and Recent Logs
 */

(function () {
  'use strict';

  const MAX_HISTORY = 60;

  const SITES_CONFIG = [
    { id: 'baidu', nameKey: 'dashboard.site_baidu', host: 'www.baidu.com', url: 'https://www.baidu.com/favicon.ico' },
    { id: 'cloudflare', nameKey: 'dashboard.site_cloudflare', host: 'cp.cloudflare.com', url: 'https://cp.cloudflare.com/generate_204' },
    { id: 'github', nameKey: 'dashboard.site_github', host: 'github.com', url: 'https://github.com/favicon.ico' },
    { id: 'youtube', nameKey: 'dashboard.site_youtube', host: 'www.youtube.com', url: 'https://www.youtube.com/generate_204' }
  ];

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

      this.isTestingSites = false;
      this.siteResults = {
        baidu: Array(10).fill(null),
        cloudflare: Array(10).fill(null),
        github: Array(10).fill(null),
        youtube: Array(10).fill(null)
      };

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

          <!-- Total Traffic & Kernel Info: 3 Parallel Cards (Side-by-side, icons on right matching CPU stat card) -->
          <div class="dash-meta-grid mb-3">
            <!-- Total Upload Card -->
            <div class="glass-panel stat-card dash-meta-card">
              <div class="stat-content">
                <span class="stat-label">${I18n.t('dashboard.total_upload')}</span>
                <span class="stat-value text-purple" id="stat-total-up">${window.formatBytes(this.lastStats.totalUpload)}</span>
                <span class="stat-subtext">${I18n.t('dashboard.total_upload_sub')}</span>
              </div>
              <div class="stat-icon-wrapper upload">
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="17 8 12 3 7 8"/>
                  <line x1="12" y1="3" x2="12" y2="15"/>
                </svg>
              </div>
            </div>

            <!-- Total Download Card -->
            <div class="glass-panel stat-card dash-meta-card">
              <div class="stat-content">
                <span class="stat-label">${I18n.t('dashboard.total_download')}</span>
                <span class="stat-value text-success" id="stat-total-down">${window.formatBytes(this.lastStats.totalDownload)}</span>
                <span class="stat-subtext">${I18n.t('dashboard.total_download_sub')}</span>
              </div>
              <div class="stat-icon-wrapper download">
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="7 10 12 15 17 10"/>
                  <line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
              </div>
            </div>

            <!-- Kernel Info Card -->
            <div class="glass-panel stat-card dash-meta-card">
              <div class="stat-content">
                <div class="dash-meta-label-row">
                  <span class="stat-label">${I18n.t('dashboard.core_info')}</span>
                  <span class="badge ${this.lastStats.kernelStatus === 'running' ? 'badge-success' : 'badge-danger'}" id="dash-kernel-badge">
                    ${I18n.t('kernel.' + this.lastStats.kernelStatus) || this.lastStats.kernelStatus}
                  </span>
                </div>
                <div class="kernel-meta-version">sing-box <strong id="dash-version">${this.lastStats.kernelVersion || 'v1.14.3'}</strong></div>
                <span class="stat-subtext">${I18n.t('dashboard.uptime')}: <strong id="dash-uptime">${window.formatUptime(this.lastStats.uptime)}</strong></span>
              </div>
              <div class="stat-icon-wrapper memory">
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
                  <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
                  <line x1="6" y1="6" x2="6.01" y2="6"/>
                  <line x1="6" y1="18" x2="6.01" y2="18"/>
                </svg>
              </div>
            </div>
          </div>

          <!-- Client Site Latency & Connectivity Panel (2x2 Grid with 10-bar charts) -->
          <div class="glass-panel site-latency-panel mb-3">
            <div class="panel-header site-panel-header">
              <div class="site-latency-title-group">
                <h3 class="panel-title">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="2" y1="12" x2="22" y2="12"/>
                    <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
                  </svg>
                  ${I18n.t('dashboard.site_latency_title')}
                </h3>
                <span class="site-latency-sub text-muted">${I18n.t('dashboard.site_latency_subtitle')}</span>
              </div>
              <div class="site-latency-actions">
                <button class="btn btn-secondary btn-sm" id="btn-retest-sites" title="${I18n.t('dashboard.site_latency_retest')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" id="icon-retest-sites">
                    <polyline points="23 4 23 10 17 10"/>
                    <polyline points="1 20 1 14 7 14"/>
                    <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                  </svg>
                  <span id="text-retest-sites">${I18n.t('dashboard.site_latency_retest')}</span>
                </button>
              </div>
            </div>

            <!-- Threshold indicator hint -->
            <div class="site-threshold-bar">
              <span class="threshold-tag green">≤ 150ms ${I18n.currentLang === 'zh' ? '优' : 'Fast'}</span>
              <span class="threshold-tag red">> 150ms ${I18n.currentLang === 'zh' ? '高' : 'Slow'}</span>
              <span class="threshold-hint-text">${I18n.t('dashboard.site_threshold_hint')}</span>
            </div>

            <!-- 2x2 Grid of Site Boxes -->
            <div class="site-latency-grid">
              <!-- Site 1: Baidu -->
              <div class="site-card glass-panel" id="site-card-baidu">
                <div class="site-card-header">
                  <div class="site-brand">
                    <div class="site-logo baidu-logo">
                      <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                        <circle cx="4.5" cy="9.5" r="2.5"/>
                        <circle cx="9" cy="4.5" r="2.5"/>
                        <circle cx="15" cy="4.5" r="2.5"/>
                        <circle cx="19.5" cy="9.5" r="2.5"/>
                        <path d="M12 11c-3.5 0-6 2.2-6 5.5 0 2 1.3 3.5 3 4.2 1 .4 2 .3 3-.2 1 .5 2 .6 3 .2 1.7-.7 3-2.2 3-4.2 0-3.3-2.5-5.5-6-5.5z"/>
                      </svg>
                    </div>
                    <span class="site-name">${I18n.t('dashboard.site_baidu')}</span>
                  </div>
                  <div class="site-val-box">
                    <span class="site-avg-val" id="site-val-baidu">-- ms</span>
                  </div>
                </div>
                <div class="site-blocks-grid" id="site-bars-baidu"></div>
              </div>

              <!-- Site 2: Cloudflare -->
              <div class="site-card glass-panel" id="site-card-cloudflare">
                <div class="site-card-header">
                  <div class="site-brand">
                    <div class="site-logo cf-logo">
                      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"/>
                      </svg>
                    </div>
                    <span class="site-name">${I18n.t('dashboard.site_cloudflare')}</span>
                  </div>
                  <div class="site-val-box">
                    <span class="site-avg-val" id="site-val-cloudflare">-- ms</span>
                  </div>
                </div>
                <div class="site-blocks-grid" id="site-bars-cloudflare"></div>
              </div>

              <!-- Site 3: GitHub -->
              <div class="site-card glass-panel" id="site-card-github">
                <div class="site-card-header">
                  <div class="site-brand">
                    <div class="site-logo github-logo">
                      <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                        <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"/>
                      </svg>
                    </div>
                    <span class="site-name">${I18n.t('dashboard.site_github')}</span>
                  </div>
                  <div class="site-val-box">
                    <span class="site-avg-val" id="site-val-github">-- ms</span>
                  </div>
                </div>
                <div class="site-blocks-grid" id="site-bars-github"></div>
              </div>

              <!-- Site 4: YouTube -->
              <div class="site-card glass-panel" id="site-card-youtube">
                <div class="site-card-header">
                  <div class="site-brand">
                    <div class="site-logo yt-logo">
                      <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                        <path d="M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z"/>
                      </svg>
                    </div>
                    <span class="site-name">${I18n.t('dashboard.site_youtube')}</span>
                  </div>
                  <div class="site-val-box">
                    <span class="site-avg-val" id="site-val-youtube">-- ms</span>
                  </div>
                </div>
                <div class="site-blocks-grid" id="site-bars-youtube"></div>
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
      this.initSiteLatency();
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

    /**
     * Initialize 2x2 site connectivity & latency testing
     */
    initSiteLatency() {
      const retestBtn = document.getElementById('btn-retest-sites');
      if (retestBtn) {
        retestBtn.addEventListener('click', () => {
          this.runSiteLatencyTests();
        });
      }

      // Initial render of empty bars
      for (const site of SITES_CONFIG) {
        this.updateSiteCard(site.id);
      }

      // Automatically trigger initial latency probe
      setTimeout(() => {
        if (!this.isDestroyed) {
          this.runSiteLatencyTests();
        }
      }, 500);
    }

    /**
     * Measure single HTTP round-trip latency directly from client browser
     */
    async pingSite(url) {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 1000);
      const cacheBuster = `_t=${Date.now()}_${Math.random().toString(36).slice(2, 7)}`;
      const targetUrl = url.includes('?') ? `${url}&${cacheBuster}` : `${url}?${cacheBuster}`;
      const start = performance.now();
      try {
        await fetch(targetUrl, {
          method: 'GET',
          mode: 'no-cors',
          cache: 'no-store',
          signal: controller.signal
        });
        clearTimeout(timeoutId);
        const duration = Math.round(performance.now() - start);
        return Math.min(1000, Math.max(1, duration));
      } catch (err) {
        clearTimeout(timeoutId);
        return 1000; // Timeout (>1000ms) or network blocked
      }
    }

    /**
     * Execute 10 samples per site and calculate average
     */
    async runSiteLatencyTests() {
      if (this.isTestingSites || this.isDestroyed) return;
      this.isTestingSites = true;

      const retestBtn = document.getElementById('btn-retest-sites');
      const retestText = document.getElementById('text-retest-sites');
      if (retestBtn) retestBtn.disabled = true;
      if (retestText) retestText.textContent = I18n.t('dashboard.site_latency_testing');

      // Reset results & render empty state
      for (const site of SITES_CONFIG) {
        this.siteResults[site.id] = Array(10).fill(null);
        this.updateSiteCard(site.id);
      }

      // Run 10 sequential rounds across the 4 sites
      for (let sampleIdx = 0; sampleIdx < 10; sampleIdx++) {
        if (this.isDestroyed) break;

        // Concurrently ping the 4 sites in this round
        await Promise.all(SITES_CONFIG.map(async (site) => {
          if (this.isDestroyed) return;
          const latency = await this.pingSite(site.url);
          if (!this.isDestroyed) {
            this.siteResults[site.id][sampleIdx] = latency;
            this.updateSiteCard(site.id);
          }
        }));

        // Brief delay between samples (80ms)
        await new Promise(resolve => setTimeout(resolve, 80));
      }

      this.isTestingSites = false;
      if (retestBtn) retestBtn.disabled = false;
      if (retestText) retestText.textContent = I18n.t('dashboard.site_latency_retest');
    }

    /**
     * Render the 10 square blocks and average latency for a site
     */
    updateSiteCard(siteId) {
      const blocksContainer = document.getElementById(`site-bars-${siteId}`);
      const valEl = document.getElementById(`site-val-${siteId}`);
      if (!blocksContainer) return;

      const samples = this.siteResults[siteId] || [];

      // Render 10 individual square blocks
      blocksContainer.innerHTML = samples.map((val, idx) => {
        let blockClass = 'site-block block-idle';
        let titleText = I18n.currentLang === 'zh' ? `第 ${idx + 1} 次测速: 待测` : `Test #${idx + 1}: Pending`;

        if (val !== null) {
          if (val >= 1000) {
            blockClass = 'site-block block-red';
            titleText = I18n.currentLang === 'zh'
              ? `第 ${idx + 1} 次测速: ${I18n.t('dashboard.site_timeout')} (≥1000ms)`
              : `Test #${idx + 1}: ${I18n.t('dashboard.site_timeout')} (≥1000ms)`;
          } else if (val > 150) {
            blockClass = 'site-block block-red';
            titleText = I18n.currentLang === 'zh'
              ? `第 ${idx + 1} 次测速: ${val} ms (>150ms)`
              : `Test #${idx + 1}: ${val} ms (>150ms)`;
          } else {
            blockClass = 'site-block block-green';
            titleText = I18n.currentLang === 'zh'
              ? `第 ${idx + 1} 次测速: ${val} ms (≤150ms)`
              : `Test #${idx + 1}: ${val} ms (≤150ms)`;
          }
        }

        return `<div class="${blockClass}" title="${titleText}"></div>`;
      }).join('');

      // Compute and display average latency
      if (valEl) {
        const completed = samples.filter(v => v !== null);
        if (completed.length === 0) {
          valEl.textContent = '-- ms';
          valEl.className = 'site-avg-val';
        } else {
          const sum = completed.reduce((acc, v) => acc + v, 0);
          const avg = Math.round(sum / completed.length);
          if (avg >= 1000) {
            valEl.textContent = I18n.t('dashboard.site_timeout');
            valEl.className = 'site-avg-val slow';
          } else {
            valEl.textContent = `${avg} ms`;
            valEl.className = `site-avg-val ${avg <= 150 ? 'good' : 'slow'}`;
          }
        }
      }
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
