# Providers

Spore supports multiple LLM providers. Local providers are preferred by default.

## Local Providers

### Ollama

The recommended local provider. Spore has native Ollama integration.

```bash
# Start with Ollama (auto-detected if running)
./spore --provider ollama --model llama3.2

# The agent can manage Ollama via built-in tools:
# - ollama_list_models  → list downloaded models
# - ollama_pull_model   → download new models
# - ollama_status       → check server health
```

**Setup:**
1. Install Ollama: https://ollama.ai
2. Pull a model: `ollama pull llama3.2`
3. Run Spore: `./spore --provider ollama`

**Native API support:**
- Chat completion via OpenAI-compatible `/v1/chat/completions`
- Model listing via `/api/tags`
- Model pulling via `/api/pull`
- Health check via `/api/tags` (connectivity test)

### llamafile

Single-file LLM server from Mozilla.

```bash
# Start llamafile (separate terminal)
./llama-3.2-3b.llamafile --server --port 8080

# Connect Spore
./spore --provider llamafile --base-url http://127.0.0.1:8080
```

## Cloud Providers

All cloud providers require an API key, set via `--api-key` flag or config file.

### OpenAI

```bash
./spore --provider openai --model gpt-4o --api-key sk-...
```

### Anthropic

```bash
./spore --provider anthropic --model claude-sonnet-4-20250514 --api-key sk-ant-...
```

Uses the native Anthropic `/v1/messages` endpoint (not OpenAI-compatible).

### Google Gemini

```bash
./spore --provider gemini --model gemini-2.0-flash --api-key AIza...
```

### Grok (xAI)

```bash
./spore --provider grok --model grok-3 --api-key xai-...
```

### OpenRouter

Access multiple models through one API key.

```bash
./spore --provider openrouter --model meta-llama/llama-3.2-90b-vision-instruct --api-key sk-or-...
```

### HuggingFace Inference API

```bash
./spore --provider huggingface --model meta-llama/Llama-3.2-3B-Instruct --api-key hf_...
```

### Custom Endpoint

Any OpenAI-compatible API:

```bash
./spore --provider custom --base-url http://my-server:8000 --model my-model
```

## Provider Selection Priority

When no provider is specified:

1. Check for running Ollama at `127.0.0.1:11434`
2. Check for running llamafile at `127.0.0.1:8080`
3. Check config file for default provider
4. Prompt user to configure

## Adding New Providers

Providers are defined in `provider/provider.go` in the `defaultProviders` slice. Any provider with an OpenAI-compatible chat completions API can be added by appending to this list.
