#!/data/data/com.termux/files/usr/bin/bash
# spore-guardian.sh — Self-healing daemon for Spore on Android
# Runs at boot, restarts on crash, maintains SSH tunnel to Dragonfly

SPORE_BIN="$HOME/bin/spore"
SPORE_PORT=8422
DRAGONFLY="adam@192.168.1.13"
TUNNEL_PORT=3000
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

# Acquire wake lock so Android doesn't kill us
termux-wake-lock 2>/dev/null
log "Wake lock acquired"

# Function: ensure SSH tunnel to Dragonfly copilot proxy
ensure_tunnel() {
    if ! pgrep -f "ssh.*-L.*${TUNNEL_PORT}:127.0.0.1:${TUNNEL_PORT}" >/dev/null 2>&1; then
        log "Starting SSH tunnel to Dragonfly:${TUNNEL_PORT}"
        ssh -f -N -o ServerAliveInterval=30 -o ServerAliveCountMax=3 \
            -o ConnectTimeout=10 -o StrictHostKeyChecking=no \
            -L ${TUNNEL_PORT}:127.0.0.1:${TUNNEL_PORT} ${DRAGONFLY} 2>>"$LOG"
        if [ $? -eq 0 ]; then
            log "Tunnel established"
        else
            log "Tunnel failed — will retry in 30s"
        fi
    fi
}

# Function: ensure sshd is running
ensure_sshd() {
    if ! pgrep sshd >/dev/null 2>&1; then
        log "Starting sshd"
        sshd 2>>"$LOG"
    fi
}

# Function: ensure Spore is running
ensure_spore() {
    if ! pgrep -f "spore start" >/dev/null 2>&1; then
        log "Starting Spore"
        nohup "$SPORE_BIN" start "$SPORE_PORT" >> "$HOME/.symbiote/logs/start.log" 2>&1 &
        sleep 3
        # Verify it's up
        if curl -s "http://127.0.0.1:${SPORE_PORT}/health" >/dev/null 2>&1; then
            log "Spore started successfully"
        else
            log "Spore failed to start — check start.log"
        fi
    fi
}

# Main loop — check every 30 seconds
while true; do
    ensure_sshd
    ensure_tunnel
    ensure_spore
    sleep 30
done
