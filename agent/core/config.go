package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key,omitempty"`
	MaxTokens   int    `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	System      string `json:"system"`
	path        string
}

func DefaultConfig() *Config {
	return &Config{
		Provider:    "local",
		Model:       "default",
		BaseURL:     "http://127.0.0.1:8080",
		MaxTokens:   4096,
		Temperature: 0.7,
		System:      "You are a capable agent. You have access to tools: exec (run shell commands), read (read files), write (write files), search (search memory). Use them to help the user. Be direct. Be concise.",
	}
}

func LoadConfig(path string) *Config {
	cfg := DefaultConfig()
	cfg.path = path

	data, err := os.ReadFile(path)
	if err != nil {
		// create workspace + default config
		os.MkdirAll(filepath.Dir(path), 0755)
		cfg.Save(path)
		return cfg
	}

	json.Unmarshal(data, cfg)
	cfg.path = path
	return cfg
}

func (c *Config) Save(path string) error {
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
		switch value {
		case "local":
			c.BaseURL = "http://127.0.0.1:8080"
		case "openai":
			c.BaseURL = "https://api.openai.com"
		case "anthropic":
			c.BaseURL = "https://api.anthropic.com"
		}
	case "model":
		c.Model = value
	case "base_url":
		c.BaseURL = value
	case "api_key":
		c.APIKey = value
	case "system":
		c.System = value
	}
}
