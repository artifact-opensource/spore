package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// Storage and sharing
	StorageDir           string `json:"storage_dir,omitempty"`
	SharedDir            string `json:"shared_dir,omitempty"`
	SecondaryFirmwareDir string `json:"secondary_firmware_dir,omitempty"`
	RGBProfile           string `json:"rgb_profile,omitempty"`

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
		Provider:             "ollama",
		Model:                "llama3.2:3b",
		BaseURL:              "http://127.0.0.1:11434",
		MaxTokens:            4096,
		Temperature:          0.7,
		MaxIterations:        25,
		System:               DefaultSystemPrompt,
		DaemonPort:           "8422",
		DiscordPrefix:        "!",
		DeviceName:           hostname,
		RGBProfile:           "spore-default",
		StorageDir:           "storage",
		SharedDir:            "shared",
		SecondaryFirmwareDir: "secondary",
	}
}

// XboxConfig returns a config preset for Xbox Dev Mode deployment.
// Uses local Ollama with qwen3.5:9b — fits in Xbox's 16GB shared memory.
func XboxConfig() *Config {
	return &Config{
		Provider:             "local",
		Model:                "qwen3.5:9b",
		BaseURL:              "http://127.0.0.1:8080/v1",
		MaxTokens:            4096,
		Temperature:          0.7,
		MaxIterations:        25,
		System:               XboxSystemPrompt,
		DaemonPort:           "8422",
		DiscordPrefix:        "!",
		DeviceName:           "xbox",
		RGBProfile:           "spore-default",
		StorageDir:           "storage",
		SharedDir:            "shared",
		SecondaryFirmwareDir: "secondary",
	}
}

const XboxSystemPrompt = `You are Spore, an autonomous agent running on Xbox in Dev Mode. You ACT — you don't describe, ask, or narrate.

RULES:
- Execute first, report briefly after. Keep responses to 1-2 sentences max.
- You have full system access: GPU monitoring, process management, file serving, network tools.
- Use gpu_status to check GPU temp/VRAM, service_manager to manage processes, system_info for hardware stats.
- Use file_server to share files across devices on the network.
- If something fails, try a different approach silently. Only report the final outcome.
- You are running on Xbox hardware with a custom AMD APU (Zen 2 + RDNA 2). dxdiag and WMI are your primary GPU info sources.`

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
		cfg.normalize()
		cfg.Save(path)
		return cfg
	}

	json.Unmarshal(data, cfg)
	cfg.path = path
	cfg.normalize()
	return cfg
}

// LoadProfile loads a named configuration profile.
// Supported profiles: "default", "xbox"
func LoadProfile(name, path string) *Config {
	switch name {
	case "xbox":
		cfg := XboxConfig()
		cfg.path = path
		cfg.normalize()
		return cfg
	default:
		return LoadConfig(path)
	}
}

func (c *Config) Save(path string) error {
	if path == "" {
		path = c.path
	}
	c.path = path
	c.normalize()
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
		info := provider.LookupProvider(value)
		if info != nil {
			c.Provider = info.ID
			c.BaseURL = info.BaseURL
		} else {
			c.Provider = value
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
	case "storage_dir":
		c.StorageDir = value
	case "shared_dir":
		oldShared := c.SharedDir
		c.SharedDir = value
		if c.SecondaryFirmwareDir == "" || (oldShared != "" && strings.HasPrefix(filepath.Clean(c.SecondaryFirmwareDir), filepath.Clean(oldShared))) {
			c.SecondaryFirmwareDir = "secondary"
		}
	case "secondary_firmware_dir":
		c.SecondaryFirmwareDir = value
	case "rgb_profile":
		c.RGBProfile = value
	case "max_iterations":
		fmt.Sscanf(value, "%d", &c.MaxIterations)
	case "max_tokens":
		fmt.Sscanf(value, "%d", &c.MaxTokens)
	case "temperature":
		fmt.Sscanf(value, "%f", &c.Temperature)
	}
	c.normalize()
}

func (c *Config) ConfigPath() string {
	return c.path
}

func (c *Config) DataDir() string {
	if c.path == "" {
		return ""
	}
	return filepath.Dir(c.path)
}

func (c *Config) normalize() {
	dataDir := c.DataDir()
	if dataDir == "" {
		return
	}

	c.StorageDir = normalizeDir(dataDir, c.StorageDir, "storage")
	c.SharedDir = normalizeDir(dataDir, c.SharedDir, "shared")
	c.SecondaryFirmwareDir = normalizeDir(c.SharedDir, c.SecondaryFirmwareDir, "secondary")
	if c.RGBProfile == "" {
		c.RGBProfile = "spore-default"
	}

	managedDirs := []struct {
		root string
		dir  string
	}{
		{root: dataDir, dir: c.StorageDir},
		{root: dataDir, dir: c.SharedDir},
		{root: c.SharedDir, dir: c.SecondaryFirmwareDir},
	}
	for _, item := range managedDirs {
		ensureDirWithinRoot(item.root, item.dir)
	}
}

func normalizeDir(base, value, fallback string) string {
	if value == "" {
		value = fallback
	}
	if filepath.IsAbs(value) {
		rel, err := filepath.Rel(base, value)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return filepath.Clean(value)
		}
		value = filepath.Base(value)
	}
	return filepath.Clean(filepath.Join(base, value))
}

func pathWithinRoot(root, candidate string) bool {
	if root == "" || candidate == "" {
		return false
	}
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func ensureDirWithinRoot(root, dir string) {
	if !pathWithinRoot(root, dir) {
		return
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return
	}
	safeDir, err := joinWithinRoot(root, rel)
	if err == nil && pathWithinRoot(root, safeDir) {
		os.MkdirAll(safeDir, 0755)
	}
}

func joinWithinRoot(root, rel string) (string, error) {
	rel = filepath.Clean(rel)
	candidate := filepath.Clean(filepath.Join(root, rel))
	relCheck, err := filepath.Rel(root, candidate)
	if err != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes root")
	}
	return candidate, nil
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
