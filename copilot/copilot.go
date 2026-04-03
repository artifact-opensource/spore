// Package copilot implements a GitHub Copilot proxy with device auth,
// automatic token refresh, VS Code fingerprinting, and robust retry.
package copilot

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	CopilotAPI     = "https://api.githubcopilot.com"
	GitHubAPI      = "https://api.github.com"
	ClientID       = "Iv1.b507a08c87ecfe98" // VS Code Copilot
	MaxRetries     = 5
	TokenRefreshBuffer = 300 // seconds before expiry to refresh
)

var retryDelays = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	15 * time.Second,
}

// Proxy is the copilot proxy server with auth, retry, and fingerprinting.
type Proxy struct {
	port      int
	dataDir   string
	auth      *Auth
	sessionID string
	machineID string
	client    *http.Client
	mu        sync.RWMutex
	stats     Stats
}

type Stats struct {
	Requests int `json:"requests"`
	Retries  int `json:"retries"`
	Errors   int `json:"errors"`
}

// New creates a copilot proxy.
func New(port int, dataDir string) *Proxy {
	p := &Proxy{
		port:      port,
		dataDir:   dataDir,
		sessionID: generateUUID(),
		machineID: getMachineID(dataDir),
		client: &http.Client{
			Timeout: 300 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        5,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     120 * time.Second,
			},
		},
	}
	p.auth = NewAuth(dataDir)
	return p
}

// ListenAndServe starts the proxy on the configured port.
func (p *Proxy) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", p.chatCompletions)
	mux.HandleFunc("/chat/completions", p.chatCompletions)
	mux.HandleFunc("/v1/models", p.listModels)
	mux.HandleFunc("/models", p.listModels)
	mux.HandleFunc("/health", p.health)
	mux.HandleFunc("/", p.health)

	addr := fmt.Sprintf("127.0.0.1:%d", p.port)
	fmt.Printf("  copilot proxy on http://%s (session=%s)\n", addr, p.sessionID[:8])
	if p.auth.githubToken == "" {
		fmt.Printf("  ⚠️  no github token — run: spore copilot auth\n")
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second,
	}
	return server.ListenAndServe()
}

func (p *Proxy) buildHeaders(token string) map[string]string {
	p.mu.Lock()
	p.stats.Requests++
	p.mu.Unlock()

	return map[string]string{
		"Authorization":           "Bearer " + token,
		"Content-Type":            "application/json",
		"X-Request-Id":            generateUUID(),
		"VScode-SessionId":        p.sessionID,
		"VScode-MachineId":        p.machineID,
		"X-GitHub-Api-Version":    "2023-07-07",
		"Copilot-Integration-Id":  "vscode-chat",
		"Editor-Version":          "vscode/1.100.0",
		"Editor-Plugin-Version":   "copilot-chat/0.25.0",
		"Openai-Organization":     "github-copilot",
		"Openai-Intent":           "conversation-panel",
		"User-Agent":              "GitHubCopilotChat/0.25.0",
	}
}

func (p *Proxy) chatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", 405)
		return
	}

	token, err := p.auth.GetCopilotToken()
	if err != nil || token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "No valid Copilot token. Run: spore copilot auth",
				"type":    "auth_error",
			},
		})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	r.Body.Close()

	// Parse to check stream flag and strip trailing assistant messages
	var req map[string]interface{}
	json.Unmarshal(body, &req)
	isStream := false
	if s, ok := req["stream"].(bool); ok {
		isStream = s
	}

	// Strip trailing assistant messages (Copilot API rejects them)
	if msgs, ok := req["messages"].([]interface{}); ok {
		for len(msgs) > 1 {
			last, ok := msgs[len(msgs)-1].(map[string]interface{})
			if !ok || last["role"] != "assistant" {
				break
			}
			msgs = msgs[:len(msgs)-1]
		}
		req["messages"] = msgs
		body, _ = json.Marshal(req)
	}

	headers := p.buildHeaders(token)

	if isStream {
		p.streamWithRetry(w, body, headers)
	} else {
		p.requestWithRetry(w, body, headers)
	}
}

func (p *Proxy) streamWithRetry(w http.ResponseWriter, body []byte, headers map[string]string) {
	flusher, canFlush := w.(http.Flusher)

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			p.mu.Lock()
			p.stats.Retries++
			p.mu.Unlock()
			delay := retryDelays[min(attempt-1, len(retryDelays)-1)]
			fmt.Printf("  [copilot] RETRY %d/%d after %v\n", attempt, MaxRetries, delay)
			time.Sleep(delay)
			// Refresh request ID
			headers["X-Request-Id"] = generateUUID()
		}

		req, err := http.NewRequest("POST", CopilotAPI+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			continue
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := p.client.Do(req)
		if err != nil {
			fmt.Printf("  [copilot] CONNECTION ERROR attempt %d: %v\n", attempt+1, err)
			p.mu.Lock()
			p.stats.Errors++
			p.mu.Unlock()
			if attempt < MaxRetries {
				continue
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			fmt.Fprintf(w, "data: %s\n\n", mustJSON(map[string]interface{}{
				"error": map[string]string{
					"message": fmt.Sprintf("Upstream unreachable after %d attempts: %v", MaxRetries+1, err),
					"type":    "connection_error",
				},
			}))
			return
		}

		if resp.StatusCode == 429 && attempt < MaxRetries {
			resp.Body.Close()
			fmt.Printf("  [copilot] 429 RATE LIMITED, retry %d\n", attempt+1)
			continue
		}

		if resp.StatusCode != 200 {
			errBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if attempt < MaxRetries && resp.StatusCode >= 500 {
				fmt.Printf("  [copilot] %d SERVER ERROR, retry %d\n", resp.StatusCode, attempt+1)
				continue
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			fmt.Fprintf(w, "data: %s\n\n", mustJSON(map[string]interface{}{
				"error": string(errBody),
			}))
			return
		}

		// Success — stream back
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(200)

		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
				if canFlush {
					flusher.Flush()
				}
			}
			if err != nil {
				break
			}
		}
		resp.Body.Close()
		return
	}
}

func (p *Proxy) requestWithRetry(w http.ResponseWriter, body []byte, headers map[string]string) {
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			p.mu.Lock()
			p.stats.Retries++
			p.mu.Unlock()
			delay := retryDelays[min(attempt-1, len(retryDelays)-1)]
			fmt.Printf("  [copilot] RETRY %d/%d after %v\n", attempt, MaxRetries, delay)
			time.Sleep(delay)
			headers["X-Request-Id"] = generateUUID()
		}

		req, err := http.NewRequest("POST", CopilotAPI+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			continue
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := p.client.Do(req)
		if err != nil {
			fmt.Printf("  [copilot] ERROR attempt %d: %v\n", attempt+1, err)
			p.mu.Lock()
			p.stats.Errors++
			p.mu.Unlock()
			if attempt < MaxRetries {
				continue
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(502)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"message": fmt.Sprintf("Upstream unreachable after %d attempts", MaxRetries+1),
					"type":    "connection_error",
				},
			})
			return
		}

		if resp.StatusCode == 429 && attempt < MaxRetries {
			resp.Body.Close()
			continue
		}

		// Forward response
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return
	}
}

func (p *Proxy) listModels(w http.ResponseWriter, r *http.Request) {
	token, err := p.auth.GetCopilotToken()
	if err != nil || token == "" {
		http.Error(w, "no copilot token", 401)
		return
	}

	req, _ := http.NewRequest("GET", CopilotAPI+"/models", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")
	req.Header.Set("VScode-SessionId", p.sessionID)
	req.Header.Set("VScode-MachineId", p.machineID)

	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func (p *Proxy) health(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	stats := p.stats
	p.mu.RUnlock()

	hasGithub := p.auth.githubToken != ""
	hasCopilot := p.auth.copilotToken != "" && time.Now().Unix() < p.auth.copilotExpires-TokenRefreshBuffer

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":              "ok",
		"github_auth":         hasGithub,
		"copilot_token_valid": hasCopilot,
		"port":                p.port,
		"session":             p.sessionID[:8],
		"stats":               stats,
	})
}

// --- Auth ---

type Auth struct {
	dataDir       string
	githubToken   string
	copilotToken  string
	copilotExpires int64
	mu            sync.Mutex
}

func NewAuth(dataDir string) *Auth {
	a := &Auth{dataDir: dataDir}
	a.loadTokens()
	return a
}

func (a *Auth) tokenDir() string {
	return filepath.Join(a.dataDir, "copilot")
}

func (a *Auth) loadTokens() {
	dir := a.tokenDir()

	if data, err := os.ReadFile(filepath.Join(dir, "github_token.json")); err == nil {
		var t struct {
			Token string `json:"github_token"`
		}
		json.Unmarshal(data, &t)
		a.githubToken = t.Token
	}

	if data, err := os.ReadFile(filepath.Join(dir, "copilot_token.json")); err == nil {
		var t struct {
			Token     string `json:"token"`
			ExpiresAt int64  `json:"expires_at"`
		}
		json.Unmarshal(data, &t)
		a.copilotToken = t.Token
		a.copilotExpires = t.ExpiresAt
	}
}

func (a *Auth) saveGitHubToken(token string) {
	dir := a.tokenDir()
	os.MkdirAll(dir, 0700)
	data, _ := json.Marshal(map[string]string{"github_token": token})
	os.WriteFile(filepath.Join(dir, "github_token.json"), data, 0600)
	a.githubToken = token
}

func (a *Auth) saveCopilotToken(token string, expiresAt int64) {
	dir := a.tokenDir()
	os.MkdirAll(dir, 0700)
	data, _ := json.Marshal(map[string]interface{}{"token": token, "expires_at": expiresAt})
	os.WriteFile(filepath.Join(dir, "copilot_token.json"), data, 0600)
	a.copilotToken = token
	a.copilotExpires = expiresAt
}

func (a *Auth) GetCopilotToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Valid token exists
	if a.copilotToken != "" && time.Now().Unix() < a.copilotExpires-TokenRefreshBuffer {
		return a.copilotToken, nil
	}

	// Need to refresh
	if a.githubToken == "" {
		return "", fmt.Errorf("no github token")
	}

	req, _ := http.NewRequest("GET", GitHubAPI+"/copilot_internal/v2/token", nil)
	req.Header.Set("Authorization", "token "+a.githubToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Editor-Version", "vscode/1.100.0")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.25.0")
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.25.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token refresh failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token refresh %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Token == "" {
		return "", fmt.Errorf("empty copilot token")
	}

	a.saveCopilotToken(result.Token, result.ExpiresAt)
	fmt.Printf("  [copilot] token refreshed (expires in %dm)\n", (result.ExpiresAt-time.Now().Unix())/60)
	return result.Token, nil
}

// DeviceAuth runs the GitHub device authorization flow interactively.
func (a *Auth) DeviceAuth() error {
	client := &http.Client{Timeout: 30 * time.Second}

	body, _ := json.Marshal(map[string]string{
		"client_id": ClientID,
		"scope":     "copilot",
	})

	resp, err := client.Post("https://github.com/login/device/code", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("device code request failed: %w", err)
	}
	defer resp.Body.Close()

	var codeResp struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		Interval        int    `json:"interval"`
	}
	// GitHub returns form-encoded by default unless we ask for JSON
	respBody, _ := io.ReadAll(resp.Body)

	// Try JSON first
	if err := json.Unmarshal(respBody, &codeResp); err != nil {
		return fmt.Errorf("unexpected response: %s", string(respBody))
	}

	if codeResp.Interval == 0 {
		codeResp.Interval = 5
	}

	fmt.Printf("\n  ══════════════════════════════════\n")
	fmt.Printf("  Go to:     %s\n", codeResp.VerificationURI)
	fmt.Printf("  Enter code: %s\n", codeResp.UserCode)
	fmt.Printf("  ══════════════════════════════════\n\n")
	fmt.Printf("  Waiting for authorization...\n")

	for {
		time.Sleep(time.Duration(codeResp.Interval) * time.Second)

		tokenBody, _ := json.Marshal(map[string]string{
			"client_id":   ClientID,
			"device_code": codeResp.DeviceCode,
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		})

		tokenReq, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewReader(tokenBody))
		tokenReq.Header.Set("Content-Type", "application/json")
		tokenReq.Header.Set("Accept", "application/json")

		tokenResp, err := client.Do(tokenReq)
		if err != nil {
			continue
		}

		var result struct {
			AccessToken string `json:"access_token"`
			Error       string `json:"error"`
			Interval    int    `json:"interval"`
		}
		json.NewDecoder(tokenResp.Body).Decode(&result)
		tokenResp.Body.Close()

		if result.AccessToken != "" {
			a.saveGitHubToken(result.AccessToken)
			fmt.Printf("  ✅ Authenticated successfully!\n")
			return nil
		}

		switch result.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			if result.Interval > 0 {
				codeResp.Interval = result.Interval
			} else {
				codeResp.Interval += 5
			}
			continue
		case "expired_token":
			return fmt.Errorf("device code expired — try again")
		case "access_denied":
			return fmt.Errorf("authorization denied")
		default:
			return fmt.Errorf("unexpected error: %s", result.Error)
		}
	}
}

// --- Helpers ---

func getMachineID(dataDir string) string {
	path := filepath.Join(dataDir, "copilot", "machine_id")
	if data, err := os.ReadFile(path); err == nil {
		return string(data)
	}
	hostname, _ := os.Hostname()
	seed := fmt.Sprintf("%s-%s-spore-copilot", hostname, runtime.GOARCH)
	h := sha256.Sum256([]byte(seed))
	id := fmt.Sprintf("%x", h)
	os.MkdirAll(filepath.Dir(path), 0700)
	os.WriteFile(path, []byte(id), 0600)
	return id
}

func generateUUID() string {
	// Simple v4 UUID without external dependency
	b := make([]byte, 16)
	// Use crypto/rand for true randomness
	f, _ := os.Open("/dev/urandom")
	f.Read(b)
	f.Close()
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func mustJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
