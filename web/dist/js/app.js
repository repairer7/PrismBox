/**
 * PrismBox - Main Application Entrypoint & SPA Core
 * Hash Router, Glassmorphism Modal System, Toast Notifications, Kernel Controls, i18n
 */

(function () {
  'use strict';

  // Available routes and their controller objects with i18n keys
  const ROUTES = {
    'dashboard': {
      titleKey: 'dashboard.title',
      subtitleKey: 'dashboard.subtitle',
      getController: () => window.Dashboard
    },
    'inbounds': {
      titleKey: 'inbound.title',
      subtitleKey: 'inbound.subtitle',
      getController: () => window.Inbounds
    },
    'outbounds': {
      titleKey: 'outbound.title',
      subtitleKey: 'outbound.subtitle',
      getController: () => window.Outbounds
    },
    'routing': {
      titleKey: 'routing.title',
      subtitleKey: 'routing.subtitle',
      getController: () => window.Routing
    },
    'dns': {
      titleKey: 'dns.title',
      subtitleKey: 'dns.subtitle',
      getController: () => window.DNS
    },
    'config': {
      titleKey: 'config.title',
      subtitleKey: 'config.subtitle',
      getController: () => window.Config
    }
  };

  class AppController {
    constructor() {
      this.currentRoute = null;
      this.currentPageController = null;
      this.kernelPollTimer = null;
      this.kernelStatus = 'unknown';
    }

    /**
     * Initialize Application
     */
    init() {
      this.registerGlobalHelpers();
      this.initModalSystem();
      this.initSidebar();
      this.initLangSwitcher();
      this.initKernelControls();
      this.initSaveApply();
      this.initRouter();
      this.initWebSocket();
      this.startKernelStatusPolling();
      this.updateStaticI18n();
    }

    /**
     * Register global utility helpers
     */
    registerGlobalHelpers() {
      /**
       * Format bytes to human readable string
       */
      window.formatBytes = function (bytes, decimals = 2) {
        if (!bytes || bytes <= 0 || isNaN(bytes)) return '0 B';
        const k = 1024;
        const dm = decimals < 0 ? 0 : decimals;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        const val = parseFloat((bytes / Math.pow(k, i)).toFixed(dm));
        return `${val} ${sizes[i] || 'TB'}`;
      };

      /**
       * Format speed to human readable string
       */
      window.formatSpeed = function (bytesPerSec, decimals = 1) {
        if (!bytesPerSec || bytesPerSec <= 0 || isNaN(bytesPerSec)) return '0 B/s';
        const k = 1024;
        const dm = decimals < 0 ? 0 : decimals;
        const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
        const i = Math.floor(Math.log(bytesPerSec) / Math.log(k));
        const val = parseFloat((bytesPerSec / Math.pow(k, i)).toFixed(dm));
        return `${val} ${sizes[i] || 'GB/s'}`;
      };

      /**
       * Format seconds to uptime string
       */
      window.formatUptime = function (seconds) {
        if (!seconds || seconds <= 0 || isNaN(seconds)) return '0s';
        const d = Math.floor(seconds / (3600 * 24));
        const h = Math.floor((seconds % (3600 * 24)) / 3600);
        const m = Math.floor((seconds % 3600) / 60);
        const s = Math.floor(seconds % 60);

        const parts = [];
        if (d > 0) parts.push(`${d}d`);
        if (h > 0) parts.push(`${h}h`);
        if (m > 0) parts.push(`${m}m`);
        parts.push(`${s}s`);
        return parts.join(' ');
      };

      /**
       * Toast notification helper
       */
      window.showToast = this.showToast.bind(this);
    }

    /**
     * Display a floating Glassmorphism toast
     */
    showToast(message, type = 'info', duration = 3500) {
      const container = document.getElementById('toast-container');
      if (!container) return;

      const toast = document.createElement('div');
      toast.className = `toast toast-${type}`;

      const iconSvg = {
        success: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>',
        error: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>',
        warning: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>',
        info: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>'
      }[type] || '';

      toast.innerHTML = `
        <div class="toast-icon">${iconSvg}</div>
        <div class="toast-content">${message}</div>
        <button class="toast-close" aria-label="Close toast">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
        <div class="toast-progress"></div>
      `;

      container.appendChild(toast);

      const closeBtn = toast.querySelector('.toast-close');
      const removeToast = () => {
        toast.style.animation = 'toastSlideIn 0.25s reverse forwards';
        setTimeout(() => toast.remove(), 250);
      };

      closeBtn.addEventListener('click', removeToast);
      setTimeout(removeToast, duration);
    }

    /**
     * Modal dialog system
     */
    initModalSystem() {
      const backdrop = document.getElementById('modal-backdrop');
      const dialog = document.getElementById('modal-dialog');
      const title = document.getElementById('modal-title');
      const body = document.getElementById('modal-body');
      const closeBtn = document.getElementById('modal-close-btn');
      const cancelBtn = document.getElementById('modal-cancel-btn');
      const saveBtn = document.getElementById('modal-save-btn');

      let currentSaveHandler = null;

      window.showModal = (opts, bodyHtmlParam, onSaveParam, extraOpts = {}) => {
        let modalTitle = '';
        let bodyHtml = '';
        let onSave = null;
        let saveText = I18n.t('common.save');
        let cancelText = I18n.t('common.cancel');
        let size = 'md';

        if (typeof opts === 'string') {
          modalTitle = opts;
          bodyHtml = bodyHtmlParam || '';
          onSave = onSaveParam || null;
          if (extraOpts.saveText) saveText = extraOpts.saveText;
          if (extraOpts.cancelText) cancelText = extraOpts.cancelText;
          if (extraOpts.size) size = extraOpts.size;
        } else if (opts && typeof opts === 'object') {
          modalTitle = opts.title || '';
          bodyHtml = opts.bodyHtml || '';
          onSave = opts.onSave || null;
          if (opts.saveText) saveText = opts.saveText;
          if (opts.cancelText) cancelText = opts.cancelText;
          if (opts.size) size = opts.size;
        }

        title.textContent = modalTitle;
        body.innerHTML = bodyHtml;
        saveBtn.textContent = saveText;
        cancelBtn.textContent = cancelText;

        dialog.className = `modal-dialog modal-${size}`;

        currentSaveHandler = onSave;
        backdrop.style.display = 'flex';
        document.body.style.overflow = 'hidden';
      };

      window.closeModal = () => {
        backdrop.style.display = 'none';
        body.innerHTML = '';
        currentSaveHandler = null;
        document.body.style.overflow = '';
      };

      closeBtn.addEventListener('click', window.closeModal);
      cancelBtn.addEventListener('click', window.closeModal);

      backdrop.addEventListener('click', (e) => {
        if (e.target === backdrop) {
          window.closeModal();
        }
      });

      saveBtn.addEventListener('click', async () => {
        if (typeof currentSaveHandler === 'function') {
          saveBtn.classList.add('loading');
          try {
            const shouldClose = await currentSaveHandler();
            if (shouldClose !== false) {
              window.closeModal();
            }
          } catch (err) {
            console.error('[Modal Save] Error:', err);
          } finally {
            saveBtn.classList.remove('loading');
          }
        }
      });

      // Escape key to close modal
      document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && backdrop.style.display === 'flex') {
          window.closeModal();
        }
      });
    }

    /**
     * Mobile sidebar toggle
     */
    initSidebar() {
      const sidebar = document.getElementById('sidebar');
      const backdrop = document.getElementById('sidebar-backdrop');
      const openBtn = document.getElementById('mobile-menu-toggle');
      const closeBtn = document.getElementById('sidebar-close');

      const closeSidebar = () => {
        sidebar?.classList.remove('mobile-open');
        backdrop?.classList.remove('active');
      };

      openBtn?.addEventListener('click', () => {
        sidebar?.classList.add('mobile-open');
        backdrop?.classList.add('active');
      });

      closeBtn?.addEventListener('click', closeSidebar);
      backdrop?.addEventListener('click', closeSidebar);

      // Close on route change on mobile
      window.addEventListener('hashchange', () => {
        if (window.innerWidth <= 820) {
          closeSidebar();
        }
      });
    }

    /**
     * Language switcher
     */
    initLangSwitcher() {
      const btn = document.getElementById('btn-lang-toggle');
      const indicator = document.getElementById('lang-indicator-text');

      const updateLangButton = () => {
        if (indicator) {
          indicator.textContent = I18n.currentLang === 'zh' ? '中文' : 'EN';
        }
      };

      updateLangButton();

      btn?.addEventListener('click', () => {
        const nextLang = I18n.currentLang === 'en' ? 'zh' : 'en';
        I18n.setLang(nextLang);
        updateLangButton();
      });
    }

    /**
     * Update all static [data-i18n] text in the DOM
     */
    updateStaticI18n() {
      document.querySelectorAll('[data-i18n]').forEach((el) => {
        const key = el.getAttribute('data-i18n');
        if (key) {
          el.textContent = I18n.t(key);
        }
      });

      const navMain = document.getElementById('nav-section-main');
      if (navMain) navMain.textContent = I18n.t('nav.main_nav');

      const langIndicator = document.getElementById('lang-indicator-text');
      if (langIndicator) {
        langIndicator.textContent = I18n.currentLang === 'zh' ? '中文' : 'EN';
      }

      this.updateTopbarTitles();
    }

    /**
     * Called whenever language changes
     */
    onLanguageChange() {
      this.updateStaticI18n();
      this.updateKernelStatusBadge(this.kernelStatus);

      // Re-render current active view
      if (this.currentPageController && typeof this.currentPageController.render === 'function') {
        this.currentPageController.render();
      }
    }

    /**
     * Kernel Start, Stop, and Restart control handlers
     */
    initKernelControls() {
      const btnStart = document.getElementById('btn-kernel-start');
      const btnRestart = document.getElementById('btn-kernel-restart');
      const btnStop = document.getElementById('btn-kernel-stop');

      btnStart?.addEventListener('click', async () => {
        btnStart.classList.add('loading');
        try {
          await API.post('kernel/start');
          this.showToast(I18n.t('kernel.start_success'), 'success');
          await this.checkKernelStatus();
        } catch (err) {
          this.showToast(err.message, 'error');
        } finally {
          btnStart.classList.remove('loading');
        }
      });

      btnRestart?.addEventListener('click', async () => {
        btnRestart.classList.add('loading');
        try {
          await API.post('kernel/restart');
          this.showToast(I18n.t('kernel.restart_success'), 'success');
          await this.checkKernelStatus();
        } catch (err) {
          this.showToast(err.message, 'error');
        } finally {
          btnRestart.classList.remove('loading');
        }
      });

      btnStop?.addEventListener('click', async () => {
        btnStop.classList.add('loading');
        try {
          await API.post('kernel/stop');
          this.showToast(I18n.t('kernel.stop_success'), 'warning');
          await this.checkKernelStatus();
        } catch (err) {
          this.showToast(err.message, 'error');
        } finally {
          btnStop.classList.remove('loading');
        }
      });
    }

    /**
     * "Save & Apply" primary action button:
     * 1. Call POST /config/validate
     * 2. If validation fails, show error toast with details
     * 3. If passes, call POST /kernel/restart
     * 4. Show success toast
     */
    initSaveApply() {
      const btnApply = document.getElementById('btn-save-apply');
      if (!btnApply) return;

      btnApply.addEventListener('click', async () => {
        btnApply.classList.add('loading');
        btnApply.disabled = true;

        try {
          this.showToast(I18n.t('kernel.validating'), 'info', 2000);

          // 1. Validate configuration
          const valRes = await API.post('config/validate', {});
          if (!valRes || valRes.valid === false) {
            const errDetail = valRes?.error || I18n.t('kernel.validate_failed');
            this.showToast(`${I18n.t('kernel.validate_failed')}: ${errDetail}`, 'error', 5000);
            return;
          }

          // 2. Restart kernel with newly validated configuration
          this.showToast(I18n.t('kernel.applying'), 'info', 2000);
          await API.post('kernel/restart');
          this.showToast(I18n.t('kernel.restart_success'), 'success', 4000);
          await this.checkKernelStatus();
        } catch (err) {
          this.showToast(`${I18n.t('kernel.restart_failed')}: ${err.message}`, 'error', 5000);
        } finally {
          btnApply.classList.remove('loading');
          btnApply.disabled = false;
        }
      });
    }

    /**
     * Polling kernel status
     */
    startKernelStatusPolling() {
      this.checkKernelStatus();
      this.kernelPollTimer = setInterval(() => this.checkKernelStatus(), 5000);
    }

    async checkKernelStatus() {
      try {
        const res = await API.get('kernel/status', { showToast: false });
        if (res) {
          const status = res.running ? 'running' : (res.status || 'stopped');
          this.kernelStatus = status;
          this.updateKernelStatusBadge(status);
        }
      } catch (e) {
        // Optimistic offline check
        this.updateKernelStatusBadge(this.kernelStatus || 'unknown');
      }
    }

    updateKernelStatusBadge(status) {
      const badge = document.getElementById('kernel-status-badge');
      const text = document.getElementById('kernel-status-text');
      if (!badge || !text) return;

      badge.className = `kernel-status-badge ${status}`;
      const statusLabels = {
        running: I18n.t('kernel.running'),
        stopped: I18n.t('kernel.stopped'),
        error: I18n.t('kernel.error'),
        unknown: I18n.t('kernel.unknown')
      };
      text.textContent = statusLabels[status] || status;
    }

    /**
     * WebSocket connection & indicator
     */
    initWebSocket() {
      const wsDot = document.getElementById('ws-status-dot');
      const wsLabel = document.getElementById('sidebar-ws-label');

      WS.onStateChange((state) => {
        if (wsDot) {
          wsDot.className = `ws-indicator ${state}`;
        }
        if (wsLabel) {
          if (state === 'connected') {
            wsLabel.textContent = I18n.t('nav.connected');
          } else if (state === 'connecting') {
            wsLabel.textContent = I18n.t('nav.connecting');
          } else {
            wsLabel.textContent = I18n.t('nav.disconnected');
          }
        }
      });

      WS.onStats((stats) => {
        if (stats.kernelStatus) {
          this.kernelStatus = stats.kernelStatus;
          this.updateKernelStatusBadge(stats.kernelStatus);
        }
      });

      WS.onKernelStatus((info) => {
        if (info && info.status) {
          this.kernelStatus = info.status;
          this.updateKernelStatusBadge(info.status);
        }
      });

      WS.connect();
    }

    /**
     * Hash Router setup
     */
    initRouter() {
      const handleHashChange = () => {
        const rawHash = window.location.hash.slice(1);
        const cleanRoute = rawHash.replace(/^\//, '').split('?')[0].split('/')[0] || 'dashboard';
        this.navigateTo(cleanRoute);
      };

      window.addEventListener('hashchange', handleHashChange);
      handleHashChange();
    }

    /**
     * Navigate to specific route
     */
    navigateTo(routeName) {
      const route = ROUTES[routeName] || ROUTES['dashboard'];
      const targetRoute = ROUTES[routeName] ? routeName : 'dashboard';

      if (this.currentRoute === targetRoute && this.currentPageController) {
        return;
      }

      this.currentRoute = targetRoute;

      // Update Navigation Links active state
      document.querySelectorAll('.sidebar-nav .nav-link').forEach((link) => {
        const linkRoute = link.getAttribute('data-route');
        if (linkRoute === targetRoute) {
          link.classList.add('active');
        } else {
          link.classList.remove('active');
        }
      });

      this.updateTopbarTitles();

      // Teardown previous controller
      if (this.currentPageController && typeof this.currentPageController.destroy === 'function') {
        try {
          this.currentPageController.destroy();
        } catch (e) {
          console.warn('[Router] Error destroying previous page:', e);
        }
      }

      // Initialize and render new controller
      this.currentPageController = route.getController();
      if (this.currentPageController && typeof this.currentPageController.render === 'function') {
        this.currentPageController.render();
      }
    }

    updateTopbarTitles() {
      const pageTitle = document.getElementById('page-title');
      const pageSubtitle = document.getElementById('page-subtitle');
      const route = ROUTES[this.currentRoute] || ROUTES['dashboard'];

      if (pageTitle && route.titleKey) {
        pageTitle.textContent = I18n.t(route.titleKey);
      }
      if (pageSubtitle && route.subtitleKey) {
        pageSubtitle.textContent = I18n.t(route.subtitleKey);
      }
    }
  }

  // Instantiate and expose application controller
  window.App = new AppController();

  document.addEventListener('DOMContentLoaded', () => {
    window.App.init();
  });
})();
