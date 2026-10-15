/**
 * PrismBox - Routing Configuration Module
 * Manages sing-box Routing Policy, Sniffing, Domain Strategy, Rule Chains, and Remote Rule Sets (.srs / .json)
 */

(function () {
  'use strict';

  class RoutingPage {
    constructor() {
      this.routing = {
        sniff: true,
        sniff_override_destination: false,
        domain_strategy: 'prefer_ipv4',
        rules: [],
        rule_sets: []
      };
      this.availableOutbounds = ['direct', 'block'];
      this.isLoading = false;
    }

    /**
     * Render Routing View
     */
    async render() {
      const app = document.getElementById('app');
      if (!app) return;

      app.innerHTML = `
        <div class="routing-page">
          <!-- Top Section: Global Settings -->
          <div class="glass-panel mb-4">
            <div class="panel-header">
              <h3 class="panel-title">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="3"/>
                  <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
                </svg>
                ${I18n.t('routing.global_settings')}
              </h3>
            </div>

            <div class="routing-settings-grid">
              <!-- Protocol Sniff -->
              <div class="setting-item glass-panel">
                <div class="setting-info">
                  <span class="setting-title">${I18n.t('routing.sniff')}</span>
                  <span class="setting-desc">${I18n.t('routing.sniff_desc')}</span>
                </div>
                <label class="toggle-switch">
                  <input type="checkbox" id="route-sniff-toggle" ${this.routing.sniff ? 'checked' : ''}>
                  <span class="toggle-slider"></span>
                </label>
              </div>

              <!-- Sniff Override Destination -->
              <div class="setting-item glass-panel">
                <div class="setting-info">
                  <span class="setting-title">${I18n.t('routing.sniff_override')}</span>
                  <span class="setting-desc">${I18n.t('routing.sniff_override_desc')}</span>
                </div>
                <label class="toggle-switch">
                  <input type="checkbox" id="route-sniff-override-toggle" ${this.routing.sniff_override_destination ? 'checked' : ''}>
                  <span class="toggle-slider"></span>
                </label>
              </div>

              <!-- Domain Strategy Selector -->
              <div class="setting-item glass-panel">
                <div class="setting-info">
                  <span class="setting-title">${I18n.t('routing.domain_strategy')}</span>
                  <span class="setting-desc">sing-box DNS domain resolution policy</span>
                </div>
                <select class="form-select" id="route-domain-strategy" style="max-width: 170px;">
                  <option value="prefer_ipv4" ${this.routing.domain_strategy === 'prefer_ipv4' ? 'selected' : ''}>${I18n.t('routing.strategy_prefer_ipv4')}</option>
                  <option value="prefer_ipv6" ${this.routing.domain_strategy === 'prefer_ipv6' ? 'selected' : ''}>${I18n.t('routing.strategy_prefer_ipv6')}</option>
                  <option value="ipv4_only" ${this.routing.domain_strategy === 'ipv4_only' ? 'selected' : ''}>${I18n.t('routing.strategy_ipv4_only')}</option>
                  <option value="ipv6_only" ${this.routing.domain_strategy === 'ipv6_only' ? 'selected' : ''}>${I18n.t('routing.strategy_ipv6_only')}</option>
                </select>
              </div>
            </div>
          </div>

          <!-- Section: Routing Rules (Unified Table Card) -->
          <div class="table-card mb-4">
            <div class="table-card-header">
              <div class="flex-center gap-sm">
                <h3 class="panel-title mb-0">
                  <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="8" y1="6" x2="21" y2="6"/>
                    <line x1="8" y1="12" x2="21" y2="12"/>
                    <line x1="8" y1="18" x2="21" y2="18"/>
                    <line x1="3" y1="6" x2="3.01" y2="6"/>
                    <line x1="3" y1="12" x2="3.01" y2="12"/>
                    <line x1="3" y1="18" x2="3.01" y2="18"/>
                  </svg>
                  ${I18n.t('routing.rules_title')}
                </h3>
                <span class="badge badge-neutral" id="rules-count-badge">0</span>
              </div>

              <button class="btn btn-primary btn-sm" id="btn-add-rule">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
                <span>${I18n.t('routing.add_rule')}</span>
              </button>
            </div>

            <div class="table-responsive" style="border: none; border-radius: 0; box-shadow: none; background: transparent;">
              <table class="glass-table" id="rules-table">
                <thead>
                  <tr>
                    <th style="width: 50px;">${I18n.t('common.enabled')}</th>
                    <th style="width: 70px;">${I18n.t('routing.reorder')}</th>
                    <th>${I18n.t('common.description')}</th>
                    <th>Match Criteria</th>
                    <th>${I18n.t('routing.target_outbound')}</th>
                    <th style="width: 100px; text-align: right;">${I18n.t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody id="rules-table-body">
                  <tr>
                    <td colspan="6" class="text-center py-4 text-muted">
                      <span class="spinner-sm"></span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Section: Remote Rule Sets (.srs / .json) -->
          <div class="rulesets-section">
            <div class="section-subheader mb-2">
              <div class="flex-center gap-sm">
                <h3 class="panel-title mb-0">
                  <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polygon points="12 2 2 7 12 12 22 7 12 2"/>
                    <polyline points="2 17 12 22 22 17"/>
                    <polyline points="2 12 12 17 22 12"/>
                  </svg>
                  ${I18n.t('routing.rulesets_title')}
                </h3>
                <span class="badge badge-neutral" id="rulesets-count-badge">0</span>
              </div>

              <button class="btn btn-secondary btn-sm" id="btn-add-ruleset">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
                <span>${I18n.t('routing.add_ruleset')}</span>
              </button>
            </div>

            <div class="rulesets-grid" id="rulesets-grid-container">
              <div class="glass-panel text-center py-3 text-muted">
                <span class="spinner-sm"></span>
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
      // Global settings toggles
      const sniffToggle = document.getElementById('route-sniff-toggle');
      if (sniffToggle) {
        sniffToggle.addEventListener('change', (e) => {
          this.routing.sniff = e.target.checked;
          this.saveRoutingSettings();
        });
      }

      const sniffOverrideToggle = document.getElementById('route-sniff-override-toggle');
      if (sniffOverrideToggle) {
        sniffOverrideToggle.addEventListener('change', (e) => {
          this.routing.sniff_override_destination = e.target.checked;
          this.saveRoutingSettings();
        });
      }

      const strategySelect = document.getElementById('route-domain-strategy');
      if (strategySelect) {
        strategySelect.addEventListener('change', (e) => {
          this.routing.domain_strategy = e.target.value;
          this.saveRoutingSettings();
        });
      }

      // Add Rule button
      const addRuleBtn = document.getElementById('btn-add-rule');
      if (addRuleBtn) {
        addRuleBtn.addEventListener('click', () => this.openRuleModal());
      }

      // Add RuleSet button
      const addRsBtn = document.getElementById('btn-add-ruleset');
      if (addRsBtn) {
        addRsBtn.addEventListener('click', () => this.openRuleSetModal());
      }
    }

    /**
     * Load routing config and available outbounds
     */
    async loadAll() {
      try {
        const [routingRes, outboundsRes, groupsRes] = await Promise.all([
          API.get('routing').catch(() => null),
          API.get('outbounds').catch(() => []),
          API.get('groups').catch(() => [])
        ]);

        if (routingRes) {
          this.routing = Object.assign(this.routing, routingRes);
          if (!this.routing.rules) this.routing.rules = [];
          if (!this.routing.rule_sets) this.routing.rule_sets = [];
        }

        const outTags = (Array.isArray(outboundsRes) ? outboundsRes : []).map(o => o.tag);
        const grpTags = (Array.isArray(groupsRes) ? groupsRes : []).map(g => g.tag);
        this.availableOutbounds = ['direct', 'block', ...grpTags, ...outTags];
      } catch (err) {
        console.error('[Routing] Load error:', err);
      } finally {
        this.updateGlobalControls();
        this.renderRules();
        this.renderRuleSets();
      }
    }

    /**
     * Synchronize global controls (toggles, selects) with loaded routing state
     */
    updateGlobalControls() {
      const sniffToggle = document.getElementById('route-sniff-toggle');
      if (sniffToggle) {
        sniffToggle.checked = this.routing.sniff !== false;
      }

      const sniffOverrideToggle = document.getElementById('route-sniff-override-toggle');
      if (sniffOverrideToggle) {
        sniffOverrideToggle.checked = !!this.routing.sniff_override_destination;
      }

      const strategySelect = document.getElementById('route-domain-strategy');
      if (strategySelect && this.routing.domain_strategy) {
        strategySelect.value = this.routing.domain_strategy;
      }
    }

    /**
     * Save updated routing configuration
     */
    async saveRoutingSettings() {
      try {
        await API.put('routing', this.routing);
        window.showToast(I18n.t('common.success'), 'success', 2000);
      } catch (err) {
        window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
      }
    }

    /**
     * Render Rules Table
     */
    renderRules() {
      const tbody = document.getElementById('rules-table-body');
      const countBadge = document.getElementById('rules-count-badge');
      if (!tbody) return;

      const rules = this.routing.rules || [];
      if (countBadge) countBadge.textContent = String(rules.length);

      if (rules.length === 0) {
        tbody.innerHTML = `
          <tr>
            <td colspan="6" class="text-center py-4 text-muted">
              ${I18n.t('routing.no_rules')}
            </td>
          </tr>
        `;
        return;
      }

      tbody.innerHTML = rules.map((rule, idx) => {
        // Collect criteria badges
        const criteriaBadges = [];
        if (rule.geosite?.length) criteriaBadges.push(`<span class="badge badge-info">geosite: ${rule.geosite.join(', ')}</span>`);
        if (rule.geoip?.length) criteriaBadges.push(`<span class="badge badge-cyan">geoip: ${rule.geoip.join(', ')}</span>`);
        if (rule.domain_suffix?.length) criteriaBadges.push(`<span class="badge badge-primary">suffix: ${rule.domain_suffix.join(', ')}</span>`);
        if (rule.domain?.length) criteriaBadges.push(`<span class="badge badge-primary">domain: ${rule.domain.join(', ')}</span>`);
        if (rule.domain_keyword?.length) criteriaBadges.push(`<span class="badge badge-primary">keyword: ${rule.domain_keyword.join(', ')}</span>`);
        if (rule.ip_cidr?.length) criteriaBadges.push(`<span class="badge badge-warning">cidr: ${rule.ip_cidr.join(', ')}</span>`);
        if (rule.rule_set?.length) criteriaBadges.push(`<span class="badge badge-pink">rule_set: ${rule.rule_set.join(', ')}</span>`);
        if (rule.protocol?.length) criteriaBadges.push(`<span class="badge badge-neutral">proto: ${rule.protocol.join(', ')}</span>`);
        if (rule.port?.length) criteriaBadges.push(`<span class="badge badge-neutral">port: ${rule.port.join(', ')}</span>`);

        const isFirst = idx === 0;
        const isLast = idx === rules.length - 1;

        return `
          <tr data-index="${idx}">
            <td>
              <label class="toggle-switch">
                <input type="checkbox" class="rule-toggle" data-index="${idx}" ${rule.enabled !== false ? 'checked' : ''}>
                <span class="toggle-slider"></span>
              </label>
            </td>
            <td>
              <div class="order-controls">
                <button class="btn-icon btn-sm btn-order-up" data-index="${idx}" ${isFirst ? 'disabled' : ''} title="${I18n.t('routing.order_up')}">
                  ▲
                </button>
                <button class="btn-icon btn-sm btn-order-down" data-index="${idx}" ${isLast ? 'disabled' : ''} title="${I18n.t('routing.order_down')}">
                  ▼
                </button>
              </div>
            </td>
            <td>
              <strong>${this.escapeHtml(rule.description || 'Rule #' + (idx + 1))}</strong>
            </td>
            <td>
              <div class="tags-list">
                ${criteriaBadges.join('') || '<span class="text-muted">Any Match</span>'}
              </div>
            </td>
            <td>
              <span class="badge badge-success">${this.escapeHtml(rule.outbound || 'direct')}</span>
            </td>
            <td style="text-align: right;">
              <div class="btn-group-row">
                <button class="btn-icon btn-sm btn-edit-rule" data-index="${idx}" title="${I18n.t('common.edit')}">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </button>
                <button class="btn-icon btn-sm btn-delete-rule text-danger" data-index="${idx}" title="${I18n.t('common.delete')}">
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

      // Bind row events
      tbody.querySelectorAll('.rule-toggle').forEach(tgl => {
        tgl.addEventListener('change', async (e) => {
          const i = parseInt(e.target.dataset.index, 10);
          this.routing.rules[i].enabled = e.target.checked;
          await this.saveRoutingSettings();
        });
      });

      tbody.querySelectorAll('.btn-order-up').forEach(btn => {
        btn.addEventListener('click', async () => {
          const i = parseInt(btn.dataset.index, 10);
          if (i > 0) {
            const temp = this.routing.rules[i];
            this.routing.rules[i] = this.routing.rules[i - 1];
            this.routing.rules[i - 1] = temp;
            this.renderRules();
            await this.saveRoutingSettings();
          }
        });
      });

      tbody.querySelectorAll('.btn-order-down').forEach(btn => {
        btn.addEventListener('click', async () => {
          const i = parseInt(btn.dataset.index, 10);
          if (i < this.routing.rules.length - 1) {
            const temp = this.routing.rules[i];
            this.routing.rules[i] = this.routing.rules[i + 1];
            this.routing.rules[i + 1] = temp;
            this.renderRules();
            await this.saveRoutingSettings();
          }
        });
      });

      tbody.querySelectorAll('.btn-edit-rule').forEach(btn => {
        btn.addEventListener('click', () => {
          const i = parseInt(btn.dataset.index, 10);
          this.openRuleModal(this.routing.rules[i], i);
        });
      });

      tbody.querySelectorAll('.btn-delete-rule').forEach(btn => {
        btn.addEventListener('click', () => {
          const i = parseInt(btn.dataset.index, 10);
          this.deleteRule(i);
        });
      });
    }

    /**
     * Render RuleSets Grid
     */
    renderRuleSets() {
      const container = document.getElementById('rulesets-grid-container');
      const countBadge = document.getElementById('rulesets-count-badge');
      if (!container) return;

      const ruleSets = this.routing.rule_sets || [];
      if (countBadge) countBadge.textContent = String(ruleSets.length);

      if (ruleSets.length === 0) {
        container.innerHTML = `
          <div class="glass-panel text-center py-3 text-muted" style="grid-column: 1 / -1;">
            ${I18n.t('routing.no_rulesets')}
          </div>
        `;
        return;
      }

      container.innerHTML = ruleSets.map(rs => {
        return `
          <div class="glass-panel ruleset-card" data-id="${rs.id}">
            <div class="ruleset-card-header">
              <div class="ruleset-title-box">
                <span class="badge badge-pink uppercase">${this.escapeHtml(rs.format || 'binary')}</span>
                <span class="ruleset-tag"><strong>${this.escapeHtml(rs.tag)}</strong></span>
              </div>
              <button class="btn-icon btn-sm btn-delete-ruleset text-danger" data-id="${rs.id}" title="${I18n.t('common.delete')}">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"/>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                </svg>
              </button>
            </div>

            <div class="ruleset-card-body mt-2">
              <div class="ruleset-url text-xs text-muted">
                <code>${this.escapeHtml(rs.url || '')}</code>
              </div>
              <div class="ruleset-meta mt-2 text-xs">
                <span>Detour: <strong>${this.escapeHtml(rs.download_detour || 'direct')}</strong></span>
                <span>Interval: <strong>${this.escapeHtml(rs.update_interval || rs.interval || '1d')}</strong></span>
              </div>
            </div>
          </div>
        `;
      }).join('');

      container.querySelectorAll('.btn-delete-ruleset').forEach(btn => {
        btn.addEventListener('click', () => {
          const rs = (this.routing.rule_sets || []).find(r => r.id === btn.dataset.id);
          if (rs) this.deleteRuleSet(rs);
        });
      });
    }

    /**
     * Open Rule Modal Form
     */
    openRuleModal(existing = null, index = -1) {
      const isEdit = existing !== null;
      const data = existing || {
        description: '',
        outbound: 'direct',
        enabled: true,
        domain_suffix: [],
        domain: [],
        domain_keyword: [],
        ip_cidr: [],
        geoip: [],
        geosite: [],
        rule_set: [],
        protocol: [],
        port: []
      };

      const outboundsOptions = this.availableOutbounds.map(tag => {
        return `<option value="${this.escapeHtml(tag)}" ${data.outbound === tag ? 'selected' : ''}>${this.escapeHtml(tag)}</option>`;
      }).join('');

      const bodyHtml = `
        <div class="rule-form-modal">
          <div class="form-row">
            <div class="form-group" style="flex: 2;">
              <label class="form-label">${I18n.t('common.description')}</label>
              <input type="text" class="form-control" id="modal-rule-desc" value="${this.escapeHtml(data.description || '')}" placeholder="Bypass domestic / Block ads">
            </div>
            <div class="form-group" style="flex: 1;">
              <label class="form-label">${I18n.t('routing.target_outbound')} *</label>
              <select class="form-select" id="modal-rule-outbound">
                ${outboundsOptions}
              </select>
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_domain_suffix')}</label>
              <input type="text" class="form-control" id="modal-rule-suffix" value="${(data.domain_suffix || []).join(', ')}" placeholder="google.com, openai.com">
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_domain')}</label>
              <input type="text" class="form-control" id="modal-rule-domain" value="${(data.domain || []).join(', ')}" placeholder="api.github.com">
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_geosite')}</label>
              <input type="text" class="form-control" id="modal-rule-geosite" value="${(data.geosite || []).join(', ')}" placeholder="cn, google, category-ads-all">
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_geoip')}</label>
              <input type="text" class="form-control" id="modal-rule-geoip" value="${(data.geoip || []).join(', ')}" placeholder="cn, private">
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_ip_cidr')}</label>
              <input type="text" class="form-control" id="modal-rule-cidr" value="${(data.ip_cidr || []).join(', ')}" placeholder="192.168.0.0/16, 10.0.0.0/8">
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_ruleset')}</label>
              <input type="text" class="form-control" id="modal-rule-ruleset" value="${(data.rule_set || []).join(', ')}" placeholder="geosite-cn, geoip-cn">
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_protocol')}</label>
              <input type="text" class="form-control" id="modal-rule-proto" value="${(data.protocol || []).join(', ')}" placeholder="http, tls, dns, bittorrent">
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.match_port')}</label>
              <input type="text" class="form-control" id="modal-rule-port" value="${(data.port || []).join(', ')}" placeholder="80, 443, 8080">
            </div>
          </div>
        </div>
      `;

      window.showModal({
        title: isEdit ? I18n.t('routing.edit_rule') : I18n.t('routing.add_rule'),
        bodyHtml,
        saveText: I18n.t('common.save'),
        cancelText: I18n.t('common.cancel'),
        size: 'lg',
        onSave: async () => {
          const desc = document.getElementById('modal-rule-desc').value.trim();
          const outbound = document.getElementById('modal-rule-outbound').value;

          const parseList = (id) => {
            const val = document.getElementById(id)?.value || '';
            return val.split(',').map(s => s.trim()).filter(Boolean);
          };

          const parsePorts = (id) => {
            return parseList(id).map(p => parseInt(p, 10)).filter(n => !isNaN(n));
          };

          const newRule = {
            id: existing ? existing.id : ('rule-' + Date.now()),
            description: desc,
            outbound,
            enabled: existing ? existing.enabled : true
          };

          const suffix = parseList('modal-rule-suffix');
          if (suffix.length) newRule.domain_suffix = suffix;

          const domain = parseList('modal-rule-domain');
          if (domain.length) newRule.domain = domain;

          const geosite = parseList('modal-rule-geosite');
          if (geosite.length) newRule.geosite = geosite;

          const geoip = parseList('modal-rule-geoip');
          if (geoip.length) newRule.geoip = geoip;

          const cidr = parseList('modal-rule-cidr');
          if (cidr.length) newRule.ip_cidr = cidr;

          const ruleset = parseList('modal-rule-ruleset');
          if (ruleset.length) newRule.rule_set = ruleset;

          const proto = parseList('modal-rule-proto');
          if (proto.length) newRule.protocol = proto;

          const ports = parsePorts('modal-rule-port');
          if (ports.length) newRule.port = ports;

          if (isEdit) {
            this.routing.rules[index] = newRule;
          } else {
            this.routing.rules.push(newRule);
          }

          await this.saveRoutingSettings();
          this.renderRules();
          return true;
        }
      });
    }

    /**
     * Delete a rule
     */
    deleteRule(index) {
      window.showModal({
        title: I18n.t('common.delete'),
        bodyHtml: `<p>${I18n.t('routing.delete_rule_confirm')}</p>`,
        saveText: I18n.t('common.delete'),
        cancelText: I18n.t('common.cancel'),
        size: 'sm',
        onSave: async () => {
          this.routing.rules.splice(index, 1);
          await this.saveRoutingSettings();
          this.renderRules();
          return true;
        }
      });
    }

    /**
     * Open Rule Set Modal
     */
    openRuleSetModal() {
      const bodyHtml = `
        <div class="ruleset-form-modal">
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.rs_tag')} *</label>
              <input type="text" class="form-control" id="modal-rs-tag" placeholder="geosite-cn" required>
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.rs_format')}</label>
              <select class="form-select" id="modal-rs-format">
                <option value="binary">${I18n.t('routing.rs_format_binary')}</option>
                <option value="source">${I18n.t('routing.rs_format_source')}</option>
              </select>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">${I18n.t('routing.rs_url')} *</label>
            <input type="url" class="form-control" id="modal-rs-url" placeholder="https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs" required>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.rs_detour')}</label>
              <select class="form-select" id="modal-rs-detour">
                ${this.availableOutbounds.map(t => `<option value="${this.escapeHtml(t)}">${this.escapeHtml(t)}</option>`).join('')}
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('routing.rs_interval')}</label>
              <input type="text" class="form-control" id="modal-rs-interval" value="1d">
            </div>
          </div>

          <div class="form-group mt-2">
            <label class="form-check-label d-flex align-items-center gap-2">
              <input type="checkbox" id="modal-rs-insecure" checked>
              <span>${I18n.t('routing.rs_insecure')}</span>
            </label>
            <div class="form-hint text-xs text-muted mt-1">${I18n.t('routing.rs_insecure_hint')}</div>
          </div>
        </div>
      `;

      window.showModal({
        title: I18n.t('routing.add_ruleset'),
        bodyHtml,
        saveText: I18n.t('common.save'),
        cancelText: I18n.t('common.cancel'),
        onSave: async () => {
          const tag = document.getElementById('modal-rs-tag')?.value.trim();
          const format = document.getElementById('modal-rs-format')?.value;
          const url = document.getElementById('modal-rs-url')?.value.trim();
          const download_detour = document.getElementById('modal-rs-detour')?.value;
          const update_interval = document.getElementById('modal-rs-interval')?.value.trim() || '1d';
          const insecure = document.getElementById('modal-rs-insecure')?.checked ?? true;

          if (!tag || !url) {
            window.showToast('Tag and URL are required', 'error');
            return false;
          }

          const payload = {
            tag,
            type: 'remote',
            format,
            url,
            download_detour,
            update_interval,
            insecure
          };

          try {
            await API.post('routing/rulesets', payload);
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
     * Delete RuleSet
     */
    deleteRuleSet(rs) {
      window.showModal({
        title: I18n.t('common.delete'),
        bodyHtml: `<p>${I18n.t('routing.delete_ruleset_confirm', { tag: rs.tag })}</p>`,
        saveText: I18n.t('common.delete'),
        cancelText: I18n.t('common.cancel'),
        size: 'sm',
        onSave: async () => {
          try {
            await API.del(`routing/rulesets/${rs.id}`);
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

  window.Routing = new RoutingPage();
})();
