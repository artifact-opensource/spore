package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

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
	Type     string         `json:"type"`
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

// ProviderConfig holds what the provider needs — no import of core
type ProviderConfig struct {
	Provider    string
	Model       string
	BaseURL     string
	APIKey      string
	MaxTokens   int
	Temperature float64
}

type Provider struct {
	config ProviderConfig
	client *http.Client
}

func New(cfg ProviderConfig) *Provider {
	return &Provider{
		config: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *Provider) Chat(system string, messages []Message, tools []ToolDef) (*Response, error) {
	if p.config.Provider == "anthropic" {
		return p.chatAnthropic(system, messages, tools)
	}
	return p.chatOpenAI(system, messages, tools)
}

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

	url := p.config.BaseURL + "/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)

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
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	choice := result.Choices[0].Message

	r := &Response{
		Content: choice.Content,
	}

	for _, tc := range choice.ToolCalls {
		r.ToolCalls = append(r.ToolCalls, ToolCall{
			ID:        tc.ID,
			Type:      tc.Type,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return r, nil
}

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

	url := p.config.BaseURL + "/v1/messages"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)

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
