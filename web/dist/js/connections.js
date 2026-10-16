/**
 * PrismBox - Active Connections Inspection & Control Module
 * Real-time connection inspection, client IP filtering, domain search, sorting, and session termination.
 */

(function () {
  'use strict';

  class ConnectionsPage {
    constructor() {
      this.connections = [];
      this.prevSnapshots = new Map(); // id -> { upload, download, timestamp }
      this.prevTotal = { upload: 0, download: 0, timestamp: 0 };
      this.filterClientIP = 'all';
      this.searchQuery = '';
      this.sortBy = 'time_desc';
      this.isPaused = false;
      this.pollTimer = null;
      this.pollIntervalMs = 1500;
      this.isDestroyed = false;
      this.isFetching = false;

      this.stats = {
        totalConnections: 0,
        activeClients: 0,
        downloadTotal: 0,
        uploadTotal: 0,
        liveDownloadSpeed: 0,
        liveUploadSpeed: 0
      };
    }

    /**
     * Render Connections View
     */
    render() {
      this.isDestroyed = false;
      const app = document.getElementById('app');
      if (!app) return;

      app.innerHTML = `
        <div class="connections-page">
          <!-- Top 4 Summary Stat Cards -->
          <div class="stats-grid conn-summary-grid mb-3">
            <div class="glass-panel stat-card conn-stat-card">
              <div class="stat-header conn-stat-header">
                <span class="stat-title" title="${I18n.t('connections.stat_active')}">${I18n.t('connections.stat_active')}</span>
                <div class="stat-icon-wrapper accent">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                    <polyline points="15 3 21 3 21 9"/>
                    <line x1="10" y1="14" x2="21" y2="3"/>
                  </svg>
                </div>
              </div>
              <div class="conn-stat-body">
                <div class="stat-value conn-stat-value" id="conn-stat-active">0</div>
                <div class="stat-sub conn-stat-sub" id="conn-stat-active-sub">${I18n.t('connections.sub_active_desc')}</div>
              </div>
            </div>

            <div class="glass-panel stat-card conn-stat-card">
              <div class="stat-header conn-stat-header">
                <span class="stat-title" title="${I18n.t('connections.stat_clients')}">${I18n.t('connections.stat_clients')}</span>
                <div class="stat-icon-wrapper cyan">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                    <circle cx="9" cy="7" r="4"/>
                    <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
                    <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
                  </svg>
                </div>
              </div>
              <div class="conn-stat-body">
                <div class="stat-value conn-stat-value" id="conn-stat-clients">0</div>
                <div class="stat-sub conn-stat-sub" id="conn-stat-clients-sub">${I18n.t('connections.sub_clients_desc')}</div>
              </div>
            </div>

            <div class="glass-panel stat-card conn-stat-card">
              <div class="stat-header conn-stat-header">
                <span class="stat-title" title="${I18n.t('connections.stat_down_speed')}">${I18n.t('connections.stat_down_speed')}</span>
                <div class="stat-icon-wrapper success">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="12" y1="5" x2="12" y2="19"/>
                    <polyline points="19 12 12 19 5 12"/>
                  </svg>
                </div>
              </div>
              <div class="conn-stat-body">
                <div class="stat-value conn-stat-value text-success" id="conn-stat-down-speed">0 B/s</div>
                <div class="stat-sub conn-stat-sub" id="conn-stat-down-total">${I18n.t('connections.stat_total_down')}: 0 B</div>
              </div>
            </div>

            <div class="glass-panel stat-card conn-stat-card">
              <div class="stat-header conn-stat-header">
                <span class="stat-title" title="${I18n.t('connections.stat_up_speed')}">${I18n.t('connections.stat_up_speed')}</span>
                <div class="stat-icon-wrapper purple">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="12" y1="19" x2="12" y2="5"/>
                    <polyline points="5 12 12 5 19 12"/>
                  </svg>
                </div>
              </div>
              <div class="conn-stat-body">
                <div class="stat-value conn-stat-value text-purple" id="conn-stat-up-speed">0 B/s</div>
                <div class="stat-sub conn-stat-sub" id="conn-stat-up-total">${I18n.t('connections.stat_total_up')}: 0 B</div>
              </div>
            </div>
          </div>

          <!-- Controls & Filter Toolbar -->
          <div class="glass-panel mb-3 conn-toolbar">
            <div class="conn-filter-group">
              <!-- Client IP Filter -->
              <div class="conn-control-item">
                <label class="conn-label" for="conn-client-ip-select">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
                    <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
                    <line x1="6" y1="6" x2="6.01" y2="6"/>
                    <line x1="6" y1="18" x2="6.01" y2="18"/>
                  </svg>
                  <span>${I18n.t('connections.filter_client')}</span>
                </label>
                <select class="form-select conn-select" id="conn-client-ip-select">
                  <option value="all">${I18n.t('connections.all_clients')} (0)</option>
                </select>
              </div>

              <!-- Keyword Search -->
              <div class="conn-control-item conn-search-box">
                <div class="search-input-box">
                  <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="11" cy="11" r="8"/>
                    <line x1="21" y1="21" x2="16.65" y2="16.65"/>
                  </svg>
                  <input type="text" class="form-control" id="conn-search-input" value="${this.escapeHtml(this.searchQuery)}" placeholder="${I18n.t('connections.search_placeholder')}">
                </div>
              </div>

              <!-- Sort Dropdown -->
              <div class="conn-control-item">
                <select class="form-select conn-select" id="conn-sort-select">
                  <option value="time_desc" ${this.sortBy === 'time_desc' ? 'selected' : ''}>${I18n.t('connections.sort_time_desc')}</option>
                  <option value="time_asc" ${this.sortBy === 'time_asc' ? 'selected' : ''}>${I18n.t('connections.sort_time_asc')}</option>
                  <option value="down_speed" ${this.sortBy === 'down_speed' ? 'selected' : ''}>${I18n.t('connections.sort_down_speed')}</option>
                  <option value="up_speed" ${this.sortBy === 'up_speed' ? 'selected' : ''}>${I18n.t('connections.sort_up_speed')}</option>
                  <option value="down_total" ${this.sortBy === 'down_total' ? 'selected' : ''}>${I18n.t('connections.sort_down_total')}</option>
                  <option value="up_total" ${this.sortBy === 'up_total' ? 'selected' : ''}>${I18n.t('connections.sort_up_total')}</option>
                </select>
              </div>
            </div>

            <div class="conn-actions-group">
              <!-- Pause / Resume Toggle -->
              <button class="btn btn-secondary btn-sm" id="btn-conn-pause" title="${this.isPaused ? I18n.t('connections.resume') : I18n.t('connections.pause')}">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" id="icon-conn-pause">
                  ${this.isPaused ? '<polygon points="5 3 19 12 5 21 5 3"/>' : '<rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/>'}
                </svg>
                <span id="text-conn-pause">${this.isPaused ? I18n.t('connections.resume') : I18n.t('connections.pause')}</span>
              </button>

              <!-- Refresh Button -->
              <button class="btn btn-secondary btn-sm btn-icon" id="btn-conn-refresh" title="${I18n.t('common.refresh')}">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="23 4 23 10 17 10"/>
                  <polyline points="1 20 1 14 7 14"/>
                  <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                </svg>
              </button>

              <!-- Disconnect All Button -->
              <button class="btn btn-danger btn-sm" id="btn-conn-close-all">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
                <span>${I18n.t('connections.close_all')}</span>
              </button>
            </div>
          </div>

          <!-- Connections Data Table -->
          <div class="table-responsive">
            <table class="glass-table conn-table" id="connections-table">
              <thead>
                <tr>
                  <th style="min-width: 240px;">${I18n.t('connections.col_network_host')}</th>
                  <th style="min-width: 160px;">${I18n.t('connections.col_source')}</th>
                  <th style="min-width: 180px;">${I18n.t('connections.col_rule_chain')}</th>
                  <th style="min-width: 140px;">${I18n.t('connections.col_speed')}</th>
                  <th style="min-width: 140px;">${I18n.t('connections.col_traffic')}</th>
                  <th style="min-width: 110px;">${I18n.t('connections.col_duration')}</th>
                  <th style="width: 70px; text-align: center;">${I18n.t('connections.col_actions')}</th>
                </tr>
              </thead>
              <tbody id="connections-table-body">
                <tr>
                  <td colspan="7" class="table-empty">
                    <div class="empty-state-box">
                      <div class="spinner" style="width: 24px; height: 24px; border-width: 2px;"></div>
                      <p>${I18n.t('common.loading')}</p>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      `;

      this.initEvents();
      this.fetchData();
      this.startPolling();
    }

    /**
     * Bind all toolbar, search, and action events
     */
    initEvents() {
      // Client IP select
      const clientSelect = document.getElementById('conn-client-ip-select');
      if (clientSelect) {
        clientSelect.addEventListener('change', (e) => {
          this.filterClientIP = e.target.value;
          this.renderRows();
        });
      }

      // Keyword Search
      const searchInput = document.getElementById('conn-search-input');
      if (searchInput) {
        searchInput.addEventListener('input', (e) => {
          this.searchQuery = e.target.value.toLowerCase().trim();
          this.renderRows();
        });
      }

      // Sort Select
      const sortSelect = document.getElementById('conn-sort-select');
      if (sortSelect) {
        sortSelect.addEventListener('change', (e) => {
          this.sortBy = e.target.value;
          this.renderRows();
        });
      }

      // Pause / Resume Toggle
      const btnPause = document.getElementById('btn-conn-pause');
      if (btnPause) {
        btnPause.addEventListener('click', () => {
          this.isPaused = !this.isPaused;
          const textEl = document.getElementById('text-conn-pause');
          const iconEl = document.getElementById('icon-conn-pause');
          if (textEl) textEl.textContent = this.isPaused ? I18n.t('connections.resume') : I18n.t('connections.pause');
          if (iconEl) {
            iconEl.innerHTML = this.isPaused
              ? '<polygon points="5 3 19 12 5 21 5 3"/>'
              : '<rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/>';
          }
          btnPause.classList.toggle('btn-warning', this.isPaused);
          btnPause.classList.toggle('btn-secondary', !this.isPaused);
          if (!this.isPaused) {
            this.fetchData();
          }
        });
      }

      // Manual Refresh
      const btnRefresh = document.getElementById('btn-conn-refresh');
      if (btnRefresh) {
        btnRefresh.addEventListener('click', () => {
          btnRefresh.classList.add('loading');
          this.fetchData().finally(() => {
            setTimeout(() => btnRefresh.classList.remove('loading'), 300);
          });
        });
      }

      // Close All Connections
      const btnCloseAll = document.getElementById('btn-conn-close-all');
      if (btnCloseAll) {
        btnCloseAll.addEventListener('click', () => {
          this.promptCloseAll();
        });
      }

      // Close single connection row event delegation
      const tableBody = document.getElementById('connections-table-body');
      if (tableBody) {
        tableBody.addEventListener('click', (e) => {
          const btn = e.target.closest('.btn-conn-close');
          if (!btn) return;
          const connId = btn.getAttribute('data-id');
          if (connId) {
            this.closeSingleConnection(connId, btn);
          }
        });
      }
    }

    /**
     * Start Background Polling Interval
     */
    startPolling() {
      this.stopPolling();
      this.pollTimer = setInterval(() => {
        if (!this.isDestroyed && !this.isPaused) {
          this.fetchData();
        }
      }, this.pollIntervalMs);
    }

    /**
     * Stop Polling
     */
    stopPolling() {
      if (this.pollTimer) {
        clearInterval(this.pollTimer);
        this.pollTimer = null;
      }
    }

    /**
     * Fetch Live Connections from backend
     */
    async fetchData() {
      if (this.isFetching) return;
      this.isFetching = true;

      try {
        const rawRes = await API.get('connections', { showToast: false });
        if (this.isDestroyed) return;

        // Unwrap response envelope if present
        const data = (rawRes && rawRes.data) ? rawRes.data : (rawRes || {});
        const now = Date.now();
        const incomingConns = Array.isArray(data.connections) ? data.connections : [];

        // 1. Calculate overall download & upload speeds
        const downloadTotal = Number(data.downloadTotal || 0);
        const uploadTotal = Number(data.uploadTotal || 0);

        if (this.prevTotal.timestamp > 0) {
          const dt = (now - this.prevTotal.timestamp) / 1000;
          if (dt > 0.4) {
            this.stats.liveDownloadSpeed = Math.max(0, (downloadTotal - this.prevTotal.download) / dt);
            this.stats.liveUploadSpeed = Math.max(0, (uploadTotal - this.prevTotal.upload) / dt);
          }
        }
        this.prevTotal = {
          download: downloadTotal,
          upload: uploadTotal,
          timestamp: now
        };
        this.stats.downloadTotal = downloadTotal;
        this.stats.uploadTotal = uploadTotal;

        // 2. Process individual connections & compute per-connection speeds
        const newSnapshots = new Map();
        const processedConns = incomingConns.map((conn) => {
          const id = conn.id;
          const download = Number(conn.download || 0);
          const upload = Number(conn.upload || 0);
          let downSpeed = 0;
          let upSpeed = 0;

          if (this.prevSnapshots.has(id)) {
            const prev = this.prevSnapshots.get(id);
            const dt = (now - prev.timestamp) / 1000;
            if (dt > 0.4) {
              downSpeed = Math.max(0, (download - prev.download) / dt);
              upSpeed = Math.max(0, (upload - prev.upload) / dt);
            }
          }

          newSnapshots.set(id, { download, upload, timestamp: now });

          // Calculate connection duration in seconds
          let durationSeconds = 0;
          if (conn.start) {
            const startMs = new Date(conn.start).getTime();
            if (!isNaN(startMs)) {
              durationSeconds = Math.max(0, Math.floor((now - startMs) / 1000));
            }
          }

          return {
            ...conn,
            download,
            upload,
            downSpeed,
            upSpeed,
            durationSeconds
          };
        });

        this.prevSnapshots = newSnapshots;
        this.connections = processedConns;

        // 3. Count unique clients
        const clientCounts = new Map();
        for (const conn of processedConns) {
          const srcIP = (conn.metadata && conn.metadata.sourceIP) ? conn.metadata.sourceIP : '127.0.0.1';
          clientCounts.set(srcIP, (clientCounts.get(srcIP) || 0) + 1);
        }

        this.stats.totalConnections = processedConns.length;
        this.stats.activeClients = clientCounts.size;

        // 4. Update Summary Cards & Client Dropdown
        this.updateSummaryCards();
        this.updateClientIPDropdown(clientCounts);

        // 5. Render Table Rows
        this.renderRows();

      } catch (err) {
        console.warn('[Connections] Fetch error:', err);
      } finally {
        this.isFetching = false;
      }
    }

    /**
     * Update top statistics cards
     */
    updateSummaryCards() {
      const activeEl = document.getElementById('conn-stat-active');
      const activeSub = document.getElementById('conn-stat-active-sub');
      const clientsEl = document.getElementById('conn-stat-clients');
      const downSpeedEl = document.getElementById('conn-stat-down-speed');
      const downTotalEl = document.getElementById('conn-stat-down-total');
      const upSpeedEl = document.getElementById('conn-stat-up-speed');
      const upTotalEl = document.getElementById('conn-stat-up-total');

      if (activeEl) activeEl.textContent = this.stats.totalConnections;
      if (activeSub) activeSub.textContent = I18n.t('connections.sub_active_desc');
      if (clientsEl) clientsEl.textContent = this.stats.activeClients;

      if (downSpeedEl) downSpeedEl.textContent = window.formatSpeed(this.stats.liveDownloadSpeed);
      if (downTotalEl) downTotalEl.textContent = `${I18n.t('connections.stat_total_down')}: ${window.formatBytes(this.stats.downloadTotal)}`;

      if (upSpeedEl) upSpeedEl.textContent = window.formatSpeed(this.stats.liveUploadSpeed);
      if (upTotalEl) upTotalEl.textContent = `${I18n.t('connections.stat_total_up')}: ${window.formatBytes(this.stats.uploadTotal)}`;
    }

    /**
     * Update Client IP filter select options while preserving selection
     */
    updateClientIPDropdown(clientCounts) {
      const select = document.getElementById('conn-client-ip-select');
      if (!select) return;

      const currentVal = this.filterClientIP;
      const sortedIPs = Array.from(clientCounts.keys()).sort();

      let optionsHtml = `<option value="all">${I18n.t('connections.all_clients')} (${this.stats.totalConnections})</option>`;
      for (const ip of sortedIPs) {
        const count = clientCounts.get(ip);
        const isSel = (ip === currentVal) ? 'selected' : '';
        optionsHtml += `<option value="${ip}" ${isSel}>${ip} (${count})</option>`;
      }

      // Only re-write DOM if options list has changed
      if (select.dataset.lastSignature !== `${sortedIPs.join(',')}:${this.stats.totalConnections}`) {
        select.innerHTML = optionsHtml;
        select.dataset.lastSignature = `${sortedIPs.join(',')}:${this.stats.totalConnections}`;
        // Restore value if still valid
        if (currentVal !== 'all' && clientCounts.has(currentVal)) {
          select.value = currentVal;
        } else if (currentVal !== 'all') {
          this.filterClientIP = 'all';
          select.value = 'all';
        }
      }
    }

    /**
     * Filter, Sort, and Render Table Rows
     */
    renderRows() {
      const tbody = document.getElementById('connections-table-body');
      if (!tbody) return;

      // Filter by Client IP
      let filtered = this.connections.filter((conn) => {
        if (this.filterClientIP !== 'all') {
          const srcIP = (conn.metadata && conn.metadata.sourceIP) ? conn.metadata.sourceIP : '';
          if (srcIP !== this.filterClientIP) return false;
        }

        // Filter by Keyword Search
        if (this.searchQuery) {
          const q = this.searchQuery;
          const meta = conn.metadata || {};
          const host = (meta.host || '').toLowerCase();
          const dstIP = (meta.destinationIP || '').toLowerCase();
          const dstPort = String(meta.destinationPort || '');
          const srcIP = (meta.sourceIP || '').toLowerCase();
          const srcPort = String(meta.sourcePort || '');
          const net = (meta.network || '').toLowerCase();
          const rule = (conn.rule || '').toLowerCase();
          const rulePayload = (conn.rulePayload || '').toLowerCase();
          const chainStr = Array.isArray(conn.chains) ? conn.chains.join(' ').toLowerCase() : '';

          const match = host.includes(q) ||
                        dstIP.includes(q) ||
                        dstPort.includes(q) ||
                        srcIP.includes(q) ||
                        srcPort.includes(q) ||
                        net.includes(q) ||
                        rule.includes(q) ||
                        rulePayload.includes(q) ||
                        chainStr.includes(q);
          if (!match) return false;
        }

        return true;
      });

      // Sort
      filtered.sort((a, b) => {
        switch (this.sortBy) {
          case 'time_desc':
            return (a.durationSeconds || 0) - (b.durationSeconds || 0); // newest first (smaller duration)
          case 'time_asc':
            return (b.durationSeconds || 0) - (a.durationSeconds || 0); // longest first (larger duration)
          case 'down_speed':
            return (b.downSpeed || 0) - (a.downSpeed || 0);
          case 'up_speed':
            return (b.upSpeed || 0) - (a.upSpeed || 0);
          case 'down_total':
            return (b.download || 0) - (a.download || 0);
          case 'up_total':
            return (b.upload || 0) - (a.upload || 0);
          default:
            return 0;
        }
      });

      if (filtered.length === 0) {
        tbody.innerHTML = `
          <tr>
            <td colspan="7" class="table-empty">
              <div class="empty-state-box">
                <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                  <polyline points="15 3 21 3 21 9"/>
                  <line x1="10" y1="14" x2="21" y2="3"/>
                </svg>
                <p>${I18n.t('connections.no_connections')}</p>
              </div>
            </td>
          </tr>
        `;
        return;
      }

      // Generate Table Rows
      tbody.innerHTML = filtered.map((conn) => {
        const meta = conn.metadata || {};
        const network = (meta.network || 'tcp').toUpperCase();
        const isUDP = network === 'UDP';
        const host = meta.host || '';
        const dstIP = meta.destinationIP || '';
        const dstPort = meta.destinationPort || '';
        const srcIP = meta.sourceIP || '127.0.0.1';
        const srcPort = meta.sourcePort || '';

        // Display Target Header: Host name if available, else destination IP
        const primaryTarget = host ? host : `${dstIP}:${dstPort}`;
        const targetSubtitle = host ? `${dstIP}:${dstPort}` : (meta.type ? `${meta.type}` : '');

        // Outbound Chain & Rule Pills
        const chains = Array.isArray(conn.chains) && conn.chains.length > 0
          ? conn.chains
          : ['direct'];
        const chainHtml = chains.map((c, idx) => {
          const isLast = idx === chains.length - 1;
          const badgeClass = isLast ? 'badge-primary' : 'badge-neutral';
          return `<span class="badge ${badgeClass} conn-chain-node">${this.escapeHtml(c)}</span>`;
        }).join('<span class="conn-chain-arrow">→</span>');

        // Rule payload pill
        let ruleHtml = '';
        if (conn.rule) {
          const ruleLabel = conn.rulePayload ? `${conn.rule}: ${conn.rulePayload}` : conn.rule;
          ruleHtml = `<div class="conn-rule-tag"><span class="badge badge-info" title="${this.escapeHtml(ruleLabel)}">${this.escapeHtml(ruleLabel)}</span></div>`;
        }

        // Speeds & Traffics
        const downSpeedFormatted = window.formatSpeed(conn.downSpeed || 0);
        const upSpeedFormatted = window.formatSpeed(conn.upSpeed || 0);
        const downTotalFormatted = window.formatBytes(conn.download || 0);
        const upTotalFormatted = window.formatBytes(conn.upload || 0);

        // Duration formatted
        const durationFormatted = this.formatDuration(conn.durationSeconds || 0);

        return `
          <tr data-conn-id="${conn.id}">
            <!-- Column 1: Network & Target Host -->
            <td>
              <div class="conn-target-cell">
                <span class="badge ${isUDP ? 'badge-warning' : 'badge-cyan'} conn-net-badge">${network}</span>
                <div class="conn-target-info">
                  <div class="conn-primary-target" title="${this.escapeHtml(primaryTarget)}">
                    ${this.escapeHtml(primaryTarget)}
                  </div>
                  ${targetSubtitle ? `<div class="conn-sub-target">${this.escapeHtml(targetSubtitle)}</div>` : ''}
                </div>
              </div>
            </td>

            <!-- Column 2: Source Client IP -->
            <td>
              <div class="conn-source-cell">
                <span class="conn-source-ip">${this.escapeHtml(srcIP)}</span>
                <span class="conn-source-port">:${this.escapeHtml(srcPort)}</span>
              </div>
            </td>

            <!-- Column 3: Rule & Outbound Chain -->
            <td>
              <div class="conn-chain-cell">
                <div class="conn-chain-list">${chainHtml}</div>
                ${ruleHtml}
              </div>
            </td>

            <!-- Column 4: Real-time Speed -->
            <td>
              <div class="conn-speed-cell">
                <div class="conn-speed-item text-success">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="19 12 12 19 5 12"/><line x1="12" y1="5" x2="12" y2="19"/></svg>
                  <span>${downSpeedFormatted}</span>
                </div>
                <div class="conn-speed-item text-purple">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="5 12 12 5 19 12"/><line x1="12" y1="19" x2="12" y2="5"/></svg>
                  <span>${upSpeedFormatted}</span>
                </div>
              </div>
            </td>

            <!-- Column 5: Total Traffic -->
            <td>
              <div class="conn-traffic-cell">
                <div class="conn-traffic-item">
                  <span class="traffic-dir">↓</span>
                  <span>${downTotalFormatted}</span>
                </div>
                <div class="conn-traffic-item">
                  <span class="traffic-dir">↑</span>
                  <span>${upTotalFormatted}</span>
                </div>
              </div>
            </td>

            <!-- Column 6: Duration -->
            <td>
              <span class="conn-duration-badge">${durationFormatted}</span>
            </td>

            <!-- Column 7: Actions (Close Single Connection) -->
            <td style="text-align: center;">
              <button class="btn btn-sm btn-icon btn-conn-close" data-id="${conn.id}" title="${I18n.t('connections.close_conn')}" aria-label="${I18n.t('connections.close_conn')}">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/>
                  <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </td>
          </tr>
        `;
      }).join('');
    }

    /**
     * Terminate a single connection by ID
     */
    async closeSingleConnection(id, buttonEl) {
      if (buttonEl) {
        buttonEl.disabled = true;
        buttonEl.classList.add('loading');
      }

      try {
        await API.del(`connections/${encodeURIComponent(id)}`);
        if (window.showToast) {
          window.showToast(I18n.t('connections.closed_single'), 'success', 2000);
        }
        // Remove locally immediately for snappy responsiveness
        this.connections = this.connections.filter(c => c.id !== id);
        this.prevSnapshots.delete(id);
        this.stats.totalConnections = this.connections.length;
        this.updateSummaryCards();
        this.renderRows();
      } catch (err) {
        if (window.showToast) {
          window.showToast(`${I18n.t('common.error')}: ${err.message}`, 'error');
        }
      } finally {
        if (buttonEl) {
          buttonEl.disabled = false;
          buttonEl.classList.remove('loading');
        }
      }
    }

    /**
     * Prompt modal confirmation to terminate all connections
     */
    promptCloseAll() {
      const modalBody = `
        <div class="confirm-modal-body">
          <div class="confirm-icon danger">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
              <line x1="12" y1="9" x2="12" y2="13"/>
              <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
          </div>
          <div class="confirm-text">
            <h4>${I18n.t('connections.close_all_title')}</h4>
            <p>${I18n.t('connections.close_all_desc')}</p>
          </div>
        </div>
      `;

      window.showModal({
        title: I18n.t('connections.close_all_title'),
        bodyHtml: modalBody,
        saveText: I18n.t('connections.close_all_confirm'),
        cancelText: I18n.t('common.cancel'),
        size: 'sm',
        onSave: async () => {
          try {
            await API.del('connections');
            if (window.showToast) {
              window.showToast(I18n.t('connections.closed_all_success'), 'success', 3000);
            }
            window.closeModal();
            // Clear connections locally and fetch
            this.connections = [];
            this.prevSnapshots.clear();
            this.stats.totalConnections = 0;
            this.updateSummaryCards();
            this.renderRows();
            await this.fetchData();
          } catch (err) {
            if (window.showToast) {
              window.showToast(`${I18n.t('common.error')}: ${err.message}`, 'error');
            }
          }
        }
      });
    }

    /**
     * Format duration seconds to human readable string (e.g. 5s, 3m 12s, 1h 45m)
     */
    formatDuration(seconds) {
      if (!seconds || seconds <= 0) return '< 1s';
      if (seconds < 60) return `${seconds}s`;
      const m = Math.floor(seconds / 60);
      const s = seconds % 60;
      if (m < 60) return `${m}m ${s}s`;
      const h = Math.floor(m / 60);
      const remM = m % 60;
      return `${h}h ${remM}m`;
    }

    /**
     * Escape HTML string
     */
    escapeHtml(str) {
      return String(str || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
    }

    /**
     * Teardown timer on navigating away
     */
    destroy() {
      this.isDestroyed = true;
      this.stopPolling();
    }
  }

  window.Connections = new ConnectionsPage();
})();
