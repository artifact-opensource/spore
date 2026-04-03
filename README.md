<div align="center">

# SYMBIOTE: Spore

**Autonomous AI agent in a single binary.**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/artifact-opensource/spore?color=green)](https://github.com/artifact-opensource/spore/releases)
[![Platforms](https://img.shields.io/badge/Platforms-Linux%20%7C%20macOS%20%7C%20Android%20%7C%20Windows-lightgrey)]()

*Runs from a flash drive. Runs on Android. Runs anywhere.*

A portable AI agent runtime — one binary, zero dependencies, full tool loop. Part of the [Symbiote](https://github.com/artifact-opensource/symbiote) family.

---

</div>

## What Is Spore

Spore is a single-binary AI agent that carries its own brain. Plug in a USB drive, run `./spore`, and you have a fully autonomous agent with tool calling, persistent memory, web chat, and Discord integration — on any machine, any OS, any architecture.

**Symbiote** is the full agent runtime (multi-provider gateway, orchestration, channels). **Spore** is Symbiote compressed into one portable executable.

## Features

| Feature | Description |
|---------|-------------|
| **Single binary** | One file. No Python, no Node, no Docker, no dependencies. |
| **Multi-provider** | Ollama, llamafile, OpenAI, Anthropic, Gemini, Grok, OpenRouter, HuggingFace — or any OpenAI-compatible endpoint. |
| **Agentic tool loop** | The agent reasons, calls tools, observes results, and iterates autonomously. |
| **Portable memory** | Persistent file-backed memory travels with the binary on a flash drive. |
| **Web chat** | Built-in HTTP daemon with embedded chat UI. |
| **Discord bot** | Full Discord integration — runs as a bot in your server. |
| **Android native** | MacDroid integration, ADB bridge, device management tools. |
| **Ollama native** | List models, pull models, check status — manages your local Ollama instance. |
| **Network discovery** | Scans local network for LLM servers (Ollama, llamafile). |
| **Shell execution** | Sandboxed shell tool with timeout and output capture. |
| **Cross-platform** | Compiles to Linux x86-64, Linux ARM64, Windows x86-64, macOS. |
| **Zero config** | Works out of the box. Optional config for advanced setups. |

## Quick Start

```bash
# Download the latest release for your platform
curl -LO https://github.com/artifact-opensource/spore/releases/latest/download/spore-linux-amd64
chmod +x spore-linux-amd64

# Interactive chat (auto-discovers local Ollama or llamafile)
./spore-linux-amd64

# Or specify a provider
./spore-linux-amd64 --provider ollama --model llama3.2

# Web chat daemon
./spore-linux-amd64 --mode serve --port 8080

# Discord bot
./spore-linux-amd64 --mode discord --token YOUR_BOT_TOKEN
```

### Build From Source

```bash
git clone https://github.com/artifact-opensource/spore.git
cd spore
go build -o spore -ldflags="-s -w" .
```

### Cross-Compile

```bash
# ARM64 (Android, Raspberry Pi, Apple Silicon)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o spore-arm64 -ldflags="-s -w" .

# Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o spore.exe -ldflags="-s -w" .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o spore-darwin-arm64 -ldflags="-s -w" .
```

## Providers

Spore connects to any LLM backend. Local-first by default — falls back to cloud providers if configured.

| Provider | Type | Default Endpoint |
|----------|------|-----------------|
| **Ollama** | Local | `http://127.0.0.1:11434` |
| **llamafile** | Local | `http://127.0.0.1:8080` |
| **OpenAI** | Cloud | `https://api.openai.com` |
| **Anthropic** | Cloud | `https://api.anthropic.com` |
| **Gemini** | Cloud | `https://generativelanguage.googleapis.com` |
| **Grok** | Cloud | `https://api.x.ai` |
| **OpenRouter** | Cloud | `https://openrouter.ai/api` |
| **HuggingFace** | Cloud | `https://api-inference.huggingface.co` |
| **Custom** | Any | User-defined endpoint |

### Ollama Integration

Spore has native Ollama support beyond the standard OpenAI-compatible chat endpoint:

```bash
# Auto-discovers local Ollama
./spore --provider ollama

# The agent can manage Ollama directly via tools:
# - ollama_list_models  — list available models
# - ollama_pull_model   — download new models
# - ollama_status       — check if Ollama is running
```

## Tools

The agent has access to a full toolbox. Tools are exposed to the LLM as function calls.

### Core

| Tool | Description |
|------|-------------|
| `shell` | Execute shell commands with timeout and output capture |
| `read_file` | Read file contents |
| `write_file` | Write content to a file |
| `list_dir` | List directory contents with metadata |
| `search_files` | Search files by name pattern |
| `http_request` | Make HTTP requests |
| `notify` | Send system notifications |
| `brightness` | Control screen brightness |

### Memory

| Tool | Description |
|------|-------------|
| `memory_store` | Store a key-value pair in persistent memory |
| `memory_recall` | Recall a value by key |
| `memory_search` | Search memory by semantic query |
| `memory_list` | List all memory keys |
| `memory_forget` | Remove a key from memory |

### Ollama

| Tool | Description |
|------|-------------|
| `ollama_list_models` | List all models in the local Ollama instance |
| `ollama_pull_model` | Download a model from the Ollama registry |
| `ollama_status` | Check Ollama availability and model count |

### Android / MacDroid

| Tool | Description |
|------|-------------|
| `device_list` | List connected Android devices |
| `device_info` | Get device details (model, OS, storage) |
| `file_push` | Push files to Android device |
| `file_pull` | Pull files from Android device |
| `file_list` | List files on Android device |
| `app_list` | List installed apps |
| `app_install` | Install APK on device |
| `screenshot` | Capture device screenshot |
| `screen_record` | Record device screen |

### Network

| Tool | Description |
|------|-------------|
| `network_scan` | Scan local network for LLM servers |
| `network_info` | Get local network interface info |

## Modes

| Mode | Flag | Description |
|------|------|-------------|
| **Chat** | `--mode chat` (default) | Interactive terminal chat |
| **Serve** | `--mode serve` | HTTP daemon with embedded web chat UI |
| **Discord** | `--mode discord` | Discord bot |
| **Daemon** | `--mode daemon` | Background daemon (all channels) |

## Architecture

```
┌──────────────────────────────────────────────────┐
│                    main.go                        │
│          CLI, flags, mode routing                 │
├──────────┬───────────┬───────────┬───────────────┤
│  core/   │ provider/ │  tools/   │   daemon/     │
│  agent   │  multi-   │  shell    │   HTTP +      │
│  loop    │  provider │  files    │   webchat     │
│  config  │  Ollama   │  android  │               │
│  session │  cloud    │  network  │               │
├──────────┤  local    │  memory   ├───────────────┤
│ memory/  │           │  ollama   │  discord/     │
│ persist  │           │           │  bot adapter  │
├──────────┤           │           ├───────────────┤
│ network/ │           │           │  shell/       │
│ scanner  │           │           │  sandboxed    │
└──────────┴───────────┴───────────┴───────────────┘
```

## Configuration

Spore works with zero config. For advanced setups, create `~/.spore/workspace/config.json`:

```json
{
  "provider": "ollama",
  "model": "llama3.2",
  "base_url": "http://127.0.0.1:11434",
  "api_key": "",
  "system_prompt": "You are a helpful assistant.",
  "max_iterations": 25,
  "temperature": 0.7,
  "discord_token": "",
  "serve_port": 8080
}
```

All fields are optional. CLI flags override config file values.

## Flash Drive Setup

The whole point of Spore — carry your AI agent on a USB stick:

```
USB Drive/
├── spore-linux-amd64      # Linux binary
├── spore-darwin-arm64     # macOS binary
├── spore.exe              # Windows binary
├── .spore/
│   └── workspace/
│       ├── config.json    # Your config
│       └── memory.json    # Persistent memory (travels with you)
└── models/                # Optional: carry llamafile models
    └── llama-3.2-3b.gguf
```

Plug in anywhere. Run. Your agent, your memory, your config — portable.

## The Symbiote Family

| Project | Description |
|---------|-------------|
| [**Symbiote**](https://github.com/artifact-opensource/symbiote) | Full agent runtime — multi-provider gateway, orchestration, channels |
| [**Spore**](https://github.com/artifact-opensource/spore) | Portable single-binary agent — this project |
| [**Singularity**](https://github.com/artifact-opensource/singularity) | Enterprise orchestration — multi-agent, C-Suite dispatch, self-optimization |

## License

Apache License 2.0 — see [LICENSE](LICENSE).

---

<div align="center">

Built by [Artifact Virtual](https://artifactvirtual.com)

**One binary. Zero dependencies. Full autonomy.**

</div>
