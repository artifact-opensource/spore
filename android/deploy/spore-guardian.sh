#!/data/data/com.termux/files/usr/bin/bash
# spore-guardian.sh — Self-healing daemon for Spore on Android
# Spore has its own embedded copilot proxy — no SSH tunnel needed

SPORE_BIN="$HOME/bin/spore"
SPORE_PORT=8422
LOG="$HOME/.symbiote/logs/guardian.log"
PIDFILE="$HOME/.symbiote/guardian.pid"

mkdir -p "$(dirname "$LOG")"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG"
}

# Prevent duplicate guardians
if [ -f "$PIDFILE" ]; then
    OLD_PID=$(cat "$PIDFILE")
    if kill -0 "$OLD_PID" 2>/dev/null; then
        log "Guardian already running (pid $OLD_PID), exiting"
        exit 0
    fi
fi
echo $$ > "$PIDFILE"

log "Guardian started (pid $$)"

# Kill orphan zombie sleep loops from previous runs
pkill -f "while true; do sleep" 2>/dev/null
pkill -f "sleep 300" 2>/dev/null
log "Cleaned up orphan processes"

# Acquire wake lock so Android doesn't kill us
termux-wake-lock 2>/dev/null
log "Wake lock acquired"

# Kill leftover SSH tunnels (no longer needed)
pkill -f "ssh.*-L.*3000:127.0.0.1:3000" 2>/dev/null

# Kill orphan sleep/bash loops from previous sessions
pkill -f "while true; do sleep" 2>/dev/null
pkill -f "sleep 300" 2>/dev/null
log "Cleaned up orphan processes"

ensure_sshd() {
    if ! pgrep sshd >/dev/null 2>&1; then
        log "Starting sshd"
        sshd 2>>"$LOG"
    fi
}

ensure_spore() {
    if ! pgrep -f "spore start" >/dev/null 2>&1; then
        log "Starting Spore"
        nohup "$SPORE_BIN" start "$SPORE_PORT" >> "$HOME/.symbiote/logs/start.log" 2>&1 &
        sleep 3
        if curl -s "http://127.0.0.1:${SPORE_PORT}/health" >/dev/null 2>&1; then
            log "Spore started successfully"
        else
            log "Spore failed to start — check start.log"
        fi
    fi
}

while true; do
    ensure_sshd
    ensure_spore
    sleep 30
done
