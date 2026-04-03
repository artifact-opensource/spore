#!/bin/bash
# Deploy Spore to AEGIS (Samsung Z Fold 5)
# Usage: bash deploy-aegis.sh [ip]

set -e

AEGIS_IP="${1:-192.168.1.7}"
AEGIS_PORT="8022"
BINARY="symbiote-android-arm64"
REMOTE_USER=""  # Termux default user is empty (whoami shows u0_aXXX)

PURPLE='\033[0;35m'
GREEN='\033[0;32m'
DIM='\033[2m'
RED='\033[0;31m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo ""
echo -e "${PURPLE}deploying spore to AEGIS ($AEGIS_IP:$AEGIS_PORT)${NC}"
echo ""

# Test connectivity
echo -e "  ${DIM}testing connection...${NC}"
if ! ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no -p $AEGIS_PORT $AEGIS_IP "echo connected" 2>/dev/null; then
    echo -e "  ${RED}cannot connect to AEGIS${NC}"
    echo -e "  ${DIM}ensure Termux SSHD is running: sshd${NC}"
    echo -e "  ${DIM}check IP: ifconfig in Termux${NC}"
    exit 1
fi

echo -e "  ${GREEN}✓${NC} connected"

# Copy binary
echo -e "  ${DIM}copying binary...${NC}"
scp -P $AEGIS_PORT "$SCRIPT_DIR/$BINARY" $AEGIS_IP:~/bin/spore
ssh -p $AEGIS_PORT $AEGIS_IP "chmod +x ~/bin/spore"
echo -e "  ${GREEN}✓${NC} binary deployed"

# Copy install script
echo -e "  ${DIM}copying installer...${NC}"
scp -P $AEGIS_PORT "$SCRIPT_DIR/install.sh" $AEGIS_IP:~/install-spore.sh
echo -e "  ${GREEN}✓${NC} installer copied"

# Create directories & configure
echo -e "  ${DIM}configuring...${NC}"
ssh -p $AEGIS_PORT $AEGIS_IP << 'REMOTE'
mkdir -p ~/bin ~/.symbiote/{memory,logs,processes} ~/workspace

# Rename binary to spore
if [ -f ~/bin/spore ]; then
    echo "  binary: $(du -h ~/bin/spore | cut -f1)"
fi

# PATH setup
grep -q 'HOME/bin' ~/.bashrc 2>/dev/null || echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
export PATH="$HOME/bin:$PATH"

# Configure provider — connect back to Dragonfly's copilot proxy via SSH tunnel
# Proxy binds to 127.0.0.1:3000 on Dragonfly, so we tunnel through SSH
spore config provider copilot 2>/dev/null
spore config base_url http://127.0.0.1:3000 2>/dev/null
spore config model gpt-4o 2>/dev/null
spore config device aegis 2>/dev/null

# Auto-start on boot
mkdir -p ~/.termux/boot
cat > ~/.termux/boot/spore.sh << 'BOOT'
#!/data/data/com.termux/files/usr/bin/bash
termux-wake-lock
sleep 5
spore daemon start >> ~/.symbiote/logs/boot.log 2>&1 &
BOOT
chmod +x ~/.termux/boot/spore.sh

# SSH tunnel boot script — forward local:3000 to Dragonfly's copilot proxy
cat > ~/.termux/boot/spore-tunnel.sh << 'TUNNEL'
#!/data/data/com.termux/files/usr/bin/bash
sleep 8
ssh -f -N -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -o StrictHostKeyChecking=no -L 3000:127.0.0.1:3000 adam@192.168.1.13 >> ~/.symbiote/logs/tunnel.log 2>&1
TUNNEL
chmod +x ~/.termux/boot/spore-tunnel.sh

# Start the tunnel now
ssh -f -N -o ServerAliveInterval=30 -o StrictHostKeyChecking=no -L 3000:127.0.0.1:3000 adam@192.168.1.13 2>/dev/null && echo "  tunnel to dragonfly: active" || echo "  tunnel to dragonfly: needs SSH key setup"

# Verify
echo ""
spore version 2>/dev/null && echo "  spore installed OK" || echo "  spore binary error"
spore status 2>/dev/null
REMOTE

echo ""
echo -e "  ${GREEN}✓ spore deployed to AEGIS${NC}"
echo ""
echo -e "  ${DIM}on AEGIS:${NC}"
echo "    spore              # interactive chat"
echo "    spore web          # open webchat in browser"
echo "    spore daemon start # background service"
echo "    spore serve        # HTTP API + webchat on :8422"
echo ""
echo -e "  ${DIM}tunnel (auto on boot, or manual):${NC}"
echo "    ssh -f -N -L 3000:127.0.0.1:3000 adam@192.168.1.13"
echo ""
echo -e "  ${DIM}from dragonfly:${NC}"
echo "    ssh aegis 'spore run \"hello\"'"
echo "    curl http://$AEGIS_IP:8422/health"
echo ""
