# Spore

Autonomous agent runtime. Single binary, multi-platform. Runs on anything with a shell.

Spore is the deployment unit of [Artifact Virtual](https://artifactvirtual.com) — a portable, self-contained AI agent that provides full system control through natural language. One binary. No dependencies. No cloud required.

## Platforms

| Platform | Architecture | Binary | Status |
|----------|-------------|--------|--------|
| Android (Termux) | ARM64 | `spore-arm64` | ✅ Production |
| Windows | x86_64 | `spore-windows-amd64.exe` | ✅ Production |
| Linux | x86_64 | `spore-linux-amd64` | ✅ Production |
| Xbox Dev Mode | x86_64 | `spore-windows-amd64.exe` | ✅ Production |

## Quick Start

```bash
# Download the binary for your platform
chmod +x spore-*

# First-time setup (creates config)
./spore setup

# For Xbox Dev Mode:
./spore setup --profile xbox

# Start everything (webchat + discord bot)
./spore start

# Or run a single command
./spore run "what's my GPU temperature?"

# Interactive chat
./spore chat
```

## Features

### Core Agent
- **Agentic loop** — tool-calling AI that executes actions, not descriptions
- **Webchat UI** — built-in HTTP server with session management
- **Discord bot** — auto-connects if token configured
- **Memory** — BM25 search over indexed files
- **Sessions** — persistent conversation history

### Platform Tools

**Universal (all platforms):**
- `exec` — shell command execution
- `read` / `write` / `edit` — file operations
- `search` — semantic memory search
- `web_fetch` — HTTP client
- `processes` / `kill_process` — process management

**Android (Termux):**
- `app_launch` / `app_stop` — control any installed app
- `brightness` / `volume` / `torch` — device hardware
- `camera_photo` — take photos
- `sms_send` / `sms_inbox` / `call` — telephony
- `clipboard_get` / `clipboard_set` — clipboard
- `tts_speak` / `toast` / `notify` — notifications
- `macro_fire` — MacroDroid integration
- `adb_connect` — wireless ADB auto-detection
- `location` / `wifi_info` / `battery` — sensors

**Windows / Xbox / Linux:**
- `gpu_status` — GPU temp, VRAM, utilization (nvidia-smi / dxdiag / WMI)
- `service_manager` — list/start/stop/kill processes (tasklist/taskkill or ps/kill)
- `network_info` — interfaces, connections, ports, DNS, ARP scan
- `system_info` — CPU, RAM, disk, GPU, OS (WMI/PowerShell or /proc)
- `file_server` — serve directory over HTTP for file transfer

### Providers

| Provider | Config |
|----------|--------|
| GitHub Copilot | `spore config provider copilot` (built-in proxy, free) |
| Ollama | `spore config provider ollama` |
| OpenAI | `spore config provider openai` |
| Anthropic | `spore config provider anthropic` |
| Local/llamafile | `spore config provider local` |
| Custom | `spore config provider custom` + `spore config base_url <url>` |

## Xbox Deployment Guide

### Prerequisites
1. Xbox in **Dev Mode** (requires Xbox Dev Mode Activation app, $20 one-time)
2. **Windows Device Portal** enabled (Settings → Dev Home → Remote Access)
3. An LLM running locally — either:
   - **Ollama** on another machine on the network
   - **llamafile** running directly on Xbox via Dev Mode shell

### Deployment Steps

```bash
# 1. Build the Windows binary (or download release)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o spore-windows-amd64.exe .

# 2. Upload to Xbox via Device Portal (https://<xbox-ip>:11443)
#    Navigate to File Explorer → LocalAppData → upload spore-windows-amd64.exe

# 3. SSH into Xbox Dev Mode shell
ssh devuser@<xbox-ip>

# 4. Setup with Xbox profile
spore-windows-amd64.exe setup --profile xbox

# 5. Configure LLM endpoint (if Ollama on another machine)
spore-windows-amd64.exe config base_url http://<ollama-host>:11434
spore-windows-amd64.exe config model qwen3.5:9b

# 6. Start Spore
spore-windows-amd64.exe start
```

### Xbox-Specific Config
The Xbox profile (`--profile xbox`) pre-configures:
- Provider: `local` (OpenAI-compatible endpoint)
- Model: `qwen3.5:9b` (fits in Xbox's ~10GB available RAM)
- Base URL: `http://127.0.0.1:8080/v1` (adjust if LLM is remote)
- System prompt: Xbox-aware (GPU tools, process management)

### What Works on Xbox
- ✅ GPU monitoring (dxdiag, WMI via PowerShell)
- ✅ Process management (tasklist/taskkill)
- ✅ Network tools (ipconfig, netstat)
- ✅ File serving (HTTP server for transfers)
- ✅ System info (systeminfo, WMI)
- ✅ Webchat UI (accessible from any device on LAN)
- ✅ Discord bot (if internet available)

## Building

```bash
# All platforms
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o spore-windows-amd64.exe .
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o spore-linux-amd64 .
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o spore-arm64 .
```

Zero CGO. No external dependencies. Single static binary per platform.

## Architecture

```
main.go              — CLI entry point, command routing
core/agent.go        — Agentic loop (tool calling, history, sessions)
core/config.go       — Config management, platform profiles
core/session.go      — Session persistence
tools/tools.go       — Universal tools (exec, read, write, search, web)
tools/android.go     — Android/Termux tools (apps, device control, sensors)
tools/xbox.go        — Xbox/Windows/Linux tools (GPU, services, network, sysinfo)
tools/proc_unix.go   — Unix process group management
tools/proc_windows.go — Windows process management
daemon/daemon.go     — Background daemon management
daemon/webchat.go    — Built-in webchat UI
discord/discord.go   — Discord bot (Gateway v10)
copilot/copilot.go   — GitHub Copilot proxy
memory/memory.go     — BM25 search index
network/network.go   — Tunnels, SOCKS proxy, network scan
shell/shell.go       — Interactive REPL
provider/provider.go — LLM provider abstraction
```

## License

Artifact Virtual © 2026. All rights reserved.
