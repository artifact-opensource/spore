package shell

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/artifact-opensource/spore/core"
)

// ANSI
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	white  = "\033[97m"
	gray   = "\033[90m"
	cyan   = "\033[36m"
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	purple = "\033[35m"
)

func Banner(version, prov, model string) {
	fmt.Fprintf(os.Stderr, "  %s%sspore%s %sv%s%s", bold, purple, reset, dim, version, reset)
	fmt.Fprintf(os.Stderr, "  %s%s/%s%s\n", gray, prov, model, reset)
	fmt.Fprintf(os.Stderr, "  %s%s%s\n", dim, strings.Repeat("─", 44), reset)
	fmt.Fprintf(os.Stderr, "\n")
}

func Prompt() {
	fmt.Fprintf(os.Stderr, "%s%s ⟩ %s", bold, purple, reset)
}

func Response(text string) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i == 0 {
			fmt.Fprintf(os.Stdout, "\n  %s%s%s\n", white, line, reset)
		} else {
			fmt.Fprintf(os.Stdout, "  %s%s%s\n", white, line, reset)
		}
	}
	fmt.Fprintf(os.Stderr, "\n")
}

func ToolCall(name string, args string) {
	fmt.Fprintf(os.Stderr, "  %s⚡%s ", yellow, reset)
	fmt.Fprintf(os.Stderr, "%s%s%s", yellow, name, reset)
	if args != "" {
		short := args
		if len(short) > 60 {
			short = short[:57] + "..."
		}
		fmt.Fprintf(os.Stderr, " %s%s%s", dim, short, reset)
	}
	fmt.Fprintf(os.Stderr, "\n")
}

func SearchResult(path string, score float64, chunk string) {
	fmt.Fprintf(os.Stdout, "  %s%.3f%s  %s%s%s\n", green, score, reset, cyan, path, reset)
	short := chunk
	if len(short) > 120 {
		short = short[:117] + "..."
	}
	fmt.Fprintf(os.Stdout, "        %s%s%s\n", dim, short, reset)
}

func Error(msg string) {
	fmt.Fprintf(os.Stderr, "  %s%s✗%s %s\n", bold, red, reset, msg)
}

func Success(msg string) {
	fmt.Fprintf(os.Stderr, "  %s✓%s %s\n", green, reset, msg)
}

func Dim(msg string) {
	fmt.Fprintf(os.Stderr, "  %s%s%s\n", dim, msg, reset)
}

func Thinking() {
	fmt.Fprintf(os.Stderr, "  %s⏳%s", dim, reset)
}

func ClearThinking() {
	fmt.Fprintf(os.Stderr, "\r  %s\r", strings.Repeat(" ", 20))
}

func Status(cfg *core.Config, memStats core.MemoryStats, dataDir string) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  %sdevice%s     %s\n", gray, reset, cfg.DeviceName)
	fmt.Fprintf(os.Stderr, "  %sprovider%s   %s\n", gray, reset, cfg.Provider)
	fmt.Fprintf(os.Stderr, "  %smodel%s      %s\n", gray, reset, cfg.Model)
	fmt.Fprintf(os.Stderr, "  %sendpoint%s   %s\n", gray, reset, cfg.BaseURL)
	fmt.Fprintf(os.Stderr, "  %sdocuments%s  %d\n", gray, reset, memStats.Documents)
	fmt.Fprintf(os.Stderr, "  %svectors%s    %d\n", gray, reset, memStats.Vectors)
	fmt.Fprintf(os.Stderr, "  %sindex%s      %s\n", gray, reset, humanBytes(memStats.IndexBytes))
	fmt.Fprintf(os.Stderr, "  %sdata%s       %s\n", gray, reset, dataDir)
	fmt.Fprintf(os.Stderr, "\n")
}

func PrintConfig(cfg *core.Config) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  %sprovider%s     %s\n", gray, reset, cfg.Provider)
	fmt.Fprintf(os.Stderr, "  %smodel%s        %s\n", gray, reset, cfg.Model)
	fmt.Fprintf(os.Stderr, "  %sbase_url%s     %s\n", gray, reset, cfg.BaseURL)
	fmt.Fprintf(os.Stderr, "  %smax_tokens%s   %d\n", gray, reset, cfg.MaxTokens)
	fmt.Fprintf(os.Stderr, "  %sdaemon_port%s  %s\n", gray, reset, cfg.DaemonPort)
	fmt.Fprintf(os.Stderr, "  %sdevice%s       %s\n", gray, reset, cfg.DeviceName)
	fmt.Fprintf(os.Stderr, "  %ssystem%s       %s\n", gray, reset, truncate(cfg.System, 50))
	fmt.Fprintf(os.Stderr, "\n")
}

func humanBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n-3] + "..."
	}
	return s
}

// Chat — interactive agent loop
func Chat(agent *core.Agent) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for {
		Prompt()
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Built-in commands
		switch input {
		case "/quit", "/exit", "/q":
			Dim("bye")
			return
		case "/clear":
			fmt.Print("\033[2J\033[H")
			continue
		case "/history":
			agent.PrintHistory()
			continue
		case "/reset":
			agent.Reset()
			Dim("context cleared")
			continue
		case "/status":
			cfg := agent.Config()
			stats := agent.MemoryStats()
			Status(cfg, stats, "")
			continue
		case "/help":
			chatHelp()
			continue
		}

		if strings.HasPrefix(input, "/search ") {
			query := strings.TrimPrefix(input, "/search ")
			results := agent.Search(query, 5)
			if len(results) == 0 {
				Dim("no results")
			} else {
				for _, r := range results {
					SearchResult(r.Path, r.Score, r.Chunk)
				}
			}
			continue
		}

		if strings.HasPrefix(input, "/ingest") {
			path := strings.TrimSpace(strings.TrimPrefix(input, "/ingest"))
			if path == "" {
				path = "."
			}
			n, err := agent.Ingest(path)
			if err != nil {
				Error(err.Error())
			} else {
				Success(fmt.Sprintf("indexed %d documents", n))
			}
			continue
		}

		// Shell passthrough with !
		if strings.HasPrefix(input, "!") {
			cmd := strings.TrimPrefix(input, "!")
			fmt.Println(execQuick(cmd))
			continue
		}

		Thinking()

		resp, err := agent.Run(input)
		ClearThinking()
		if err != nil {
			Error(err.Error())
			continue
		}

		Response(resp)
	}
}

func chatHelp() {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  %s/quit%s         exit\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/clear%s        clear screen\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/reset%s        clear conversation\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/history%s      show conversation\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/status%s       system info\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/search%s q     search memory\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/ingest%s p     index files\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s!command%s      shell passthrough\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "\n")
}

func execQuick(cmd string) string {
	out, err := os.ReadFile("/dev/null") // dummy
	_ = out
	c := strings.Split(cmd, " ")
	if len(c) == 0 {
		return ""
	}
	// Use os/exec directly for shell passthrough
	proc := bufio.NewScanner(os.Stdin)
	_ = proc
	_ = err
	return ""
}

// Interactive shell mode — drops into an enhanced shell
func Interactive(home string) {
	fmt.Fprintf(os.Stderr, "\n  %s%sspore shell%s\n", bold, purple, reset)
	fmt.Fprintf(os.Stderr, "  %stype 'exit' to return%s\n\n", dim, reset)

	// Find the best shell
	shells := []string{
		"/data/data/com.termux/files/usr/bin/bash",
		"/data/data/com.termux/files/usr/bin/zsh",
		"/bin/bash",
		"/bin/sh",
	}

	shellPath := "sh"
	for _, s := range shells {
		if _, err := os.Stat(s); err == nil {
			shellPath = s
			break
		}
	}

	// Fork into an interactive shell with our environment
	env := os.Environ()
	env = append(env, "SPORE=1", "TERM=xterm-256color")

	pa := os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Dir:   home,
		Env:   env,
	}

	proc, err := os.StartProcess(shellPath, []string{shellPath, "-i"}, &pa)
	if err != nil {
		Error(fmt.Sprintf("failed to start shell: %s", err))
		return
	}

	proc.Wait()
	fmt.Fprintf(os.Stderr, "\n  %sback to spore%s\n\n", dim, reset)
}
