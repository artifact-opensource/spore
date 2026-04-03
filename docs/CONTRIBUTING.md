# Contributing

## Development

```bash
# Clone
git clone https://github.com/artifact-opensource/spore.git
cd spore

# Build
go build -o spore .

# Vet
go vet ./...

# Run
./spore --provider ollama --model llama3.2
```

## Project Structure

```
spore/
├── main.go            # CLI entry point
├── core/              # Agent loop, config, sessions
├── provider/          # LLM provider abstraction
├── tools/             # Tool definitions and execution
├── daemon/            # HTTP server + web chat
├── discord/           # Discord bot adapter
├── memory/            # Persistent memory backend
├── network/           # Network scanning
├── shell/             # Shell execution
└── docs/              # Documentation
```

## Adding a Tool

1. Define the tool schema in `tools/tools.go` (or a new file in `tools/`)
2. Add the execution case in `core/agent.go` `executeTool()` switch
3. Run `go vet ./...` to verify

## Adding a Provider

1. Add entry to `defaultProviders` in `provider/provider.go`
2. If the provider uses a non-OpenAI protocol, add a case in `Complete()`
3. Run `go vet ./...` to verify

## Releases

Releases are tagged with semver (e.g. `v0.1.0`). Binaries for all platforms are attached to GitHub releases.

## License

Apache 2.0. By contributing, you agree that your contributions will be licensed under the same license.
