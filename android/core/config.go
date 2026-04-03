package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/artifact-virtual/symbiote-android/provider"
)

type Config struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	BaseURL     string  `json:"base_url"`
	APIKey      string  `json:"api_key,omitempty"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	System      string  `json:"system"`

	// Daemon settings
	DaemonPort string `json:"daemon_port,omitempty"`

	// Agent settings
	MaxIterations int `json:"max_iterations,omitempty"`

	// Discord bot
	DiscordToken  string `json:"discord_token,omitempty"`
	DiscordPrefix string `json:"discord_prefix,omitempty"`

	// Identity
	DeviceName string `json:"device_name,omitempty"`
	DeviceID   string `json:"device_id,omitempty"`

	path string
}

func DefaultConfig() *Config {
	hostname, _ := os.Hostname()
	return &Config{
		Provider:      "ollama",
		Model:         "llama3.2:3b",
		BaseURL:       "http://127.0.0.1:11434",
		MaxTokens:     4096,
		Temperature:   0.7,
		MaxIterations: 25,
		System:        DefaultSystemPrompt,
		DaemonPort:    "8422",
		DiscordPrefix: "!",
		DeviceName:    hostname,
	}
}

const DefaultSystemPrompt = `You are Spore, an autonomous agent on AEGIS (Samsung Z Fold 5, Android/Termux). You ACT — you don't describe, ask, or narrate.

RULES:
- Execute first, report briefly after. "Launch chrome" → call app_launch, say "Done." Not a paragraph.
- NEVER ask clarifying questions unless genuinely ambiguous. Default to action.
- Keep responses to 1-2 sentences max. The tool results speak for themselves.
- If something fails, try a different approach silently. Only report the final outcome.
- You have full device control: apps, shell, files, brightness, volume, torch, clipboard, TTS, camera, SMS, notifications, MacroDroid.
- If the user says "it didn't work" — investigate (check processes, try alternative commands) instead of asking what they mean.`

func LoadConfig(path string) *Config {
	cfg := DefaultConfig()
	cfg.path = path

	data, err := os.ReadFile(path)
	if err != nil {
		os.MkdirAll(filepath.Dir(path), 0755)
		cfg.Save(path)
		return cfg
	}

	json.Unmarshal(data, cfg)
	cfg.path = path
	return cfg
}

func (c *Config) Save(path string) error {
	if path == "" {
		path = c.path
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *Config) Set(key, value string) {
	switch key {
	case "provider":
		c.Provider = value
		// Auto-set base URL from registry
		info := provider.LookupProvider(value)
		if info != nil {
			c.BaseURL = info.BaseURL
		}
	case "model":
		c.Model = value
	case "base_url":
		c.BaseURL = value
	case "api_key":
		c.APIKey = value
	case "system":
		c.System = value
	case "daemon_port":
		c.DaemonPort = value
	case "device_name":
		c.DeviceName = value
	case "discord_token":
		c.DiscordToken = value
	case "discord_prefix":
		c.DiscordPrefix = value
	case "max_iterations":
		fmt.Sscanf(value, "%d", &c.MaxIterations)
	case "max_tokens":
		fmt.Sscanf(value, "%d", &c.MaxTokens)
	case "temperature":
		fmt.Sscanf(value, "%f", &c.Temperature)
	}
}

func (c *Config) ConfigPath() string {
	return c.path
}

// ToProviderConfig converts to provider.ProviderConfig
func (c *Config) ToProviderConfig() provider.ProviderConfig {
	return provider.ProviderConfig{
		Provider:    c.Provider,
		Model:       c.Model,
		BaseURL:     c.BaseURL,
		APIKey:      c.APIKey,
		MaxTokens:   c.MaxTokens,
		Temperature: c.Temperature,
	}
}
