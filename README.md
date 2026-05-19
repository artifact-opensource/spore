# 🧬 Spore

**Single-binary autonomous agent runtime. Runs on anything with a pulse.**

Spore is the deployment unit of [Artifact Virtual](https://artifactvirtual.com) — a portable, self-contained AI agent that colonizes any device through a single executable. No dependencies. No containers. No cloud. One binary lands, the host comes alive.


## What Is This

Spore is an **agentic runtime** — an AI that doesn't just talk, it *does*. Give it a shell, give it an LLM endpoint, and it will manage the machine it lives on. Files, processes, network, hardware, sensors — everything the host offers becomes a tool.

It spreads across architectures:

| Platform | Binary | Size |
|----------|--------|------|
| Linux x86_64 | `spore-linux-amd64` | 8.4 MB |
| Windows x86_64 | `spore-windows-amd64.exe` | 8.6 MB |
| Android ARM64 | `spore-android-arm64` | 8.1 MB |
| Android x86_64 | `spore-android-x86_64` | 8.3 MB |

Zero CGO. Static linking. Cross-compiled from pure Go.

## Quick Start

```bash
# Download binary for your platform from releases/
chmod +x spore-*

# First run — creates config, picks provider
./spore setup

# Start the daemon (webchat UI + Discord bot)
./spore start

# Or just run a command
./spore run "show me disk usage and kill any zombie processes"
```

## Architecture

Spore has two generations living in this repo:

```
agent/     — Gen 1: Terminal agent (REPL, provider abstraction, tools)
android/   — Gen 2: Full runtime (webchat, Discord, sessions, multi-platform tools)
releases/  — Pre-built binaries
```

Gen 2 (`android/`) is the production runtime despite the directory name — it runs on Android, Windows, Linux, and Xbox. The name is vestigial from its origin on a Samsung Z Fold 5.

### Core Components

```
core/agent.go        — Agentic loop: observe → think → act → repeat
core/session.go      — Persistent conversation memory
tools/tools.go       — Universal: exec, read, write, search, web_fetch
tools/android.go     — Android: apps, camera, SMS, sensors, ADB
tools/xbox.go        — Xbox/Windows/Linux: GPU, services, network, sysinfo
daemon/webchat.go    — Built-in web UI with session management
discord/discord.go   — Discord Gateway v10 bot
copilot/copilot.go   — Embedded GitHub Copilot proxy (free LLM access)
memory/memory.go     — BM25 search over indexed files
```

## LLM Providers

Spore is model-agnostic. It speaks OpenAI-compatible API to any backend:

- **GitHub Copilot** — built-in proxy, free tier (default)
- **Ollama** — local models, any size
- **OpenAI / Anthropic / Custom** — any endpoint that speaks `/v1/chat/completions`
- **Artifact Engine** — our own Vulkan inference engine (coming soon)

## The Vision

Every device is a potential host. A phone in a drawer. A gaming console. A server in a closet. A Raspberry Pi on a shelf. Spore doesn't need much — a binary, a network connection, and something to think with. It lands, it roots, it serves.

Part of the **Symbiote** family:
- **Mach6** — the nervous system (orchestrator)
- **Spore** — the seed (device agent)
- **OSymbiote** — the organism (bootable agent OS)

## Building

```bash
cd android/

# All platforms at once
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -o ../releases/spore-linux-amd64 .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o ../releases/spore-windows-amd64.exe .
GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -o ../releases/spore-android-arm64 .
```

## License

[Apache 2.0](LICENSE)

---

*Artifact Virtual © 2026*
