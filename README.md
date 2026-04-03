# Symbiote for Android

Portable agentic runtime for Android via Termux. Single binary, zero dependencies, full shell control.

## What It Does

- **Full agentic AI runtime** on your phone — tool use, shell commands, file ops, memory search
- **Background daemon** with auto-start on boot via Termux:Boot
- **HTTP API server** for remote control from any device on the network
- **SSH tunnels** — forward and reverse, connect back to Dragonfly
- **Network scanning** — nmap when available, pure Go fallback
- **SOCKS5 proxy** — route traffic through the phone
- **BM25 memory search** — index and search files locally
- **Process management** — spawn, monitor, kill background processes
- **Android integration** — battery, notifications, storage via Termux:API

## Install

```bash
# In Termux:
bash install.sh
```

## Quick Start

```bash
# Interactive chat
symbiote

# Single-shot
symbiote run "list all files in my home directory"

# Background daemon
symbiote daemon start

# HTTP API
symbiote serve 8422

# Connect to Dragonfly
symbiote config provider copilot
symbiote config base_url http://192.168.1.13:3000
symbiote config model gpt-4o
```

## Architecture

```
symbiote (5.7 MB ARM64 binary)
├── core/       agent loop + config
├── provider/   OpenAI, Anthropic, Copilot, local, custom
├── tools/      exec, read, write, edit, search, web_fetch, processes, notify
├── memory/     BM25 full-text search + document index
├── daemon/     background service + HTTP API
├── network/    SSH tunnels, port scanning, SOCKS5 proxy
└── shell/      interactive terminal + shell passthrough
```

## Binary Targets

| Target | File | Size |
|--------|------|------|
| Android ARM64 (phone/tablet) | `symbiote-android-arm64` | 5.7 MB |
| Android x86_64 (emulator) | `symbiote-android-x86_64` | 5.8 MB |
| Linux ARM64 (Chromebook) | `symbiote-linux-arm64` | 5.2 MB |

## Data Layout

```
~/.symbiote/
├── config.json     provider, model, device settings
├── memory/
│   └── index.json  BM25 search index
├── logs/
│   └── daemon.log
└── processes/
```

## Tools (12)

| Tool | Description |
|------|-------------|
| `exec` | Run any shell command (full Termux access) |
| `read` | Read files with offset/limit |
| `write` | Write files, auto-create dirs |
| `edit` | Surgical find-and-replace |
| `search` | BM25 memory search |
| `web_fetch` | HTTP GET with size limits |
| `list` | Directory listing |
| `processes` | List running processes |
| `kill_process` | Kill by PID or name |
| `env` | Show environment |
| `device_info` | Battery, storage, network, IP |
| `notify` | Android notification via Termux:API |

## Network Features

```bash
# Forward tunnel (access remote service locally)
symbiote tunnel 8080:80:example.com

# Reverse tunnel (let Dragonfly reach your phone)
symbiote tunnel reverse 9422:8422:adam@192.168.1.13

# Network scan
symbiote scan 192.168.1.0/24
symbiote scan 192.168.1.13

# SOCKS5 proxy
symbiote proxy 1080
```

## Providers

| Provider | Config |
|----------|--------|
| Local (llamafile) | `symbiote config provider local` |
| OpenAI | `symbiote config provider openai` + api_key |
| Anthropic | `symbiote config provider anthropic` + api_key |
| Copilot proxy | `symbiote config provider copilot` |
| Any OpenAI-compatible | `symbiote config provider custom` + base_url |

## Requirements

- Android device with [Termux](https://f-droid.org/packages/com.termux/)
- Optional: [Termux:Boot](https://f-droid.org/packages/com.termux.boot/) for auto-start
- Optional: [Termux:API](https://f-droid.org/packages/com.termux.api/) for notifications/device info

## Part of Artifact Virtual

Built by AVA. Sibling to [Spore](../agent/) (desktop) and [Mach6](https://github.com/artifact-opensource/symbiote) (enterprise).
