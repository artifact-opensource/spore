# Architecture

Spore is organized into 8 packages, all compiled into a single binary.

## Package Map

```
spore/
├── main.go           # Entry point — CLI parsing, mode routing
├── core/             # Agent brain
│   ├── agent.go      # Agentic loop — reason → tool call → observe → iterate
│   ├── config.go     # Configuration loading and defaults
│   └── session.go    # Conversation session management
├── provider/         # LLM providers
│   └── provider.go   # Multi-provider abstraction + Ollama native API
├── tools/            # Tool registry
│   ├── tools.go      # Core tools — shell, files, HTTP, memory, Ollama
│   └── android.go    # Android/MacDroid tools — ADB, file transfer
├── daemon/           # HTTP server
│   ├── daemon.go     # HTTP daemon with API endpoints
│   └── webchat.go    # Embedded web chat UI (HTML/JS/CSS)
├── discord/          # Discord adapter
│   └── discord.go    # Discord bot — gateway, message handling
├── memory/           # Persistent memory
│   └── memory.go     # File-backed key-value store with search
├── network/          # Network utilities
│   └── network.go    # Local network scanning for LLM servers
└── shell/            # Shell execution
    └── shell.go      # Sandboxed command execution with timeout
```

## Data Flow

```
User Input (terminal / web / Discord)
    │
    ▼
┌─────────────────────────┐
│    core/agent.go        │
│    Agent Loop            │
│                         │
│  1. Build messages      │
│  2. Call LLM provider   │◄──── provider/provider.go
│  3. Parse tool calls    │
│  4. Execute tools       │◄──── tools/tools.go
│  5. Append results      │
│  6. Repeat until done   │
│  7. Return final answer │
└─────────────────────────┘
    │
    ▼
Response (terminal / web / Discord)
```

## Agent Loop

The agent runs a ReAct-style loop:

1. **System prompt** + conversation history → LLM
2. LLM responds with either:
   - **Text** → return to user
   - **Tool call(s)** → execute, append results, go to step 1
3. Loop continues until max iterations or LLM returns text-only

Tool calls are JSON-encoded in the LLM response and parsed by the agent.

## Provider Chain

Providers are tried in order:
1. **Local first** — Ollama, llamafile (zero-latency, free, private)
2. **Cloud fallback** — OpenAI, Anthropic, etc. (requires API key)

All providers use the OpenAI chat completions format. Anthropic uses its native `/v1/messages` endpoint. Ollama additionally exposes native APIs for model management.

## Memory Model

Memory is file-backed JSON (`~/.spore/workspace/memory.json`). It travels with the binary when on a flash drive.

- **Store/Recall** — key-value pairs
- **Search** — substring matching across all stored values
- **Persistence** — survives restarts, portable across machines

## Build Targets

| Target | GOOS | GOARCH | CGO | Typical Size |
|--------|------|--------|-----|-------------|
| Linux x86-64 | linux | amd64 | disabled | ~5.5 MB |
| Linux ARM64 | linux | arm64 | disabled | ~5.4 MB |
| Windows | windows | amd64 | disabled | ~5.9 MB |
| macOS Intel | darwin | amd64 | disabled | ~5.7 MB |
| macOS ARM | darwin | arm64 | disabled | ~5.5 MB |

All builds are static (CGO_ENABLED=0) with stripped symbols (-ldflags="-s -w").
