# PrismBox

<p align="center">
  <img src="https://img.shields.io/badge/Platform-Linux%20x86__64-blue?style=flat-square" />
  <img src="https://img.shields.io/badge/Language-Go%201.22-00ADD8?style=flat-square&logo=go" />
  <img src="https://img.shields.io/badge/sing--box-1.14.3-purple?style=flat-square" />
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" />
  <img src="https://img.shields.io/badge/CGO-Disabled-orange?style=flat-square" />
</p>

> **A high-performance, modern Glassmorphism WebUI client for [sing-box](https://github.com/SagerNet/sing-box), packaged as a single static binary for Linux x86_64.**

[中文文档](README_zh.md)

---

## ✨ Features

### 🎨 Glassmorphism WebUI
- Dark cyberpunk aesthetic with frosted glass effects and ambient glow
- Fully responsive — works on desktop, tablet, and mobile
- Real-time dashboard with live traffic charts via WebSocket
- i18n support: English & 中文 one-click switching

### 🚀 Zero-Dependency Binary
- Single `prismbox` binary (~15MB) with embedded WebUI
- Pure Go, zero CGO — static compilation for Linux x86_64
- No Node.js, no npm, no build tools required at runtime

### 🔧 External Kernel Management
- Manages the official `sing-box` binary as an external child process
- Auto-detects `./sing-box` in the same directory, falls back to `$PATH`
- Users can independently upgrade `sing-box` without rebuilding PrismBox
- Full process lifecycle: start, stop, restart with crash detection & auto-recovery

### 📝 Configuration Management
- Visual form-based editing for all sing-box configuration sections
- Built-in JSON editor with schema reference
- **Config validation before restart** — runs `sing-box check -c` to catch errors
- "Save & Apply" button with real-time error feedback
- DNS "leave empty" option to avoid DNS loops on routers

### 🌐 Proxy Node Management
- Import share links: Hysteria2, VLESS (Reality/Vision/gRPC), VMess, Trojan, Shadowsocks, TUIC v5
- Subscription import: Base64 & Clash YAML format with auto-update
- Node speed testing (URL-Test latency)
- Strategy groups: Selector (manual) & URLTest (auto)

### 🔀 landscape-router Flow Integration
- Default TProxy inbound on port `12345` — ready for transparent proxy redirection
- Pre-configured sniffing and transparent proxy marks
- Seamless integration with Docker-based router setups

---

## 📦 Quick Start

### Prerequisites
- Linux x86_64 system
- [sing-box](https://github.com/SagerNet/sing-box/releases) v1.14.x binary

### Install

```bash
# Download the latest release
wget https://github.com/repairer7/PrismBox/releases/latest/download/prismbox
chmod +x prismbox

# Place sing-box binary in the same directory
wget https://github.com/SagerNet/sing-box/releases/download/v1.14.3/sing-box-1.14.3-linux-amd64.tar.gz
tar xzf sing-box-1.14.3-linux-amd64.tar.gz
cp sing-box-1.14.3-linux-amd64/sing-box .

# Start PrismBox
./prismbox
```

Open your browser at **http://127.0.0.1:9090**

### Command Line Options

```bash
./prismbox [flags]

Flags:
  --listen    Address to listen on (default ":9090")
  --data-dir  Data directory for configs (default "~/.prismbox")
  --kernel    Path to sing-box binary (default: auto-detect)
```

---

## 🏗️ Architecture

```
┌────────────────────────────────────┐
│         PrismBox Binary            │
│    (Pure Go, CGO_ENABLED=0)        │
│                                    │
│  ┌─────────┐  ┌─────────────────┐  │
│  │ Embedded │  │  REST API       │  │
│  │ WebUI    │──│  + WebSocket    │  │
│  │ (SPA)    │  │  Server         │  │
│  └─────────┘  └────────┬────────┘  │
│                         │          │
│  ┌──────────────────────┴────────┐ │
│  │    Kernel Manager             │ │
│  │  (Process Lifecycle Control)  │ │
│  └──────────────┬────────────────┘ │
└─────────────────┼──────────────────┘
                  │ exec / signal
                  ▼
        ┌──────────────────┐
        │   sing-box        │
        │ (External Binary) │
        └──────────────────┘
```

PrismBox manages sing-box as an **external child process** — not as an embedded library. This means:
- ✅ Replace `sing-box` binary at any time without recompiling
- ✅ No CGO dependency chain, no complex build environment
- ✅ Config validation via `sing-box check -c` before applying changes
- ✅ Process crash detection with automatic restart and backoff

---

## 📱 WebUI Modules

| Module | Description |
|--------|-------------|
| **Dashboard** | Real-time CPU, memory, traffic speed charts, kernel status, PID, uptime |
| **Inbounds** | TProxy (default 12345), TUN, Mixed mode configuration |
| **Outbounds** | Node management, share link import, subscription, speed test, groups |
| **Routing** | Sniff rules, DNS hijack, remote rule sets (.srs/.json), custom rules |
| **DNS** | Enable/disable custom DNS, DoH/DoT/UDP servers, DNS rules |
| **Config** | Visual form + JSON editor, export/import, validation |

---

## 🐳 Docker

```yaml
version: '3.8'
services:
  prismbox:
    image: ghcr.io/repairer7/prismbox:latest
    container_name: prismbox
    restart: unless-stopped
    network_mode: host
    cap_add:
      - NET_ADMIN
      - NET_RAW
    volumes:
      - ./data:/root/.prismbox
      - ./sing-box:/usr/local/bin/sing-box
    ports:
      - "9090:9090"
      - "12345:12345"
```

---

## 🔨 Building from Source

```bash
# Clone
git clone https://github.com/repairer7/PrismBox.git
cd PrismBox

# Build (requires Go 1.22+)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o prismbox .

# Run
./prismbox --listen :9090
```

---

## 🌐 Share Link Formats Supported

| Protocol | Format | Example |
|----------|--------|---------|
| Hysteria2 | `hysteria2://pass@host:port?sni=...#name` | `hysteria2://samplePassword%21@198.51.100.1:443?sni=example.com&insecure=1#Sample-Node` |
| VLESS | `vless://uuid@host:port?security=reality&pbk=...#name` | Reality, Vision, gRPC, WS |
| VMess | `vmess://base64(json)` | Standard V2Ray format |
| Trojan | `trojan://pass@host:port?sni=...#name` | With WS/gRPC transport |
| Shadowsocks | `ss://base64(method:pass)@host:port#name` | SIP002 & Legacy |
| TUIC v5 | `tuic://uuid:pass@host:port?cc=bbr#name` | With congestion control |

---

## 📄 License

[MIT License](LICENSE) © PrismBox Contributors

---

<p align="center">
  <strong>Built with ❤️ for the open-source community</strong><br/>
  <sub>PrismBox is not affiliated with the sing-box project.</sub>
</p>