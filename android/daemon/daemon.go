package daemon

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/artifact-virtual/symbiote-android/core"
)

const pidFile = "daemon.pid"
const logFile = "daemon.log"

// Start the daemon in the foreground (Termux handles backgrounding via &/nohup)
func Start(cfg *core.Config, dataDir string) {
	pidPath := filepath.Join(dataDir, pidFile)

	// Check if already running
	if isRunning(pidPath) {
		fmt.Println("  daemon already running")
		return
	}

	// Write PID
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0644)

	// Log file
	logPath := filepath.Join(dataDir, "logs", logFile)
	os.MkdirAll(filepath.Dir(logPath), 0755)
	logF, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("  error opening log: %s\n", err)
		return
	}
	defer logF.Close()

	log := func(msg string) {
		ts := time.Now().Format("2006-01-02 15:04:05")
		line := fmt.Sprintf("[%s] %s\n", ts, msg)
		logF.WriteString(line)
		fmt.Print(line)
	}

	log("symbiote daemon starting")
	log(fmt.Sprintf("pid: %d", os.Getpid()))
	log(fmt.Sprintf("provider: %s/%s", cfg.Provider, cfg.Model))

	// Handle shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// Heartbeat loop
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	log("daemon running — ctrl+c to stop")

	for {
		select {
		case <-sig:
			log("daemon stopping")
			os.Remove(pidPath)
			return
		case <-ticker.C:
			log("heartbeat")
			// Could add: health checks, auto-reconnect, memory cleanup
		}
	}
}

func Stop(dataDir string) {
	pidPath := filepath.Join(dataDir, pidFile)
	data, err := os.ReadFile(pidPath)
	if err != nil {
		fmt.Println("  daemon not running")
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		fmt.Println("  invalid pid file")
		os.Remove(pidPath)
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("  process not found")
		os.Remove(pidPath)
		return
	}

	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		fmt.Printf("  error: %s\n", err)
		os.Remove(pidPath)
		return
	}

	os.Remove(pidPath)
	fmt.Printf("  stopped daemon (pid %d)\n", pid)
}

func Status(dataDir string) {
	pidPath := filepath.Join(dataDir, pidFile)
	if isRunning(pidPath) {
		data, _ := os.ReadFile(pidPath)
		fmt.Printf("  daemon: \033[32mrunning\033[0m (pid %s)\n", strings.TrimSpace(string(data)))
	} else {
		fmt.Println("  daemon: \033[31mstopped\033[0m")
	}
}

func Logs(dataDir string, lines int) {
	logPath := filepath.Join(dataDir, "logs", logFile)
	data, err := os.ReadFile(logPath)
	if err != nil {
		fmt.Println("  no logs")
		return
	}
	all := strings.Split(string(data), "\n")
	start := 0
	if len(all) > lines {
		start = len(all) - lines
	}
	for _, l := range all[start:] {
		fmt.Println(l)
	}
}

func isRunning(pidPath string) bool {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds — check if alive
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func autoConnect(spec string, log func(string)) {
	// spec format: user@host:port
	// Creates reverse SSH tunnel from remote back to this device
	// This lets Dragonfly (or any server) reach into the phone

	for {
		log(fmt.Sprintf("connecting tunnel: %s", spec))
		// Use exec to run ssh — Termux has openssh
		cmd := fmt.Sprintf("ssh -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -o StrictHostKeyChecking=no -N %s", spec)
		_ = cmd // would use tools.Exec in real implementation
		time.Sleep(30 * time.Second) // reconnect delay
	}
}

// --- HTTP API Server ---

type apiHandler struct {
	agent *core.Agent
}

func ServeHTTP(agent *core.Agent, port string, openBrowser bool) {
	h := &apiHandler{agent: agent}

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.webchat)
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/run", h.run)
	mux.HandleFunc("/search", h.search)
	mux.HandleFunc("/status", h.status)
	mux.HandleFunc("/exec", h.execCmd)
	mux.HandleFunc("/api/sessions", h.sessions)
	mux.HandleFunc("/api/sessions/", h.sessionByID)

	addr := "0.0.0.0:" + port
	fmt.Printf("  serving on %s\n", addr)

	if openBrowser {
		go func() {
			time.Sleep(300 * time.Millisecond)
			openURL(fmt.Sprintf("http://localhost:%s", port))
		}()
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("  server error: %s\n", err)
	}
}

func openURL(url string) {
	// On Android/Termux, faccessat2 syscall may be blocked by seccomp,
	// causing exec.Command/LookPath to crash with SIGSYS.
	// Use direct path lookup instead of exec.Command which calls LookPath.
	paths := []string{
		"/data/data/com.termux/files/usr/bin/termux-open-url",
		"/data/data/com.termux/files/usr/bin/am",
		"/usr/bin/xdg-open",
		"/usr/bin/open",
	}
	args := map[string][]string{
		"termux-open-url": {url},
		"am":              {"start", "-a", "android.intent.action.VIEW", "-d", url},
		"xdg-open":        {url},
		"open":            {url},
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		base := filepath.Base(p)
		proc := &exec.Cmd{Path: p, Args: append([]string{base}, args[base]...)}
		if err := proc.Start(); err == nil {
			return
		}
	}
}

func (h *apiHandler) webchat(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(webchatHTML))
}

func (h *apiHandler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"version": "0.1.0",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *apiHandler) execCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", 405)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var req struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout"` // seconds, default 30
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if req.Command == "" {
		http.Error(w, "command required", 400)
		return
	}
	if req.Timeout <= 0 {
		req.Timeout = 30
	}

	cmd := &exec.Cmd{
		Path: "/data/data/com.termux/files/usr/bin/sh",
		Args: []string{"sh", "-c", req.Command},
	}
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"output":    string(out),
		"exit_code": exitCode,
	})
}

func (h *apiHandler) run(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", 405)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	result, err := h.agent.Run(req.Prompt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": result})
}

func (h *apiHandler) search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "?q= required", 400)
		return
	}
	k := 5
	if ks := r.URL.Query().Get("k"); ks != "" {
		if ki, err := strconv.Atoi(ks); err == nil {
			k = ki
		}
	}

	results := h.agent.Search(query, k)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *apiHandler) status(w http.ResponseWriter, r *http.Request) {
	stats := h.agent.MemoryStats()
	cfg := h.agent.Config()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider":  cfg.Provider,
		"model":     cfg.Model,
		"documents": stats.Documents,
		"vectors":   stats.Vectors,
	})
}

// --- Session API ---

func (h *apiHandler) sessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		list := h.agent.ListSessions()
		if list == nil {
			list = []core.SessionMeta{}
		}
		active := ""
		if s := h.agent.ActiveSession(); s != nil {
			active = s.ID
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": list,
			"active":   active,
		})
	case "POST":
		var req struct {
			Title string `json:"title"`
		}
		body, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
		json.Unmarshal(body, &req)
		sess := h.agent.NewSession(req.Title)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    sess.ID,
			"title": sess.Title,
		})
	default:
		http.Error(w, "GET or POST", 405)
	}
}

func (h *apiHandler) sessionByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from /api/sessions/{id}[/action]
	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	if id == "" {
		http.Error(w, "session id required", 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == "GET" && action == "":
		// Load session
		sess, err := h.agent.LoadSession(id)
		if err != nil {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       sess.ID,
			"title":    sess.Title,
			"messages": sess.Messages,
		})
	case r.Method == "DELETE" && action == "":
		// Delete session
		if err := h.agent.DeleteSession(id); err != nil {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	case r.Method == "POST" && action == "rename":
		var req struct {
			Title string `json:"title"`
		}
		body, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
		json.Unmarshal(body, &req)
		sess, err := h.agent.LoadSession(id)
		if err != nil {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		sess.Title = req.Title
		json.NewEncoder(w).Encode(map[string]string{"status": "renamed", "title": sess.Title})
	default:
		http.Error(w, "not found", 404)
	}
}

// --- Process Management ---

type ManagedProcess struct {
	PID     int    `json:"pid"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Started string `json:"started"`
}

func ListProcesses(dataDir string) {
	procDir := filepath.Join(dataDir, "processes")
	entries, err := os.ReadDir(procDir)
	if err != nil || len(entries) == 0 {
		fmt.Println("  no managed processes")
		return
	}

	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(procDir, e.Name()))
		if err != nil {
			continue
		}
		var p ManagedProcess
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}

		// Check if still alive
		proc, err := os.FindProcess(p.PID)
		alive := err == nil && proc.Signal(syscall.Signal(0)) == nil

		status := "\033[32malive\033[0m"
		if !alive {
			status = "\033[31mdead\033[0m"
		}

		fmt.Printf("  %d  %s  %s  %s\n", p.PID, status, p.Name, p.Command)
	}
}

func KillProcess(dataDir, target string) {
	pid, err := strconv.Atoi(target)
	if err != nil {
		fmt.Printf("  invalid pid: %s\n", target)
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("  process not found: %d\n", pid)
		return
	}
	err = proc.Kill()
	if err != nil {
		fmt.Printf("  error killing %d: %s\n", pid, err)
		return
	}

	// Clean up process file
	procFile := filepath.Join(dataDir, "processes", fmt.Sprintf("%d.json", pid))
	os.Remove(procFile)

	fmt.Printf("  killed %d\n", pid)
}
