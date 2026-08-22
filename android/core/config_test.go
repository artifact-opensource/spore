package core

import (
	"path/filepath"
	"testing"
)

func TestLoadConfigCreatesSharedRuntimeDirs(t *testing.T) {
	tmp := t.TempDir()
	cfg := LoadConfig(filepath.Join(tmp, "config.json"))

	if cfg.SharedDir != filepath.Join(tmp, "shared") {
		t.Fatalf("shared dir = %q", cfg.SharedDir)
	}
	if cfg.StorageDir != filepath.Join(tmp, "storage") {
		t.Fatalf("storage dir = %q", cfg.StorageDir)
	}
	if cfg.SecondaryFirmwareDir != filepath.Join(tmp, "shared", "secondary") {
		t.Fatalf("secondary dir = %q", cfg.SecondaryFirmwareDir)
	}
}

func TestSetSharedDirMovesSecondaryFirmwareDir(t *testing.T) {
	tmp := t.TempDir()
	cfg := LoadConfig(filepath.Join(tmp, "config.json"))
	cfg.Set("shared_dir", "dropbox")

	if cfg.SharedDir != filepath.Join(tmp, "dropbox") {
		t.Fatalf("shared dir = %q", cfg.SharedDir)
	}
	if cfg.SecondaryFirmwareDir != filepath.Join(tmp, "dropbox", "secondary") {
		t.Fatalf("secondary dir = %q", cfg.SecondaryFirmwareDir)
	}
}

func TestSetProviderCanonicalizesGeminiAlias(t *testing.T) {
	tmp := t.TempDir()
	cfg := LoadConfig(filepath.Join(tmp, "config.json"))
	cfg.Set("provider", "gemini")

	if cfg.Provider != "google" {
		t.Fatalf("provider = %q", cfg.Provider)
	}
	if cfg.BaseURL != "https://generativelanguage.googleapis.com/v1beta/openai" {
		t.Fatalf("base url = %q", cfg.BaseURL)
	}
}
