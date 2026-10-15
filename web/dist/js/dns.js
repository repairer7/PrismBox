/**
 * PrismBox - DNS Configuration Module
 * Manages sing-box DNS Configuration, master toggle, DoH/DoT/UDP servers, split-DNS rules, and system default fallbacks
 */

(function () {
  'use strict';

  class DNSPage {
    constructor() {
      this.config = {
        enabled: false,
        strategy: 'prefer_ipv4',
        disable_cache: false,
        disable_expire: false,
        servers: [],
        rules: []
      };
      this.availableOutbounds = ['direct'];
    }

    /**
     * Render DNS View
     */
    async render() {
      const app = document.getElementById('app');
      if (!app) return;

      app.innerHTML = `
        <div class="dns-page">
          <!-- Master Toggle Banner -->
          <div class="dns-banner glass-panel mb-4">
            <div class="dns-banner-left">
              <div class="dns-banner-icon">
                <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
                  <circle cx="12" cy="11" r="2.5"/>
                </svg>
              </div>
              <div class="dns-banner-text">
                <h3>${I18n.t('dns.enabled')}</h3>
                <p>${I18n.t('dns.enabled_desc')}</p>
              </div>
            </div>

            <label class="toggle-switch toggle-lg">
              <input type="checkbox" id="master-dns-toggle" ${this.config.enabled ? 'checked' : ''}>
              <span class="toggle-slider"></span>
            </label>
          </div>

          <!-- Section when Custom DNS is OFF (Disabled) -->
          <div id="dns-disabled-section" style="display: ${this.config.enabled ? 'none' : 'block'};">
            <div class="glass-panel text-center py-5">
              <div class="empty-icon-box mb-3">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--cyan-light)" stroke-width="1.5">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="2" y1="12" x2="22" y2="12"/>
                  <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
                </svg>
              </div>
              <h3 class="text-white mb-2">${I18n.t('dns.disabled_hint')}</h3>
              <p class="text-muted max-w-lg mx-auto mb-4">${I18n.t('dns.disabled_desc')}</p>
              <button class="btn btn-primary" id="btn-enable-dns-cta">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
                <span>${I18n.t('dns.enabled')}</span>
              </button>
            </div>
          </div>

          <!-- Section when Custom DNS is ON (Enabled) -->
          <div id="dns-enabled-section" style="display: ${this.config.enabled ? 'block' : 'none'};">
            <!-- Global DNS Settings -->
            <div class="glass-panel mb-4">
              <div class="panel-header mb-2">
                <h3 class="panel-title">
                  <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="3"/>
                    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
                  </svg>
                  ${I18n.t('dns.global_settings')}
                </h3>
              </div>

              <div class="routing-settings-grid">
                <!-- Strategy -->
                <div class="setting-item glass-panel">
                  <div class="setting-info">
                    <span class="setting-title">${I18n.t('dns.strategy')}</span>
                    <span class="setting-desc">Domain address query strategy</span>
                  </div>
                  <select class="form-select" id="dns-global-strategy" style="max-width: 170px;">
                    <option value="prefer_ipv4" ${this.config.strategy === 'prefer_ipv4' ? 'selected' : ''}>Prefer IPv4</option>
                    <option value="prefer_ipv6" ${this.config.strategy === 'prefer_ipv6' ? 'selected' : ''}>Prefer IPv6</option>
                    <option value="ipv4_only" ${this.config.strategy === 'ipv4_only' ? 'selected' : ''}>IPv4 Only</option>
                    <option value="ipv6_only" ${this.config.strategy === 'ipv6_only' ? 'selected' : ''}>IPv6 Only</option>
                  </select>
                </div>

                <!-- Disable Cache -->
                <div class="setting-item glass-panel">
                  <div class="setting-info">
                    <span class="setting-title">${I18n.t('dns.disable_cache')}</span>
                    <span class="setting-desc">Disable in-memory DNS query caching</span>
                  </div>
                  <label class="toggle-switch">
                    <input type="checkbox" id="dns-disable-cache-toggle" ${this.config.disable_cache ? 'checked' : ''}>
                    <span class="toggle-slider"></span>
                  </label>
                </div>

                <!-- Disable Expire -->
                <div class="setting-item glass-panel">
                  <div class="setting-info">
                    <span class="setting-title">${I18n.t('dns.disable_expire')}</span>
                    <span class="setting-desc">Prevent DNS cache entries from expiring</span>
                  </div>
                  <label class="toggle-switch">
                    <input type="checkbox" id="dns-disable-expire-toggle" ${this.config.disable_expire ? 'checked' : ''}>
                    <span class="toggle-slider"></span>
                  </label>
                </div>
              </div>
            </div>

            <!-- DNS Servers Section -->
            <div class="mb-4">
              <div class="panel-header mb-2">
                <div class="flex-center gap-sm">
                  <h3 class="panel-title">
                    <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
                      <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
                      <line x1="6" y1="6" x2="6.01" y2="6"/>
                      <line x1="6" y1="18" x2="6.01" y2="18"/>
                    </svg>
                    ${I18n.t('dns.servers_title')}
                  </h3>
                  <span class="badge badge-neutral" id="dns-servers-count-badge">0</span>
                </div>

                <button class="btn btn-primary btn-sm" id="btn-add-dns-server">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <line x1="12" y1="5" x2="12" y2="19"/>
                    <line x1="5" y1="12" x2="19" y2="12"/>
                  </svg>
                  <span>${I18n.t('dns.add_server')}</span>
                </button>
              </div>

              <div class="table-responsive">
                <table class="glass-table" id="dns-servers-table">
                  <thead>
                    <tr>
                      <th>${I18n.t('dns.server_tag')}</th>
                      <th>${I18n.t('dns.server_addr')}</th>
                      <th>${I18n.t('dns.server_resolver')}</th>
                      <th>${I18n.t('dns.server_detour')}</th>
                      <th style="width: 100px; text-align: right;">${I18n.t('common.actions')}</th>
                    </tr>
                  </thead>
                  <tbody id="dns-servers-tbody">
                    <!-- Populated dynamically -->
                  </tbody>
                </table>
              </div>
            </div>

            <!-- DNS Rules Section -->
            <div class="mb-4">
              <div class="panel-header mb-2">
                <div class="flex-center gap-sm">
                  <h3 class="panel-title">
                    <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="9 11 12 14 22 4"/>
                      <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/>
                    </svg>
                    ${I18n.t('dns.rules_title')}
                  </h3>
                  <span class="badge badge-neutral" id="dns-rules-count-badge">0</span>
                </div>

                <button class="btn btn-secondary btn-sm" id="btn-add-dns-rule">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <line x1="12" y1="5" x2="12" y2="19"/>
                    <line x1="5" y1="12" x2="19" y2="12"/>
                  </svg>
                  <span>${I18n.t('dns.add_rule')}</span>
                </button>
              </div>

              <div class="table-responsive">
                <table class="glass-table" id="dns-rules-table">
                  <thead>
                    <tr>
                      <th>Match Domain / GeoSite</th>
                      <th>${I18n.t('dns.server_tag')}</th>
                      <th style="width: 100px; text-align: right;">${I18n.t('common.actions')}</th>
                    </tr>
                  </thead>
                  <tbody id="dns-rules-tbody">
                    <!-- Populated dynamically -->
                  </tbody>
                </table>
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
      // Master toggle
      const masterToggle = document.getElementById('master-dns-toggle');
      if (masterToggle) {
        masterToggle.addEventListener('change', (e) => this.toggleMasterDNS(e.target.checked));
      }

      // CTA Enable button
      const ctaBtn = document.getElementById('btn-enable-dns-cta');
      if (ctaBtn) {
        ctaBtn.addEventListener('click', () => {
          if (masterToggle) {
            masterToggle.checked = true;
            this.toggleMasterDNS(true);
          }
        });
      }

      // Global Settings
      const stratSelect = document.getElementById('dns-global-strategy');
      if (stratSelect) {
        stratSelect.addEventListener('change', (e) => {
          this.config.strategy = e.target.value;
          this.saveDNSConfig();
        });
      }

      const cacheToggle = document.getElementById('dns-disable-cache-toggle');
      if (cacheToggle) {
        cacheToggle.addEventListener('change', (e) => {
          this.config.disable_cache = e.target.checked;
          this.saveDNSConfig();
        });
      }

      const expireToggle = document.getElementById('dns-disable-expire-toggle');
      if (expireToggle) {
        expireToggle.addEventListener('change', (e) => {
          this.config.disable_expire = e.target.checked;
          this.saveDNSConfig();
        });
      }

      // Add server button
      const addServerBtn = document.getElementById('btn-add-dns-server');
      if (addServerBtn) {
        addServerBtn.addEventListener('click', () => this.openServerModal());
      }

      // Add rule button
      const addRuleBtn = document.getElementById('btn-add-dns-rule');
      if (addRuleBtn) {
        addRuleBtn.addEventListener('click', () => this.openRuleModal());
      }
    }

    /**
     * Toggle Master DNS Enable/Disable
     */
    async toggleMasterDNS(enabled) {
      this.config.enabled = enabled;
      const disabledSec = document.getElementById('dns-disabled-section');
      const enabledSec = document.getElementById('dns-enabled-section');

      if (enabled) {
        if (disabledSec) disabledSec.style.display = 'none';
        if (enabledSec) enabledSec.style.display = 'block';

        // Provide defaults if empty
        if (!this.config.servers || this.config.servers.length === 0) {
          this.config.servers = [
            {
              tag: 'dns-remote',
              address: 'https://1.1.1.1/dns-query',
              address_resolver: 'dns-direct',
              detour: ''
            },
            {
              tag: 'dns-direct',
              address: 'https://223.5.5.5/dns-query',
              address_resolver: '',
              detour: ''
            }
          ];
        }
        if (!this.config.rules || this.config.rules.length === 0) {
          this.config.rules = [
            {
              id: 'rule-cn',
              domain_suffix: ['.cn'],
              server: 'dns-direct'
            }
          ];
        }
        this.renderServers();
        this.renderRules();
      } else {
        if (disabledSec) disabledSec.style.display = 'block';
        if (enabledSec) enabledSec.style.display = 'none';
      }

      await this.saveDNSConfig();
    }

    /**
     * Load DNS configuration from backend
     */
    async loadAll() {
      try {
        const [dnsRes, outboundsRes, groupsRes] = await Promise.all([
          API.get('dns').catch(() => null),
          API.get('outbounds').catch(() => []),
          API.get('groups').catch(() => [])
        ]);

        if (dnsRes) {
          this.config = Object.assign(this.config, dnsRes);
          if (!this.config.servers) this.config.servers = [];
          if (!this.config.rules) this.config.rules = [];

          // Clean up any direct detours to avoid sing-box fatal crash
          this.config.servers.forEach(s => {
            if (s.detour === 'direct') s.detour = '';
            if (s.address_resolver === 'direct') s.address_resolver = '';
          });

          // Migrate legacy geosite field in rules to domain_suffix
          this.config.rules.forEach(r => {
            if (r.geosite && r.geosite.length > 0) {
              if (!r.domain_suffix) r.domain_suffix = [];
              r.geosite.forEach(g => {
                if (g === 'cn' && !r.domain_suffix.includes('.cn')) {
                  r.domain_suffix.push('.cn');
                }
              });
            }
          });
        }

        const outTags = (Array.isArray(outboundsRes) ? outboundsRes : []).map(o => o.tag);
        const grpTags = (Array.isArray(groupsRes) ? groupsRes : []).map(g => g.tag);
        this.availableOutbounds = ['direct', 'block', ...grpTags, ...outTags];
      } catch (err) {
        console.error('[DNS] Load error:', err);
      } finally {
        this.updateGlobalControls();
        this.renderServers();
        this.renderRules();
      }
    }

    /**
     * Synchronize global controls (toggles, selects) with loaded DNS state
     */
    updateGlobalControls() {
      const masterToggle = document.getElementById('master-dns-toggle');
      if (masterToggle) {
        masterToggle.checked = !!this.config.enabled;
      }

      const disabledSec = document.getElementById('dns-disabled-section');
      const enabledSec = document.getElementById('dns-enabled-section');
      if (disabledSec) {
        disabledSec.style.display = this.config.enabled ? 'none' : 'block';
      }
      if (enabledSec) {
        enabledSec.style.display = this.config.enabled ? 'block' : 'none';
      }

      const stratSelect = document.getElementById('dns-global-strategy');
      if (stratSelect && this.config.strategy) {
        stratSelect.value = this.config.strategy;
      }

      const cacheToggle = document.getElementById('dns-disable-cache-toggle');
      if (cacheToggle) {
        cacheToggle.checked = !!this.config.disable_cache;
      }

      const expireToggle = document.getElementById('dns-disable-expire-toggle');
      if (expireToggle) {
        expireToggle.checked = !!this.config.disable_expire;
      }
    }

    /**
     * Save DNS configuration
     */
    async saveDNSConfig() {
      try {
        await API.put('dns', this.config);
        window.showToast(I18n.t('common.success'), 'success', 2000);
      } catch (err) {
        window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
      }
    }

    /**
     * Render DNS Servers Table
     */
    renderServers() {
      const tbody = document.getElementById('dns-servers-tbody');
      const countBadge = document.getElementById('dns-servers-count-badge');
      if (!tbody) return;

      const servers = this.config.servers || [];
      if (countBadge) countBadge.textContent = String(servers.length);

      if (servers.length === 0) {
        tbody.innerHTML = `
          <tr>
            <td colspan="5" class="text-center py-4 text-muted">
              ${I18n.t('dns.no_servers')}
            </td>
          </tr>
        `;
        return;
      }

      tbody.innerHTML = servers.map((srv, idx) => {
        let typeBadge = 'badge-primary';
        if (srv.address?.startsWith('https://')) typeBadge = 'badge-cyan';
        else if (srv.address?.startsWith('tls://')) typeBadge = 'badge-purple';
        else if (srv.address?.startsWith('rcode://')) typeBadge = 'badge-danger';

        return `
          <tr data-index="${idx}">
            <td>
              <span class="badge ${typeBadge} mr-1">${this.escapeHtml(srv.tag)}</span>
            </td>
            <td>
              <code>${this.escapeHtml(srv.address)}</code>
            </td>
            <td>
              ${srv.address_resolver ? `<span class="badge badge-neutral">${this.escapeHtml(srv.address_resolver)}</span>` : '<span class="text-muted">-</span>'}
            </td>
            <td>
              <span class="badge badge-success">${this.escapeHtml(srv.detour || 'direct')}</span>
            </td>
            <td style="text-align: right;">
              <div class="btn-group-row">
                <button class="btn-icon btn-sm btn-edit-dns-server" data-index="${idx}" title="${I18n.t('common.edit')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </button>
                <button class="btn-icon btn-sm btn-delete-dns-server text-danger" data-index="${idx}" title="${I18n.t('common.delete')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                  </svg>
                </button>
              </div>
            </td>
          </tr>
        `;
      }).join('');

      tbody.querySelectorAll('.btn-edit-dns-server').forEach(btn => {
        btn.addEventListener('click', () => {
          const i = parseInt(btn.dataset.index, 10);
          this.openServerModal(this.config.servers[i], i);
        });
      });

      tbody.querySelectorAll('.btn-delete-dns-server').forEach(btn => {
        btn.addEventListener('click', () => {
          const i = parseInt(btn.dataset.index, 10);
          this.deleteServer(i);
        });
      });
    }

    /**
     * Render DNS Rules Table
     */
    renderRules() {
      const tbody = document.getElementById('dns-rules-tbody');
      const countBadge = document.getElementById('dns-rules-count-badge');
      if (!tbody) return;

      const rules = this.config.rules || [];
      if (countBadge) countBadge.textContent = String(rules.length);

      if (rules.length === 0) {
        tbody.innerHTML = `
          <tr>
            <td colspan="3" class="text-center py-4 text-muted">
              ${I18n.t('dns.no_rules')}
            </td>
          </tr>
        `;
        return;
      }

      tbody.innerHTML = rules.map((r, idx) => {
        const criteria = [];
        if (r.geosite?.length) criteria.push(`<span class="badge badge-info">geosite: ${r.geosite.join(', ')}</span>`);
        if (r.domain?.length) criteria.push(`<span class="badge badge-primary">domain: ${r.domain.join(', ')}</span>`);
        if (r.domain_suffix?.length) criteria.push(`<span class="badge badge-primary">suffix: ${r.domain_suffix.join(', ')}</span>`);
        if (r.rule_set?.length) criteria.push(`<span class="badge badge-pink">rule_set: ${r.rule_set.join(', ')}</span>`);

        return `
          <tr data-index="${idx}">
            <td>
              <div class="tags-list">
                ${criteria.join('') || '<span class="text-muted">Any Domain</span>'}
              </div>
            </td>
            <td>
              <span class="badge badge-success">${this.escapeHtml(r.server)}</span>
            </td>
            <td style="text-align: right;">
              <div class="btn-group-row">
                <button class="btn-icon btn-sm btn-edit-dns-rule" data-index="${idx}" title="${I18n.t('common.edit')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </button>
                <button class="btn-icon btn-sm btn-delete-dns-rule text-danger" data-index="${idx}" title="${I18n.t('common.delete')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                  </svg>
                </button>
              </div>
            </td>
          </tr>
        `;
      }).join('');

      tbody.querySelectorAll('.btn-edit-dns-rule').forEach(btn => {
        btn.addEventListener('click', () => {
          const i = parseInt(btn.dataset.index, 10);
          this.openRuleModal(this.config.rules[i], i);
        });
      });

      tbody.querySelectorAll('.btn-delete-dns-rule').forEach(btn => {
        btn.addEventListener('click', () => {
          const i = parseInt(btn.dataset.index, 10);
          this.deleteRule(i);
        });
      });
    }

    /**
     * Add/Edit DNS Server Modal
     */
    openServerModal(existing = null, index = -1) {
      const isEdit = existing !== null;
      const data = existing || {
        tag: 'dns-remote',
        address: 'https://1.1.1.1/dns-query',
        address_resolver: '',
        detour: ''
      };

      const serverTags = (this.config.servers || []).map(s => s.tag);
      const proxyOutbounds = this.availableOutbounds.filter(t => t !== 'direct' && t !== 'block');
      const detourOptions = [
        `<option value="" ${(!data.detour || data.detour === 'direct') ? 'selected' : ''}>Direct (Default)</option>`,
        ...proxyOutbounds.map(t => {
          return `<option value="${this.escapeHtml(t)}" ${data.detour === t ? 'selected' : ''}>${this.escapeHtml(t)}</option>`;
        })
      ].join('');

      const bodyHtml = `
        <div class="dns-server-form">
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('dns.server_tag')} *</label>
              <input type="text" class="form-control" id="modal-dns-tag" value="${this.escapeHtml(data.tag || '')}" required>
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('dns.server_detour')}</label>
              <select class="form-select" id="modal-dns-detour">
                ${detourOptions}
              </select>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">${I18n.t('dns.server_addr')} * (e.g. https://1.1.1.1/dns-query, tls://8.8.8.8, 223.5.5.5)</label>
            <input type="text" class="form-control" id="modal-dns-address" value="${this.escapeHtml(data.address || '')}" required>
          </div>

          <div class="form-group">
            <label class="form-label">${I18n.t('dns.server_resolver')} (Optional Server Tag)</label>
            <select class="form-select" id="modal-dns-resolver">
              <option value="">None (Direct IP)</option>
              ${serverTags.filter(t => t !== data.tag).map(t => `
                <option value="${this.escapeHtml(t)}" ${data.address_resolver === t ? 'selected' : ''}>${this.escapeHtml(t)}</option>
              `).join('')}
            </select>
          </div>
        </div>
      `;

      window.showModal({
        title: isEdit ? I18n.t('dns.edit_server') : I18n.t('dns.add_server'),
        bodyHtml,
        saveText: I18n.t('common.save'),
        cancelText: I18n.t('common.cancel'),
        onSave: async () => {
          const tag = document.getElementById('modal-dns-tag')?.value.trim();
          const address = document.getElementById('modal-dns-address')?.value.trim();
          const rawDetour = document.getElementById('modal-dns-detour')?.value || '';
          const detour = (rawDetour === 'direct') ? '' : rawDetour;
          const address_resolver = document.getElementById('modal-dns-resolver')?.value;

          if (!tag || !address) {
            window.showToast('Tag and Address are required', 'error');
            return false;
          }

          const srv = {
            tag,
            address,
            detour,
            address_resolver
          };

          if (isEdit) {
            this.config.servers[index] = srv;
          } else {
            this.config.servers.push(srv);
          }

          await this.saveDNSConfig();
          this.renderServers();
          return true;
        }
      });
    }

    /**
     * Delete DNS Server
     */
    deleteServer(index) {
      const srv = this.config.servers[index];
      window.showModal({
        title: I18n.t('common.delete'),
        bodyHtml: `<p>${I18n.t('dns.delete_server_confirm', { tag: srv?.tag })}</p>`,
        saveText: I18n.t('common.delete'),
        cancelText: I18n.t('common.cancel'),
        size: 'sm',
        onSave: async () => {
          this.config.servers.splice(index, 1);
          await this.saveDNSConfig();
          this.renderServers();
          return true;
        }
      });
    }

    /**
     * Add/Edit DNS Rule Modal
     */
    openRuleModal(existing = null, index = -1) {
      const isEdit = existing !== null;
      const data = existing || {
        server: this.config.servers?.[0]?.tag || 'dns-direct',
        geosite: [],
        domain: [],
        domain_suffix: [],
        rule_set: []
      };

      const serverOptions = (this.config.servers || []).map(s => {
        return `<option value="${this.escapeHtml(s.tag)}" ${data.server === s.tag ? 'selected' : ''}>${this.escapeHtml(s.tag)}</option>`;
      }).join('');

      const bodyHtml = `
        <div class="dns-rule-form">
          <div class="form-group">
            <label class="form-label">${I18n.t('dns.server_tag')} *</label>
            <select class="form-select" id="modal-dns-rule-server">
              ${serverOptions}
            </select>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_geosite')}</label>
              <input type="text" class="form-control" id="modal-dns-rule-geosite" value="${(data.geosite || []).join(', ')}" placeholder="cn, google">
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_domain_suffix')}</label>
              <input type="text" class="form-control" id="modal-dns-rule-suffix" value="${(data.domain_suffix || []).join(', ')}" placeholder="example.com">
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_domain')}</label>
              <input type="text" class="form-control" id="modal-dns-rule-domain" value="${(data.domain || []).join(', ')}" placeholder="api.github.com">
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_ruleset')}</label>
              <input type="text" class="form-control" id="modal-dns-rule-ruleset" value="${(data.rule_set || []).join(', ')}" placeholder="geosite-cn">
            </div>
          </div>
        </div>
      `;

      window.showModal({
        title: isEdit ? I18n.t('dns.edit_rule') : I18n.t('dns.add_rule'),
        bodyHtml,
        saveText: I18n.t('common.save'),
        cancelText: I18n.t('common.cancel'),
        onSave: async () => {
          const server = document.getElementById('modal-dns-rule-server')?.value;
          if (!server) {
            window.showToast('Server tag is required', 'error');
            return false;
          }

          const parseList = (id) => {
            const val = document.getElementById(id)?.value || '';
            return val.split(',').map(s => s.trim()).filter(Boolean);
          };

          const newRule = {
            id: existing ? existing.id : ('dns-rule-' + Date.now()),
            server
          };

          const geosite = parseList('modal-dns-rule-geosite');
          if (geosite.length) {
            newRule.geosite = geosite;
            if (geosite.includes('cn')) {
              if (!newRule.domain_suffix) newRule.domain_suffix = [];
              if (!newRule.domain_suffix.includes('.cn')) newRule.domain_suffix.push('.cn');
            }
          }

          const suffix = parseList('modal-dns-rule-suffix');
          if (suffix.length) {
            if (!newRule.domain_suffix) newRule.domain_suffix = [];
            suffix.forEach(s => {
              if (!newRule.domain_suffix.includes(s)) newRule.domain_suffix.push(s);
            });
          }

          const domain = parseList('modal-dns-rule-domain');
          if (domain.length) newRule.domain = domain;

          const ruleset = parseList('modal-dns-rule-ruleset');
          if (ruleset.length) newRule.rule_set = ruleset;

          if (isEdit) {
            this.config.rules[index] = newRule;
          } else {
            this.config.rules.push(newRule);
          }

          await this.saveDNSConfig();
          this.renderRules();
          return true;
        }
      });
    }

    /**
     * Delete DNS Rule
     */
    deleteRule(index) {
      window.showModal({
        title: I18n.t('common.delete'),
        bodyHtml: `<p>${I18n.t('dns.delete_rule_confirm')}</p>`,
        saveText: I18n.t('common.delete'),
        cancelText: I18n.t('common.cancel'),
        size: 'sm',
        onSave: async () => {
          this.config.rules.splice(index, 1);
          await this.saveDNSConfig();
          this.renderRules();
          return true;
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

  window.DNS = new DNSPage();
})();
