package terminal

import (
	"fmt"
	"os"
	"strings"

	"github.com/artifact-virtual/spore/core"
)

// ANSI
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	white   = "\033[97m"
	gray    = "\033[90m"
	cyan    = "\033[36m"
	green   = "\033[32m"
	red     = "\033[31m"
	yellow  = "\033[33m"
)

func Banner(version, prov, model string) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  %s%sspore%s %sv%s%s", bold, white, reset, dim, version, reset)
	fmt.Fprintf(os.Stderr, "  %s%s/%s%s\n", gray, prov, model, reset)
	fmt.Fprintf(os.Stderr, "  %s%s%s\n", dim, strings.Repeat("-", 40), reset)
	fmt.Fprintf(os.Stderr, "\n")
}

func Prompt() {
	fmt.Fprintf(os.Stderr, "%s%s > %s", bold, cyan, reset)
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
	fmt.Fprintf(os.Stderr, "  %s%s%s", dim, gray, reset)
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
	fmt.Fprintf(os.Stderr, "  %s%serror%s %s\n", bold, red, reset, msg)
}

func Success(msg string) {
	fmt.Fprintf(os.Stderr, "  %s%s%s\n", green, msg, reset)
}

func Dim(msg string) {
	fmt.Fprintf(os.Stderr, "  %s%s%s\n", dim, msg, reset)
}

func Thinking() {
	fmt.Fprintf(os.Stderr, "  %s...%s", dim, reset)
}

func ClearThinking() {
	fmt.Fprintf(os.Stderr, "\r  %s\r", strings.Repeat(" ", 20))
}

func Status(cfg *core.Config, memStats core.MemoryStats) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  %sprovider%s   %s\n", gray, reset, cfg.Provider)
	fmt.Fprintf(os.Stderr, "  %smodel%s      %s\n", gray, reset, cfg.Model)
	fmt.Fprintf(os.Stderr, "  %sendpoint%s   %s\n", gray, reset, cfg.BaseURL)
	fmt.Fprintf(os.Stderr, "  %sdocuments%s  %d\n", gray, reset, memStats.Documents)
	fmt.Fprintf(os.Stderr, "  %svectors%s    %d\n", gray, reset, memStats.Vectors)
	fmt.Fprintf(os.Stderr, "  %sindex%s      %s\n", gray, reset, humanBytes(memStats.IndexBytes))
	fmt.Fprintf(os.Stderr, "\n")
}

func PrintConfig(cfg *core.Config) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  %sprovider%s    %s\n", gray, reset, cfg.Provider)
	fmt.Fprintf(os.Stderr, "  %smodel%s       %s\n", gray, reset, cfg.Model)
	fmt.Fprintf(os.Stderr, "  %sbase_url%s    %s\n", gray, reset, cfg.BaseURL)
	fmt.Fprintf(os.Stderr, "  %smax_tokens%s  %d\n", gray, reset, cfg.MaxTokens)
	fmt.Fprintf(os.Stderr, "  %ssystem%s      %s\n", gray, reset, truncate(cfg.System, 50))
	fmt.Fprintf(os.Stderr, "\n")
}

func Help() {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  %s%sspore%s  portable agentic runtime\n\n", bold, white, reset)
	fmt.Fprintf(os.Stderr, "  %susage%s\n", gray, reset)
	fmt.Fprintf(os.Stderr, "    spore                    interactive chat\n")
	fmt.Fprintf(os.Stderr, "    spore run <prompt>        single-shot execution\n")
	fmt.Fprintf(os.Stderr, "    spore search <query>      search memory\n")
	fmt.Fprintf(os.Stderr, "    spore ingest [path]       index files into memory\n")
	fmt.Fprintf(os.Stderr, "    spore config [key val]    view or set configuration\n")
	fmt.Fprintf(os.Stderr, "    spore status              system status\n")
	fmt.Fprintf(os.Stderr, "    spore version             print version\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  %sproviders%s\n", gray, reset)
	fmt.Fprintf(os.Stderr, "    spore config provider local       llamafile (default)\n")
	fmt.Fprintf(os.Stderr, "    spore config provider openai      OpenAI API\n")
	fmt.Fprintf(os.Stderr, "    spore config provider anthropic   Anthropic API\n")
	fmt.Fprintf(os.Stderr, "    spore config provider custom      any OpenAI-compatible\n")
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
