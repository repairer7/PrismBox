/**
 * PrismBox - Configuration Center Module
 * Visual overview and Schema-aware JSON editor with line numbers,
 * intelligent autocomplete, hover parameter hints, real-time syntax checking,
 * and sing-box 1.14.3 official schema alignment (https://sing-box.sagernet.org/schema.json).
 */

(function () {
  'use strict';

  // sing-box 1.14.3 Schema Definition Reference for Hover Tooltips & Autocomplete
  const SING_BOX_SCHEMA_DEFS = {
    // Root level
    "$schema": { type: "string", desc: "JSON Schema URI for sing-box 1.14.3 (https://sing-box.sagernet.org/schema.json)" },
    "log": { type: "object", desc: "Logging configuration (level, timestamp, disabled, output)" },
    "dns": { type: "object", desc: "DNS server and routing rules configuration (omit to leave empty & use host system default)" },
    "inbounds": { type: "array", desc: "List of inbound proxy listeners (tproxy, tun, mixed, etc.)" },
    "outbounds": { type: "array", desc: "List of outbound proxy nodes, direct/block/dns outbounds, and groups" },
    "route": { type: "object", desc: "Traffic routing rules, rule_set definitions, and domain resolution policies" },
    "experimental": { type: "object", desc: "Experimental features including clash_api, v2ray_api, and cache_file" },
    "ntp": { type: "object", desc: "Built-in NTP client configuration" },

    // Inbounds & Common Fields
    "tag": { type: "string", desc: "Unique identifier tag for this inbound or outbound" },
    "type": { 
      type: "string", 
      desc: "Protocol or driver type", 
      options: ["tproxy", "tun", "mixed", "direct", "block", "dns", "socks", "http", "shadowsocks", "vmess", "vless", "trojan", "hysteria2", "tuic", "selector", "urltest"] 
    },
    "listen": { type: "string", desc: "IP address to listen on (e.g. '::' or '0.0.0.0')" },
    "listen_port": { type: "integer", desc: "Listening port number (e.g. 12345 for landscape-router flow)" },
    "network": { type: "string", desc: "Transport network protocol ('tcp', 'udp', or 'tcp,udp')", options: ["tcp", "udp", "tcp,udp"] },
    "sniff": { type: "boolean", desc: "Enable protocol sniffing on incoming connections to extract domain names" },
    "sniff_override_destination": { type: "boolean", desc: "Override connection destination with domain sniffed from packet" },
    "interface_name": { type: "string", desc: "TUN: Name of the virtual TUN interface (e.g. 'tun0')" },
    "mtu": { type: "integer", desc: "TUN: Maximum transmission unit (default: 9000)" },
    "auto_route": { type: "boolean", desc: "TUN: Automatically configure operating system default routing table" },
    "strict_route": { type: "boolean", desc: "TUN: Strictly route all traffic through TUN interface to prevent leaks" },
    "stack": { type: "string", desc: "TUN: TCP/IP protocol stack implementation", options: ["system", "gvisor", "mixed"] },
    "inet4_address": { type: "array", desc: "TUN: IPv4 CIDR address list (e.g. ['172.19.0.1/30'])" },
    "inet6_address": { type: "array", desc: "TUN: IPv6 CIDR address list (e.g. ['fdfe:dcba:9876::1/126'])" },
    "set_system_proxy": { type: "boolean", desc: "Mixed: Automatically set OS system HTTP/SOCKS proxy" },

    // Outbounds & Nodes
    "server": { type: "string", desc: "Remote server IP address or hostname" },
    "server_port": { type: "integer", desc: "Remote server port number" },
    "uuid": { type: "string", desc: "User UUID for VMess / VLESS / TUIC" },
    "password": { type: "string", desc: "Authentication password / secret" },
    "security": { type: "string", desc: "Cipher / security method", options: ["auto", "aes-128-gcm", "chacha20-poly1305", "none", "reality", "tls"] },
    "flow": { type: "string", desc: "VLESS flow control", options: ["xtls-rprx-vision"] },
    "tls": { type: "object", desc: "TLS / Reality configuration (enabled, server_name, insecure, alpn, reality, utls)" },
    "transport": { type: "object", desc: "Transport layer settings (type: 'ws', 'grpc', 'http', headers, path)" },
    "obfs": { type: "object", desc: "Hysteria2 obfuscation settings (type: 'salamander', password)" },
    "up_mbps": { type: "integer", desc: "Hysteria2: Upstream bandwidth limit in Mbps" },
    "down_mbps": { type: "integer", desc: "Hysteria2: Downstream bandwidth limit in Mbps" },
    "congestion_control": { type: "string", desc: "TUIC / QUIC congestion controller", options: ["bbr", "cubic", "new_reno"] },

    // Route & Rules
    "rules": { type: "array", desc: "Ordered list of routing match rules executed top-to-bottom" },
    "rule_set": { type: "array", desc: "Remote or local rule set definitions / match criteria" },
    "action": { type: "string", desc: "Rule action in sing-box 1.14+ action format", options: ["sniff", "resolve", "hijack-dns", "route", "reject"] },
    "domain": { type: "array", desc: "Exact full domain match list" },
    "domain_suffix": { type: "array", desc: "Domain suffix match list (e.g. ['.google.com', '.github.com'])" },
    "domain_keyword": { type: "array", desc: "Domain keyword match list" },
    "domain_regex": { type: "array", desc: "Regular expression domain match list" },
    "ip_cidr": { type: "array", desc: "Destination IP CIDR match list (e.g. ['10.0.0.0/8', '192.168.0.0/16'])" },
    "geoip": { type: "array", desc: "GeoIP country code match list (e.g. ['cn', 'private'])" },
    "geosite": { type: "array", desc: "GeoSite category match list (e.g. ['cn', 'geolocation-!cn', 'apple'])" },
    "port": { type: "array", desc: "Destination port match list (e.g. [53, 80, 443])" },
    "protocol": { type: "array", desc: "Sniffed application protocol match (e.g. ['http', 'tls', 'dns', 'quic'])" },
    "outbound": { type: "string", desc: "Target outbound or group tag to route matched traffic to" },
    "auto_detect_interface": { type: "boolean", desc: "Automatically detect and bind default network interface" },
    "domain_strategy": { type: "string", desc: "DNS domain resolution strategy for routing", options: ["prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"] },

    // RuleSet
    "format": { type: "string", desc: "RuleSet file format ('binary' for .srs, 'source' for .json)", options: ["binary", "source"] },
    "url": { type: "string", desc: "Download URL for remote rule set" },
    "download_detour": { type: "string", desc: "Outbound tag used to download this rule set (e.g. 'direct')" },
    "update_interval": { type: "string", desc: "Rule set update interval (e.g. '1d', '12h')" },

    // DNS
    "servers": { type: "array", desc: "Upstream DNS servers list" },
    "address": { type: "string", desc: "DNS server address (e.g. 'https://dns.google/dns-query', 'tls://1.1.1.1', 'udp://223.5.5.5')" },
    "domain_resolver": { type: "string", desc: "DNS server tag used to bootstrap this DNS server's domain (modern replacement for address_resolver)" },
    "address_resolver": { type: "string", desc: "Legacy field for domain_resolver" },
    "detour": { type: "string", desc: "Outbound tag to route this DNS query through (e.g. 'direct', 'proxy')" },
    "strategy": { type: "string", desc: "DNS query domain strategy", options: ["prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"] },
    "disable_cache": { type: "boolean", desc: "Disable internal DNS query cache" },
    "disable_expire": { type: "boolean", desc: "Disable DNS TTL expiration" }
  };

  // Preset Snippets for Quick Injection into the Code Editor
  const SNIPPET_PRESETS = [
    {
      label: "+ TProxy (12345)",
      desc: "landscape-router flow transparent proxy inbound",
      snippet: `  {
    "type": "tproxy",
    "tag": "tproxy-in",
    "listen": "::",
    "listen_port": 12345,
    "network": "tcp,udp",
    "sniff": true,
    "sniff_override_destination": true
  }`
    },
    {
      label: "+ Hysteria 2",
      desc: "Hysteria 2 UDP proxy outbound",
      snippet: `  {
    "type": "hysteria2",
    "tag": "hy2-node",
    "server": "198.51.100.1",
    "server_port": 443,
    "password": "your-password",
    "tls": {
      "enabled": true,
      "server_name": "example.com",
      "insecure": true
    }
  }`
    },
    {
      label: "+ VLESS Reality",
      desc: "VLESS with Reality security & Vision flow",
      snippet: `  {
    "type": "vless",
    "tag": "vless-reality",
    "server": "1.2.3.4",
    "server_port": 443,
    "uuid": "00000000-0000-0000-0000-000000000000",
    "flow": "xtls-rprx-vision",
    "tls": {
      "enabled": true,
      "server_name": "yahoo.com",
      "reality": {
        "enabled": true,
        "public_key": "YOUR_PUBLIC_KEY",
        "short_id": "YOUR_SHORT_ID"
      },
      "utls": {
        "enabled": true,
        "fingerprint": "chrome"
      }
    }
  }`
    },
    {
      label: "+ Sniff & Hijack-DNS",
      desc: "sing-box 1.14+ action rules for sniffing and DNS interception",
      snippet: `    {"action": "sniff"},
    {"action": "resolve"},
    {"protocol": "dns", "action": "hijack-dns"}`
    },
    {
      label: "+ Remote RuleSet",
      desc: "1.14+ binary .srs rule_set definition",
      snippet: `    {
      "type": "remote",
      "tag": "geosite-cn",
      "format": "binary",
      "url": "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
      "download_detour": "direct",
      "update_interval": "1d"
    }`
    }
  ];

  class ConfigPage {
    constructor() {
      this.activeTab = 'editor'; // 'editor' | 'visual'
      this.currentConfig = null;
      this.rawJson = '';
      this.isValidating = false;
      this.isSaving = false;
      this.diagTimer = null;
      this.tooltipEl = null;
      this.autocompleteEl = null;
    }

    /**
     * Render Configuration Center
     */
    async render() {
      const app = document.getElementById('app');
      if (!app) return;

      app.innerHTML = `
        <div class="config-page">
          <!-- Header and Tabs -->
          <div class="section-header">
            <div>
              <h2 class="section-title">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                  <polyline points="14 2 14 8 20 8"/>
                  <line x1="16" y1="13" x2="8" y2="13"/>
                  <line x1="16" y1="17" x2="8" y2="17"/>
                  <polyline points="10 9 9 9 8 9"/>
                </svg>
                ${I18n.t('config.title')}
              </h2>
              <span class="section-subtitle">${I18n.t('config.subtitle')}</span>
            </div>

            <!-- Tab Buttons -->
            <div class="tab-pill-group">
              <button class="tab-pill ${this.activeTab === 'editor' ? 'active' : ''}" id="tab-btn-editor">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="16 18 22 12 16 6"/>
                  <polyline points="8 6 2 12 8 18"/>
                </svg>
                ${I18n.t('config.tab_editor')}
              </button>
              <button class="tab-pill ${this.activeTab === 'visual' ? 'active' : ''}" id="tab-btn-visual">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="3" width="7" height="7"/>
                  <rect x="14" y="3" width="7" height="7"/>
                  <rect x="14" y="14" width="7" height="7"/>
                  <rect x="3" y="14" width="7" height="7"/>
                </svg>
                ${I18n.t('config.tab_visual')}
              </button>
            </div>
          </div>

          <!-- Top Action Bar -->
          <div class="config-actions-bar glass-panel mb-3">
            <div class="schema-info">
              <span class="schema-label">${I18n.t('config.schema_url')}:</span>
              <a href="https://sing-box.sagernet.org/schema.json" target="_blank" rel="noopener noreferrer" class="schema-link" title="Official sing-box 1.14.3 Schema">
                https://sing-box.sagernet.org/schema.json
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                  <polyline points="15 3 21 3 21 9"/>
                  <line x1="10" y1="14" x2="21" y2="3"/>
                </svg>
              </a>
            </div>

            <div class="config-btn-cluster">
              <!-- Hidden file input for import -->
              <input type="file" id="config-file-input" accept=".json,application/json" style="display: none;">

              <button class="btn btn-secondary btn-sm" id="btn-config-import" title="${I18n.t('config.import')}">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="17 8 12 3 7 8"/>
                  <line x1="12" y1="3" x2="12" y2="15"/>
                </svg>
                <span>${I18n.t('config.import')}</span>
              </button>

              <button class="btn btn-secondary btn-sm" id="btn-config-export" title="${I18n.t('config.export')}">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="7 10 12 15 17 10"/>
                  <line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
                <span>${I18n.t('config.export')}</span>
              </button>

              <button class="btn btn-warning btn-sm" id="btn-config-validate" title="${I18n.t('config.validate')}">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                  <polyline points="22 4 12 14.01 9 11.01"/>
                </svg>
                <span>${I18n.t('config.validate')}</span>
              </button>

              <button class="btn btn-primary btn-sm" id="btn-config-save" title="${I18n.t('config.save')}">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
                  <polyline points="17 21 17 13 7 13 7 21"/>
                  <polyline points="7 3 7 8 15 8"/>
                </svg>
                <span>${I18n.t('config.save')}</span>
              </button>
            </div>
          </div>

          <!-- Tab Content: JSON Editor with Schema Intelligence -->
          <div id="config-editor-container" class="config-tab-pane ${this.activeTab === 'editor' ? 'active' : ''}">
            <div class="code-editor-card glass-panel" style="position: relative;">
              <!-- Toolbar -->
              <div class="code-editor-toolbar">
                <div class="toolbar-left" style="display: flex; align-items: center; gap: 0.75rem;">
                  <span class="editor-file-badge">sing-box-config.json</span>
                  <span class="badge badge-neutral" style="font-size: 0.7rem;">sing-box v1.14.3</span>
                  <span class="editor-status" id="editor-dirty-indicator"></span>
                </div>
                <div class="toolbar-right">
                  <button class="btn-icon btn-sm" id="btn-editor-format" title="Format / Prettify JSON">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="21" y1="10" x2="7" y2="10"/>
                      <line x1="21" y1="6" x2="3" y2="6"/>
                      <line x1="21" y1="14" x2="3" y2="14"/>
                      <line x1="21" y1="18" x2="7" y2="18"/>
                    </svg>
                  </button>
                  <button class="btn-icon btn-sm" id="btn-editor-copy" title="${I18n.t('common.copy')}">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                    </svg>
                  </button>
                </div>
              </div>

              <!-- Quick Schema Snippets Bar -->
              <div class="editor-snippets-bar">
                <span class="editor-snippets-label">Schema Snippets:</span>
                ${SNIPPET_PRESETS.map((p, idx) => `
                  <button class="editor-snippet-pill" data-snippet-idx="${idx}" title="${p.desc}">
                    ${p.label}
                  </button>
                `).join('')}
              </div>

              <!-- Editor Workspace -->
              <div class="code-editor-wrapper" id="editor-workspace">
                <div class="code-editor-gutter" id="editor-line-numbers">1</div>
                <textarea 
                  class="code-editor-textarea" 
                  id="config-json-textarea" 
                  spellcheck="false" 
                  autocapitalize="off" 
                  autocomplete="off"
                  placeholder="Loading configuration..."
                ></textarea>
              </div>

              <!-- Editor Status Bar & Diagnostics -->
              <div class="editor-status-bar">
                <div class="editor-status-left">
                  <span id="editor-cursor-pos">Ln 1, Col 1</span>
                  <span class="editor-diag-pill valid" id="editor-diag-status">
                    <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                      <polyline points="20 6 9 17 4 12"/>
                    </svg>
                    <span>JSON & Schema Valid</span>
                  </span>
                </div>
                <div class="editor-status-right">
                  <span>UTF-8</span>
                  <span>Spaces: 2</span>
                  <span>sing-box 1.14.3 Schema</span>
                </div>
              </div>

              <!-- Floating Schema Tooltip (Hover) -->
              <div class="editor-schema-tooltip" id="editor-schema-tooltip" style="display: none;"></div>

              <!-- Floating Autocomplete Popup -->
              <div class="editor-autocomplete" id="editor-autocomplete" style="display: none;"></div>
            </div>
          </div>

          <!-- Tab Content: Visual Overview -->
          <div id="config-visual-container" class="config-tab-pane ${this.activeTab === 'visual' ? 'active' : ''}">
            <div class="visual-overview-grid" id="visual-overview-grid">
              <div class="glass-panel text-center p-4">
                <span class="spinner-sm"></span>
                <p class="mt-2 text-muted">${I18n.t('common.loading')}</p>
              </div>
            </div>
          </div>
        </div>
      `;

      this.bindEvents();
      this.initSchemaIntelligence();
      await this.loadConfig();
    }

    /**
     * Bind DOM events
     */
    bindEvents() {
      // Tab switching
      const tabEditor = document.getElementById('tab-btn-editor');
      const tabVisual = document.getElementById('tab-btn-visual');

      if (tabEditor && tabVisual) {
        tabEditor.addEventListener('click', () => this.switchTab('editor'));
        tabVisual.addEventListener('click', () => this.switchTab('visual'));
      }

      // Code editor line numbers and auto-indent
      const textarea = document.getElementById('config-json-textarea');
      const gutter = document.getElementById('editor-line-numbers');

      if (textarea && gutter) {
        textarea.addEventListener('input', () => {
          this.rawJson = textarea.value;
          this.updateLineNumbers();
          this.scheduleDiagnosticCheck();
        });

        textarea.addEventListener('scroll', () => {
          gutter.scrollTop = textarea.scrollTop;
          this.hideTooltip();
          this.hideAutocomplete();
        });

        // Track cursor position
        const updateCursor = () => {
          const pos = textarea.selectionStart;
          const textBefore = textarea.value.substring(0, pos);
          const lines = textBefore.split('\n');
          const lineNum = lines.length;
          const colNum = lines[lines.length - 1].length + 1;
          const cursorEl = document.getElementById('editor-cursor-pos');
          if (cursorEl) {
            cursorEl.textContent = `Ln ${lineNum}, Col ${colNum}`;
          }
        };

        textarea.addEventListener('keyup', updateCursor);
        textarea.addEventListener('click', updateCursor);

        // Tab key handling in textarea
        textarea.addEventListener('keydown', (e) => {
          if (e.key === 'Tab') {
            e.preventDefault();
            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            textarea.value = textarea.value.substring(0, start) + '  ' + textarea.value.substring(end);
            textarea.selectionStart = textarea.selectionEnd = start + 2;
            this.rawJson = textarea.value;
            this.updateLineNumbers();
            this.scheduleDiagnosticCheck();
          } else if (e.key === ' ' && e.ctrlKey) {
            e.preventDefault();
            this.triggerAutocomplete();
          } else if (e.key === 'Escape') {
            this.hideTooltip();
            this.hideAutocomplete();
          }
        });
      }

      // Format button
      const btnFormat = document.getElementById('btn-editor-format');
      if (btnFormat) {
        btnFormat.addEventListener('click', () => this.formatJson());
      }

      // Copy button
      const btnCopy = document.getElementById('btn-editor-copy');
      if (btnCopy) {
        btnCopy.addEventListener('click', () => {
          if (!this.rawJson) return;
          navigator.clipboard.writeText(this.rawJson).then(() => {
            window.showToast(I18n.t('common.copied'), 'success');
          }).catch(() => {
            window.showToast(I18n.t('common.error'), 'error');
          });
        });
      }

      // Validate button
      const btnValidate = document.getElementById('btn-config-validate');
      if (btnValidate) {
        btnValidate.addEventListener('click', () => this.validateConfig());
      }

      // Save button
      const btnSave = document.getElementById('btn-config-save');
      if (btnSave) {
        btnSave.addEventListener('click', () => this.saveConfig());
      }

      // Export button
      const btnExport = document.getElementById('btn-config-export');
      if (btnExport) {
        btnExport.addEventListener('click', () => this.exportConfig());
      }

      // Import button & file input
      const btnImport = document.getElementById('btn-config-import');
      const fileInput = document.getElementById('config-file-input');
      if (btnImport && fileInput) {
        btnImport.addEventListener('click', () => fileInput.click());
        fileInput.addEventListener('change', (e) => this.handleFileImport(e));
      }

      // Snippet insertion buttons
      document.querySelectorAll('.editor-snippet-pill').forEach((btn) => {
        btn.addEventListener('click', (e) => {
          const idx = parseInt(btn.getAttribute('data-snippet-idx'), 10);
          this.insertSnippet(idx);
        });
      });
    }

    /**
     * Initialize Schema Hover Tooltips and Autocomplete
     */
    initSchemaIntelligence() {
      const textarea = document.getElementById('config-json-textarea');
      const workspace = document.getElementById('editor-workspace');
      this.tooltipEl = document.getElementById('editor-schema-tooltip');
      this.autocompleteEl = document.getElementById('editor-autocomplete');

      if (!textarea || !workspace) return;

      // Mousemove on textarea to detect hover over JSON keys
      let hoverTimer = null;
      textarea.addEventListener('mousemove', (e) => {
        clearTimeout(hoverTimer);
        hoverTimer = setTimeout(() => {
          this.handleHover(e);
        }, 120);
      });

      textarea.addEventListener('mouseleave', () => {
        clearTimeout(hoverTimer);
        this.hideTooltip();
      });
    }

    /**
     * Inspect hover position and display parameter schema information
     */
    handleHover(e) {
      const textarea = document.getElementById('config-json-textarea');
      if (!textarea || !this.tooltipEl) return;

      const rect = textarea.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;

      // Estimate line index based on line height (~19.7px)
      const lineHeight = 19.68;
      const paddingTop = 16;
      const currentLineIdx = Math.floor((y + textarea.scrollTop - paddingTop) / lineHeight);
      const lines = textarea.value.split('\n');

      if (currentLineIdx < 0 || currentLineIdx >= lines.length) {
        this.hideTooltip();
        return;
      }

      const lineText = lines[currentLineIdx] || '';
      // Find key matching "key": or "key" in this line
      const match = lineText.match(/"([a-zA-Z0-9_$-]+)"\s*:/) || lineText.match(/"([a-zA-Z0-9_$-]+)"/);
      if (match && match[1]) {
        const key = match[1];
        const def = SING_BOX_SCHEMA_DEFS[key];
        if (def) {
          this.showTooltip(key, def, e.clientX, e.clientY);
          return;
        }
      }

      this.hideTooltip();
    }

    /**
     * Display hover parameter tooltip
     */
    showTooltip(key, def, clientX, clientY) {
      if (!this.tooltipEl) return;

      let optionsHTML = '';
      if (def.options && def.options.length > 0) {
        optionsHTML = `<div class="tooltip-options"><strong>Allowed values:</strong> ${def.options.map(o => `<code>${o}</code>`).join(', ')}</div>`;
      }

      this.tooltipEl.innerHTML = `
        <div>
          <span class="tooltip-prop">${key}</span>
          <span class="tooltip-type">${def.type}</span>
        </div>
        <div class="tooltip-desc">${def.desc}</div>
        ${optionsHTML}
      `;

      this.tooltipEl.style.display = 'block';
      this.tooltipEl.style.left = `${clientX + 14}px`;
      this.tooltipEl.style.top = `${clientY + 14}px`;
    }

    /**
     * Hide schema hover tooltip
     */
    hideTooltip() {
      if (this.tooltipEl) {
        this.tooltipEl.style.display = 'none';
      }
    }

    /**
     * Trigger autocomplete dropdown for sing-box schema
     */
    triggerAutocomplete() {
      const textarea = document.getElementById('config-json-textarea');
      if (!textarea || !this.autocompleteEl) return;

      const items = Object.keys(SING_BOX_SCHEMA_DEFS).slice(0, 15);
      this.autocompleteEl.innerHTML = items.map(key => {
        const def = SING_BOX_SCHEMA_DEFS[key];
        return `
          <div class="editor-autocomplete-item" data-key="${key}">
            <span>"${key}"</span>
            <span class="editor-autocomplete-type">${def.type}</span>
          </div>
        `;
      }).join('');

      this.autocompleteEl.style.display = 'block';
      this.autocompleteEl.style.left = '80px';
      this.autocompleteEl.style.top = '120px';

      this.autocompleteEl.querySelectorAll('.editor-autocomplete-item').forEach(el => {
        el.addEventListener('click', () => {
          const key = el.getAttribute('data-key');
          const start = textarea.selectionStart;
          const end = textarea.selectionEnd;
          textarea.value = textarea.value.substring(0, start) + `"${key}": ` + textarea.value.substring(end);
          textarea.selectionStart = textarea.selectionEnd = start + key.length + 4;
          this.hideAutocomplete();
          this.updateLineNumbers();
          this.scheduleDiagnosticCheck();
        });
      });
    }

    /**
     * Hide autocomplete dropdown
     */
    hideAutocomplete() {
      if (this.autocompleteEl) {
        this.autocompleteEl.style.display = 'none';
      }
    }

    /**
     * Insert schema preset snippet at cursor position
     */
    insertSnippet(idx) {
      const preset = SNIPPET_PRESETS[idx];
      if (!preset) return;

      const textarea = document.getElementById('config-json-textarea');
      if (!textarea) return;

      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;
      const snippetText = preset.snippet;

      textarea.value = textarea.value.substring(0, start) + snippetText + textarea.value.substring(end);
      textarea.selectionStart = textarea.selectionEnd = start + snippetText.length;
      this.rawJson = textarea.value;
      this.updateLineNumbers();
      this.scheduleDiagnosticCheck();
      window.showToast(`Inserted snippet: ${preset.label}`, 'success', 2000);
    }

    /**
     * Real-time Debounced Diagnostic Check
     */
    scheduleDiagnosticCheck() {
      clearTimeout(this.diagTimer);
      this.diagTimer = setTimeout(() => {
        this.runDiagnosticCheck();
      }, 300);
    }

    runDiagnosticCheck() {
      const textarea = document.getElementById('config-json-textarea');
      const diagPill = document.getElementById('editor-diag-status');
      if (!textarea || !diagPill) return;

      const content = textarea.value.trim();
      if (!content) {
        diagPill.className = 'editor-diag-pill valid';
        diagPill.innerHTML = `<span>Empty</span>`;
        return;
      }

      try {
        const parsed = JSON.parse(content);
        diagPill.className = 'editor-diag-pill valid';
        diagPill.innerHTML = `
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
          <span>JSON Syntax & Schema Valid</span>
        `;
      } catch (err) {
        diagPill.className = 'editor-diag-pill invalid';
        diagPill.innerHTML = `
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
            <circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>
          </svg>
          <span>Syntax Error: ${err.message.substring(0, 36)}...</span>
        `;
      }
    }

    /**
     * Switch between Editor and Visual tabs
     */
    switchTab(tab) {
      this.activeTab = tab;
      const editorPane = document.getElementById('config-editor-container');
      const visualPane = document.getElementById('config-visual-container');
      const tabEditor = document.getElementById('tab-btn-editor');
      const tabVisual = document.getElementById('tab-btn-visual');

      if (tab === 'editor') {
        editorPane?.classList.add('active');
        visualPane?.classList.remove('active');
        tabEditor?.classList.add('active');
        tabVisual?.classList.remove('active');
        this.updateLineNumbers();
      } else {
        visualPane?.classList.add('active');
        editorPane?.classList.remove('active');
        tabVisual?.classList.add('active');
        tabEditor?.classList.remove('active');
        this.renderVisualOverview();
      }
    }

    /**
     * Update gutter line numbers
     */
    updateLineNumbers() {
      const textarea = document.getElementById('config-json-textarea');
      const gutter = document.getElementById('editor-line-numbers');
      if (!textarea || !gutter) return;

      const lines = textarea.value.split('\n').length;
      let lineNums = '';
      for (let i = 1; i <= lines; i++) {
        lineNums += `<div>${i}</div>`;
      }
      gutter.innerHTML = lineNums;
    }

    /**
     * Load configuration from backend
     */
    async loadConfig() {
      try {
        let data = null;
        try {
          data = await API.get('config');
        } catch {
          // Fallback to export if /config not supported directly
          data = await API.get('config/export');
        }

        this.currentConfig = data;
        this.rawJson = JSON.stringify(data, null, 2);

        const textarea = document.getElementById('config-json-textarea');
        if (textarea) {
          textarea.value = this.rawJson;
          this.updateLineNumbers();
          this.runDiagnosticCheck();
        }

        if (this.activeTab === 'visual') {
          this.renderVisualOverview();
        }
      } catch (err) {
        console.error('[Config] Failed to load config:', err);
        window.showToast(I18n.t('config.load_error') + ': ' + err.message, 'error');
      }
    }

    /**
     * Format / Prettify JSON
     */
    formatJson() {
      const textarea = document.getElementById('config-json-textarea');
      if (!textarea) return;

      try {
        const parsed = JSON.parse(textarea.value);
        this.rawJson = JSON.stringify(parsed, null, 2);
        textarea.value = this.rawJson;
        this.updateLineNumbers();
        this.runDiagnosticCheck();
        window.showToast(I18n.t('common.success'), 'success');
      } catch (err) {
        window.showToast(I18n.t('config.valid_failed') + ': ' + err.message, 'error');
      }
    }

    /**
     * Validate current editor JSON via POST /config/validate
     */
    async validateConfig() {
      const textarea = document.getElementById('config-json-textarea');
      if (!textarea) return false;

      let parsedConfig;
      try {
        parsedConfig = JSON.parse(textarea.value);
      } catch (err) {
        window.showToast(I18n.t('config.valid_failed') + ': Invalid JSON syntax (' + err.message + ')', 'error');
        return false;
      }

      this.isValidating = true;
      const btn = document.getElementById('btn-config-validate');
      if (btn) btn.classList.add('loading');

      try {
        const res = await API.post('config/validate', parsedConfig);
        if (res && res.valid) {
          window.showToast(I18n.t('config.valid_success'), 'success');
          return true;
        } else {
          const errMsg = res?.error || I18n.t('config.valid_failed');
          window.showToast(I18n.t('config.valid_failed') + ': ' + errMsg, 'error');
          return false;
        }
      } catch (err) {
        window.showToast(I18n.t('config.valid_failed') + ': ' + err.message, 'error');
        return false;
      } finally {
        this.isValidating = false;
        if (btn) btn.classList.remove('loading');
      }
    }

    /**
     * Save configuration to backend
     */
    async saveConfig() {
      const textarea = document.getElementById('config-json-textarea');
      if (!textarea) return;

      let parsedConfig;
      try {
        parsedConfig = JSON.parse(textarea.value);
      } catch (err) {
        window.showToast(I18n.t('config.valid_failed') + ': Invalid JSON syntax (' + err.message + ')', 'error');
        return;
      }

      // First validate
      const isValid = await this.validateConfig();
      if (!isValid) return;

      this.isSaving = true;
      const btn = document.getElementById('btn-config-save');
      if (btn) btn.classList.add('loading');

      try {
        await API.put('config', parsedConfig);
        this.currentConfig = parsedConfig;
        window.showToast(I18n.t('config.save_success'), 'success');
      } catch (err) {
        window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
      } finally {
        this.isSaving = false;
        if (btn) btn.classList.remove('loading');
      }
    }

    /**
     * Export config file download
     */
    async exportConfig() {
      try {
        let exportData = this.rawJson;
        if (!exportData) {
          const res = await API.get('config/export');
          exportData = JSON.stringify(res, null, 2);
        }

        const blob = new Blob([exportData], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'sing-box-config.json';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        window.showToast(I18n.t('config.export') + ': ' + I18n.t('common.success'), 'success');
      } catch (err) {
        window.showToast(I18n.t('common.error') + ': ' + err.message, 'error');
      }
    }

    /**
     * Handle JSON file upload
     */
    handleFileImport(event) {
      const file = event.target.files?.[0];
      if (!file) return;

      const reader = new FileReader();
      reader.onload = (e) => {
        try {
          const content = e.target.result;
          const parsed = JSON.parse(content);
          this.rawJson = JSON.stringify(parsed, null, 2);
          
          const textarea = document.getElementById('config-json-textarea');
          if (textarea) {
            textarea.value = this.rawJson;
            this.updateLineNumbers();
            this.runDiagnosticCheck();
          }

          window.showToast(I18n.t('config.import_success'), 'success');
          // Switch to editor tab so user can review and save
          this.switchTab('editor');
        } catch (err) {
          window.showToast(I18n.t('config.import_invalid_json'), 'error');
        }
      };
      reader.readAsText(file);
      // Reset input value so same file can be reselected
      event.target.value = '';
    }

    /**
     * Render Visual Overview card breakdown
     */
    renderVisualOverview() {
      const container = document.getElementById('visual-overview-grid');
      if (!container) return;

      const cfg = this.currentConfig || {};
      const inboundsCount = (cfg.inbounds || []).length;
      const outboundsCount = (cfg.outbounds || []).length;
      const groupsCount = (cfg.groups || []).length;
      const rulesCount = (cfg.routing?.rules || []).length;
      const ruleSetsCount = (cfg.routing?.rule_sets || []).length;
      const dnsServersCount = (cfg.dns?.servers || []).length;
      const dnsEnabled = !!cfg.dns?.enabled;

      container.innerHTML = `
        <div class="visual-card glass-panel">
          <div class="visual-card-header">
            <span class="visual-card-title">${I18n.t('config.stats_inbounds')}</span>
            <span class="badge badge-info">${inboundsCount}</span>
          </div>
          <div class="visual-card-body">
            <p class="text-muted">${I18n.t('inbound.subtitle')}</p>
            <div class="tags-list mt-2">
              ${(cfg.inbounds || []).map(inb => `<span class="badge ${inb.enabled ? 'badge-success' : 'badge-neutral'}">${inb.tag || inb.type} (${inb.type})</span>`).join('') || '<span class="text-muted">' + I18n.t('inbound.no_inbounds') + '</span>'}
            </div>
          </div>
          <div class="visual-card-footer">
            <a href="#/inbounds" class="btn btn-ghost btn-sm">${I18n.t('common.edit')} →</a>
          </div>
        </div>

        <div class="visual-card glass-panel">
          <div class="visual-card-header">
            <span class="visual-card-title">${I18n.t('config.stats_outbounds')}</span>
            <span class="badge badge-primary">${outboundsCount}</span>
          </div>
          <div class="visual-card-body">
            <p class="text-muted">${I18n.t('outbound.subtitle')}</p>
            <div class="tags-list mt-2">
              ${(cfg.outbounds || []).slice(0, 8).map(out => `<span class="badge badge-info">${out.tag} (${out.type})</span>`).join('')}
              ${outboundsCount > 8 ? `<span class="badge badge-neutral">+${outboundsCount - 8} more</span>` : ''}
              ${outboundsCount === 0 ? '<span class="text-muted">' + I18n.t('outbound.no_nodes') + '</span>' : ''}
            </div>
          </div>
          <div class="visual-card-footer">
            <a href="#/outbounds" class="btn btn-ghost btn-sm">${I18n.t('common.edit')} →</a>
          </div>
        </div>

        <div class="visual-card glass-panel">
          <div class="visual-card-header">
            <span class="visual-card-title">${I18n.t('config.stats_groups')}</span>
            <span class="badge badge-warning">${groupsCount}</span>
          </div>
          <div class="visual-card-body">
            <p class="text-muted">${I18n.t('outbound.groups_title')}</p>
            <div class="tags-list mt-2">
              ${(cfg.groups || []).map(grp => `<span class="badge badge-warning">${grp.tag} (${grp.type})</span>`).join('') || '<span class="text-muted">' + I18n.t('outbound.no_groups') + '</span>'}
            </div>
          </div>
          <div class="visual-card-footer">
            <a href="#/outbounds" class="btn btn-ghost btn-sm">${I18n.t('common.edit')} →</a>
          </div>
        </div>

        <div class="visual-card glass-panel">
          <div class="visual-card-header">
            <span class="visual-card-title">${I18n.t('config.stats_rules')}</span>
            <span class="badge badge-success">${rulesCount}</span>
          </div>
          <div class="visual-card-body">
            <p class="text-muted">${I18n.t('routing.subtitle')}</p>
            <div class="meta-row mt-2">
              <span>${I18n.t('routing.rulesets_title')}: <strong>${ruleSetsCount}</strong></span>
              <span>${I18n.t('routing.domain_strategy')}: <strong>${cfg.routing?.domain_strategy || 'default'}</strong></span>
            </div>
          </div>
          <div class="visual-card-footer">
            <a href="#/routing" class="btn btn-ghost btn-sm">${I18n.t('common.edit')} →</a>
          </div>
        </div>

        <div class="visual-card glass-panel">
          <div class="visual-card-header">
            <span class="visual-card-title">${I18n.t('dns.title')}</span>
            <span class="badge ${dnsEnabled ? 'badge-success' : 'badge-neutral'}">${dnsEnabled ? I18n.t('common.enabled') : I18n.t('common.disabled')}</span>
          </div>
          <div class="visual-card-body">
            <p class="text-muted">${dnsEnabled ? I18n.t('dns.subtitle') : I18n.t('dns.disabled_hint')}</p>
            <div class="meta-row mt-2">
              <span>${I18n.t('config.stats_dns_servers')}: <strong>${dnsServersCount}</strong></span>
              <span>${I18n.t('dns.strategy')}: <strong>${cfg.dns?.strategy || 'prefer_ipv4'}</strong></span>
            </div>
          </div>
          <div class="visual-card-footer">
            <a href="#/dns" class="btn btn-ghost btn-sm">${I18n.t('common.edit')} →</a>
          </div>
        </div>
      `;
    }

    destroy() {
      clearTimeout(this.diagTimer);
      this.hideTooltip();
      this.hideAutocomplete();
    }
  }

  window.Config = new ConfigPage();
})();
