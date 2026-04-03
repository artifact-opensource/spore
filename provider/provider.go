package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ── Types ────────────────────────────────────────────────────────────────

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"function_name"`
	Arguments string `json:"function_arguments"`
}

type ToolDef struct {
	Type     string          `json:"type"`
	Function ToolDefFunction `json:"function"`
}

type ToolDefFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type Response struct {
	Content   string
	ToolCalls []ToolCall
}

type ProviderConfig struct {
	Provider    string
	Model       string
	BaseURL     string
	APIKey      string
	MaxTokens   int
	Temperature float64
}

// ── Provider Registry ────────────────────────────────────────────────────

// ProviderInfo describes a supported LLM provider
type ProviderInfo struct {
	ID          string
	Name        string
	BaseURL     string // default API base URL
	AuthHeader  string // "Authorization" or "x-api-key" etc
	AuthPrefix  string // "Bearer " or ""
	Models      []string
	KeyEnvVar   string // env var name for the API key
	RequiresKey bool
	Protocol    string // "openai" or "anthropic"
}

var Registry = []ProviderInfo{
	{
		ID:          "openai",
		Name:        "OpenAI",
		BaseURL:     "https://api.openai.com",
		AuthHeader:  "Authorization",
		AuthPrefix:  "Bearer ",
		Models:      []string{"gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano", "o4-mini", "o3-mini"},
		KeyEnvVar:   "OPENAI_API_KEY",
		RequiresKey: true,
		Protocol:    "openai",
	},
	{
		ID:          "anthropic",
		Name:        "Anthropic",
		BaseURL:     "https://api.anthropic.com",
		AuthHeader:  "x-api-key",
		AuthPrefix:  "",
		Models:      []string{"claude-sonnet-4-20250514", "claude-3-5-haiku-20241022", "claude-3-5-sonnet-20241022"},
		KeyEnvVar:   "ANTHROPIC_API_KEY",
		RequiresKey: true,
		Protocol:    "anthropic",
	},
	{
		ID:          "gemini",
		Name:        "Google Gemini",
		BaseURL:     "https://generativelanguage.googleapis.com/v1beta/openai",
		AuthHeader:  "Authorization",
		AuthPrefix:  "Bearer ",
		Models:      []string{"gemini-2.5-flash-preview-04-17", "gemini-2.0-flash", "gemini-2.5-pro-preview-03-25"},
		KeyEnvVar:   "GEMINI_API_KEY",
		RequiresKey: true,
		Protocol:    "openai",
	},
	{
		ID:          "grok",
		Name:        "xAI Grok",
		BaseURL:     "https://api.x.ai",
		AuthHeader:  "Authorization",
		AuthPrefix:  "Bearer ",
		Models:      []string{"grok-3", "grok-3-mini", "grok-2"},
		KeyEnvVar:   "XAI_API_KEY",
		RequiresKey: true,
		Protocol:    "openai",
	},
	{
		ID:          "openrouter",
		Name:        "OpenRouter",
		BaseURL:     "https://openrouter.ai/api",
		AuthHeader:  "Authorization",
		AuthPrefix:  "Bearer ",
		Models:      []string{"anthropic/claude-sonnet-4", "openai/gpt-4o", "google/gemini-2.5-flash-preview", "meta-llama/llama-4-maverick"},
		KeyEnvVar:   "OPENROUTER_API_KEY",
		RequiresKey: true,
		Protocol:    "openai",
	},
	{
		ID:          "huggingface",
		Name:        "HuggingFace Inference",
		BaseURL:     "https://api-inference.huggingface.co/v1",
		AuthHeader:  "Authorization",
		AuthPrefix:  "Bearer ",
		Models:      []string{"meta-llama/Llama-3.3-70B-Instruct", "mistralai/Mistral-7B-Instruct-v0.3"},
		KeyEnvVar:   "HF_API_KEY",
		RequiresKey: true,
		Protocol:    "openai",
	},
	{
		ID:          "ollama",
		Name:        "Ollama (Local)",
		BaseURL:     "http://127.0.0.1:11434",
		AuthHeader:  "",
		AuthPrefix:  "",
		Models:      []string{"llama3.2:3b", "llama3.1:8b", "mistral:7b", "qwen2.5:7b", "gemma2:9b", "phi3:mini"},
		KeyEnvVar:   "",
		RequiresKey: false,
		Protocol:    "openai",
	},
	{
		ID:          "llamafile",
		Name:        "Llamafile (Local)",
		BaseURL:     "http://127.0.0.1:8080",
		AuthHeader:  "",
		AuthPrefix:  "",
		Models:      []string{"local"},
		KeyEnvVar:   "",
		RequiresKey: false,
		Protocol:    "openai",
	},
	{
		ID:          "custom",
		Name:        "Custom OpenAI-Compatible",
		BaseURL:     "",
		AuthHeader:  "Authorization",
		AuthPrefix:  "Bearer ",
		Models:      []string{},
		KeyEnvVar:   "",
		RequiresKey: false,
		Protocol:    "openai",
	},
}

// LookupProvider returns provider info by ID
func LookupProvider(id string) *ProviderInfo {
	id = strings.ToLower(id)
	for i := range Registry {
		if Registry[i].ID == id {
			return &Registry[i]
		}
	}
	return nil
}

// ── Provider ─────────────────────────────────────────────────────────────

type Provider struct {
	config ProviderConfig
	client *http.Client
	info   *ProviderInfo
}

func New(cfg ProviderConfig) *Provider {
	p := &Provider{
		config: cfg,
		client: &http.Client{
			Timeout: 180 * time.Second,
		},
		info: LookupProvider(cfg.Provider),
	}
	return p
}

func (p *Provider) Chat(system string, messages []Message, tools []ToolDef) (*Response, error) {
	protocol := "openai"
	if p.info != nil {
		protocol = p.info.Protocol
	} else if p.config.Provider == "anthropic" {
		protocol = "anthropic"
	}

	switch protocol {
	case "anthropic":
		return p.chatAnthropic(system, messages, tools)
	default:
		return p.chatOpenAI(system, messages, tools)
	}
}

// ── OpenAI-compatible protocol ───────────────────────────────────────────

func (p *Provider) chatOpenAI(system string, messages []Message, tools []ToolDef) (*Response, error) {
	msgs := []map[string]interface{}{}

	if system != "" {
		msgs = append(msgs, map[string]interface{}{
			"role":    "system",
			"content": system,
		})
	}

	for _, m := range messages {
		msg := map[string]interface{}{
			"role":    m.Role,
			"content": m.Content,
		}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			tcs := []map[string]interface{}{}
			for _, tc := range m.ToolCalls {
				tcs = append(tcs, map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Name,
						"arguments": tc.Arguments,
					},
				})
			}
			msg["tool_calls"] = tcs
		}
		msgs = append(msgs, msg)
	}

	body := map[string]interface{}{
		"model":       p.config.Model,
		"messages":    msgs,
		"max_tokens":  p.config.MaxTokens,
		"temperature": p.config.Temperature,
	}

	if len(tools) > 0 {
		body["tools"] = tools
	}

	data, _ := json.Marshal(body)

	url := strings.TrimRight(p.config.BaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Set auth header based on provider info
	if p.config.APIKey != "" {
		authHeader := "Authorization"
		authPrefix := "Bearer "
		if p.info != nil && p.info.AuthHeader != "" {
			authHeader = p.info.AuthHeader
			authPrefix = p.info.AuthPrefix
		}
		req.Header.Set(authHeader, authPrefix+p.config.APIKey)
	}

	// OpenRouter requires extra headers
	if p.config.Provider == "openrouter" {
		req.Header.Set("HTTP-Referer", "https://github.com/artifact-virtual/spore")
		req.Header.Set("X-Title", "Spore Agent")
	}

	fmt.Fprintf(os.Stderr, "[provider] POST %s model=%s msgs=%d tools=%d\n",
		url, p.config.Model, len(msgs), len(tools))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)

	fmt.Fprintf(os.Stderr, "[provider] HTTP %d, %d bytes\n", resp.StatusCode, len(respData))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respData))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("parse error: %w\nBody: %s", err, string(respData[:min(500, len(respData))]))
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty response from %s", p.config.Provider)
	}

	r := &Response{}
	for _, c := range result.Choices {
		if c.Message.Content != "" && r.Content == "" {
			r.Content = c.Message.Content
		}
		for _, tc := range c.Message.ToolCalls {
			r.ToolCalls = append(r.ToolCalls, ToolCall{
				ID:        tc.ID,
				Type:      tc.Type,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}

	fmt.Fprintf(os.Stderr, "[provider] content=%d chars, tool_calls=%d\n", len(r.Content), len(r.ToolCalls))
	for _, tc := range r.ToolCalls {
		fmt.Fprintf(os.Stderr, "[provider]   tool: %s(%s)\n", tc.Name, tc.Arguments[:min(80, len(tc.Arguments))])
	}

	return r, nil
}

// ── Anthropic protocol ───────────────────────────────────────────────────

func (p *Provider) chatAnthropic(system string, messages []Message, tools []ToolDef) (*Response, error) {
	msgs := []map[string]interface{}{}
	for _, m := range messages {
		if m.Role == "tool" {
			msgs = append(msgs, map[string]interface{}{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type":        "tool_result",
						"tool_use_id": m.ToolCallID,
						"content":     m.Content,
					},
				},
			})
			continue
		}
		msg := map[string]interface{}{
			"role":    m.Role,
			"content": m.Content,
		}
		if len(m.ToolCalls) > 0 {
			content := []map[string]interface{}{}
			if m.Content != "" {
				content = append(content, map[string]interface{}{
					"type": "text",
					"text": m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				var input interface{}
				json.Unmarshal([]byte(tc.Arguments), &input)
				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Name,
					"input": input,
				})
			}
			msg["content"] = content
		}
		msgs = append(msgs, msg)
	}

	anthropicTools := []map[string]interface{}{}
	for _, t := range tools {
		anthropicTools = append(anthropicTools, map[string]interface{}{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": t.Function.Parameters,
		})
	}

	body := map[string]interface{}{
		"model":      p.config.Model,
		"max_tokens": p.config.MaxTokens,
		"messages":   msgs,
	}
	if system != "" {
		body["system"] = system
	}
	if len(anthropicTools) > 0 {
		body["tools"] = anthropicTools
	}

	data, _ := json.Marshal(body)

	url := strings.TrimRight(p.config.BaseURL, "/") + "/v1/messages"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	fmt.Fprintf(os.Stderr, "[provider] POST %s model=%s msgs=%d tools=%d\n",
		url, p.config.Model, len(msgs), len(tools))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)

	fmt.Fprintf(os.Stderr, "[provider] HTTP %d, %d bytes\n", resp.StatusCode, len(respData))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respData))
	}

	var result struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	r := &Response{}
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			r.Content += block.Text
		case "tool_use":
			args, _ := json.Marshal(block.Input)
			r.ToolCalls = append(r.ToolCalls, ToolCall{
				ID:        block.ID,
				Type:      "function",
				Name:      block.Name,
				Arguments: string(args),
			})
		}
	}

	return r, nil
}
