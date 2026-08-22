package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/artifact-virtual/symbiote-android/core"
	"github.com/artifact-virtual/symbiote-android/provider"
)

type commandCentreState struct {
	Config          *core.Config       `json:"config"`
	Providers       []providerInfoView `json:"providers"`
	Storage         storageView        `json:"storage"`
	SharedFiles     []assetView        `json:"shared_files"`
	FirmwareCatalog []assetView        `json:"firmware_catalog"`
	RGBProfiles     []rgbProfileView   `json:"rgb_profiles"`
	Tools           []toolStatusView   `json:"tools"`
	ProjectDir      string             `json:"project_dir,omitempty"`
	SharedURL       string             `json:"shared_url"`
	ChatURL         string             `json:"chat_url"`
}

type providerInfoView struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	BaseURL     string   `json:"base_url"`
	Models      []string `json:"models"`
	KeyEnvVar   string   `json:"key_env_var,omitempty"`
	RequiresKey bool     `json:"requires_key"`
}

type storageView struct {
	DataDir              string `json:"data_dir"`
	StorageDir           string `json:"storage_dir"`
	SharedDir            string `json:"shared_dir"`
	SecondaryFirmwareDir string `json:"secondary_firmware_dir"`
}

type assetView struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
	Type   string `json:"type"`
	Size   int64  `json:"size"`
	URL    string `json:"url,omitempty"`
}

type rgbProfileView struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Colors      map[string]string `json:"colors"`
	Patterns    []string          `json:"patterns"`
}

type toolStatusView struct {
	Name        string `json:"name"`
	Available   bool   `json:"available"`
	Command     string `json:"command,omitempty"`
	Description string `json:"description"`
}

type flashJob struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Source    string `json:"source"`
	Port      string `json:"port,omitempty"`
	Address   string `json:"address,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func (h *apiHandler) commandCentrePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/command-centre" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(commandCentreHTML))
}

func (h *apiHandler) commandCentreAPI(w http.ResponseWriter, r *http.Request) {
	cfg := h.agent.Config()

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, h.buildCommandCentreState(cfg))
	case http.MethodPost:
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, key := range []string{"provider", "model", "base_url", "api_key", "daemon_port", "device_name", "storage_dir", "shared_dir", "secondary_firmware_dir", "rgb_profile"} {
			if val, ok := payload[key]; ok {
				str := fmt.Sprintf("%v", val)
				if key == "api_key" && str == "" {
					continue
				}
				cfg.Set(key, str)
			}
		}
		if err := cfg.Save(cfg.ConfigPath()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, h.buildCommandCentreState(cfg))
	default:
		http.Error(w, "GET or POST", http.StatusMethodNotAllowed)
	}
}

func (h *apiHandler) commandCentreFirmware(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	cfg := h.agent.Config()

	var req struct {
		Action  string `json:"action"`
		Source  string `json:"source"`
		Port    string `json:"port"`
		Baud    int    `json:"baud"`
		Address string `json:"address"`
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Source == "" {
		http.Error(w, "source required", http.StatusBadRequest)
		return
	}
	if req.Baud <= 0 {
		req.Baud = 460800
	}
	if req.Address == "" {
		req.Address = "0x0"
	}

	source, err := resolveCommandCentrePath(cfg, req.Source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "install", "":
		target, err := joinUnderRoot(cfg.SecondaryFirmwareDir, filepath.Base(source))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := copyFile(filepath.Dir(source), source, cfg.SecondaryFirmwareDir, target); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "installed", "target": target})
	case "flash":
		tool, args := detectESPFlashTool(req.Port, req.Baud, req.Address, source)
		if tool == "" || req.Port == "" {
			job, err := queueFlashJob(cfg, req, source)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"status": "queued", "job": job})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, tool, args...)
		out, err := cmd.CombinedOutput()
		status := "flashed"
		if err != nil {
			status = "failed"
		}
		writeJSON(w, map[string]interface{}{"status": status, "command": append([]string{tool}, args...), "output": string(out)})
	default:
		http.Error(w, "unsupported action", http.StatusBadRequest)
	}
}

func (h *apiHandler) buildCommandCentreState(cfg *core.Config) commandCentreState {
	projectDir := detectESP32Project()
	configView := *cfg
	configView.APIKey = ""
	return commandCentreState{
		Config:          &configView,
		Providers:       providerViews(),
		Storage:         storageView{DataDir: cfg.DataDir(), StorageDir: cfg.StorageDir, SharedDir: cfg.SharedDir, SecondaryFirmwareDir: cfg.SecondaryFirmwareDir},
		SharedFiles:     listAssets(cfg.SharedDir, cfg.SharedDir, "shared"),
		FirmwareCatalog: listFirmwareAssets(cfg, projectDir),
		RGBProfiles:     defaultRGBProfiles(),
		Tools:           detectESPTools(),
		ProjectDir:      projectDir,
		SharedURL:       "/shared/",
		ChatURL:         "/",
	}
}

func providerViews() []providerInfoView {
	out := make([]providerInfoView, 0, len(provider.Registry))
	for _, p := range provider.Registry {
		out = append(out, providerInfoView{
			ID:          p.ID,
			Name:        p.Name,
			BaseURL:     p.BaseURL,
			Models:      p.Models,
			KeyEnvVar:   p.KeyEnvVar,
			RequiresKey: p.RequiresKey,
		})
	}
	return out
}

func listFirmwareAssets(cfg *core.Config, projectDir string) []assetView {
	seen := map[string]bool{}
	var out []assetView
	roots := []string{cfg.SharedDir, cfg.SecondaryFirmwareDir}
	if projectDir != "" {
		roots = append(roots, projectDir)
	}
	for _, root := range roots {
		for _, item := range listAssets(root, root, firmwareSourcePrefix(root, cfg, projectDir)) {
			if !isFirmwareAsset(item.Path) || seen[item.Path] {
				continue
			}
			seen[item.Path] = true
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func listAssets(root, labelBase, sourcePrefix string) []assetView {
	if root == "" {
		return nil
	}
	entries := []assetView{}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(labelBase, path)
		if err != nil {
			rel = filepath.Base(path)
		}
		if info.IsDir() {
			if strings.Count(rel, string(os.PathSeparator)) > 1 {
				return filepath.SkipDir
			}
			return nil
		}
		relPath := filepath.ToSlash(rel)
		entries = append(entries, assetView{
			Name:   info.Name(),
			Path:   relPath,
			Source: sourcePrefix + ":" + relPath,
			Type:   strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
			Size:   info.Size(),
			URL:    sharedURLForPath(root, path),
		})
		if len(entries) >= 100 {
			return io.EOF
		}
		return nil
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries
}

func sharedURLForPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	return "/shared/" + strings.ReplaceAll(rel, string(os.PathSeparator), "/")
}

func defaultRGBProfiles() []rgbProfileView {
	return []rgbProfileView{
		{
			ID:          "spore-default",
			Name:        "Spore Default",
			Description: "GPIO48 RGB shows green when idle, cyan while thinking, amber on tool activity, red on failures, and magenta during firmware work.",
			Colors: map[string]string{
				"idle":     "#00ff88",
				"thinking": "#00d4ff",
				"tool":     "#ffb000",
				"error":    "#ff3355",
				"flash":    "#ff44ff",
			},
			Patterns: []string{"solid", "pulse", "chase", "blink"},
		},
		{
			ID:          "quiet-night",
			Name:        "Quiet Night",
			Description: "Dimmer profile for always-on devices and bedside hardware.",
			Colors: map[string]string{
				"idle":     "#002b1d",
				"thinking": "#003344",
				"tool":     "#332200",
				"error":    "#440011",
				"flash":    "#2a0040",
			},
			Patterns: []string{"solid", "pulse"},
		},
	}
}

func detectESPTools() []toolStatusView {
	defs := []struct {
		name string
		desc string
	}{
		{name: "idf.py", desc: "ESP-IDF build and monitor workflow"},
		{name: "esptool.py", desc: "Flash ESP32/ESP32-S3 firmware images"},
		{name: "mpremote", desc: "Manage MicroPython boards and files"},
		{name: "ampy", desc: "Legacy MicroPython file transfer helper"},
	}
	out := make([]toolStatusView, 0, len(defs))
	for _, def := range defs {
		path, err := exec.LookPath(def.name)
		out = append(out, toolStatusView{
			Name:        def.name,
			Available:   err == nil,
			Command:     path,
			Description: def.desc,
		})
	}
	out = append(out, toolStatusView{
		Name:        "espnow",
		Available:   true,
		Description: "ESP-NOW is provisioned in the companion firmware and simulated via activity events.",
	})
	return out
}

func detectESP32Project() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "esp32_s3")
		if _, err := os.Stat(filepath.Join(candidate, "main", "spore_esp32_s3.c")); err == nil {
			return candidate
		}
		if dir == filepath.Dir(dir) {
			return ""
		}
	}
}

func firmwareSourcePrefix(root string, cfg *core.Config, projectDir string) string {
	switch root {
	case cfg.SharedDir:
		return "shared"
	case cfg.SecondaryFirmwareDir:
		return "secondary"
	case projectDir:
		return "project"
	default:
		return "shared"
	}
}

func commandCentreRoot(prefix string, cfg *core.Config, projectDir string) (string, error) {
	switch prefix {
	case "shared":
		return cfg.SharedDir, nil
	case "secondary":
		return cfg.SecondaryFirmwareDir, nil
	case "project":
		if projectDir == "" {
			return "", fmt.Errorf("project directory unavailable")
		}
		return projectDir, nil
	default:
		return "", fmt.Errorf("unsupported source: %s", prefix)
	}
}

func joinUnderRoot(root, rel string) (string, error) {
	rel = filepath.Clean(rel)
	candidate := filepath.Join(root, rel)
	candidate = filepath.Clean(candidate)
	relCheck, err := filepath.Rel(root, candidate)
	if err != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes root")
	}
	return candidate, nil
}

func pathWithinRoot(root, candidate string) bool {
	if root == "" || candidate == "" {
		return false
	}
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func isFirmwareAsset(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".bin", ".uf2", ".hex", ".img", ".py", ".lua", ".json", ".c":
		return true
	default:
		return false
	}
}

func resolveCommandCentrePath(cfg *core.Config, requested string) (string, error) {
	parts := strings.SplitN(requested, ":", 2)
	if len(parts) != 2 || parts[1] == "" {
		return "", fmt.Errorf("invalid source: %s", requested)
	}
	root, err := commandCentreRoot(parts[0], cfg, detectESP32Project())
	if err != nil {
		return "", err
	}
	candidate, err := joinUnderRoot(root, parts[1])
	if err != nil {
		return "", err
	}
	if !pathWithinRoot(root, candidate) {
		return "", fmt.Errorf("path escapes root")
	}
	if _, err := os.Stat(candidate); err != nil {
		return "", fmt.Errorf("source not found: %s", requested)
	}
	return candidate, nil
}

func copyFile(srcRoot, src, dstRoot, dst string) error {
	if !pathWithinRoot(srcRoot, src) || !pathWithinRoot(dstRoot, dst) {
		return fmt.Errorf("path escapes managed roots")
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func detectESPFlashTool(port string, baud int, address, source string) (string, []string) {
	tool, err := exec.LookPath("esptool.py")
	if err != nil {
		return "", nil
	}
	args := []string{"--chip", "esp32s3"}
	if port != "" {
		args = append(args, "--port", port)
	}
	if baud > 0 {
		args = append(args, "--baud", fmt.Sprintf("%d", baud))
	}
	args = append(args, "write_flash", address, source)
	return tool, args
}

func queueFlashJob(cfg *core.Config, req struct {
	Action  string `json:"action"`
	Source  string `json:"source"`
	Port    string `json:"port"`
	Baud    int    `json:"baud"`
	Address string `json:"address"`
}, source string) (flashJob, error) {
	job := flashJob{
		ID:        fmt.Sprintf("job-%d", time.Now().UnixNano()),
		Action:    req.Action,
		Source:    source,
		Port:      req.Port,
		Address:   req.Address,
		Status:    "pending",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	jobsDir := filepath.Join(cfg.DataDir(), "flash_jobs")
	if err := os.MkdirAll(jobsDir, 0755); err != nil {
		return flashJob{}, err
	}
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return flashJob{}, err
	}
	return job, os.WriteFile(filepath.Join(jobsDir, job.ID+".json"), data, 0644)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

const commandCentreHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Spore Command Centre</title>
<style>
:root{--bg:#09090f;--panel:#13131c;--panel2:#171724;--line:#27273a;--text:#ececff;--muted:#9da0bc;--accent:#8b5cf6;--ok:#00d084;--warn:#f59e0b;--err:#f43f5e}
*{box-sizing:border-box}body{margin:0;font:14px/1.4 system-ui;background:var(--bg);color:var(--text)}a{color:#c4b5fd}.wrap{display:grid;grid-template-columns:1.2fr 1fr;min-height:100vh}.chat,.side{padding:20px}.chat{border-right:1px solid var(--line)}.card{background:var(--panel);border:1px solid var(--line);border-radius:16px;padding:16px;margin-bottom:16px;box-shadow:0 10px 30px rgba(0,0,0,.18)}h1,h2,h3{margin:0 0 12px}.muted{color:var(--muted)}textarea,input,select{width:100%;background:var(--panel2);color:var(--text);border:1px solid var(--line);border-radius:12px;padding:10px}textarea{min-height:96px;resize:vertical}button{background:var(--accent);color:white;border:0;border-radius:12px;padding:10px 14px;cursor:pointer}.row{display:grid;grid-template-columns:1fr 1fr;gap:10px}.stack{display:grid;gap:10px}.log{white-space:pre-wrap;background:#0c0c14;border:1px solid var(--line);border-radius:12px;padding:12px;min-height:120px}.pill{display:inline-block;padding:4px 8px;border-radius:999px;font-size:12px;margin-left:8px}.ok{background:rgba(0,208,132,.15);color:#7bf0bb}.bad{background:rgba(244,63,94,.16);color:#ff93a9}.item{padding:10px 0;border-top:1px solid var(--line)}.item:first-child{border-top:0}.item small{display:block;color:var(--muted)}.actions{display:flex;gap:8px;flex-wrap:wrap;margin-top:8px}.ghost{background:transparent;border:1px solid var(--line)}@media(max-width:980px){.wrap{grid-template-columns:1fr}.chat{border-right:0;border-bottom:1px solid var(--line)}}
</style>
</head>
<body>
<div class="wrap">
  <section class="chat">
    <div class="card">
      <h1>Spore Command Centre</h1>
      <div class="muted">Chat, provider controls, storage, shared firmware, GPIO48 RGB profiles, and ESP32-S3 companion staging.</div>
      <div style="margin-top:10px"><a href="/">Open full chat UI</a> · <a href="/shared/" target="_blank">Browse shared directory</a></div>
    </div>
    <div class="card stack">
      <h2>Chat interface</h2>
      <textarea id="prompt" placeholder="Ask Spore to inspect the shared directory, build firmware, or run tools."></textarea>
      <div class="actions"><button onclick="sendPrompt()">Send</button></div>
      <div class="log" id="chatLog">ready</div>
    </div>
    <div class="card">
      <h2>Firmware catalogue</h2>
      <div id="firmware"></div>
    </div>
  </section>
  <aside class="side">
    <div class="card stack">
      <h2>LLM + storage</h2>
      <div class="row">
        <div><small>Provider</small><select id="provider"></select></div>
        <div><small>Model</small><input id="model"></div>
      </div>
      <div><small>Base URL</small><input id="baseUrl"></div>
      <div><small>API key</small><input id="apiKey" type="password"></div>
      <div class="row">
        <div><small>Shared directory</small><input id="sharedDir"></div>
        <div><small>Storage directory</small><input id="storageDir"></div>
      </div>
      <div class="row">
        <div><small>Secondary firmware directory</small><input id="secondaryDir"></div>
        <div><small>GPIO48 RGB profile</small><select id="rgbProfile"></select></div>
      </div>
      <div class="actions"><button onclick="saveSettings()">Save settings</button><span class="muted" id="saveState"></span></div>
    </div>
    <div class="card">
      <h2>ESP tooling</h2>
      <div id="tools"></div>
    </div>
    <div class="card">
      <h2>Shared directory</h2>
      <div id="sharedFiles"></div>
    </div>
  </aside>
</div>
<script>
let state=null;
async function loadState(){
  const res=await fetch('/api/command-centre');
  state=await res.json();
  renderState();
}
function renderState(){
  const cfg=state.config;
  const providers=document.getElementById('provider');
  providers.innerHTML=state.providers.map(function(p){return '<option value="'+p.id+'" '+(p.id===cfg.provider?'selected':'')+'>'+p.name+'</option>';}).join('');
  document.getElementById('model').value=cfg.model||'';
  document.getElementById('baseUrl').value=cfg.base_url||'';
  document.getElementById('apiKey').value=cfg.api_key||'';
  document.getElementById('sharedDir').value=cfg.shared_dir||'';
  document.getElementById('storageDir').value=cfg.storage_dir||'';
  document.getElementById('secondaryDir').value=cfg.secondary_firmware_dir||'';
  const rgb=document.getElementById('rgbProfile');
  rgb.innerHTML=state.rgb_profiles.map(function(p){return '<option value="'+p.id+'" '+(p.id===cfg.rgb_profile?'selected':'')+'>'+p.name+'</option>';}).join('');
  document.getElementById('tools').innerHTML=state.tools.map(function(t){return '<div class="item"><strong>'+t.name+'</strong><span class="pill '+(t.available?'ok':'bad')+'">'+(t.available?'available':'not found')+'</span><small>'+t.description+(t.command?' · '+t.command:'')+'</small></div>';}).join('') || '<div class="muted">No tooling detected.</div>';
  document.getElementById('sharedFiles').innerHTML=(state.shared_files||[]).map(function(f){return '<div class="item"><strong>'+f.name+'</strong><small>'+f.path+'</small></div>';}).join('') || '<div class="muted">Shared directory is empty.</div>';
  document.getElementById('firmware').innerHTML=(state.firmware_catalog||[]).map(function(f){return '<div class="item"><strong>'+f.name+'</strong><small>'+f.path+'</small><div class="actions"><button data-source="'+esc(f.source)+'" onclick="installFirmware(this.dataset.source)">Install</button><button class="ghost" data-source="'+esc(f.source)+'" onclick="flashFirmware(this.dataset.source)">Flash</button></div></div>';}).join('') || '<div class="muted">No firmware or Lua/MicroPython assets detected yet.</div>';
}
function esc(v){return String(v).replace(/'/g,"&#39;")}
async function sendPrompt(){
  const prompt=document.getElementById('prompt').value.trim();
  if(!prompt)return;
  const log=document.getElementById('chatLog');
  log.textContent='thinking...';
  const res=await fetch('/run',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({prompt})});
  const data=await res.json();
  log.textContent=data.result||data.error||JSON.stringify(data,null,2);
}
async function saveSettings(){
  document.getElementById('saveState').textContent='saving...';
  const payload={provider:provider.value,model:model.value,base_url:baseUrl.value,api_key:apiKey.value,shared_dir:sharedDir.value,storage_dir:storageDir.value,secondary_firmware_dir:secondaryDir.value,rgb_profile:rgbProfile.value};
  const res=await fetch('/api/command-centre',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});
  state=await res.json();
  renderState();
  document.getElementById('saveState').textContent='saved';
}
async function installFirmware(path){
  const res=await fetch('/api/command-centre/firmware',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action:'install',source:path})});
  const data=await res.json();
  alert(data.status + (data.target ? '\n'+data.target : ''));
  await loadState();
}
async function flashFirmware(path){
  const port=prompt('Serial port for esptool (leave blank to queue a flash job):','');
  const res=await fetch('/api/command-centre/firmware',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action:'flash',source:path,port:port||''})});
  const data=await res.json();
  alert((data.status||'done') + (data.output ? '\n\n'+data.output : ''));
}
loadState();
</script>
</body>
</html>`
