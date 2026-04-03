package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"

	"github.com/artifact-opensource/spore/core"
)

// customDialer uses Google/Cloudflare DNS instead of system resolver
// Fixes Termux where [::1]:53 doesn't exist
var customResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 5 * time.Second}
		// Try Google DNS first, then Cloudflare
		conn, err := d.DialContext(ctx, "udp", "8.8.8.8:53")
		if err != nil {
			conn, err = d.DialContext(ctx, "udp", "1.1.1.1:53")
		}
		return conn, err
	},
}

var customTransport = &http.Transport{
	DialContext: (&net.Dialer{
		Resolver: customResolver,
		Timeout:  10 * time.Second,
	}).DialContext,
}

var customHTTPClient = &http.Client{
	Transport: customTransport,
	Timeout:   30 * time.Second,
}

const (
	gatewayURL = "wss://gateway.discord.gg/?v=10&encoding=json"
	apiBase    = "https://discord.com/api/v10"
)

// Bot is a minimal Discord bot that pipes messages through an Agent
type Bot struct {
	token  string
	prefix string
	agent  *core.Agent
	botID  string
	ws     *websocket.Conn
	seq    *int64
	mu     sync.Mutex
	log    func(string)
}

// gateway payloads
type gatewayPayload struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d,omitempty"`
	S  *int64          `json:"s,omitempty"`
	T  string          `json:"t,omitempty"`
}

type identifyPayload struct {
	Token      string            `json:"token"`
	Intents    int               `json:"intents"`
	Properties map[string]string `json:"properties"`
}

type readyData struct {
	User struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Bot      bool   `json:"bot"`
	} `json:"user"`
	SessionID string `json:"session_id"`
}

type messageData struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id"`
	Content   string `json:"content"`
	Author    struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Bot      bool   `json:"bot"`
	} `json:"author"`
	Mentions []struct {
		ID string `json:"id"`
	} `json:"mentions"`
}

func New(token string, prefix string, agent *core.Agent, logFn func(string)) *Bot {
	if prefix == "" {
		prefix = "!"
	}
	if logFn == nil {
		logFn = func(s string) { log.Println("[discord]", s) }
	}
	return &Bot{
		token:  token,
		prefix: prefix,
		agent:  agent,
		log:    logFn,
	}
}

// Run connects to Discord and blocks until context is cancelled
func (b *Bot) Run(ctx context.Context) error {
	for {
		err := b.connect(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		b.log(fmt.Sprintf("disconnected: %v — reconnecting in 5s", err))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func (b *Bot) connect(ctx context.Context) error {
	b.log("connecting to Discord gateway...")
	conn, _, err := websocket.Dial(ctx, gatewayURL, &websocket.DialOptions{
		HTTPClient: customHTTPClient,
	})
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "bye")
	conn.SetReadLimit(1 << 20) // 1MB

	b.ws = conn

	// Read Hello (op 10)
	var hello gatewayPayload
	if err := b.readPayload(ctx, &hello); err != nil {
		return fmt.Errorf("hello: %w", err)
	}
	if hello.Op != 10 {
		return fmt.Errorf("expected op 10 hello, got %d", hello.Op)
	}

	var helloData struct {
		HeartbeatInterval int `json:"heartbeat_interval"`
	}
	json.Unmarshal(hello.D, &helloData)

	// Send Identify (op 2)
	// Intents: GUILDS (1) | GUILD_MESSAGES (512) | MESSAGE_CONTENT (32768) | DMs (4096)
	identify := identifyPayload{
		Token:   b.token,
		Intents: 1 | 512 | 4096 | 32768,
		Properties: map[string]string{
			"os":      "android",
			"browser": "spore",
			"device":  "spore",
		},
	}
	if err := b.sendOp(ctx, 2, identify); err != nil {
		return fmt.Errorf("identify: %w", err)
	}

	// Start heartbeat
	go b.heartbeat(ctx, time.Duration(helloData.HeartbeatInterval)*time.Millisecond)

	// Event loop
	for {
		var payload gatewayPayload
		if err := b.readPayload(ctx, &payload); err != nil {
			return err
		}

		if payload.S != nil {
			b.mu.Lock()
			b.seq = payload.S
			b.mu.Unlock()
		}

		switch payload.Op {
		case 0: // Dispatch
			b.handleEvent(ctx, payload.T, payload.D)
		case 1: // Heartbeat request
			b.sendHeartbeat(ctx)
		case 7: // Reconnect
			return fmt.Errorf("server requested reconnect")
		case 9: // Invalid Session
			return fmt.Errorf("invalid session")
		case 11: // Heartbeat ACK
			// ok
		}
	}
}

func (b *Bot) handleEvent(ctx context.Context, t string, d json.RawMessage) {
	switch t {
	case "READY":
		var ready readyData
		json.Unmarshal(d, &ready)
		b.botID = ready.User.ID
		b.log(fmt.Sprintf("connected as %s (%s)", ready.User.Username, ready.User.ID))

	case "MESSAGE_CREATE":
		var msg messageData
		json.Unmarshal(d, &msg)

		// Ignore own messages
		if msg.Author.Bot || msg.Author.ID == b.botID {
			return
		}

		// Check if mentioned or has prefix or is DM
		isDM := msg.GuildID == ""
		isMentioned := false
		for _, m := range msg.Mentions {
			if m.ID == b.botID {
				isMentioned = true
				break
			}
		}
		hasPrefix := strings.HasPrefix(msg.Content, b.prefix)

		if !isDM && !isMentioned && !hasPrefix {
			return
		}

		// Strip mention and prefix
		content := msg.Content
		content = strings.ReplaceAll(content, "<@"+b.botID+">", "")
		content = strings.ReplaceAll(content, "<@!"+b.botID+">", "")
		if hasPrefix {
			content = strings.TrimPrefix(content, b.prefix)
		}
		content = strings.TrimSpace(content)

		if content == "" {
			content = "hello"
		}

		b.log(fmt.Sprintf("[%s] %s: %s", msg.ChannelID, msg.Author.Username, content))

		// Send typing indicator
		go b.sendTyping(msg.ChannelID)

		// Run through agent (fresh context per message to prevent history poisoning)
		go func() {
			b.log(fmt.Sprintf("[%s] processing request from %s...", msg.ChannelID, msg.Author.Username))
			b.agent.Reset() // fresh history per Discord message
			result, err := b.agent.Run(content)
			if err != nil {
				b.log(fmt.Sprintf("[%s] agent error: %v", msg.ChannelID, err))
				result = "error: " + err.Error()
			}
			b.log(fmt.Sprintf("[%s] response ready (%d chars)", msg.ChannelID, len(result)))

			// Discord message limit is 2000 chars
			if len(result) > 1990 {
				// Split into multiple messages
				for len(result) > 0 {
					chunk := result
					if len(chunk) > 1990 {
						// Try to split at newline
						idx := strings.LastIndex(chunk[:1990], "\n")
						if idx < 500 {
							idx = 1990
						}
						chunk = result[:idx]
						result = result[idx:]
					} else {
						result = ""
					}
					b.sendMessage(msg.ChannelID, strings.TrimSpace(chunk))
				}
			} else {
				if err := b.sendMessage(msg.ChannelID, result); err != nil {
					b.log(fmt.Sprintf("[%s] send error: %v", msg.ChannelID, err))
				} else {
					b.log(fmt.Sprintf("[%s] message sent", msg.ChannelID))
				}
			}
		}()
	}
}

func (b *Bot) heartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.sendHeartbeat(ctx)
		}
	}
}

func (b *Bot) sendHeartbeat(ctx context.Context) {
	b.mu.Lock()
	seq := b.seq
	b.mu.Unlock()
	b.sendOp(ctx, 1, seq)
}

func (b *Bot) readPayload(ctx context.Context, p *gatewayPayload) error {
	_, data, err := b.ws.Read(ctx)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, p)
}

func (b *Bot) sendOp(ctx context.Context, op int, d interface{}) error {
	raw, _ := json.Marshal(d)
	payload := gatewayPayload{Op: op, D: json.RawMessage(raw)}
	data, _ := json.Marshal(payload)
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ws.Write(ctx, websocket.MessageText, data)
}

// --- REST API ---

func (b *Bot) sendMessage(channelID, content string) error {
	url := fmt.Sprintf("%s/channels/%s/messages", apiBase, channelID)
	body, _ := json.Marshal(map[string]string{"content": content})
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bot "+b.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Spore (https://github.com/artifact-opensource/spore, 1.0)")

	resp, err := customHTTPClient.Do(req)
	if err != nil {
		b.log(fmt.Sprintf("send error: %v", err))
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		b.log(fmt.Sprintf("send %d: %s", resp.StatusCode, string(body)))
		return fmt.Errorf("discord API %d", resp.StatusCode)
	}
	return nil
}

func (b *Bot) sendTyping(channelID string) {
	url := fmt.Sprintf("%s/channels/%s/typing", apiBase, channelID)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bot "+b.token)
	req.Header.Set("User-Agent", "Spore (https://github.com/artifact-opensource/spore, 1.0)")
	customHTTPClient.Do(req)
}
