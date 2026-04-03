#!/bin/bash
# deploy-aegis.sh — Push Spore + guardian to AEGIS
set -e

AEGIS="aegis"  # SSH alias
SRC="$(dirname "$0")/.."
BINARY="$SRC/spore-arm64"

echo "Building arm64 binary..."
cd "$SRC"
GOOS=linux GOARCH=arm64 go build -o spore-arm64 -ldflags="-s -w" .
echo "  Built: $(ls -lh spore-arm64 | awk '{print $5}')"

echo "Stopping remote Spore..."
ssh $AEGIS "pkill -f 'spore start' 2>/dev/null; pkill -f spore-guardian 2>/dev/null; sleep 1" || true

echo "Deploying binary + guardian..."
scp "$BINARY" $AEGIS:~/bin/spore
scp "deploy/spore-guardian.sh" $AEGIS:~/bin/spore-guardian.sh
ssh $AEGIS "chmod +x ~/bin/spore ~/bin/spore-guardian.sh"

echo "Setting up boot persistence..."
ssh $AEGIS "mkdir -p ~/.termux/boot"
scp "deploy/boot.sh" $AEGIS:~/.termux/boot/spore.sh
ssh $AEGIS "chmod +x ~/.termux/boot/spore.sh"

echo "Launching guardian..."
ssh $AEGIS "nohup bash ~/bin/spore-guardian.sh > /dev/null 2>&1 &"
sleep 5

echo "Checking health..."
HEALTH=$(ssh $AEGIS "curl -s http://127.0.0.1:8422/health 2>/dev/null" || echo "FAILED")
echo "  $HEALTH"

echo "Done."
