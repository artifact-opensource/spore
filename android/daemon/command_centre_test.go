package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/artifact-virtual/symbiote-android/core"
	"github.com/artifact-virtual/symbiote-android/tools"
)

type stubMemory struct{}

func (stubMemory) Search(string, int) []core.SearchResult { return nil }
func (stubMemory) Ingest(string) (int, error)             { return 0, nil }
func (stubMemory) Stats() core.MemoryStats                { return core.MemoryStats{} }

func newTestHandler(t *testing.T) (*apiHandler, *core.Config) {
	t.Helper()
	dataDir := t.TempDir()
	cfg := core.LoadConfig(filepath.Join(dataDir, "config.json"))
	tb := tools.New(dataDir)
	agent := core.NewAgent(cfg, stubMemory{}, tb)
	return &apiHandler{agent: agent}, cfg
}

func TestCommandCentreAPIListsSharedFirmware(t *testing.T) {
	h, cfg := newTestHandler(t)
	firmware := filepath.Join(cfg.SharedDir, "firmware.bin")
	if err := os.WriteFile(firmware, []byte("bin"), 0644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/command-centre", nil)
	w := httptest.NewRecorder()
	h.commandCentreAPI(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var state commandCentreState
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if len(state.Providers) == 0 {
		t.Fatal("expected providers")
	}
	found := false
	for _, item := range state.FirmwareCatalog {
		if item.Source == "shared:firmware.bin" && item.Path == "firmware.bin" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected firmware %s in catalog", firmware)
	}
}

func TestCommandCentreAPIPersistsSettings(t *testing.T) {
	h, cfg := newTestHandler(t)
	body := strings.NewReader(`{"provider":"google","shared_dir":"exchange","rgb_profile":"quiet-night"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/command-centre", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.commandCentreAPI(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	if cfg.Provider != "google" {
		t.Fatalf("provider = %q", cfg.Provider)
	}
	if cfg.SharedDir != filepath.Join(cfg.DataDir(), "exchange") {
		t.Fatalf("shared dir = %q", cfg.SharedDir)
	}
	if cfg.RGBProfile != "quiet-night" {
		t.Fatalf("rgb profile = %q", cfg.RGBProfile)
	}
}

func TestCommandCentreFirmwareInstallCopiesAsset(t *testing.T) {
	h, cfg := newTestHandler(t)
	source := filepath.Join(cfg.SharedDir, "micropython.uf2")
	if err := os.WriteFile(source, []byte("fw"), 0644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/command-centre/firmware", strings.NewReader(`{"action":"install","source":"shared:micropython.uf2"}`))
	w := httptest.NewRecorder()
	h.commandCentreFirmware(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	target := filepath.Join(cfg.SecondaryFirmwareDir, "micropython.uf2")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected installed firmware: %v", err)
	}
}

func TestCommandCentreFirmwareFlashQueuesWithoutTooling(t *testing.T) {
	h, cfg := newTestHandler(t)
	source := filepath.Join(cfg.SharedDir, "bundle.bin")
	if err := os.WriteFile(source, []byte("fw"), 0644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/command-centre/firmware", strings.NewReader(`{"action":"flash","source":"shared:bundle.bin"}`))
	w := httptest.NewRecorder()
	h.commandCentreFirmware(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	jobsDir := filepath.Join(cfg.DataDir(), "flash_jobs")
	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected queued flash job")
	}
}
