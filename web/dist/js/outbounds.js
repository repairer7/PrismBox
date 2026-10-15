/**
 * PrismBox - Outbounds & Proxy Nodes Management Module
 * Comprehensive support for VLESS, VMess, Trojan, Shadowsocks, Hysteria 2, TUIC
 * Link import parser, Subscription manager, Group manager, and Latency Speed Test
 */

(function () {
  'use strict';

  class OutboundsPage {
    constructor() {
      this.nodes = [];
      this.groups = [];
      this.selectedNodeIds = new Set();
      this.isSpeedTesting = false;
      this.searchQuery = '';
      this.filterType = 'all';
    }

    /**
     * Render Outbounds View
     */
    async render() {
      const app = document.getElementById('app');
      if (!app) return;

      app.innerHTML = `
        <div class="outbounds-page">
          <!-- Page Header & Action Cluster -->
          <div class="section-header">
            <div>
              <h2 class="section-title">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="2" y1="12" x2="22" y2="12"/>
                  <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
                </svg>
                ${I18n.t('outbound.title')}
              </h2>
              <span class="section-subtitle">${I18n.t('outbound.subtitle')}</span>
            </div>

            <div class="actions-row">
              <button class="btn btn-warning btn-sm" id="btn-speedtest-all">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
                </svg>
                <span id="speedtest-btn-text">${I18n.t('outbound.speedtest')}</span>
              </button>

              <button class="btn btn-secondary btn-sm" id="btn-import-links">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                  <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                </svg>
                <span>${I18n.t('outbound.import')}</span>
              </button>

              <button class="btn btn-secondary btn-sm" id="btn-add-subscription">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M4 11a9 9 0 0 1 9 9"/>
                  <path d="M4 4a16 16 0 0 1 16 16"/>
                  <circle cx="5" cy="19" r="1"/>
                </svg>
                <span>${I18n.t('outbound.subscribe')}</span>
              </button>

              <button class="btn btn-primary btn-sm" id="btn-add-node">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
                <span>${I18n.t('outbound.add')}</span>
              </button>
            </div>
          </div>

          <!-- Section: Policy Groups -->
          <div class="groups-section mb-4">
            <div class="panel-header mb-2">
              <h3 class="panel-title">
                <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="2" y="7" width="20" height="14" rx="2" ry="2"/>
                  <path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>
                </svg>
                ${I18n.t('outbound.groups_title')}
              </h3>
              <button class="btn btn-ghost btn-sm" id="btn-add-group">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
                <span>${I18n.t('outbound.add_group')}</span>
              </button>
            </div>

            <div class="groups-grid" id="groups-grid-container">
              <div class="glass-panel text-center py-3 text-muted">
                <span class="spinner-sm"></span>
              </div>
            </div>
          </div>

          <!-- Section: Nodes List Header with Search & Filters -->
          <div class="nodes-section">
            <div class="panel-header mb-3">
              <div class="flex-center gap-sm">
                <h3 class="panel-title">
                  <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <polygon points="16.24 7.76 14.12 14.12 7.76 16.24 9.88 9.88 16.24 7.76"/>
                  </svg>
                  ${I18n.t('outbound.nodes_title')}
                </h3>
                <span class="badge badge-neutral" id="nodes-count-badge">0</span>
              </div>

              <div class="actions-row">
                <div class="filter-pills" id="node-type-filters">
                  <button class="tab-pill ${this.filterType === 'all' ? 'active' : ''}" data-type="all">${I18n.t('common.all')}</button>
                  <button class="tab-pill ${this.filterType === 'vless' ? 'active' : ''}" data-type="vless">VLESS</button>
                  <button class="tab-pill ${this.filterType === 'hysteria2' ? 'active' : ''}" data-type="hysteria2">Hy2</button>
                  <button class="tab-pill ${this.filterType === 'trojan' ? 'active' : ''}" data-type="trojan">Trojan</button>
                  <button class="tab-pill ${this.filterType === 'shadowsocks' ? 'active' : ''}" data-type="shadowsocks">SS</button>
                  <button class="tab-pill ${this.filterType === 'vmess' ? 'active' : ''}" data-type="vmess">VMess</button>
                </div>

                <div class="search-input-box">
                  <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="11" cy="11" r="8"/>
                    <line x1="21" y1="21" x2="16.65" y2="16.65"/>
                  </svg>
                  <input type="text" class="form-control" id="nodes-search" placeholder="${I18n.t('common.search')}">
                </div>
              </div>
            </div>

            <!-- Nodes Cards Grid -->
            <div class="nodes-grid" id="nodes-grid-container">
              <div class="glass-panel text-center py-4 text-muted">
                <span class="spinner-sm"></span>
                <p class="mt-2">${I18n.t('common.loading')}</p>
              </div>
            </div>
          </div>
        </div>
      `;

      this.bindEvents();
      await this.loadAll();
    }

    /**
     * Bind DOM events
     */
    bindEvents() {
      // Speed test button
      const speedtestBtn = document.getElementById('btn-speedtest-all');
      if (speedtestBtn) {
        speedtestBtn.addEventListener('click', () => this.runSpeedTest());
      }

      // Import links button
      const importBtn = document.getElementById('btn-import-links');
      if (importBtn) {
        importBtn.addEventListener('click', () => this.openImportModal());
      }

      // Add subscription button
      const subBtn = document.getElementById('btn-add-subscription');
      if (subBtn) {
        subBtn.addEventListener('click', () => this.openSubscriptionModal());
      }

      // Add node button
      const addNodeBtn = document.getElementById('btn-add-node');
      if (addNodeBtn) {
        addNodeBtn.addEventListener('click', () => this.openNodeModal());
      }

      // Add group button
      const addGroupBtn = document.getElementById('btn-add-group');
      if (addGroupBtn) {
        addGroupBtn.addEventListener('click', () => this.openGroupModal());
      }

      // Search input
      const searchInput = document.getElementById('nodes-search');
      if (searchInput) {
        searchInput.addEventListener('input', (e) => {
          this.searchQuery = e.target.value.toLowerCase().trim();
          this.renderNodes();
        });
      }

      // Type filter pills
      const filterContainer = document.getElementById('node-type-filters');
      if (filterContainer) {
        filterContainer.querySelectorAll('.tab-pill').forEach(btn => {
          btn.addEventListener('click', (e) => {
            filterContainer.querySelectorAll('.tab-pill').forEach(b => b.classList.remove('active'));
            e.currentTarget.classList.add('active');
            this.filterType = e.currentTarget.dataset.type;
            this.renderNodes();
          });
        });
      }
    }

    /**
     * Load outbounds & groups from backend
     */
    async loadAll() {
      try {
        const [nodesRes, groupsRes] = await Promise.all([
          API.get('outbounds').catch(() => []),
          API.get('groups').catch(() => [])
        ]);

        this.nodes = Array.isArray(nodesRes) ? nodesRes : [];
        this.groups = Array.isArray(groupsRes) ? groupsRes : [];
      } catch (err) {
        console.error('[Outbounds] Load error:', err);
      } finally {
        this.renderGroups();
        this.renderNodes();
      }
    }

    /**
     * Render Groups Cards
     */
    renderGroups() {
      const container = document.getElementById('groups-grid-container');
      if (!container) return;

      if (!this.groups || this.groups.length === 0) {
        container.innerHTML = `
          <div class="glass-panel text-center py-3 text-muted" style="grid-column: 1 / -1;">
            ${I18n.t('outbound.no_groups')}
          </div>
        `;
        return;
      }

      container.innerHTML = this.groups.map(grp => {
        const isSelector = grp.type === 'selector';
        const typeBadge = isSelector ? 'badge-primary' : 'badge-warning';
        const typeLabel = isSelector ? I18n.t('outbound.group_selector') : I18n.t('outbound.group_urltest');
        const includedTags = grp.outbounds || [];

        return `
          <div class="glass-panel group-card" data-id="${grp.id}">
            <div class="group-card-header">
              <div class="group-info">
                <span class="group-tag"><strong>${this.escapeHtml(grp.tag)}</strong></span>
                <span class="badge ${typeBadge} badge-sm">${typeLabel}</span>
              </div>
              <div class="group-actions">
                <button class="btn-icon btn-sm btn-edit-group" data-id="${grp.id}" title="${I18n.t('common.edit')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </button>
                <button class="btn-icon btn-sm btn-delete-group text-danger" data-id="${grp.id}" title="${I18n.t('common.delete')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                  </svg>
                </button>
              </div>
            </div>

            <div class="group-card-body mt-2">
              <div class="active-node-selector">
                <label class="form-label text-xs">${I18n.t('outbound.active_node')}:</label>
                <select class="form-select form-select-sm group-active-select" data-id="${grp.id}">
                  ${includedTags.map(tag => `<option value="${this.escapeHtml(tag)}">${this.escapeHtml(tag)}</option>`).join('')}
                </select>
              </div>

              ${!isSelector ? `
                <div class="group-meta mt-2 text-xs text-muted">
                  <span>URL: <code>${this.escapeHtml(grp.url || 'http://cp.cloudflare.com/generate_204')}</code></span>
                  <span>Interval: <code>${this.escapeHtml(grp.interval || '3m')}</code></span>
                </div>
              ` : ''}
            </div>
          </div>
        `;
      }).join('');

      // Bind group card events
      container.querySelectorAll('.btn-edit-group').forEach(btn => {
        btn.addEventListener('click', () => {
          const grp = this.groups.find(g => g.id === btn.dataset.id);
          if (grp) this.openGroupModal(grp);
        });
      });

      container.querySelectorAll('.btn-delete-group').forEach(btn => {
        btn.addEventListener('click', () => {
          const grp = this.groups.find(g => g.id === btn.dataset.id);
          if (grp) this.deleteGroup(grp);
        });
      });
    }

    /**
     * Render Node Cards Grid
     */
    renderNodes() {
      const container = document.getElementById('nodes-grid-container');
      const countBadge = document.getElementById('nodes-count-badge');
      if (!container) return;

      const filtered = this.nodes.filter(n => {
        if (this.filterType !== 'all' && n.type !== this.filterType) return false;
        if (!this.searchQuery) return true;
        const q = this.searchQuery;
        return (n.tag || '').toLowerCase().includes(q) ||
               (n.server || '').toLowerCase().includes(q) ||
               (n.type || '').toLowerCase().includes(q) ||
               (n.subscription_tag || '').toLowerCase().includes(q);
      });

      if (countBadge) {
        countBadge.textContent = `${filtered.length} / ${this.nodes.length}`;
      }

      if (filtered.length === 0) {
        container.innerHTML = `
          <div class="glass-panel text-center py-4 text-muted" style="grid-column: 1 / -1;">
            ${I18n.t('outbound.no_nodes')}
          </div>
        `;
        return;
      }

      container.innerHTML = filtered.map(node => {
        const typeBadges = {
          vless: 'badge-primary',
          hysteria2: 'badge-cyan',
          trojan: 'badge-warning',
          shadowsocks: 'badge-purple',
          vmess: 'badge-info',
          tuic: 'badge-pink',
          direct: 'badge-success',
          block: 'badge-danger'
        };

        let latencyBadge = '';
        if (typeof node.latency === 'number' && node.latency > 0) {
          const latencyClass = node.latency < 100 ? 'badge-success' : (node.latency < 250 ? 'badge-warning' : 'badge-danger');
          latencyBadge = `<span class="badge ${latencyClass} latency-badge">${node.latency} ms</span>`;
        } else if (node.latency === -1) {
          latencyBadge = `<span class="badge badge-danger latency-badge">Timeout</span>`;
        } else {
          latencyBadge = `<span class="badge badge-neutral latency-badge">--</span>`;
        }

        return `
          <div class="glass-panel node-card" data-id="${node.id}">
            <div class="node-card-header">
              <div class="node-title-box">
                <span class="badge ${typeBadges[node.type] || 'badge-neutral'} uppercase">${node.type}</span>
                <span class="node-tag" title="${this.escapeHtml(node.tag)}"><strong>${this.escapeHtml(node.tag)}</strong></span>
              </div>
              <div class="node-latency-box">
                ${latencyBadge}
              </div>
            </div>

            <div class="node-card-body mt-2">
              <div class="node-endpoint">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
                  <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
                  <line x1="6" y1="6" x2="6.01" y2="6"/>
                  <line x1="6" y1="18" x2="6.01" y2="18"/>
                </svg>
                <span class="endpoint-addr">
                  ${node.server ? `${this.escapeHtml(node.server)}:${node.server_port || ''}` : '<span class="text-muted">Direct / Built-in</span>'}
                </span>
              </div>

              ${node.subscription_tag ? `
                <div class="node-sub-tag mt-1">
                  <span class="badge badge-neutral badge-sm">Sub: ${this.escapeHtml(node.subscription_tag)}</span>
                </div>
              ` : ''}
            </div>

            <div class="node-card-footer mt-3">
              <div class="footer-left">
                <button class="btn-icon btn-sm btn-test-single" data-id="${node.id}" title="${I18n.t('outbound.speedtest')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
                  </svg>
                </button>
              </div>

              <div class="footer-right">
                <button class="btn-icon btn-sm btn-edit-node" data-id="${node.id}" title="${I18n.t('common.edit')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </button>
                <button class="btn-icon btn-sm btn-delete-node text-danger" data-id="${node.id}" title="${I18n.t('common.delete')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        `;
      }).join('');

      // Bind node actions
      container.querySelectorAll('.btn-test-single').forEach(btn => {
        btn.addEventListener('click', () => this.runSpeedTest([btn.dataset.id]));
      });

      container.querySelectorAll('.btn-edit-node').forEach(btn => {
        btn.addEventListener('click', () => {
          const node = this.nodes.find(n => n.id === btn.dataset.id);
          if (node) this.openNodeModal(node);
        });
      });

      container.querySelectorAll('.btn-delete-node').forEach(btn => {
        btn.addEventListener('click', () => {
          const node = this.nodes.find(n => n.id === btn.dataset.id);
          if (node) this.deleteNode(node);
        });
      });
    }

    /**
     * Run Speed Test for all or specific nodes
     */
    async runSpeedTest(ids = null) {
      if (this.isSpeedTesting) return;
      this.isSpeedTesting = true;

      const btn = document.getElementById('btn-speedtest-all');
      const text = document.getElementById('speedtest-btn-text');
      if (btn) btn.classList.add('loading');
      if (text) text.textContent = I18n.t('outbound.speedtesting');

      try {
        const payload = ids ? { ids } : {};
        const res = await API.post('outbounds/speedtest', payload);

        if (res && Array.isArray(res.results)) {
          for (const item of res.results) {
            const node = this.nodes.find(n => n.id === item.id);
            if (node) {
              node.latency = item.latency;
            }
          }
          this.renderNodes();
          window.showToast(I18n.t('common.success'), 'success', 2000);
        }
      } catch (err) {
        window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
      } finally {
        this.isSpeedTesting = false;
        if (btn) btn.classList.remove('loading');
        if (text) text.textContent = I18n.t('outbound.speedtest');
      }
    }

    /**
     * Import Links modal (pasting share links: vless://, vmess://, trojan://, ss://, hysteria2://, tuic://)
     */
    openImportModal() {
      const bodyHtml = `
        <div class="import-links-modal">
          <div class="form-group">
            <label class="form-label">${I18n.t('outbound.import_hint')}</label>
            <textarea class="form-control code-editor-textarea" id="import-links-textarea" rows="10" placeholder="vless://...\nhysteria2://...\ntrojan://...\nss://..."></textarea>
          </div>
        </div>
      `;

      window.showModal({
        title: I18n.t('outbound.import'),
        bodyHtml,
        saveText: I18n.t('outbound.import'),
        cancelText: I18n.t('common.cancel'),
        size: 'lg',
        onSave: async () => {
          const textarea = document.getElementById('import-links-textarea');
          if (!textarea) return false;

          const lines = textarea.value.split('\n').map(l => l.trim()).filter(Boolean);
          if (lines.length === 0) {
            window.showToast(I18n.t('outbound.import_hint'), 'warning');
            return false;
          }

          try {
            const res = await API.post('outbounds/import-links', { links: textarea.value });
            if (res && res.success) {
              const count = res.count || (res.outbounds && res.outbounds.length) || 1;
              window.showToast(I18n.t('outbound.import_success', { count }), 'success');
              await this.loadAll();
              return true;
            }
          } catch (e) {
            console.warn('[Import Link] API call failed, trying client-side fallback:', e);
          }

          let addedCount = 0;
          for (const line of lines) {
            const parsed = this.parseShareLink(line);
            if (parsed) {
              try {
                await API.post('outbounds', parsed);
                addedCount++;
              } catch (e) {
                console.warn('[Import Link] Error adding node:', e);
              }
            }
          }

          if (addedCount > 0) {
            window.showToast(I18n.t('outbound.import_success', { count: addedCount }), 'success');
            await this.loadAll();
            return true;
          } else {
            window.showToast(I18n.t('common.error'), 'error');
            return false;
          }
        }
      });
    }

    /**
     * Client-side Parser for standard proxy links
     */
    parseShareLink(link) {
      try {
        const u = new URL(link);
        const protocol = u.protocol.replace(':', '').toLowerCase();

        const insecure = u.searchParams.get('insecure') === '1' ||
          u.searchParams.get('insecure') === 'true' ||
          u.searchParams.get('allowInsecure') === '1' ||
          u.searchParams.get('allowInsecure') === 'true';

        let node = {
          type: protocol,
          tag: decodeURIComponent(u.hash ? u.hash.slice(1) : `${protocol}-${u.hostname}`),
          server: u.hostname,
          server_port: parseInt(u.port, 10) || 443,
          enabled: true,
          settings: {}
        };

        if (protocol === 'vless') {
          node.settings.uuid = u.username;
          const flow = u.searchParams.get('flow');
          if (flow) node.settings.flow = flow;
          const security = u.searchParams.get('security');
          if (security === 'reality') {
            node.settings.tls = {
              enabled: true,
              server_name: u.searchParams.get('sni') || u.hostname,
              reality: {
                enabled: true,
                public_key: u.searchParams.get('pbk') || '',
                short_id: u.searchParams.get('sid') || ''
              }
            };
          } else if (security === 'tls') {
            node.settings.tls = {
              enabled: true,
              server_name: u.searchParams.get('sni') || u.hostname,
              insecure: insecure
            };
          }
        } else if (protocol === 'hysteria2' || protocol === 'hy2') {
          node.type = 'hysteria2';
          node.settings.password = decodeURIComponent(u.username || u.password || '');
          const sni = u.searchParams.get('sni');
          const tls = { enabled: true };
          if (sni) tls.server_name = sni;
          if (insecure) tls.insecure = true;
          node.settings.tls = tls;
        } else if (protocol === 'trojan') {
          node.settings.password = decodeURIComponent(u.username || '');
          const sni = u.searchParams.get('sni');
          const tls = { enabled: true };
          if (sni) tls.server_name = sni;
          if (insecure) tls.insecure = true;
          node.settings.tls = tls;
        } else if (protocol === 'ss') {
          node.type = 'shadowsocks';
          let userInfo = u.username;
          if (!u.password && userInfo) {
            try {
              userInfo = atob(userInfo);
            } catch (e) {}
          } else if (u.password) {
            userInfo = `${u.username}:${u.password}`;
          }
          const [method, pass] = userInfo.split(':');
          node.settings.method = method || '2022-blake3-aes-128-gcm';
          node.settings.password = pass || '';
        } else if (protocol === 'tuic') {
          node.settings.uuid = u.username;
          node.settings.password = u.password;
          const sni = u.searchParams.get('sni');
          const tls = { enabled: true };
          if (sni) tls.server_name = sni;
          if (insecure) tls.insecure = true;
          node.settings.tls = tls;
        } else {
          return null;
        }

        return node;
      } catch (e) {
        return null;
      }
    }

    /**
     * Add Subscription Modal
     */
    openSubscriptionModal() {
      const bodyHtml = `
        <div class="sub-form-modal">
          <div class="form-group">
            <label class="form-label">${I18n.t('outbound.sub_url')} *</label>
            <input type="url" class="form-control" id="modal-sub-url" placeholder="https://example.com/api/v1/client/subscribe?token=..." required>
          </div>
          <div class="form-group">
            <label class="form-label">${I18n.t('outbound.sub_tag')}</label>
            <input type="text" class="form-control" id="modal-sub-tag" placeholder="My-Subscription">
          </div>
        </div>
      `;

      window.showModal({
        title: I18n.t('outbound.subscribe'),
        bodyHtml,
        saveText: I18n.t('common.confirm'),
        cancelText: I18n.t('common.cancel'),
        onSave: async () => {
          const url = document.getElementById('modal-sub-url')?.value.trim();
          if (!url) {
            window.showToast(I18n.t('outbound.sub_url') + ' is required', 'error');
            return false;
          }

          try {
            window.showToast(I18n.t('common.loading'), 'info', 2000);
            const res = await API.post('outbounds/subscribe', { url });
            const count = res?.count || (res?.outbounds || []).length;
            window.showToast(I18n.t('outbound.import_success', { count }), 'success');
            await this.loadAll();
            return true;
          } catch (err) {
            window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
            return false;
          }
        }
      });
    }

    /**
     * Add or Edit Node Modal
     */
    openNodeModal(existing = null) {
      const isEdit = !!existing;
      const data = existing || {
        tag: 'node-1',
        type: 'vless',
        server: '',
        server_port: 443,
        enabled: true,
        settings: {}
      };

      const bodyHtml = `
        <div class="node-form-modal">
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('outbound.type')}</label>
              <select class="form-select" id="modal-node-type">
                <option value="vless" ${data.type === 'vless' ? 'selected' : ''}>VLESS</option>
                <option value="hysteria2" ${data.type === 'hysteria2' ? 'selected' : ''}>Hysteria 2</option>
                <option value="trojan" ${data.type === 'trojan' ? 'selected' : ''}>Trojan</option>
                <option value="shadowsocks" ${data.type === 'shadowsocks' ? 'selected' : ''}>Shadowsocks</option>
                <option value="vmess" ${data.type === 'vmess' ? 'selected' : ''}>VMess</option>
                <option value="tuic" ${data.type === 'tuic' ? 'selected' : ''}>TUIC</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('common.tag')} *</label>
              <input type="text" class="form-control" id="modal-node-tag" value="${this.escapeHtml(data.tag || '')}" required>
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('outbound.server')} *</label>
              <input type="text" class="form-control" id="modal-node-server" value="${this.escapeHtml(data.server || '')}" required>
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('outbound.port')} *</label>
              <input type="number" class="form-control" id="modal-node-port" value="${data.server_port || 443}" min="1" max="65535" required>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">Password / UUID / Secret</label>
            <input type="text" class="form-control" id="modal-node-secret" value="${this.escapeHtml(data.settings?.password || data.settings?.uuid || '')}">
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">SNI / Server Name</label>
              <input type="text" class="form-control" id="modal-node-sni" value="${this.escapeHtml(data.settings?.tls?.server_name || '')}" placeholder="e.g. bing.com">
            </div>
            <div class="form-group d-flex flex-column justify-content-center">
              <label class="form-label" style="margin-bottom: 8px;">TLS 证书校验</label>
              <div class="form-check">
                <input type="checkbox" class="form-check-input" id="modal-node-insecure" ${data.settings?.tls?.insecure ? 'checked' : ''}>
                <label class="form-check-label" for="modal-node-insecure">跳过证书校验 (Insecure / 允许自签证书)</label>
              </div>
            </div>
          </div>
        </div>
      `;

      window.showModal({
        title: isEdit ? I18n.t('common.edit') : I18n.t('outbound.add'),
        bodyHtml,
        saveText: I18n.t('common.save'),
        cancelText: I18n.t('common.cancel'),
        onSave: async () => {
          const type = document.getElementById('modal-node-type').value;
          const tag = document.getElementById('modal-node-tag').value.trim();
          const server = document.getElementById('modal-node-server').value.trim();
          const port = parseInt(document.getElementById('modal-node-port').value, 10);
          const secret = document.getElementById('modal-node-secret').value.trim();
          const sni = document.getElementById('modal-node-sni')?.value.trim();
          const insecure = document.getElementById('modal-node-insecure')?.checked;

          if (!tag || !server || !port) {
            window.showToast('Tag, Server, and Port are required', 'error');
            return false;
          }

          const payload = {
            tag,
            type,
            server,
            server_port: port,
            enabled: true,
            settings: {
              ...(existing?.settings || {})
            }
          };

          if (type === 'vless' || type === 'vmess' || type === 'tuic') {
            payload.settings.uuid = secret;
          } else {
            payload.settings.password = secret;
          }

          if (type === 'hysteria2' || type === 'trojan' || type === 'vless' || type === 'tuic' || sni || insecure) {
            payload.settings.tls = payload.settings.tls || { enabled: true };
            if (sni) {
              payload.settings.tls.server_name = sni;
            }
            if (insecure) {
              payload.settings.tls.insecure = true;
            } else {
              delete payload.settings.tls.insecure;
            }
          }

          try {
            if (isEdit) {
              await API.put(`outbounds/${existing.id}`, payload);
            } else {
              await API.post('outbounds', payload);
            }
            window.showToast(I18n.t('common.success'), 'success');
            await this.loadAll();
            return true;
          } catch (err) {
            window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
            return false;
          }
        }
      });
    }

    /**
     * Add or Edit Group Modal
     */
    openGroupModal(existing = null) {
      const isEdit = !!existing;
      const data = existing || {
        tag: 'PROXY',
        type: 'selector',
        outbounds: [],
        url: 'https://cp.cloudflare.com/generate_204',
        interval: '3m',
        tolerance: 50
      };

      const allNodeTags = this.nodes.map(n => n.tag);

      const bodyHtml = `
        <div class="group-form-modal">
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('outbound.group_type')}</label>
              <select class="form-select" id="modal-group-type">
                <option value="selector" ${data.type === 'selector' ? 'selected' : ''}>${I18n.t('outbound.group_selector')}</option>
                <option value="urltest" ${data.type === 'urltest' ? 'selected' : ''}>${I18n.t('outbound.group_urltest')}</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('common.tag')} *</label>
              <input type="text" class="form-control" id="modal-group-tag" value="${this.escapeHtml(data.tag || '')}" required>
            </div>
          </div>

          <!-- Included Nodes Selection -->
          <div class="form-group">
            <label class="form-label">${I18n.t('outbound.group_outbounds')}</label>
            <div class="nodes-multiselect-box" style="max-height: 180px; overflow-y: auto; border: 1px solid var(--border-glass); border-radius: var(--radius-sm); padding: 0.5rem;">
              ${allNodeTags.map(tag => `
                <label class="checkbox-label" style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.35rem; cursor: pointer;">
                  <input type="checkbox" class="grp-node-checkbox" value="${this.escapeHtml(tag)}" ${(data.outbounds || []).includes(tag) ? 'checked' : ''}>
                  <span>${this.escapeHtml(tag)}</span>
                </label>
              `).join('')}
            </div>
          </div>

          <div id="urltest-settings-row" style="display: ${data.type === 'urltest' ? 'block' : 'none'};">
            <div class="form-group">
              <label class="form-label">${I18n.t('outbound.group_url')}</label>
              <input type="url" class="form-control" id="modal-group-url" value="${this.escapeHtml(data.url || 'https://cp.cloudflare.com/generate_204')}">
            </div>
            <div class="form-row">
              <div class="form-group">
                <label class="form-label">${I18n.t('outbound.group_interval')}</label>
                <input type="text" class="form-control" id="modal-group-interval" value="${this.escapeHtml(data.interval || '3m')}">
              </div>
              <div class="form-group">
                <label class="form-label">${I18n.t('outbound.group_tolerance')}</label>
                <input type="number" class="form-control" id="modal-group-tolerance" value="${data.tolerance || 50}">
              </div>
            </div>
          </div>
        </div>
      `;

      window.showModal({
        title: isEdit ? I18n.t('outbound.edit_group') : I18n.t('outbound.add_group'),
        bodyHtml,
        saveText: I18n.t('common.save'),
        cancelText: I18n.t('common.cancel'),
        onSave: async () => {
          const type = document.getElementById('modal-group-type').value;
          const tag = document.getElementById('modal-group-tag').value.trim();

          if (!tag) {
            window.showToast('Group tag is required', 'error');
            return false;
          }

          const selectedOutbounds = [];
          document.querySelectorAll('.grp-node-checkbox:checked').forEach(cb => {
            selectedOutbounds.push(cb.value);
          });

          const payload = {
            tag,
            type,
            outbounds: selectedOutbounds
          };

          if (type === 'urltest') {
            payload.url = document.getElementById('modal-group-url')?.value.trim() || 'https://cp.cloudflare.com/generate_204';
            payload.interval = document.getElementById('modal-group-interval')?.value.trim() || '3m';
            payload.tolerance = parseInt(document.getElementById('modal-group-tolerance')?.value, 10) || 50;
          }

          try {
            if (isEdit) {
              await API.put(`groups/${existing.id}`, payload);
            } else {
              await API.post('groups', payload);
            }
            window.showToast(I18n.t('common.success'), 'success');
            await this.loadAll();
            return true;
          } catch (err) {
            window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
            return false;
          }
        }
      });

      // Type switch
      const typeSelect = document.getElementById('modal-group-type');
      if (typeSelect) {
        typeSelect.addEventListener('change', (e) => {
          const urltestSec = document.getElementById('urltest-settings-row');
          if (urltestSec) {
            urltestSec.style.display = e.target.value === 'urltest' ? 'block' : 'none';
          }
        });
      }
    }

    /**
     * Delete node
     */
    deleteNode(node) {
      window.showModal({
        title: I18n.t('common.delete'),
        bodyHtml: `<p>${this.escapeHtml(I18n.t('outbound.delete_node_confirm', { tag: node.tag }))}</p>`,
        saveText: I18n.t('common.delete'),
        cancelText: I18n.t('common.cancel'),
        size: 'sm',
        onSave: async () => {
          try {
            await API.del(`outbounds/${node.id}`);
            window.showToast(I18n.t('common.success'), 'success');
            await this.loadAll();
            return true;
          } catch (err) {
            window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
            return false;
          }
        }
      });
    }

    /**
     * Delete group
     */
    deleteGroup(group) {
      window.showModal({
        title: I18n.t('common.delete'),
        bodyHtml: `<p>${this.escapeHtml(I18n.t('outbound.delete_group_confirm', { tag: group.tag }))}</p>`,
        saveText: I18n.t('common.delete'),
        cancelText: I18n.t('common.cancel'),
        size: 'sm',
        onSave: async () => {
          try {
            await API.del(`groups/${group.id}`);
            window.showToast(I18n.t('common.success'), 'success');
            await this.loadAll();
            return true;
          } catch (err) {
            window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
            return false;
          }
        }
      });
    }

    escapeHtml(str) {
      return String(str || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
    }

    destroy() {
      // Cleanup
    }
  }

  window.Outbounds = new OutboundsPage();
})();
