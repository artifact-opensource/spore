package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/artifact-virtual/spore/core"
	"github.com/artifact-virtual/spore/memory"
	"github.com/artifact-virtual/spore/terminal"
	"github.com/artifact-virtual/spore/tools"
)

const version = "0.1.0"

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		runChat()
		return
	}

	switch args[0] {
	case "chat":
		runChat()
	case "run":
		if len(args) < 2 {
			terminal.Error("usage: spore run <prompt>")
			os.Exit(1)
		}
		runOnce(strings.Join(args[1:], " "))
	case "search":
		if len(args) < 2 {
			terminal.Error("usage: spore search <query>")
			os.Exit(1)
		}
		runSearch(strings.Join(args[1:], " "))
	case "ingest":
		path := "."
		if len(args) > 1 {
			path = args[1]
		}
		runIngest(path)
	case "config":
		runConfig(args[1:])
	case "status":
		runStatus()
	case "version":
		fmt.Printf("spore %s\n", version)
	case "help":
		printHelp()
	default:
		runOnce(strings.Join(args, " "))
	}
}

func runChat() {
	cfg := core.LoadConfig(configPath())
	mem := memory.New(workspacePath())
	t := tools.New(workspacePath())
	agent := core.NewAgent(cfg, mem, t)

	terminal.Banner(version, cfg.Provider, cfg.Model)
	terminal.Chat(agent)
}

func runOnce(prompt string) {
	cfg := core.LoadConfig(configPath())
	mem := memory.New(workspacePath())
	t := tools.New(workspacePath())
	agent := core.NewAgent(cfg, mem, t)

	resp, err := agent.Run(prompt)
	if err != nil {
		terminal.Error(err.Error())
		os.Exit(1)
	}
	fmt.Print(resp)
}

func runSearch(query string) {
	mem := memory.New(workspacePath())
	results := mem.Search(query, 5)
	if len(results) == 0 {
		terminal.Dim("no results")
		return
	}
	for _, r := range results {
		terminal.SearchResult(r.Path, r.Score, r.Chunk)
	}
}

func runIngest(path string) {
	mem := memory.New(workspacePath())
	n, err := mem.Ingest(path)
	if err != nil {
		terminal.Error(err.Error())
		os.Exit(1)
	}
	terminal.Success(fmt.Sprintf("indexed %d documents", n))
}

func runConfig(args []string) {
	cfg := core.LoadConfig(configPath())
	if len(args) == 0 {
		terminal.PrintConfig(cfg)
		return
	}
	if len(args) < 2 {
		terminal.Error("usage: spore config <key> <value>")
		os.Exit(1)
	}
	cfg.Set(args[0], args[1])
	cfg.Save(configPath())
	terminal.Success(fmt.Sprintf("%s = %s", args[0], args[1]))
}

func runStatus() {
	cfg := core.LoadConfig(configPath())
	mem := memory.New(workspacePath())
	terminal.Status(cfg, mem.Stats())
}

func printHelp() {
	terminal.Help()
}

func workspacePath() string {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(dir, "workspace")); err == nil {
		return filepath.Join(dir, "workspace")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".spore", "workspace")
}

func configPath() string {
	ws := workspacePath()
	return filepath.Join(ws, "config.json")
}
