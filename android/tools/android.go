package tools

import (
	"fmt"
	"strings"
)

// --- App name lookup table ---

var appNames = map[string]string{
	"chrome":      "com.android.chrome",
	"settings":    "com.android.settings",
	"camera":      "com.sec.android.app.camera",
	"gallery":     "com.sec.android.gallery3d",
	"phone":       "com.samsung.android.dialer",
	"messages":    "com.samsung.android.messaging",
	"calendar":    "com.samsung.android.calendar",
	"clock":       "com.sec.android.app.clockpackage",
	"calculator":  "com.sec.android.app.popupcalculator",
	"files":       "com.sec.android.app.myfiles",
	"youtube":     "com.google.android.youtube",
	"maps":        "com.google.android.apps.maps",
	"gmail":       "com.google.android.gm",
	"whatsapp":    "com.whatsapp",
	"telegram":    "org.telegram.messenger",
	"twitter":     "com.twitter.android",
	"x":           "com.twitter.android",
	"facebook":    "com.facebook.katana",
	"instagram":   "com.instagram.android",
	"reddit":      "com.reddit.frontpage",
	"spotify":     "com.spotify.music",
	"discord":     "com.discord",
	"slack":       "com.Slack",
	"termux":      "com.termux",
	"macrodroid":  "com.arlosoft.macrodroid",
	"outlook":     "com.microsoft.office.outlook",
	"netflix":     "com.netflix.mediaclient",
	"sketchbook":  "com.adsk.sketchbook",
	"exness":      "com.exness.investor",
	"firefox":     "org.mozilla.firefox",
	"brave":       "com.brave.browser",
	"edge":        "com.microsoft.emmx",
	"opera":       "com.opera.browser",
	"samsung internet": "com.sec.android.app.sbrowser",
	"samsung browser":  "com.sec.android.app.sbrowser",
	"notes":       "com.samsung.android.app.notes",
	"drive":       "com.google.android.apps.docs",
	"sheets":      "com.google.android.apps.docs.editors.sheets",
	"docs":        "com.google.android.apps.docs.editors.docs",
	"photos":      "com.google.android.apps.photos",
	"play store":  "com.android.vending",
	"store":       "com.android.vending",
	"tiktok":      "com.zhiliaoapp.musically",
	"snapchat":    "com.snapchat.android",
	"signal":      "org.thoughtcrime.securesms",
	"teams":       "com.microsoft.teams",
	"zoom":        "us.zoom.videomeetings",
	"mt5":         "net.metaquotes.metatrader5",
	"metatrader":  "net.metaquotes.metatrader5",
}

// resolveApp resolves a friendly name or package name to a package name
func resolveApp(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	if pkg, ok := appNames[lower]; ok {
		return pkg
	}
	// If it looks like a package name already, return as-is
	if strings.Contains(name, ".") {
		return name
	}
	// Fuzzy match: check if name is substring of any key or value
	for friendly, pkg := range appNames {
		if strings.Contains(friendly, lower) || strings.Contains(strings.ToLower(pkg), lower) {
			return pkg
		}
	}
	return name
}

// ==================== Device Control ====================

func (t *Toolbox) Brightness(level int) string {
	if level < 0 || level > 255 {
		return "error: brightness must be 0-255"
	}
	return t.ExecTimeout(fmt.Sprintf("termux-brightness %d", level), 5)
}

func (t *Toolbox) Volume(stream string, level int) string {
	if stream == "" {
		stream = "music"
	}
	return t.ExecTimeout(fmt.Sprintf("termux-volume %s %d", stream, level), 5)
}

func (t *Toolbox) Torch(state string) string {
	if state == "" {
		state = "on"
	}
	return t.ExecTimeout(fmt.Sprintf("termux-torch %s", state), 5)
}

func (t *Toolbox) Vibrate(durationMs int) string {
	if durationMs <= 0 {
		durationMs = 500
	}
	return t.ExecTimeout(fmt.Sprintf("termux-vibrate -d %d", durationMs), 5)
}

func (t *Toolbox) ClipboardGet() string {
	return t.ExecTimeout("termux-clipboard-get", 5)
}

func (t *Toolbox) ClipboardSet(text string) string {
	cmd := fmt.Sprintf("termux-clipboard-set %q", text)
	result := t.ExecTimeout(cmd, 5)
	if result == "" {
		return "clipboard set"
	}
	return result
}

func (t *Toolbox) TtsSpeak(text string) string {
	cmd := fmt.Sprintf("termux-tts-speak %q", text)
	result := t.ExecTimeout(cmd, 15)
	if result == "" {
		return "speaking"
	}
	return result
}

func (t *Toolbox) Toast(text string) string {
	cmd := fmt.Sprintf("termux-toast %q", text)
	result := t.ExecTimeout(cmd, 5)
	if result == "" {
		return "toast shown"
	}
	return result
}

func (t *Toolbox) WifiInfo() string {
	return t.ExecTimeout("termux-wifi-connectioninfo", 5)
}

func (t *Toolbox) Location() string {
	return t.ExecTimeout("termux-location -p gps -r once", 15)
}

func (t *Toolbox) CameraPhoto(cameraID int, outputPath string) string {
	if outputPath == "" {
		outputPath = "/data/data/com.termux/files/home/photo.jpg"
	}
	cmd := fmt.Sprintf("termux-camera-photo -c %d %s", cameraID, outputPath)
	result := t.ExecTimeout(cmd, 10)
	if result == "" {
		return fmt.Sprintf("photo saved to %s", outputPath)
	}
	return result
}

func (t *Toolbox) MediaControl(action string) string {
	if action == "" {
		action = "pause"
	}
	return t.ExecTimeout(fmt.Sprintf("termux-media-player %s", action), 5)
}

func (t *Toolbox) SmsSend(number, message string) string {
	cmd := fmt.Sprintf("termux-sms-send -n %q %q", number, message)
	result := t.ExecTimeout(cmd, 10)
	if result == "" {
		return fmt.Sprintf("SMS sent to %s", number)
	}
	return result
}

func (t *Toolbox) SmsInbox(limit int) string {
	if limit <= 0 {
		limit = 10
	}
	return t.ExecTimeout(fmt.Sprintf("termux-sms-list -l %d", limit), 10)
}

func (t *Toolbox) Call(number string) string {
	cmd := fmt.Sprintf("termux-telephony-call %q", number)
	result := t.ExecTimeout(cmd, 5)
	if result == "" {
		return fmt.Sprintf("calling %s", number)
	}
	return result
}

func (t *Toolbox) ScreenState() string {
	return t.ExecTimeout("dumpsys power 2>/dev/null | grep -E 'Display Power|mWakefulness|mScreenOn'", 5)
}

func (t *Toolbox) BatteryStatus() string {
	return t.ExecTimeout("termux-battery-status", 5)
}

func (t *Toolbox) Sensor(sensor string, limit int) string {
	if limit <= 0 {
		limit = 1
	}
	if sensor == "" {
		return t.ExecTimeout("termux-sensor -l", 5)
	}
	return t.ExecTimeout(fmt.Sprintf("termux-sensor -s %q -n %d", sensor, limit), 10)
}

// ==================== App Management ====================

// knownActivities maps common app names to their launch components
// Using am start -n <component> properly foregrounds the app on screen
var knownActivities = map[string]string{
	"com.android.chrome":                 "com.android.chrome/com.google.android.apps.chrome.Main",
	"com.android.settings":               "com.android.settings/.Settings",
	"com.google.android.youtube":         "com.google.android.youtube/.HomeActivity",
	"com.google.android.apps.nbu.files":  "com.google.android.apps.nbu.files/.home.HomeActivity",
	"com.google.android.apps.maps":       "com.google.android.apps.maps/.MapsActivity",
	"com.google.android.gm":              "com.google.android.gm/.GmailActivity",
	"com.google.android.apps.messaging":  "com.google.android.apps.messaging/.ui.ConversationListActivity",
	"com.google.android.dialer":          "com.google.android.dialer/.extensions.GoogleDialtactsActivity",
	"com.google.android.apps.photos":     "com.google.android.apps.photos/.home.HomeActivity",
	"com.google.android.calendar":        "com.google.android.calendar/.AllInOneActivity",
	"com.google.android.deskclock":       "com.google.android.deskclock/.DeskClock",
	"com.google.android.calculator":      "com.google.android.calculator/.Calculator",
	"com.google.android.contacts":        "com.google.android.contacts/.activities.PeopleActivity",
	"com.whatsapp":                        "com.whatsapp/.HomeActivity",
	"com.instagram.android":               "com.instagram.android/.activity.MainTabActivity",
	"com.twitter.android":                 "com.twitter.android/.StartActivity",
	"com.facebook.katana":                 "com.facebook.katana/.LoginActivity",
	"com.spotify.music":                   "com.spotify.music/.MainActivity",
	"com.termux":                           "com.termux/.app.TermuxActivity",
	"org.telegram.messenger":              "org.telegram.messenger/.DefaultIcon",
	"com.samsung.android.app.notes":       "com.samsung.android.app.notes/.main.MainActivity",
	"com.discord":                         "com.discord/.main.MainActivity",
	"com.Slack":                           "com.Slack/.ui.HomeActivity",
	"com.arlosoft.macrodroid":             "com.arlosoft.macrodroid/.homescreen.NewHomeScreenActivity",
	"com.sec.android.app.camera":          "com.sec.android.app.camera/.Camera",
	"com.sec.android.app.myfiles":         "com.sec.android.app.myfiles/.common.MainActivity",
	"com.samsung.android.dialer":          "com.samsung.android.dialer/.DialtactsActivity",
	"com.samsung.android.messaging":       "com.samsung.android.messaging/.ui.view.main.MainActivityStart",
	"com.samsung.android.calendar":        "com.samsung.android.calendar/.CalendarActivity",
	"com.sec.android.app.clockpackage":    "com.sec.android.app.clockpackage/.ClockPackage",
	"com.reddit.frontpage":                "com.reddit.frontpage/.StartActivity",
	"com.netflix.mediaclient":             "com.netflix.mediaclient/.ui.launch.UIWebViewActivity",
	"com.microsoft.office.outlook":        "com.microsoft.office.outlook/.MainActivity",
	"com.exness.investor":                 "com.exness.investor/.ui.splash.SplashActivity",
	"org.mozilla.firefox":                 "org.mozilla.firefox/.App",
	"com.brave.browser":                   "com.brave.browser/.BraveActivity",
	"com.sec.android.app.sbrowser":        "com.sec.android.app.sbrowser/.SBrowserMainActivity",
	"com.android.vending":                 "com.android.vending/.AssetBrowserActivity",
	"net.metaquotes.metatrader5":           "net.metaquotes.metatrader5/.MainActivityNew",
}

// appURISchemes maps package names to URI schemes that Android will route to the correct app.
// Using termux-open with a URI scheme goes through Termux's foreground context,
// bypassing Android 13+ background activity start restrictions.
var appURISchemes = map[string]string{
	"com.android.chrome":                "googlechrome://navigate?url=about:blank",
	"com.google.android.youtube":        "vnd.youtube://",
	"com.google.android.apps.maps":      "geo:0,0",
	"com.google.android.gm":             "mailto:",
	"com.whatsapp":                       "whatsapp://",
	"com.instagram.android":              "instagram://",
	"com.twitter.android":                "twitter://",
	"com.facebook.katana":                "fb://",
	"com.spotify.music":                  "spotify://",
	"org.telegram.messenger":             "tg://",
	"com.discord":                        "discord://",
	"com.Slack":                          "slack://",
	"com.reddit.frontpage":               "reddit://",
	"com.netflix.mediaclient":            "nflx://",
}

func (t *Toolbox) AppLaunch(name string) string {
	pkg := resolveApp(name)

	// Method 1: URI scheme via termux-open (best for Android 13+ — uses foreground context)
	if uri, ok := appURISchemes[pkg]; ok {
		result := t.ExecTimeout(fmt.Sprintf("termux-open '%s' 2>&1", uri), 10)
		if !strings.Contains(result, "Error") && !strings.Contains(result, "error") {
			return fmt.Sprintf("launched %s", name)
		}
	}

	// Method 2: am start via termux-am (works for settings, system apps, and when URI unavailable)
	if activity, ok := knownActivities[pkg]; ok {
		cmd := fmt.Sprintf("am start -n %s 2>&1", activity)
		result := t.ExecTimeout(cmd, 10)
		if !strings.Contains(result, "Error") && !strings.Contains(result, "error") {
			return fmt.Sprintf("launched %s (may need to switch to it manually — Android restricts background app starts)", name)
		}
	}

	// Method 3: monkey launch — works for ANY installed package (no activity name needed)
	cmd := fmt.Sprintf("monkey -p %s -c android.intent.category.LAUNCHER 1 2>&1", pkg)
	result := t.ExecTimeout(cmd, 10)
	if strings.Contains(result, "Events injected: 1") {
		return fmt.Sprintf("launched %s", name)
	}

	// Method 4: dynamic lookup — resolve the launch activity from package manager
	dumpResult := t.ExecTimeout(fmt.Sprintf("pm dump %s 2>/dev/null | grep -A1 'android.intent.action.MAIN' | grep -oP '[\\w.]+/[\\w.]+' | head -1", pkg), 10)
	dumpResult = strings.TrimSpace(dumpResult)
	if dumpResult != "" && strings.Contains(dumpResult, "/") {
		launchCmd := fmt.Sprintf("am start -n %s 2>&1", dumpResult)
		launchResult := t.ExecTimeout(launchCmd, 10)
		if !strings.Contains(launchResult, "Error") {
			return fmt.Sprintf("launched %s", name)
		}
	}

	// Method 5: try as-is with pm resolve — find any launchable activity
	resolveResult := t.ExecTimeout(fmt.Sprintf("pm resolve-activity --brief -a android.intent.action.MAIN -c android.intent.category.LAUNCHER %s 2>/dev/null | tail -1", pkg), 10)
	resolveResult = strings.TrimSpace(resolveResult)
	if resolveResult != "" && strings.Contains(resolveResult, "/") {
		launchResult := t.ExecTimeout(fmt.Sprintf("am start -n %s 2>&1", resolveResult), 10)
		if !strings.Contains(launchResult, "Error") {
			return fmt.Sprintf("launched %s", name)
		}
	}

	return fmt.Sprintf("failed to launch %s — package '%s' not found or has no launchable activity. Try: app_list to find the correct package name.", name, pkg)
}

func (t *Toolbox) AppStop(name string) string {
	pkg := resolveApp(name)
	// am force-stop may not work on Android 13+ without system permissions
	result := t.ExecTimeout(fmt.Sprintf("am force-stop %s 2>&1", pkg), 5)
	if strings.Contains(result, "Error") || strings.Contains(result, "Permission") {
		// Fallback: try killing via pid
		pidResult := t.ExecTimeout(fmt.Sprintf("pidof %s 2>/dev/null", pkg), 3)
		if pidResult != "" {
			t.ExecTimeout(fmt.Sprintf("kill %s 2>/dev/null", strings.TrimSpace(pidResult)), 3)
			return fmt.Sprintf("killed %s process", name)
		}
		return fmt.Sprintf("cannot stop %s — Android restricts this from Termux. Use the app switcher to close it.", name)
	}
	return fmt.Sprintf("stopped %s", name)
}

func (t *Toolbox) AppList(filter string) string {
	cmd := "pm list packages"
	if filter != "" {
		cmd = fmt.Sprintf("pm list packages 2>/dev/null | grep -i %q", filter)
	}
	result := t.ExecTimeout(cmd+" | head -50", 10)
	// Count total
	countResult := t.ExecTimeout(cmd+" | wc -l", 5)
	return fmt.Sprintf("%s\n--- %s total", result, strings.TrimSpace(countResult))
}

func (t *Toolbox) AppInfo(name string) string {
	pkg := resolveApp(name)
	return t.ExecTimeout(fmt.Sprintf("dumpsys package %s 2>/dev/null | head -60", pkg), 10)
}

func (t *Toolbox) AppSwitch(name string) string {
	// Same as launch — Android brings existing instance to foreground
	return t.AppLaunch(name)
}

// ==================== MacroDroid Integration ====================

func (t *Toolbox) MacroFire(triggerName string) string {
	cmd := fmt.Sprintf("am broadcast -a com.arlosoft.macrodroid.ACTION_FIRE_TRIGGER -e trigger_name %q 2>&1", triggerName)
	result := t.ExecTimeout(cmd, 5)
	if strings.Contains(result, "Broadcast sent") {
		return fmt.Sprintf("fired MacroDroid trigger: %s", triggerName)
	}
	return result
}

func (t *Toolbox) MacroFireWith(triggerName string, vars map[string]string) string {
	// Build extras string
	extras := ""
	for k, v := range vars {
		extras += fmt.Sprintf(" -e %q %q", k, v)
	}
	cmd := fmt.Sprintf("am broadcast -a com.arlosoft.macrodroid.ACTION_FIRE_TRIGGER -e trigger_name %q%s 2>&1", triggerName, extras)
	result := t.ExecTimeout(cmd, 5)
	if strings.Contains(result, "Broadcast sent") {
		return fmt.Sprintf("fired MacroDroid trigger: %s (with %d vars)", triggerName, len(vars))
	}
	return result
}

// ==================== Tool Definitions ====================

func AndroidToolDefs() []ToolDef {
	return []ToolDef{
		// --- Device Control ---
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "brightness",
				Description: "Set screen brightness (0=off, 255=max).",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"level": map[string]interface{}{
							"type": "number", "description": "Brightness 0-255",
						},
					},
					"required": []string{"level"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "volume",
				Description: "Set device volume. Streams: music, ring, notification, alarm, system.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"stream": map[string]interface{}{
							"type": "string", "description": "Audio stream (music/ring/notification/alarm/system)",
						},
						"level": map[string]interface{}{
							"type": "number", "description": "Volume level (0-15)",
						},
					},
					"required": []string{"level"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "torch",
				Description: "Toggle flashlight on or off.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"state": map[string]interface{}{
							"type": "string", "description": "on or off",
						},
					},
					"required": []string{"state"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "vibrate",
				Description: "Vibrate the device.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"duration_ms": map[string]interface{}{
							"type": "number", "description": "Duration in milliseconds (default 500)",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "clipboard_get",
				Description: "Read the current clipboard content.",
				Parameters: map[string]interface{}{
					"type": "object", "properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "clipboard_set",
				Description: "Copy text to clipboard.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text": map[string]interface{}{
							"type": "string", "description": "Text to copy",
						},
					},
					"required": []string{"text"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "tts_speak",
				Description: "Speak text aloud using Android text-to-speech.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text": map[string]interface{}{
							"type": "string", "description": "Text to speak",
						},
					},
					"required": []string{"text"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "toast",
				Description: "Show a brief Android toast notification on screen.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text": map[string]interface{}{
							"type": "string", "description": "Toast message",
						},
					},
					"required": []string{"text"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "wifi_info",
				Description: "Get current WiFi connection details (SSID, IP, speed, etc).",
				Parameters: map[string]interface{}{
					"type": "object", "properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "location",
				Description: "Get current GPS location (latitude, longitude, altitude).",
				Parameters: map[string]interface{}{
					"type": "object", "properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "camera_photo",
				Description: "Take a photo with the device camera.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"camera_id": map[string]interface{}{
							"type": "number", "description": "0=back camera, 1=front camera (default 0)",
						},
						"output_path": map[string]interface{}{
							"type": "string", "description": "Save path (default ~/photo.jpg)",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "media_control",
				Description: "Control media playback: play, pause, stop, next, previous.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{
							"type": "string", "description": "play, pause, stop, next, previous",
						},
					},
					"required": []string{"action"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "sms_send",
				Description: "Send an SMS message.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"number": map[string]interface{}{
							"type": "string", "description": "Phone number",
						},
						"message": map[string]interface{}{
							"type": "string", "description": "Message text",
						},
					},
					"required": []string{"number", "message"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "sms_inbox",
				Description: "Read recent SMS messages from inbox.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"limit": map[string]interface{}{
							"type": "number", "description": "Number of messages to return (default 10)",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "call",
				Description: "Make a phone call.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"number": map[string]interface{}{
							"type": "string", "description": "Phone number to call",
						},
					},
					"required": []string{"number"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "screen_state",
				Description: "Check if screen is on/off and display power state.",
				Parameters: map[string]interface{}{
					"type": "object", "properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "battery",
				Description: "Get battery status (level, charging, temperature).",
				Parameters: map[string]interface{}{
					"type": "object", "properties": map[string]interface{}{},
				},
			},
		},
		// --- App Management ---
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "app_launch",
				Description: "Launch an Android app by name (e.g. 'chrome', 'whatsapp', 'youtube') or package name. Opens the app in foreground.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string", "description": "App name (chrome, whatsapp, youtube, etc) or full package name",
						},
					},
					"required": []string{"name"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "app_stop",
				Description: "Force-stop an Android app.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string", "description": "App name or package name",
						},
					},
					"required": []string{"name"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "app_list",
				Description: "List installed apps, optionally filtered by keyword.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"filter": map[string]interface{}{
							"type": "string", "description": "Filter keyword (optional)",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "app_info",
				Description: "Get detailed info about an installed app.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string", "description": "App name or package name",
						},
					},
					"required": []string{"name"},
				},
			},
		},
		// --- MacroDroid ---
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "macro_fire",
				Description: "Fire a MacroDroid automation trigger by name. Requires MacroDroid installed with a matching 'Intent Received' trigger.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"trigger_name": map[string]interface{}{
							"type": "string", "description": "Name of the MacroDroid trigger to fire",
						},
					},
					"required": []string{"trigger_name"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolDefFunc{
				Name:        "macro_fire_with",
				Description: "Fire a MacroDroid trigger with extra variables (key-value pairs passed as intent extras).",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"trigger_name": map[string]interface{}{
							"type": "string", "description": "Trigger name",
						},
						"variables": map[string]interface{}{
							"type": "object", "description": "Key-value pairs to pass as extras",
						},
					},
					"required": []string{"trigger_name"},
				},
			},
		},
	}
}
