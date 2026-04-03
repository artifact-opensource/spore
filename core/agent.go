package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/artifact-virtual/symbiote-android/provider"
	"github.com/artifact-virtual/symbiote-android/tools"
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
	prov     *provider.Provider
	memory   MemoryBackend
	tools    *tools.Toolbox
	config   *Config
	history  []provider.Message
	sessions *SessionStore
	session  *Session // active session (nil = ephemeral)
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

	// Session store lives in data dir
	dataDir := filepath.Dir(cfg.path)
	if dataDir == "." {
		dataDir = os.Getenv("HOME")
		if dataDir == "" {
			dataDir = "/tmp"
		}
		dataDir = filepath.Join(dataDir, ".symbiote")
	}

	return &Agent{
		prov:     provider.New(pcfg),
		memory:   mem,
		tools:    t,
		config:   cfg,
		history:  []provider.Message{},
		sessions: NewSessionStore(dataDir),
	}
}

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
	// Auto-create session if none active
	if a.session == nil {
		a.NewSession("")
	}

	a.history = append(a.history, provider.Message{
		Role:    "user",
		Content: input,
	})

	provTools := convertToolDefs(a.tools.Definitions())

	maxIter := a.config.MaxIterations
	if maxIter <= 0 {
		maxIter = 25
	}
	for i := 0; i < maxIter; i++ {
		resp, err := a.prov.Chat(a.config.System, a.history, provTools)
		if err != nil {
			return "", err
		}

		if len(resp.ToolCalls) == 0 {
			a.history = append(a.history, provider.Message{
				Role:    "assistant",
				Content: resp.Content,
			})
			a.saveSession()
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

	a.saveSession()
	return "[max iterations reached]", nil
}

func (a *Agent) executeTool(name string, args map[string]interface{}) string {
	switch name {
	case "exec":
		cmd, _ := args["command"].(string)
		timeout := 30
		if t, ok := args["timeout"].(float64); ok {
			timeout = int(t)
		}
		return a.tools.ExecTimeout(cmd, timeout)
	case "read":
		path, _ := args["path"].(string)
		offset := 0
		limit := 0
		if o, ok := args["offset"].(float64); ok {
			offset = int(o)
		}
		if l, ok := args["limit"].(float64); ok {
			limit = int(l)
		}
		return a.tools.ReadLines(path, offset, limit)
	case "write":
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)
		return a.tools.Write(path, content)
	case "edit":
		path, _ := args["path"].(string)
		old, _ := args["old_text"].(string)
		new_, _ := args["new_text"].(string)
		return a.tools.Edit(path, old, new_)
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
		maxBytes := 50000
		if m, ok := args["max_bytes"].(float64); ok {
			maxBytes = int(m)
		}
		return a.tools.WebFetch(url, maxBytes)
	case "list":
		path, _ := args["path"].(string)
		if path == "" {
			path = "."
		}
		return a.tools.List(path)
	case "processes":
		return a.tools.Processes()
	case "kill_process":
		pid, _ := args["pid"].(string)
		return a.tools.KillProcess(pid)
	case "env":
		return a.tools.Env()
	case "device_info":
		return a.tools.DeviceInfo()
	case "notify":
		title, _ := args["title"].(string)
		body, _ := args["body"].(string)
		return a.tools.Notify(title, body)

	// --- Android Device Control ---
	case "brightness":
		level, _ := args["level"].(float64)
		return a.tools.Brightness(int(level))
	case "volume":
		stream, _ := args["stream"].(string)
		level, _ := args["level"].(float64)
		return a.tools.Volume(stream, int(level))
	case "torch":
		state, _ := args["state"].(string)
		return a.tools.Torch(state)
	case "vibrate":
		dur, _ := args["duration_ms"].(float64)
		return a.tools.Vibrate(int(dur))
	case "clipboard_get":
		return a.tools.ClipboardGet()
	case "clipboard_set":
		text, _ := args["text"].(string)
		return a.tools.ClipboardSet(text)
	case "tts_speak":
		text, _ := args["text"].(string)
		return a.tools.TtsSpeak(text)
	case "toast":
		text, _ := args["text"].(string)
		return a.tools.Toast(text)
	case "wifi_info":
		return a.tools.WifiInfo()
	case "location":
		return a.tools.Location()
	case "camera_photo":
		camID, _ := args["camera_id"].(float64)
		output, _ := args["output_path"].(string)
		return a.tools.CameraPhoto(int(camID), output)
	case "media_control":
		action, _ := args["action"].(string)
		return a.tools.MediaControl(action)
	case "sms_send":
		number, _ := args["number"].(string)
		message, _ := args["message"].(string)
		return a.tools.SmsSend(number, message)
	case "sms_inbox":
		limit, _ := args["limit"].(float64)
		return a.tools.SmsInbox(int(limit))
	case "call":
		number, _ := args["number"].(string)
		return a.tools.Call(number)
	case "screen_state":
		return a.tools.ScreenState()
	case "battery":
		return a.tools.BatteryStatus()

	// --- App Management ---
	case "app_launch":
		name, _ := args["name"].(string)
		return a.tools.AppLaunch(name)
	case "app_stop":
		name, _ := args["name"].(string)
		return a.tools.AppStop(name)
	case "app_list":
		filter, _ := args["filter"].(string)
		return a.tools.AppList(filter)
	case "app_info":
		name, _ := args["name"].(string)
		return a.tools.AppInfo(name)

	// --- MacroDroid ---
	case "macro_fire":
		trigger, _ := args["trigger_name"].(string)
		return a.tools.MacroFire(trigger)
	case "macro_fire_with":
		trigger, _ := args["trigger_name"].(string)
		vars := make(map[string]string)
		if v, ok := args["variables"].(map[string]interface{}); ok {
			for k, val := range v {
				vars[k] = fmt.Sprintf("%v", val)
			}
		}
		return a.tools.MacroFireWith(trigger, vars)

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
	a.session = nil
}

// --- Session management ---

func (a *Agent) NewSession(title string) *Session {
	sess := a.sessions.Create(title)
	a.session = sess
	a.history = []provider.Message{}
	return sess
}

func (a *Agent) LoadSession(id string) (*Session, error) {
	sess, err := a.sessions.Load(id)
	if err != nil {
		return nil, err
	}
	a.session = sess
	a.history = sess.Messages
	return sess, nil
}

func (a *Agent) DeleteSession(id string) error {
	if a.session != nil && a.session.ID == id {
		a.session = nil
		a.history = []provider.Message{}
	}
	return a.sessions.Delete(id)
}

func (a *Agent) ListSessions() []SessionMeta {
	return a.sessions.List()
}

func (a *Agent) ActiveSession() *Session {
	return a.session
}

func (a *Agent) saveSession() {
	if a.session == nil {
		return
	}
	a.session.Messages = a.history
	a.sessions.UpdateTitle(a.session)
	a.sessions.Save(a.session)
}

func truncStr(s string, n int) string {
	if len(s) > n {
		return s[:n-3] + "..."
	}
	return s
}
