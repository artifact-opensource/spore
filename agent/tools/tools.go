package tools

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Toolbox struct {
	workspace string
}

func New(workspace string) *Toolbox {
	return &Toolbox{workspace: workspace}
}

// ToolDef — self-contained, no external imports
type ToolDef struct {
	Type     string      `json:"type"`
	Function ToolDefFunc `json:"function"`
}

type ToolDefFunc struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

func (t *Toolbox) Definitions() []ToolDef {
	return []ToolDef{
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "exec",
				Description: "Execute a shell command. Returns stdout+stderr.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "Shell command to execute",
						},
					},
					"required": []string{"command"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "read",
				Description: "Read the contents of a file.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to read",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "write",
				Description: "Write content to a file. Creates parent directories.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to write to",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Content to write",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "search",
				Description: "Search indexed memory for relevant documents.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search query",
						},
						"k": map[string]interface{}{
							"type":        "number",
							"description": "Number of results (default 5)",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "web_fetch",
				Description: "Fetch content from a URL. Returns text.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url": map[string]interface{}{
							"type":        "string",
							"description": "URL to fetch",
						},
					},
					"required": []string{"url"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "list",
				Description: "List files in a directory.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Directory path (default: current)",
						},
					},
				},
			},
		},
	}
}

func (t *Toolbox) Exec(command string) string {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = t.workspace

	output, err := cmd.CombinedOutput()
	result := string(output)
	if err != nil {
		result += "\n[exit: " + err.Error() + "]"
	}

	if len(result) > 16384 {
		result = result[:16384] + "\n[truncated]"
	}

	return result
}

func (t *Toolbox) Read(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(t.workspace, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}

	content := string(data)
	if len(content) > 32768 {
		content = content[:32768] + "\n[truncated]"
	}

	return content
}

func (t *Toolbox) Write(path, content string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(t.workspace, path)
	}

	os.MkdirAll(filepath.Dir(path), 0755)

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}

	return fmt.Sprintf("wrote %d bytes to %s", len(content), path)
}

func (t *Toolbox) WebFetch(url string) string {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 50000))
	if err != nil {
		return fmt.Sprintf("error reading body: %s", err)
	}

	return string(body)
}

func (t *Toolbox) List(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(t.workspace, path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}

	var sb strings.Builder
	for _, e := range entries {
		info, _ := e.Info()
		if info == nil {
			continue
		}
		prefix := "  "
		if e.IsDir() {
			prefix = "d "
		}
		sb.WriteString(fmt.Sprintf("%s%-40s %s\n", prefix, e.Name(), humanSize(info.Size())))
	}

	return sb.String()
}

func humanSize(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}
