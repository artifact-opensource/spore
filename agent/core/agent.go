package core

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/artifact-virtual/spore/provider"
	"github.com/artifact-virtual/spore/tools"
)

type MemoryStats struct {
	Documents  int
	Vectors    int
	IndexBytes int64
}

type SearchResult struct {
	Path  string
	Score float64
	Chunk string
}

type MemoryBackend interface {
	Search(query string, k int) []SearchResult
	Ingest(path string) (int, error)
	Stats() MemoryStats
}

type Agent struct {
	prov    *provider.Provider
	memory  MemoryBackend
	tools   *tools.Toolbox
	config  *Config
	history []provider.Message
}

func NewAgent(cfg *Config, mem MemoryBackend, t *tools.Toolbox) *Agent {
	pcfg := provider.ProviderConfig{
		Provider:    cfg.Provider,
		Model:       cfg.Model,
		BaseURL:     cfg.BaseURL,
		APIKey:      cfg.APIKey,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	}
	return &Agent{
		prov:    provider.New(pcfg),
		memory:  mem,
		tools:   t,
		config:  cfg,
		history: []provider.Message{},
	}
}

// convert tools.ToolDef to provider.ToolDef
func convertToolDefs(defs []tools.ToolDef) []provider.ToolDef {
	out := make([]provider.ToolDef, len(defs))
	for i, d := range defs {
		out[i] = provider.ToolDef{
			Type: d.Type,
			Function: provider.ToolDefFunction{
				Name:        d.Function.Name,
				Description: d.Function.Description,
				Parameters:  d.Function.Parameters,
			},
		}
	}
	return out
}

func (a *Agent) Run(input string) (string, error) {
	a.history = append(a.history, provider.Message{
		Role:    "user",
		Content: input,
	})

	provTools := convertToolDefs(a.tools.Definitions())

	for i := 0; i < 10; i++ {
		resp, err := a.prov.Chat(a.config.System, a.history, provTools)
		if err != nil {
			return "", err
		}

		if len(resp.ToolCalls) == 0 {
			a.history = append(a.history, provider.Message{
				Role:    "assistant",
				Content: resp.Content,
			})
			return resp.Content, nil
		}

		a.history = append(a.history, provider.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		for _, tc := range resp.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Arguments), &args)

			result := a.executeTool(tc.Name, args)

			a.history = append(a.history, provider.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return "[max iterations reached]", nil
}

func (a *Agent) executeTool(name string, args map[string]interface{}) string {
	switch name {
	case "exec":
		cmd, _ := args["command"].(string)
		return a.tools.Exec(cmd)
	case "read":
		path, _ := args["path"].(string)
		return a.tools.Read(path)
	case "write":
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)
		return a.tools.Write(path, content)
	case "search":
		query, _ := args["query"].(string)
		k := 5
		if kf, ok := args["k"].(float64); ok {
			k = int(kf)
		}
		results := a.memory.Search(query, k)
		if len(results) == 0 {
			return "no results"
		}
		var sb strings.Builder
		for _, r := range results {
			sb.WriteString(fmt.Sprintf("[%.3f] %s: %s\n", r.Score, r.Path, r.Chunk))
		}
		return sb.String()
	case "web_fetch":
		url, _ := args["url"].(string)
		return a.tools.WebFetch(url)
	case "list":
		path, _ := args["path"].(string)
		if path == "" {
			path = "."
		}
		return a.tools.List(path)
	default:
		return fmt.Sprintf("unknown tool: %s", name)
	}
}

func (a *Agent) Search(query string, k int) []SearchResult {
	return a.memory.Search(query, k)
}

func (a *Agent) Ingest(path string) (int, error) {
	return a.memory.Ingest(path)
}

func (a *Agent) Config() *Config {
	return a.config
}

func (a *Agent) MemoryStats() MemoryStats {
	return a.memory.Stats()
}

func (a *Agent) PrintHistory() {
	for _, m := range a.history {
		fmt.Printf("[%s] %s\n", m.Role, truncStr(m.Content, 200))
	}
}

func (a *Agent) Reset() {
	a.history = []provider.Message{}
}

func truncStr(s string, n int) string {
	if len(s) > n {
		return s[:n-3] + "..."
	}
	return s
}
