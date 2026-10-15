/**
 * PrismBox - Inbounds Management Module
 * Manages sing-box Inbound Listeners (TProxy, TUN, Mixed)
 */

(function () {
  'use strict';

  class InboundsPage {
    constructor() {
      this.inbounds = [];
      this.isLoading = false;
      this.searchQuery = '';
    }

    /**
     * Render the Inbounds management page
     */
    async render() {
      const app = document.getElementById('app');
      if (!app) return;

      app.innerHTML = `
        <div class="inbounds-page">
          <div class="section-header">
            <div>
              <h2 class="section-title">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="7 10 12 15 17 10"/>
                  <line x1="12" y1="15" x2="12" y2="3"/>
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                </svg>
                ${I18n.t('inbound.title')}
              </h2>
              <span class="section-subtitle">${I18n.t('inbound.subtitle')}</span>
            </div>

            <div class="actions-row">
              <div class="search-input-box">
                <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="11" cy="11" r="8"/>
                  <line x1="21" y1="21" x2="16.65" y2="16.65"/>
                </svg>
                <input type="text" class="form-control" id="inbounds-search" placeholder="${I18n.t('common.search')}">
              </div>
              <button class="btn btn-primary" id="btn-add-inbound">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
                <span>${I18n.t('inbound.add')}</span>
              </button>
            </div>
          </div>

          <!-- Inbounds Table Card -->
          <div class="table-responsive">
            <table class="glass-table" id="inbounds-table">
              <thead>
                <tr>
                  <th style="width: 70px;">${I18n.t('inbound.status')}</th>
                  <th>${I18n.t('inbound.tag')}</th>
                  <th>${I18n.t('inbound.type')}</th>
                  <th>${I18n.t('inbound.listen')}</th>
                  <th>${I18n.t('inbound.port')}</th>
                  <th>${I18n.t('inbound.details')}</th>
                  <th style="width: 110px; text-align: right;">${I18n.t('common.actions')}</th>
                </tr>
              </thead>
              <tbody id="inbounds-table-body">
                <tr>
                  <td colspan="7" class="text-center py-4 text-muted">
                    <span class="spinner-sm"></span>
                    <span class="ml-2">${I18n.t('common.loading')}</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      `;

      this.bindEvents();
      await this.loadInbounds();
    }

    /**
     * Bind DOM events
     */
    bindEvents() {
      const addBtn = document.getElementById('btn-add-inbound');
      if (addBtn) {
        addBtn.addEventListener('click', () => this.openInboundModal());
      }

      const searchInput = document.getElementById('inbounds-search');
      if (searchInput) {
        searchInput.addEventListener('input', (e) => {
          this.searchQuery = e.target.value.toLowerCase().trim();
          this.renderRows();
        });
      }
    }

    /**
     * Fetch all inbounds from backend
     */
    async loadInbounds() {
      this.isLoading = true;
      try {
        const data = await API.get('inbounds');
        this.inbounds = Array.isArray(data) ? data : [];
      } catch (err) {
        console.error('[Inbounds] Failed to load:', err);
      } finally {
        this.isLoading = false;
        this.renderRows();
      }
    }

    /**
     * Render Inbound table rows
     */
    renderRows() {
      const tbody = document.getElementById('inbounds-table-body');
      if (!tbody) return;

      const filtered = this.inbounds.filter(inb => {
        if (!this.searchQuery) return true;
        const q = this.searchQuery;
        return (inb.tag || '').toLowerCase().includes(q) ||
               (inb.type || '').toLowerCase().includes(q) ||
               (inb.listen || '').toLowerCase().includes(q) ||
               String(inb.port || '').includes(q);
      });

      if (filtered.length === 0) {
        tbody.innerHTML = `
          <tr>
            <td colspan="7" class="text-center py-4 text-muted">
              ${I18n.t('inbound.no_inbounds')}
            </td>
          </tr>
        `;
        return;
      }

      tbody.innerHTML = filtered.map(inb => {
        let detailsHtml = '';
        if (inb.type === 'tun') {
          detailsHtml = `
            <span class="badge badge-info">dev: ${inb.interface_name || 'singbox-tun'}</span>
            <span class="badge badge-neutral">mtu: ${inb.mtu || 1500}</span>
            <span class="badge badge-neutral">stack: ${inb.stack || 'system'}</span>
            ${inb.auto_route ? `<span class="badge badge-success">${I18n.t('inbound.tun_auto_route')}</span>` : ''}
          `;
        } else if (inb.type === 'tproxy') {
          detailsHtml = `
            <span class="badge badge-warning">net: ${inb.network || 'tcp,udp'}</span>
            <span class="badge badge-info">${I18n.t('inbound.sniff')}</span>
          `;
        } else if (inb.type === 'mixed') {
          detailsHtml = `
            <span class="badge badge-neutral">SOCKS5 + HTTP</span>
            ${inb.set_system_proxy ? `<span class="badge badge-success">${I18n.t('inbound.mixed_sysproxy')}</span>` : ''}
          `;
        }

        const typeBadges = {
          tproxy: 'badge-warning',
          tun: 'badge-primary',
          mixed: 'badge-info'
        };

        return `
          <tr data-id="${inb.id}">
            <td>
              <label class="toggle-switch">
                <input type="checkbox" class="inbound-toggle" data-id="${inb.id}" ${inb.enabled ? 'checked' : ''}>
                <span class="toggle-slider"></span>
              </label>
            </td>
            <td>
              <strong class="inbound-tag-name">${this.escapeHtml(inb.tag || '-')}</strong>
            </td>
            <td>
              <span class="badge ${typeBadges[inb.type] || 'badge-neutral'} uppercase">${inb.type}</span>
            </td>
            <td>
              <code>${this.escapeHtml(inb.listen || (inb.type === 'tun' ? '-' : '::'))}</code>
            </td>
            <td>
              <code>${inb.port ? inb.port : (inb.type === 'tun' ? '-' : '-')}</code>
            </td>
            <td>
              <div class="tags-list">
                ${detailsHtml}
              </div>
            </td>
            <td style="text-align: right;">
              <div class="btn-group-row">
                <button class="btn-icon btn-sm btn-edit-inbound" data-id="${inb.id}" title="${I18n.t('common.edit')}">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </button>
                <button class="btn-icon btn-sm btn-delete-inbound text-danger" data-id="${inb.id}" title="${I18n.t('common.delete')}">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                  </svg>
                </button>
              </div>
            </td>
          </tr>
        `;
      }).join('');

      // Attach row action listeners
      tbody.querySelectorAll('.inbound-toggle').forEach(tgl => {
        tgl.addEventListener('change', (e) => this.toggleInbound(e.target.dataset.id, e.target.checked));
      });

      tbody.querySelectorAll('.btn-edit-inbound').forEach(btn => {
        btn.addEventListener('click', () => {
          const inb = this.inbounds.find(x => x.id === btn.dataset.id);
          if (inb) this.openInboundModal(inb);
        });
      });

      tbody.querySelectorAll('.btn-delete-inbound').forEach(btn => {
        btn.addEventListener('click', () => {
          const inb = this.inbounds.find(x => x.id === btn.dataset.id);
          if (inb) this.deleteInbound(inb);
        });
      });
    }

    /**
     * Toggle Inbound enabled status
     */
    async toggleInbound(id, enabled) {
      const inb = this.inbounds.find(x => x.id === id);
      if (!inb) return;

      const updated = Object.assign({}, inb, { enabled });
      try {
        await API.put(`inbounds/${id}`, updated);
        inb.enabled = enabled;
        window.showToast(I18n.t('common.success'), 'success', 2000);
      } catch (err) {
        window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
        this.renderRows(); // revert
      }
    }

    /**
     * Open Inbound modal form
     * Default TProxy preset: port 12345, listen "::", network "tcp,udp", sniff enabled
     */
    openInboundModal(existing = null) {
      const isEdit = !!existing;
      const initialType = existing ? existing.type : 'tproxy';

      const presetTProxy = {
        type: 'tproxy',
        tag: 'tproxy-in',
        listen: '::',
        port: 12345,
        network: 'tcp,udp',
        enabled: true
      };

      const data = existing || presetTProxy;

      const bodyHtml = `
        <div class="inbound-form-modal">
          <!-- Type selector: TProxy (default), TUN, Mixed -->
          <div class="form-group">
            <label class="form-label">${I18n.t('inbound.type')}</label>
            <select class="form-select" id="modal-inbound-type" ${isEdit ? 'disabled' : ''}>
              <option value="tproxy" ${data.type === 'tproxy' ? 'selected' : ''}>TProxy (Transparent Proxy - Linux / Router)</option>
              <option value="tun" ${data.type === 'tun' ? 'selected' : ''}>TUN (Virtual Network Interface)</option>
              <option value="mixed" ${data.type === 'mixed' ? 'selected' : ''}>Mixed (SOCKS5 / HTTP Proxy)</option>
            </select>
          </div>

          <!-- Common fields -->
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">${I18n.t('inbound.tag')} *</label>
              <input type="text" class="form-control" id="modal-inbound-tag" value="${this.escapeHtml(data.tag || '')}" required>
            </div>
            <div class="form-group flex-center justify-start mt-4">
              <label class="toggle-switch mr-2">
                <input type="checkbox" id="modal-inbound-enabled" ${data.enabled !== false ? 'checked' : ''}>
                <span class="toggle-slider"></span>
              </label>
              <label class="form-label mb-0" for="modal-inbound-enabled">${I18n.t('inbound.enabled')}</label>
            </div>
          </div>

          <!-- TProxy & Mixed Listen Address and Port -->
          <div id="section-listen-port" class="form-row" style="display: ${data.type === 'tun' ? 'none' : 'flex'};">
            <div class="form-group">
              <label class="form-label">${I18n.t('inbound.listen')}</label>
              <input type="text" class="form-control" id="modal-inbound-listen" value="${this.escapeHtml(data.listen || '::')}">
            </div>
            <div class="form-group">
              <label class="form-label">${I18n.t('inbound.port')}</label>
              <input type="number" class="form-control" id="modal-inbound-port" value="${data.port || (data.type === 'tproxy' ? 12345 : 2080)}" min="1" max="65535">
            </div>
          </div>

          <!-- TProxy Specific Fields -->
          <div id="section-tproxy-fields" style="display: ${data.type === 'tproxy' ? 'block' : 'none'};">
            <div class="form-group">
              <label class="form-label">${I18n.t('inbound.network')}</label>
              <select class="form-select" id="modal-inbound-network">
                <option value="tcp,udp" ${data.network === 'tcp,udp' ? 'selected' : ''}>TCP + UDP (Both)</option>
                <option value="tcp" ${data.network === 'tcp' ? 'selected' : ''}>TCP only</option>
                <option value="udp" ${data.network === 'udp' ? 'selected' : ''}>UDP only</option>
              </select>
            </div>
          </div>

          <!-- TUN Specific Fields -->
          <div id="section-tun-fields" style="display: ${data.type === 'tun' ? 'block' : 'none'};">
            <div class="form-row">
              <div class="form-group">
                <label class="form-label">${I18n.t('inbound.tun_interface')}</label>
                <input type="text" class="form-control" id="modal-inbound-ifname" value="${this.escapeHtml(data.interface_name || 'singbox-tun')}">
              </div>
              <div class="form-group">
                <label class="form-label">${I18n.t('inbound.tun_mtu')}</label>
                <input type="number" class="form-control" id="modal-inbound-mtu" value="${data.mtu || 1500}">
              </div>
            </div>

            <div class="form-group">
              <label class="form-label">${I18n.t('inbound.tun_stack')}</label>
              <select class="form-select" id="modal-inbound-stack">
                <option value="system" ${data.stack === 'system' ? 'selected' : ''}>system</option>
                <option value="gvisor" ${data.stack === 'gvisor' ? 'selected' : ''}>gvisor</option>
                <option value="mixed" ${data.stack === 'mixed' ? 'selected' : ''}>mixed</option>
              </select>
            </div>

            <div class="form-row">
              <div class="form-group">
                <label class="form-label">${I18n.t('inbound.tun_inet4')}</label>
                <input type="text" class="form-control" id="modal-inbound-inet4" value="${(data.inet4_address || ['172.19.0.1/30']).join(', ')}">
              </div>
              <div class="form-group">
                <label class="form-label">${I18n.t('inbound.tun_inet6')}</label>
                <input type="text" class="form-control" id="modal-inbound-inet6" value="${(data.inet6_address || ['fdfe:dcba:9876::1/126']).join(', ')}">
              </div>
            </div>

            <div class="form-row mt-2">
              <div class="form-group flex-center justify-start">
                <label class="toggle-switch mr-2">
                  <input type="checkbox" id="modal-inbound-autoroute" ${data.auto_route !== false ? 'checked' : ''}>
                  <span class="toggle-slider"></span>
                </label>
                <label class="form-label mb-0" for="modal-inbound-autoroute">${I18n.t('inbound.tun_auto_route')}</label>
              </div>

              <div class="form-group flex-center justify-start">
                <label class="toggle-switch mr-2">
                  <input type="checkbox" id="modal-inbound-strictroute" ${data.strict_route ? 'checked' : ''}>
                  <span class="toggle-slider"></span>
                </label>
                <label class="form-label mb-0" for="modal-inbound-strictroute">${I18n.t('inbound.tun_strict_route')}</label>
              </div>
            </div>
          </div>

          <!-- Mixed Specific Fields -->
          <div id="section-mixed-fields" style="display: ${data.type === 'mixed' ? 'block' : 'none'};">
            <div class="form-group flex-center justify-start mt-2">
              <label class="toggle-switch mr-2">
                <input type="checkbox" id="modal-inbound-sysproxy" ${data.set_system_proxy ? 'checked' : ''}>
                <span class="toggle-slider"></span>
              </label>
              <label class="form-label mb-0" for="modal-inbound-sysproxy">${I18n.t('inbound.mixed_sysproxy')}</label>
            </div>
          </div>
        </div>
      `;

      window.showModal({
        title: isEdit ? I18n.t('inbound.edit') : I18n.t('inbound.add'),
        bodyHtml,
        saveText: I18n.t('common.save'),
        cancelText: I18n.t('common.cancel'),
        size: 'lg',
        onSave: async () => {
          const type = document.getElementById('modal-inbound-type').value;
          const tag = document.getElementById('modal-inbound-tag').value.trim();
          const enabled = document.getElementById('modal-inbound-enabled').checked;

          if (!tag) {
            window.showToast(I18n.t('inbound.tag') + ' is required', 'error');
            return false;
          }

          const payload = {
            tag,
            type,
            enabled
          };

          if (isEdit) {
            payload.id = existing.id;
          }

          if (type !== 'tun') {
            payload.listen = document.getElementById('modal-inbound-listen').value.trim() || '::';
            payload.port = parseInt(document.getElementById('modal-inbound-port').value, 10) || 12345;
          }

          if (type === 'tproxy') {
            payload.network = document.getElementById('modal-inbound-network').value;
          } else if (type === 'tun') {
            payload.interface_name = document.getElementById('modal-inbound-ifname').value.trim() || 'singbox-tun';
            payload.mtu = parseInt(document.getElementById('modal-inbound-mtu').value, 10) || 1500;
            payload.stack = document.getElementById('modal-inbound-stack').value;
            payload.auto_route = document.getElementById('modal-inbound-autoroute').checked;
            payload.strict_route = document.getElementById('modal-inbound-strictroute').checked;

            const inet4 = document.getElementById('modal-inbound-inet4').value.split(',').map(s => s.trim()).filter(Boolean);
            const inet6 = document.getElementById('modal-inbound-inet6').value.split(',').map(s => s.trim()).filter(Boolean);
            if (inet4.length) payload.inet4_address = inet4;
            if (inet6.length) payload.inet6_address = inet6;
          } else if (type === 'mixed') {
            payload.set_system_proxy = document.getElementById('modal-inbound-sysproxy').checked;
          }

          try {
            if (isEdit) {
              await API.put(`inbounds/${existing.id}`, payload);
            } else {
              await API.post('inbounds', payload);
            }
            window.showToast(I18n.t('common.success'), 'success');
            await this.loadInbounds();
            return true;
          } catch (err) {
            window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
            return false;
          }
        }
      });

      // Type change dynamically toggles field sections
      const typeSelect = document.getElementById('modal-inbound-type');
      if (typeSelect) {
        typeSelect.addEventListener('change', (e) => {
          const newType = e.target.value;
          const listenPortSec = document.getElementById('section-listen-port');
          const tproxySec = document.getElementById('section-tproxy-fields');
          const tunSec = document.getElementById('section-tun-fields');
          const mixedSec = document.getElementById('section-mixed-fields');
          const tagInput = document.getElementById('modal-inbound-tag');

          listenPortSec.style.display = newType === 'tun' ? 'none' : 'flex';
          tproxySec.style.display = newType === 'tproxy' ? 'block' : 'none';
          tunSec.style.display = newType === 'tun' ? 'block' : 'none';
          mixedSec.style.display = newType === 'mixed' ? 'block' : 'none';

          if (!isEdit && tagInput) {
            tagInput.value = `${newType}-in`;
          }
        });
      }
    }

    /**
     * Delete an inbound
     */
    deleteInbound(inb) {
      const confirmText = I18n.t('inbound.delete_confirm', { tag: inb.tag });
      window.showModal({
        title: I18n.t('common.delete'),
        bodyHtml: `<p>${this.escapeHtml(confirmText)}</p>`,
        saveText: I18n.t('common.delete'),
        cancelText: I18n.t('common.cancel'),
        size: 'sm',
        onSave: async () => {
          try {
            await API.del(`inbounds/${inb.id}`);
            window.showToast(I18n.t('common.success'), 'success');
            await this.loadInbounds();
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
      // Cleanup if any
    }
  }

  window.Inbounds = new InboundsPage();
})();
