package tools

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Toolbox struct {
	home string
}

func New(home string) *Toolbox {
	return &Toolbox{home: home}
}

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
	defs := []ToolDef{
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "exec",
				Description: "Execute a shell command. Full Termux shell access. Returns stdout+stderr. Use for anything: apt, python, nmap, ssh, git, curl, etc.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "Shell command to execute",
						},
						"timeout": map[string]interface{}{
							"type":        "number",
							"description": "Timeout in seconds (default 30)",
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
				Description: "Read contents of a file. Supports offset/limit for large files.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "File path to read",
						},
						"offset": map[string]interface{}{
							"type":        "number",
							"description": "Start line (1-indexed, optional)",
						},
						"limit": map[string]interface{}{
							"type":        "number",
							"description": "Max lines to read (optional)",
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
				Description: "Write content to a file. Creates parent directories automatically.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "File path to write",
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
				Name:        "edit",
				Description: "Edit a file by replacing exact text. Surgical precision.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "File path",
						},
						"old_text": map[string]interface{}{
							"type":        "string",
							"description": "Exact text to find",
						},
						"new_text": map[string]interface{}{
							"type":        "string",
							"description": "Replacement text",
						},
					},
					"required": []string{"path", "old_text", "new_text"},
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
				Description: "Fetch content from a URL. Returns response body.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url": map[string]interface{}{
							"type":        "string",
							"description": "URL to fetch",
						},
						"max_bytes": map[string]interface{}{
							"type":        "number",
							"description": "Max response bytes (default 50000)",
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
				Description: "List files and directories.",
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
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "processes",
				Description: "List running processes with PID, name, CPU, memory.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "kill_process",
				Description: "Kill a process by PID.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pid": map[string]interface{}{
							"type":        "string",
							"description": "Process ID to kill",
						},
					},
					"required": []string{"pid"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "env",
				Description: "Show environment variables.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "device_info",
				Description: "Get Android device information: model, battery, storage, network, IP.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "notify",
				Description: "Send an Android notification via Termux:API.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Notification title",
						},
						"body": map[string]interface{}{
							"type":        "string",
							"description": "Notification body",
						},
					},
					"required": []string{"title", "body"},
				},
			},
		},
	}

	// Append Android-specific tools
	defs = append(defs, AndroidToolDefs()...)
	return defs
}

// --- Implementations ---

func (t *Toolbox) Exec(command string) string {
	return t.ExecTimeout(command, 30)
}

func (t *Toolbox) ExecTimeout(command string, timeout int) string {
	shell := findShell()
	cmd := exec.Command(shell, "-c", command)
	cmd.Dir = t.home

	// Create new process group so we can kill all children on timeout
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Termux environment
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
	)

	done := make(chan struct{})
	var output []byte
	var err error

	go func() {
		output, err = cmd.CombinedOutput()
		close(done)
	}()

	select {
	case <-done:
		// completed
	case <-time.After(time.Duration(timeout) * time.Second):
		// Kill entire process group (negative PID) to prevent zombie children
		if cmd.Process != nil {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return fmt.Sprintf("[timeout after %ds]", timeout)
	}

	result := string(output)
	if err != nil {
		result += "\n[exit: " + err.Error() + "]"
	}
	if len(result) > 32768 {
		result = result[:32768] + "\n[truncated]"
	}
	return result
}

func (t *Toolbox) Read(path string) string {
	return t.ReadLines(path, 0, 0)
}

func (t *Toolbox) ReadLines(path string, offset, limit int) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(t.home, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}

	content := string(data)

	if offset > 0 || limit > 0 {
		lines := strings.Split(content, "\n")
		start := 0
		if offset > 0 {
			start = offset - 1 // 1-indexed
		}
		if start >= len(lines) {
			return ""
		}
		end := len(lines)
		if limit > 0 && start+limit < end {
			end = start + limit
		}
		content = strings.Join(lines[start:end], "\n")
	}

	if len(content) > 65536 {
		content = content[:65536] + "\n[truncated at 64KB — use offset/limit]"
	}
	return content
}

func (t *Toolbox) Write(path, content string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(t.home, path)
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(content), path)
}

func (t *Toolbox) Edit(path, oldText, newText string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(t.home, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	content := string(data)
	if !strings.Contains(content, oldText) {
		return "error: old_text not found in file"
	}
	count := strings.Count(content, oldText)
	if count > 1 {
		return fmt.Sprintf("error: old_text found %d times — must be unique", count)
	}
	content = strings.Replace(content, oldText, newText, 1)
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return fmt.Sprintf("error writing: %s", err)
	}
	return fmt.Sprintf("edited %s (replaced %d chars with %d chars)", filepath.Base(path), len(oldText), len(newText))
}

func (t *Toolbox) WebFetch(url string, maxBytes int) string {
	if maxBytes <= 0 {
		maxBytes = 50000
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))
	if err != nil {
		return fmt.Sprintf("error reading: %s", err)
	}
	return string(body)
}

func (t *Toolbox) List(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(t.home, path)
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

func (t *Toolbox) Processes() string {
	out := t.ExecTimeout("ps aux 2>/dev/null || ps -ef 2>/dev/null || ps", 5)
	return out
}

func (t *Toolbox) KillProcess(pid string) string {
	p, err := strconv.Atoi(pid)
	if err != nil {
		// try by name
		return t.ExecTimeout(fmt.Sprintf("pkill -f %s", pid), 5)
	}
	proc, err := os.FindProcess(p)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	err = proc.Kill()
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	return fmt.Sprintf("killed PID %d", p)
}

func (t *Toolbox) Env() string {
	var sb strings.Builder
	for _, e := range os.Environ() {
		sb.WriteString(e)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (t *Toolbox) DeviceInfo() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("os: android (%s/%s)\n", runtime.GOOS, runtime.GOARCH))

	// Termux-specific info via termux-api
	cmds := map[string]string{
		"battery":  "termux-battery-status 2>/dev/null",
		"wifi":     "termux-wifi-connectioninfo 2>/dev/null",
		"storage":  "df -h /data 2>/dev/null | tail -1",
		"ip":       "ip addr show wlan0 2>/dev/null | grep 'inet ' | awk '{print $2}'",
		"hostname": "hostname 2>/dev/null || getprop net.hostname 2>/dev/null",
		"uptime":   "uptime 2>/dev/null",
		"memory":   "free -h 2>/dev/null | head -2",
	}

	for name, cmd := range cmds {
		out := t.ExecTimeout(cmd, 3)
		out = strings.TrimSpace(out)
		if out != "" && !strings.Contains(out, "error") && !strings.Contains(out, "not found") {
			sb.WriteString(fmt.Sprintf("%s: %s\n", name, out))
		}
	}

	return sb.String()
}

func (t *Toolbox) Notify(title, body string) string {
	cmd := fmt.Sprintf("termux-notification --title %q --content %q", title, body)
	return t.ExecTimeout(cmd, 5)
}

// --- Helpers ---

func findShell() string {
	// Termux bash
	if _, err := os.Stat("/data/data/com.termux/files/usr/bin/bash"); err == nil {
		return "/data/data/com.termux/files/usr/bin/bash"
	}
	// Standard
	for _, sh := range []string{"/bin/bash", "/bin/sh", "/system/bin/sh"} {
		if _, err := os.Stat(sh); err == nil {
			return sh
		}
	}
	return "sh"
}

func humanSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1fG", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}
