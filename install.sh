#!/data/data/com.termux/files/usr/bin/bash
# Spore for Android — Installer
# Run in Termux: bash install.sh

set -e

PURPLE='\033[0;35m'
GREEN='\033[0;32m'
DIM='\033[2m'
NC='\033[0m'
BOLD='\033[1m'

echo ""
echo -e "${PURPLE}${BOLD}"
echo "  ███████╗██████╗  ██████╗ ██████╗ ███████╗"
echo "  ██╔════╝██╔══██╗██╔═══██╗██╔══██╗██╔════╝"
echo "  ███████╗██████╔╝██║   ██║██████╔╝█████╗  "
echo "  ╚════██║██╔═══╝ ██║   ██║██╔══██╗██╔══╝  "
echo "  ███████║██║     ╚██████╔╝██║  ██║███████╗"
echo "  ╚══════╝╚═╝      ╚═════╝ ╚═╝  ╚═╝╚══════╝"
echo -e "${NC}"
echo -e "  ${DIM}spore installer v0.1.0${NC}"
echo ""

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    aarch64|arm64) BINARY="symbiote-android-arm64" ;;
    x86_64|amd64)  BINARY="symbiote-android-x86_64" ;;
    *)
        echo "  unsupported architecture: $ARCH"
        exit 1
        ;;
esac

echo -e "  ${DIM}arch: $ARCH${NC}"
echo -e "  ${DIM}binary: $BINARY${NC}"
echo ""

# Install core packages
echo -e "  ${DIM}installing packages...${NC}"
pkg update -y -q 2>/dev/null
for p in openssh curl wget git python nmap net-tools iproute2 termux-api; do
    pkg install -y -q $p 2>/dev/null || true
done

# Create directories
echo -e "  ${DIM}creating directories...${NC}"
mkdir -p ~/bin
mkdir -p ~/.symbiote/{memory,logs,processes}
mkdir -p ~/workspace

# Install binary as 'spore'
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [ -f "$SCRIPT_DIR/$BINARY" ]; then
    cp "$SCRIPT_DIR/$BINARY" ~/bin/spore
elif [ -f "$SCRIPT_DIR/symbiote-android-arm64" ]; then
    cp "$SCRIPT_DIR/symbiote-android-arm64" ~/bin/spore
else
    echo "  error: binary not found. Place $BINARY in the same directory."
    exit 1
fi
chmod +x ~/bin/spore

# PATH
if ! grep -q 'HOME/bin' ~/.bashrc 2>/dev/null; then
    echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
fi
export PATH="$HOME/bin:$PATH"

# SSH key
if [ ! -f ~/.ssh/id_ed25519 ]; then
    echo -e "  ${DIM}generating SSH key...${NC}"
    mkdir -p ~/.ssh
    ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -N "" -q
fi

# Termux:Boot auto-start
mkdir -p ~/.termux/boot
cat > ~/.termux/boot/spore.sh << 'BOOT'
#!/data/data/com.termux/files/usr/bin/bash
termux-wake-lock
sleep 5
spore daemon start >> ~/.symbiote/logs/boot.log 2>&1 &
BOOT
chmod +x ~/.termux/boot/spore.sh

# Storage access
termux-setup-storage 2>/dev/null || true

# Verify
echo ""
if command -v spore &>/dev/null; then
    echo -e "  ${GREEN}✓${NC} installed: $(spore version)"
    echo -e "  ${GREEN}✓${NC} binary: ~/bin/spore ($(du -h ~/bin/spore | cut -f1))"
    echo -e "  ${GREEN}✓${NC} data: ~/.symbiote/"
    echo -e "  ${GREEN}✓${NC} boot: ~/.termux/boot/spore.sh"
    echo -e "  ${GREEN}✓${NC} ssh key: ~/.ssh/id_ed25519.pub"
    echo ""
    echo -e "  ${DIM}configure:${NC}"
    echo "    spore config provider copilot"
    echo "    spore config base_url http://127.0.0.1:3000"
    echo "    spore config model gpt-4o"
    echo ""
    echo -e "  ${DIM}tunnel to dragonfly:${NC}"
    echo "    ssh -f -N -L 3000:127.0.0.1:3000 adam@192.168.1.13"
    echo ""
    echo -e "  ${DIM}run:${NC}"
    echo "    spore              # interactive"
    echo "    spore web          # open webchat in browser"
    echo "    spore daemon start # background service"
    echo "    spore serve        # HTTP API + webchat on :8422"
    echo ""
else
    echo "  error: installation failed"
    exit 1
fi
