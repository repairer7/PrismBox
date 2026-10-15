# PrismBox

<p align="center">
  <img src="https://img.shields.io/badge/平台-Linux%20x86__64-blue?style=flat-square" />
  <img src="https://img.shields.io/badge/语言-Go%201.22-00ADD8?style=flat-square&logo=go" />
  <img src="https://img.shields.io/badge/sing--box-1.14.3-purple?style=flat-square" />
  <img src="https://img.shields.io/badge/许可证-MIT-green?style=flat-square" />
  <img src="https://img.shields.io/badge/CGO-禁用-orange?style=flat-square" />
</p>

> **一个高性能、现代毛玻璃风格的 [sing-box](https://github.com/SagerNet/sing-box) WebUI 客户端，打包为 Linux x86_64 单文件静态二进制。**

[English Documentation](README.md)

---

## ✨ 核心特性

### 🎨 毛玻璃 WebUI
- 深色赛博朋克美学，磨砂玻璃效果与环境光晕
- 全响应式设计 —— 适配桌面、平板和手机
- 实时仪表盘，通过 WebSocket 提供实时流量图表
- 国际化支持：中英文一键切换

### 🚀 零依赖单二进制
- 单个 `prismbox` 二进制文件 (~15MB)，内嵌 WebUI
- 纯 Go 编译，零 CGO —— Linux x86_64 静态编译
- 运行时无需 Node.js、npm 或其他构建工具

### 🔧 外部内核管理
- 以外部子进程方式管理官方 `sing-box` 二进制
- 自动检测同目录下的 `./sing-box`，回退检查系统 `$PATH`
- 用户可独立升级 `sing-box`，无需重新编译 PrismBox
- 完整进程生命周期：启动、停止、重启，含崩溃检测和自动恢复

### 📝 配置管理
- 所有 sing-box 配置项的可视化表单编辑
- 内置 JSON 编辑器，支持 Schema 参考
- **重启前配置校验** —— 执行 `sing-box check -c` 捕获错误
- "保存并应用" 按钮，实时错误反馈
- DNS "留空" 选项，避免路由器上的 DNS 回环

### 🌐 代理节点管理
- 导入分享链接：Hysteria2、VLESS（Reality/Vision/gRPC）、VMess、Trojan、Shadowsocks、TUIC v5
- 订阅导入：Base64 和 Clash YAML 格式，支持自动更新
- 节点延迟测速（URL-Test）
- 策略组：Selector（手动选择）& URLTest（自动优选）

### 🔀 landscape-router 流量对接
- 默认 TProxy 入站端口 `12345` —— 开箱即用的透明代理重定向
- 预配置协议嗅探与透明代理标记
- 无缝对接基于 Docker 的软路由系统

---

## 📦 快速开始

### 环境要求
- Linux x86_64 系统
- [sing-box](https://github.com/SagerNet/sing-box/releases) v1.14.x 二进制文件

### 安装

```bash
# 下载最新发布版
wget https://github.com/repairer7/PrismBox/releases/latest/download/prismbox
chmod +x prismbox

# 将 sing-box 二进制放在同目录
wget https://github.com/SagerNet/sing-box/releases/download/v1.14.3/sing-box-1.14.3-linux-amd64.tar.gz
tar xzf sing-box-1.14.3-linux-amd64.tar.gz
cp sing-box-1.14.3-linux-amd64/sing-box .

# 启动 PrismBox
./prismbox
```

打开浏览器访问 **http://127.0.0.1:9090**

### 命令行参数

```bash
./prismbox [参数]

参数:
  --listen    监听地址（默认 ":9090"）
  --data-dir  数据目录（默认 "~/.prismbox"）
  --kernel    sing-box 二进制路径（默认：自动检测）
```

---

## 🏗️ 架构设计

```
┌────────────────────────────────────┐
│         PrismBox 二进制            │
│    (纯 Go, CGO_ENABLED=0)          │
│                                    │
│  ┌─────────┐  ┌─────────────────┐  │
│  │ 内嵌     │  │  REST API       │  │
│  │ WebUI   │──│  + WebSocket    │  │
│  │ (SPA)   │  │  服务器          │  │
│  └─────────┘  └────────┬────────┘  │
│                         │          │
│  ┌──────────────────────┴────────┐ │
│  │    内核管理器                  │ │
│  │  (进程生命周期控制)            │ │
│  └──────────────┬────────────────┘ │
└─────────────────┼──────────────────┘
                  │ exec / signal
                  ▼
        ┌──────────────────┐
        │   sing-box        │
        │ (外部二进制文件)   │
        └──────────────────┘
```

PrismBox 以**外部子进程**方式管理 sing-box —— 而非嵌入式库。这意味着：
- ✅ 随时替换 `sing-box` 二进制文件，无需重新编译
- ✅ 无 CGO 依赖链，无复杂构建环境
- ✅ 通过 `sing-box check -c` 在应用更改前进行配置校验
- ✅ 进程崩溃检测，自动重启并指数退避

---

## 📱 WebUI 模块

| 模块 | 说明 |
|------|------|
| **运行情况** | 实时 CPU、内存、流量速率图表，内核状态、PID、运行时长 |
| **入口方式** | TProxy（默认 12345）、TUN、Mixed 模式配置 |
| **出口节点** | 节点管理、分享链接导入、订阅、测速、策略组 |
| **分流配置** | 嗅探规则、DNS 劫持、远程规则集（.srs/.json）、自定义规则 |
| **DNS 配置** | 启用/禁用自定义 DNS、DoH/DoT/UDP 服务器、DNS 规则 |
| **配置中心** | 可视化表单 + JSON 编辑器，导出/导入，校验 |

---

## 🐳 Docker 部署

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

## 🔨 从源码构建

```bash
# 克隆仓库
git clone https://github.com/repairer7/PrismBox.git
cd PrismBox

# 构建（需要 Go 1.22+）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o prismbox .

# 运行
./prismbox --listen :9090
```

---

## 🌐 支持的分享链接格式

| 协议 | 格式 | 示例 |
|------|------|------|
| Hysteria2 | `hysteria2://密码@主机:端口?sni=...#名称` | `hysteria2://samplePassword%21@198.51.100.1:443?sni=example.com&insecure=1#测试节点` |
| VLESS | `vless://uuid@主机:端口?security=reality&pbk=...#名称` | Reality、Vision、gRPC、WS |
| VMess | `vmess://base64(json)` | 标准 V2Ray 格式 |
| Trojan | `trojan://密码@主机:端口?sni=...#名称` | 支持 WS/gRPC 传输层 |
| Shadowsocks | `ss://base64(加密方式:密码)@主机:端口#名称` | SIP002 和旧版格式 |
| TUIC v5 | `tuic://uuid:密码@主机:端口?cc=bbr#名称` | 支持拥塞控制 |

---

## 📄 开源许可

[MIT 许可证](LICENSE) © PrismBox 贡献者

---

<p align="center">
  <strong>用 ❤️ 为开源社区打造</strong><br/>
  <sub>PrismBox 与 sing-box 项目无官方关联。</sub>
</p>
