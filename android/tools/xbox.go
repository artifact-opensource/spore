package tools

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// ==================== Xbox / Windows Tools ====================
// These tools provide system management capabilities for Xbox Dev Mode
// and Windows deployments. They use standard Windows CLI tools (tasklist,
// taskkill, systeminfo, netstat, ipconfig) which are available in Xbox
// Dev Mode via the Device Portal shell.

// --- GPU Status ---

func (t *Toolbox) GpuStatus() string {
	var sb strings.Builder

	// Try nvidia-smi first (NVIDIA GPUs on PC)
	nv := t.ExecTimeout("nvidia-smi --query-gpu=name,temperature.gpu,utilization.gpu,memory.used,memory.total --format=csv,noheader,nounits 2>/dev/null", 5)
	nv = strings.TrimSpace(nv)
	if nv != "" && !strings.Contains(nv, "not found") && !strings.Contains(nv, "FAILED") {
		parts := strings.Split(nv, ", ")
		if len(parts) >= 5 {
			sb.WriteString(fmt.Sprintf("GPU: %s\n", parts[0]))
			sb.WriteString(fmt.Sprintf("Temperature: %s°C\n", parts[1]))
			sb.WriteString(fmt.Sprintf("Utilization: %s%%\n", parts[2]))
			sb.WriteString(fmt.Sprintf("VRAM: %sMB / %sMB\n", parts[3], parts[4]))
			return sb.String()
		}
		sb.WriteString(nv + "\n")
		return sb.String()
	}

	// Try dxdiag parsing (Xbox / Windows without NVIDIA)
	dx := t.ExecTimeout("dxdiag /t dxdiag_out.txt && timeout /t 3 >nul && type dxdiag_out.txt && del dxdiag_out.txt", 15)
	if dx != "" && !strings.Contains(dx, "not recognized") {
		lines := strings.Split(dx, "\n")
		for _, line := range lines {
			l := strings.TrimSpace(line)
			if strings.HasPrefix(l, "Card name:") ||
				strings.HasPrefix(l, "Display Memory:") ||
				strings.HasPrefix(l, "Dedicated Memory:") ||
				strings.HasPrefix(l, "Shared Memory:") ||
				strings.HasPrefix(l, "Driver Version:") ||
				strings.HasPrefix(l, "Current Mode:") {
				sb.WriteString(l + "\n")
			}
		}
		if sb.Len() > 0 {
			return sb.String()
		}
	}

	// Fallback: WMI query via PowerShell
	wmi := t.ExecTimeout(`powershell -Command "Get-CimInstance Win32_VideoController | Select-Object Name, AdapterRAM, DriverVersion, VideoProcessor | Format-List" 2>/dev/null`, 10)
	wmi = strings.TrimSpace(wmi)
	if wmi != "" && !strings.Contains(wmi, "not recognized") {
		return wmi
	}

	// Linux fallback: lspci
	lspci := t.ExecTimeout("lspci 2>/dev/null | grep -i vga", 5)
	lspci = strings.TrimSpace(lspci)
	if lspci != "" {
		return lspci
	}

	return "GPU info unavailable — no nvidia-smi, dxdiag, WMI, or lspci found"
}

// --- Service/Process Manager ---

func (t *Toolbox) ServiceManager(action, target string) string {
	switch action {
	case "list", "":
		// Try tasklist (Windows/Xbox), fall back to ps
		result := t.ExecTimeout("tasklist /FO TABLE /NH 2>/dev/null", 10)
		if result != "" && !strings.Contains(result, "not recognized") {
			return result
		}
		return t.ExecTimeout("ps aux 2>/dev/null || ps -ef", 10)

	case "start":
		if target == "" {
			return "error: target required for start action"
		}
		// Try start command (Windows)
		result := t.ExecTimeout(fmt.Sprintf("start /B %s 2>&1", target), 10)
		if strings.Contains(result, "not recognized") {
			// Linux: try nohup
			result = t.ExecTimeout(fmt.Sprintf("nohup %s &>/dev/null & echo $!", target), 5)
		}
		return result

	case "stop", "kill":
		if target == "" {
			return "error: target required for stop action"
		}
		// Check if target is a PID (numeric)
		isPID := true
		for _, c := range target {
			if c < '0' || c > '9' {
				isPID = false
				break
			}
		}

		if isPID {
			// Kill by PID
			result := t.ExecTimeout(fmt.Sprintf("taskkill /PID %s /F 2>/dev/null", target), 5)
			if strings.Contains(result, "not recognized") {
				return t.ExecTimeout(fmt.Sprintf("kill -9 %s", target), 5)
			}
			return result
		}
		// Kill by name
		result := t.ExecTimeout(fmt.Sprintf("taskkill /IM %s /F 2>/dev/null", target), 5)
		if strings.Contains(result, "not recognized") {
			return t.ExecTimeout(fmt.Sprintf("pkill -f %s", target), 5)
		}
		return result

	case "find":
		if target == "" {
			return "error: target required for find action"
		}
		result := t.ExecTimeout(fmt.Sprintf("tasklist /FI \"IMAGENAME eq %s\" 2>/dev/null", target), 5)
		if strings.Contains(result, "not recognized") {
			return t.ExecTimeout(fmt.Sprintf("ps aux | grep -i %s | grep -v grep", target), 5)
		}
		return result

	default:
		return fmt.Sprintf("error: unknown action '%s' — use list, start, stop, kill, find", action)
	}
}

// --- Network Info ---

func (t *Toolbox) NetworkInfo(action string) string {
	switch action {
	case "interfaces", "ip", "":
		// Try ipconfig (Windows/Xbox), fall back to ip addr
		result := t.ExecTimeout("ipconfig 2>/dev/null", 5)
		if result != "" && !strings.Contains(result, "not recognized") {
			return result
		}
		return t.ExecTimeout("ip addr show 2>/dev/null || ifconfig 2>/dev/null", 5)

	case "connections", "netstat":
		result := t.ExecTimeout("netstat -an 2>/dev/null", 10)
		if result == "" {
			result = t.ExecTimeout("ss -tulpn 2>/dev/null", 10)
		}
		return result

	case "ports":
		result := t.ExecTimeout("netstat -an 2>/dev/null | findstr LISTENING 2>/dev/null", 10)
		if strings.Contains(result, "not recognized") {
			result = t.ExecTimeout("ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null", 10)
		}
		return result

	case "dns":
		result := t.ExecTimeout("ipconfig /displaydns 2>/dev/null | head -100", 10)
		if strings.Contains(result, "not recognized") {
			return t.ExecTimeout("cat /etc/resolv.conf 2>/dev/null", 5)
		}
		return result

	case "scan":
		// Quick local network scan
		return t.ExecTimeout("arp -a 2>/dev/null", 10)

	default:
		return fmt.Sprintf("error: unknown action '%s' — use interfaces, connections, ports, dns, scan", action)
	}
}

// --- System Info ---

func (t *Toolbox) SystemInfo(component string) string {
	switch component {
	case "all", "":
		// Quick composite view
		var sb strings.Builder

		// CPU
		cpu := t.ExecTimeout(`powershell -Command "Get-CimInstance Win32_Processor | Select-Object Name, NumberOfCores, NumberOfLogicalProcessors, MaxClockSpeed | Format-List" 2>/dev/null`, 10)
		if cpu != "" && !strings.Contains(cpu, "not recognized") {
			sb.WriteString("=== CPU ===\n" + cpu + "\n")
		} else {
			sb.WriteString("=== CPU ===\n" + t.ExecTimeout("lscpu 2>/dev/null | head -15 || cat /proc/cpuinfo | head -15", 5) + "\n")
		}

		// RAM
		ram := t.ExecTimeout(`powershell -Command "Get-CimInstance Win32_OperatingSystem | Select-Object TotalVisibleMemorySize, FreePhysicalMemory | Format-List" 2>/dev/null`, 10)
		if ram != "" && !strings.Contains(ram, "not recognized") {
			sb.WriteString("=== RAM ===\n" + ram + "\n")
		} else {
			sb.WriteString("=== RAM ===\n" + t.ExecTimeout("free -h 2>/dev/null | head -3", 5) + "\n")
		}

		// Disk
		disk := t.ExecTimeout(`powershell -Command "Get-CimInstance Win32_LogicalDisk | Select-Object DeviceID, Size, FreeSpace | Format-List" 2>/dev/null`, 10)
		if disk != "" && !strings.Contains(disk, "not recognized") {
			sb.WriteString("=== Disk ===\n" + disk + "\n")
		} else {
			sb.WriteString("=== Disk ===\n" + t.ExecTimeout("df -h 2>/dev/null | head -10", 5) + "\n")
		}

		// GPU
		sb.WriteString("=== GPU ===\n" + t.GpuStatus() + "\n")

		return sb.String()

	case "cpu":
		result := t.ExecTimeout(`powershell -Command "Get-CimInstance Win32_Processor | Format-List" 2>/dev/null`, 10)
		if result != "" && !strings.Contains(result, "not recognized") {
			return result
		}
		return t.ExecTimeout("lscpu 2>/dev/null || cat /proc/cpuinfo", 5)

	case "ram", "memory":
		result := t.ExecTimeout(`powershell -Command "Get-CimInstance Win32_OperatingSystem | Select-Object TotalVisibleMemorySize, FreePhysicalMemory | Format-List" 2>/dev/null`, 10)
		if result != "" && !strings.Contains(result, "not recognized") {
			return result
		}
		return t.ExecTimeout("free -h 2>/dev/null", 5)

	case "disk", "storage":
		result := t.ExecTimeout(`powershell -Command "Get-CimInstance Win32_LogicalDisk | Format-Table DeviceID, @{N='SizeGB';E={[math]::Round($_.Size/1GB,1)}}, @{N='FreeGB';E={[math]::Round($_.FreeSpace/1GB,1)}}" 2>/dev/null`, 10)
		if result != "" && !strings.Contains(result, "not recognized") {
			return result
		}
		return t.ExecTimeout("df -h 2>/dev/null", 5)

	case "gpu":
		return t.GpuStatus()

	case "os":
		result := t.ExecTimeout("systeminfo 2>/dev/null | findstr /B /C:\"OS\" /C:\"System\" /C:\"Total\" /C:\"Available\"", 10)
		if result != "" && !strings.Contains(result, "not recognized") {
			return result
		}
		return t.ExecTimeout("uname -a && cat /etc/os-release 2>/dev/null", 5)

	default:
		return fmt.Sprintf("error: unknown component '%s' — use all, cpu, ram, disk, gpu, os", component)
	}
}

// --- File Server ---

func (t *Toolbox) FileServer(action, path string, port int) string {
	switch action {
	case "start", "":
		if path == "" {
			path = t.home
		}
		if port <= 0 {
			port = 9090
		}

		// Get local IP for display
		localIP := getLocalIP()

		// Start HTTP file server in background
		mux := http.NewServeMux()
		fs := http.FileServer(http.Dir(path))
		mux.Handle("/", fs)

		server := &http.Server{
			Addr:         fmt.Sprintf("0.0.0.0:%d", port),
			Handler:      mux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		}

		go func() {
			server.ListenAndServe()
		}()

		// Brief wait to verify it started
		time.Sleep(200 * time.Millisecond)

		return fmt.Sprintf("file server started\n  serving: %s\n  local:   http://127.0.0.1:%d\n  LAN:     http://%s:%d\n  stop with: service_manager stop action on this process",
			path, port, localIP, port)

	case "stop":
		// Kill any running file server by port
		result := t.ExecTimeout(fmt.Sprintf("fuser -k %d/tcp 2>/dev/null || netstat -tlnp 2>/dev/null | grep :%d", port, port), 5)
		return "file server stopped\n" + result

	default:
		return fmt.Sprintf("error: unknown action '%s' — use start, stop", action)
	}
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}

// ==================== Xbox Tool Definitions ====================

func XboxToolDefs() []ToolDef {
	return []ToolDef{
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "gpu_status",
				Description: "Get GPU information: name, temperature, utilization, VRAM usage. Works on NVIDIA (nvidia-smi), Xbox/Windows (dxdiag/WMI), and Linux (lspci).",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "service_manager",
				Description: "Manage background processes. Actions: list (show all), start <target>, stop/kill <target> (by name or PID), find <target>. Cross-platform: uses tasklist/taskkill on Windows/Xbox, ps/kill on Linux.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{
							"type":        "string",
							"description": "Action: list, start, stop, kill, find (default: list)",
						},
						"target": map[string]interface{}{
							"type":        "string",
							"description": "Process name or PID (required for start/stop/kill/find)",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "network_info",
				Description: "Network diagnostics. Actions: interfaces/ip (show IPs), connections/netstat (all connections), ports (listening ports), dns (DNS cache/config), scan (ARP table / local devices). Cross-platform.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{
							"type":        "string",
							"description": "Action: interfaces, connections, ports, dns, scan (default: interfaces)",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "system_info",
				Description: "System hardware info. Components: all (composite view), cpu, ram/memory, disk/storage, gpu, os. Cross-platform: uses WMI/PowerShell on Windows/Xbox, /proc + cli tools on Linux.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"component": map[string]interface{}{
							"type":        "string",
							"description": "Component: all, cpu, ram, disk, gpu, os (default: all)",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "file_server",
				Description: "Serve a directory over HTTP for easy file transfer. Actions: start (begin serving), stop. Accessible from any device on the network.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{
							"type":        "string",
							"description": "Action: start, stop (default: start)",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Directory to serve (default: home directory)",
						},
						"port": map[string]interface{}{
							"type":        "number",
							"description": "Port number (default: 9090)",
						},
					},
				},
			},
		},
	}
}
