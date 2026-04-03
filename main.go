package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/artifact-virtual/symbiote-android/copilot"
	"github.com/artifact-virtual/symbiote-android/core"
	"github.com/artifact-virtual/symbiote-android/daemon"
	discordbot "github.com/artifact-virtual/symbiote-android/discord"
	"github.com/artifact-virtual/symbiote-android/memory"
	"github.com/artifact-virtual/symbiote-android/network"
	"github.com/artifact-virtual/symbiote-android/shell"
	"github.com/artifact-virtual/symbiote-android/tools"
)

const version = "0.1.0"
const banner = `
  ███████╗██╗   ██╗███╗   ███╗██████╗ ██╗ ██████╗ ████████╗███████╗
  ██╔════╝╚██╗ ██╔╝████╗ ████║██╔══██╗██║██╔═══██╗╚══██╔══╝██╔════╝
  ███████╗ ╚████╔╝ ██╔████╔██║██████╔╝██║██║   ██║   ██║   █████╗  
  ╚════██║  ╚██╔╝  ██║╚██╔╝██║██╔══██╗██║██║   ██║   ██║   ██╔══╝  
  ███████║   ██║   ██║ ╚═╝ ██║██████╔╝██║╚██████╔╝   ██║   ███████╗
  ╚══════╝   ╚═╝   ╚═╝     ╚═╝╚═════╝ ╚═╝ ╚═════╝    ╚═╝   ╚══════╝
`

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		runInteractive()
		return
	}

	switch args[0] {
	// --- Agent modes ---
	case "chat":
		runInteractive()
	case "run":
		if len(args) < 2 {
			fatal("usage: symbiote run <prompt>")
		}
		runOnce(strings.Join(args[1:], " "))

	// --- Start / Stop (primary interface) ---
	case "start":
		runStart(args[1:])
	case "stop":
		runStop()

	// --- Daemon / service ---
	case "daemon":
		runDaemon(args[1:])
	case "serve":
		runServe(args[1:])

	// --- Shell & exec ---
	case "sh", "shell":
		runShell(args[1:])
	case "exec":
		if len(args) < 2 {
			fatal("usage: symbiote exec <command>")
		}
		runExec(strings.Join(args[1:], " "))

	// --- Memory ---
	case "search":
		if len(args) < 2 {
			fatal("usage: symbiote search <query>")
		}
		runSearch(strings.Join(args[1:], " "))
	case "ingest":
		path := "."
		if len(args) > 1 {
			path = args[1]
		}
		runIngest(path)

	// --- Network ---
	case "tunnel":
		runTunnel(args[1:])
	case "scan":
		runScan(args[1:])
	case "proxy":
		runProxy(args[1:])

	// --- Process management ---
	case "ps":
		runPS()
	case "kill":
		if len(args) < 2 {
			fatal("usage: symbiote kill <pid|name>")
		}
		runKill(args[1])

	// --- Config ---
	case "config":
		runConfig(args[1:])
	case "status":
		runStatus()
	case "setup":
		runSetup()
	case "web":
		runWeb(args[1:])

	// --- Copilot ---
	case "copilot":
		runCopilot(args[1:])

	// --- Discord ---
	case "discord":
		runDiscord(args[1:])

	case "version":
		fmt.Printf("symbiote %s (android)\n", version)
	case "help":
		printHelp()

	default:
		// treat as prompt
		runOnce(strings.Join(args, " "))
	}
}

// --- Agent ---

func runInteractive() {
	cfg := core.LoadConfig(configPath())
	mem := memory.New(dataPath())
	t := tools.New(homePath())
	agent := core.NewAgent(cfg, mem, t)

	fmt.Fprint(os.Stderr, banner)
	shell.Banner(version, cfg.Provider, cfg.Model)
	shell.Chat(agent)
}

func runOnce(prompt string) {
	cfg := core.LoadConfig(configPath())
	mem := memory.New(dataPath())
	t := tools.New(homePath())
	agent := core.NewAgent(cfg, mem, t)

	resp, err := agent.Run(prompt)
	if err != nil {
		fatal(err.Error())
	}
	fmt.Print(resp)
}

// --- Start / Stop ---

func runStart(args []string) {
	port := "8422"
	if len(args) > 0 {
		port = args[0]
	}

	// Check if already running
	pidFile := filepath.Join(dataPath(), "spore.pid")
	if data, err := os.ReadFile(pidFile); err == nil {
		pid := strings.TrimSpace(string(data))
		// Check if process is actually alive
		if _, err := os.FindProcess(atoi(pid)); err == nil {
			// Check /proc/<pid> on Linux/Android
			if _, err := os.Stat(fmt.Sprintf("/proc/%s", pid)); err == nil {
				fmt.Printf("  spore already running (pid %s)\n", pid)
				fmt.Printf("  webchat: http://0.0.0.0:%s\n", port)
				return
			}
		}
	}

	// Write our PID
	os.MkdirAll(dataPath(), 0755)
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)

	// Cleanup PID on exit
	defer os.Remove(pidFile)

	cfg := core.LoadConfig(configPath())

	// Auto-start copilot proxy if provider is copilot
	if cfg.Provider == "copilot" {
		copilotPort := cfg.CopilotPort
		if copilotPort == 0 {
			copilotPort = 3000
		}
		proxy := copilot.New(copilotPort, dataPath())
		go func() {
			if err := proxy.ListenAndServe(); err != nil {
				fmt.Printf("  copilot proxy error: %s\n", err)
			}
		}()
		time.Sleep(200 * time.Millisecond)
	}

	mem := memory.New(dataPath())
	t := tools.New(homePath())
	agent := core.NewAgent(cfg, mem, t)

	// Auto-start Discord bot if token configured
	if cfg.DiscordToken != "" {
		bot := discordbot.New(cfg.DiscordToken, cfg.DiscordPrefix, agent, func(s string) {
			fmt.Printf("  [discord] %s\n", s)
		})
		go func() {
			ctx := context.Background()
			if err := bot.Run(ctx); err != nil {
				fmt.Printf("  discord bot error: %s\n", err)
			}
		}()
		fmt.Println("  discord bot starting...")
	}

	fmt.Fprint(os.Stderr, banner)
	fmt.Printf("  spore started (pid %d)\n", os.Getpid())
	fmt.Printf("  webchat: http://0.0.0.0:%s\n", port)
	if cfg.DiscordToken != "" {
		fmt.Println("  discord: connected")
	}
	fmt.Println("  press Ctrl+C to stop")

	daemon.ServeHTTP(agent, port, false) // don't auto-open browser
}

func runStop() {
	pidFile := filepath.Join(dataPath(), "spore.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		// Try pkill as fallback
		fmt.Println("  no pidfile — trying pkill...")
		exec.Command("pkill", "-f", "spore start").Run()
		exec.Command("pkill", "-f", "spore serve").Run()
		os.Remove(pidFile)
		success("spore stopped")
		return
	}

	pid := strings.TrimSpace(string(data))
	fmt.Printf("  stopping spore (pid %s)...\n", pid)

	proc, err := os.FindProcess(atoi(pid))
	if err != nil {
		os.Remove(pidFile)
		success("spore stopped (process not found)")
		return
	}

	// Send SIGTERM
	proc.Signal(os.Interrupt)
	time.Sleep(500 * time.Millisecond)
	proc.Kill() // force if still alive

	os.Remove(pidFile)
	success("spore stopped")
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// --- Daemon ---

func runDaemon(args []string) {
	if len(args) == 0 {
		args = []string{"start"}
	}
	cfg := core.LoadConfig(configPath())

	switch args[0] {
	case "start":
		daemon.Start(cfg, dataPath())
	case "stop":
		daemon.Stop(dataPath())
	case "status":
		daemon.Status(dataPath())
	case "log", "logs":
		daemon.Logs(dataPath(), 50)
	default:
		fatal("daemon: start|stop|status|logs")
	}
}

func runServe(args []string) {
	port := "8422"
	if len(args) > 0 {
		port = args[0]
	}
	cfg := core.LoadConfig(configPath())

	// Auto-start copilot proxy if provider is copilot
	if cfg.Provider == "copilot" {
		copilotPort := cfg.CopilotPort
		if copilotPort == 0 {
			copilotPort = 3000
		}
		proxy := copilot.New(copilotPort, dataPath())
		go func() {
			if err := proxy.ListenAndServe(); err != nil {
				fmt.Printf("  copilot proxy error: %s\n", err)
			}
		}()
		// Wait briefly for proxy to start
		time.Sleep(200 * time.Millisecond)
	}

	mem := memory.New(dataPath())
	t := tools.New(homePath())
	agent := core.NewAgent(cfg, mem, t)

	// Auto-start Discord bot if token configured
	if cfg.DiscordToken != "" {
		bot := discordbot.New(cfg.DiscordToken, cfg.DiscordPrefix, agent, func(s string) {
			fmt.Printf("  [discord] %s\n", s)
		})
		go func() {
			ctx := context.Background()
			if err := bot.Run(ctx); err != nil {
				fmt.Printf("  discord bot error: %s\n", err)
			}
		}()
		fmt.Println("  discord bot starting...")
	}

	daemon.ServeHTTP(agent, port, true) // open webchat in browser
}

// --- Shell ---

func runShell(args []string) {
	shell.Interactive(homePath())
}

func runExec(command string) {
	t := tools.New(homePath())
	fmt.Print(t.Exec(command))
}

// --- Memory ---

func runSearch(query string) {
	mem := memory.New(dataPath())
	results := mem.Search(query, 5)
	if len(results) == 0 {
		dim("no results")
		return
	}
	for _, r := range results {
		shell.SearchResult(r.Path, r.Score, r.Chunk)
	}
}

func runIngest(path string) {
	mem := memory.New(dataPath())
	n, err := mem.Ingest(path)
	if err != nil {
		fatal(err.Error())
	}
	success(fmt.Sprintf("indexed %d documents", n))
}

// --- Network ---

func runTunnel(args []string) {
	if len(args) < 1 {
		fatal("usage: symbiote tunnel <local:remote:host> | symbiote tunnel reverse <remote:local:host>")
	}
	if args[0] == "reverse" {
		if len(args) < 2 {
			fatal("usage: symbiote tunnel reverse <remote:local:host>")
		}
		network.ReverseTunnel(args[1])
	} else {
		network.ForwardTunnel(args[0])
	}
}

func runScan(args []string) {
	target := "192.168.1.0/24"
	if len(args) > 0 {
		target = args[0]
	}
	network.Scan(target)
}

func runProxy(args []string) {
	port := "1080"
	if len(args) > 0 {
		port = args[0]
	}
	network.SOCKSProxy(port)
}

// --- Process management ---

func runPS() {
	daemon.ListProcesses(dataPath())
}

func runKill(target string) {
	daemon.KillProcess(dataPath(), target)
}

// --- Config ---

func runConfig(args []string) {
	cfg := core.LoadConfig(configPath())
	if len(args) == 0 {
		shell.PrintConfig(cfg)
		return
	}
	if len(args) < 2 {
		fatal("usage: symbiote config <key> <value>")
	}
	cfg.Set(args[0], args[1])
	cfg.Save(configPath())
	success(fmt.Sprintf("%s = %s", args[0], args[1]))
}

func runStatus() {
	cfg := core.LoadConfig(configPath())
	mem := memory.New(dataPath())
	shell.Status(cfg, mem.Stats(), dataPath())
}

func runSetup() {
	core.Setup(homePath(), dataPath())
}

func runWeb(args []string) {
	port := "8422"
	if len(args) > 0 {
		port = args[0]
	}
	cfg := core.LoadConfig(configPath())
	mem := memory.New(dataPath())
	t := tools.New(homePath())
	agent := core.NewAgent(cfg, mem, t)
	fmt.Println("  opening webchat...")
	daemon.ServeHTTP(agent, port, true)
}

func runDiscord(args []string) {
	cfg := core.LoadConfig(configPath())
	token := cfg.DiscordToken
	if token == "" {
		fatal("no discord_token configured — run: spore config discord_token <token>")
	}
	mem := memory.New(dataPath())
	t := tools.New(homePath())
	agent := core.NewAgent(cfg, mem, t)

	bot := discordbot.New(token, cfg.DiscordPrefix, agent, func(s string) {
		fmt.Printf("  [discord] %s\n", s)
	})
	fmt.Println("  starting Discord bot...")
	ctx := context.Background()
	if err := bot.Run(ctx); err != nil {
		fatal(err.Error())
	}
}

func runCopilot(args []string) {
	if len(args) == 0 {
		args = []string{"start"}
	}
	switch args[0] {
	case "auth":
		auth := copilot.NewAuth(dataPath())
		if err := auth.DeviceAuth(); err != nil {
			fatal(err.Error())
		}
	case "start":
		port := 3000
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &port)
		}
		proxy := copilot.New(port, dataPath())
		if err := proxy.ListenAndServe(); err != nil {
			fatal(err.Error())
		}
	case "health":
		// Quick health check
		resp, err := http.Get("http://127.0.0.1:3000/health")
		if err != nil {
			fatal("proxy not running: " + err.Error())
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
	default:
		fatal("copilot: auth | start [port] | health")
	}
}

// --- Paths (Termux-aware) ---

func homePath() string {
	// Termux sets HOME to /data/data/com.termux/files/home
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	// fallback: check PREFIX (Termux env)
	if p := os.Getenv("PREFIX"); p != "" {
		return filepath.Join(filepath.Dir(p), "home")
	}
	return "/data/data/com.termux/files/home"
}

func dataPath() string {
	return filepath.Join(homePath(), ".symbiote")
}

func configPath() string {
	return filepath.Join(dataPath(), "config.json")
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `
  %s  symbiote for android — full agentic runtime

  %sstart / stop%s
    symbiote start [port]      start everything (webchat + discord + copilot)
    symbiote stop              stop spore

  %sagent%s
    symbiote                   interactive chat
    symbiote run <prompt>      single-shot agent
    symbiote chat              interactive mode

  %sdaemon%s
    symbiote daemon start      start background agent
    symbiote daemon stop       stop daemon
    symbiote daemon status     check daemon
    symbiote daemon logs       tail daemon logs
    symbiote serve [port]      HTTP API server (default: 8422)
    symbiote web [port]        open webchat in browser

  %sshell%s
    symbiote sh                interactive shell with tools
    symbiote exec <command>    run a command

  %smemory%s
    symbiote search <query>    search indexed files
    symbiote ingest [path]     index files

  %snetwork%s
    symbiote tunnel L:R:host   forward tunnel
    symbiote tunnel reverse R:L:host   reverse tunnel
    symbiote scan [target]     network scan (default: 192.168.1.0/24)
    symbiote proxy [port]      SOCKS5 proxy (default: 1080)

  %sprocess%s
    symbiote ps                list managed processes
    symbiote kill <pid|name>   kill a managed process

  %sconfig%s
    symbiote config            show config
    symbiote config <k> <v>    set config value
    symbiote setup             first-time setup
    symbiote status            full system status

  %sproviders%s
    symbiote config provider copilot     GitHub Copilot (default, built-in proxy)
    symbiote config provider ollama      Ollama (local, toggle with: config ollama on)
    symbiote config provider openai      OpenAI API
    symbiote config provider anthropic   Anthropic API  
    symbiote config provider local       llamafile
    symbiote config provider custom      any OpenAI-compatible

  %scopilot%s
    symbiote copilot auth       authenticate with GitHub
    symbiote copilot start      start proxy standalone
    symbiote copilot health     check proxy status

`, banner, bold, reset, bold, reset, bold, reset, bold, reset, bold, reset, bold, reset, bold, reset, bold, reset, bold, reset, bold, reset)
}

// --- Output helpers ---
const (
	reset = "\033[0m"
	bold  = "\033[1m"
	dim_  = "\033[2m"
	red   = "\033[31m"
	green = "\033[32m"
)

func fatal(msg string) {
	fmt.Fprintf(os.Stderr, "  %s%serror%s %s\n", bold, red, reset, msg)
	os.Exit(1)
}

func success(msg string) {
	fmt.Fprintf(os.Stderr, "  %s%s%s\n", green, msg, reset)
}

func dim(msg string) {
	fmt.Fprintf(os.Stderr, "  %s%s%s\n", dim_, msg, reset)
}
