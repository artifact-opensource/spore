package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/artifact-virtual/spore/core"
)

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

		// built-in commands
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
			runStatusInChat(agent)
			continue
		case "/help":
			chatHelp()
			continue
		}

		// check for /search command
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

		// check for /ingest command
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

func runStatusInChat(agent *core.Agent) {
	cfg := agent.Config()
	stats := agent.MemoryStats()
	Status(cfg, stats)
}

func chatHelp() {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  %s/quit%s        exit\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/clear%s       clear screen\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/reset%s       clear conversation\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/history%s     show conversation\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/status%s      system info\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/search%s q    search memory\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "  %s/ingest%s p    index files\n", cyan, reset)
	fmt.Fprintf(os.Stderr, "\n")
}
