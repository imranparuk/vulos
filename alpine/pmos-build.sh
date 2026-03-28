#!/bin/sh
# Vula OS — postmarketOS Mobile Image Builder
# Builds a flashable image for phones/tablets.
#
# Prerequisites:
#   pip install pmbootstrap
#   pmbootstrap init  (select your device)
#
# Usage:
#   ./alpine/pmos-build.sh                     # default: pine64-pinephone
#   ./alpine/pmos-build.sh samsung-a52q        # specify device
#   ./alpine/pmos-build.sh oneplus-enchilada   # OnePlus 6

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DEVICE="${1:-pine64-pinephone}"
OUTDIR="$ROOT_DIR/output-pmos"

echo "╔══════════════════════════════════════╗"
echo "║  Vula OS — postmarketOS Builder      ║"
echo "╠══════════════════════════════════════╣"
echo "║ Device: $DEVICE"
echo "║ Output: $OUTDIR"
echo "╚══════════════════════════════════════╝"
echo ""

mkdir -p "$OUTDIR"

# ═══════════════════════════════════
# 1. Build Go binaries (ARM64)
# ═══════════════════════════════════
echo "▸ Building Go binaries (arm64)..."
cd "$ROOT_DIR/backend"
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTDIR/vulos-server" ./cmd/server
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTDIR/vulos-init" ./cmd/init
echo "  ✓ vulos-server, vulos-init (arm64)"

# ═══════════════════════════════════
# 2. Build frontend
# ═══════════════════════════════════
echo "▸ Building frontend..."
cd "$ROOT_DIR"
npm ci --silent 2>/dev/null || npm install --silent
npm run build
cp -r dist "$OUTDIR/webroot"
echo "  ✓ webroot/"

# ═══════════════════════════════════
# 3. Copy app services
# ═══════════════════════════════════
echo "▸ Copying app services..."
mkdir -p "$OUTDIR/apps"
for app in browser notes gallery; do
    if [ -d "$ROOT_DIR/apps/$app" ]; then
        cp -r "$ROOT_DIR/apps/$app" "$OUTDIR/apps/"
    fi
done

# ═══════════════════════════════════
# 4. Install into pmOS rootfs
# ═══════════════════════════════════
echo "▸ Configuring pmbootstrap..."

if ! command -v pmbootstrap >/dev/null 2>&1; then
    echo "  ✗ pmbootstrap not found. Install: pip install pmbootstrap"
    exit 1
fi

WORK="$(pmbootstrap config work)"
ROOTFS="$WORK/chroot_rootfs_${DEVICE}"

echo "▸ Installing packages..."
pmbootstrap install \
    cage wlroots wpewebkit cog wlr-randr brightnessctl \
    iproute2 iptables wpa_supplicant dhcpcd \
    bluez bluez-utils \
    pipewire pipewire-pulse wireplumber \
    restic python3 \
    iw ethtool curl jq \
    dbus eudev font-dejavu \
    2>&1

echo "▸ Copying vulos into image..."
# Binaries
sudo cp "$OUTDIR/vulos-server" "$ROOTFS/usr/local/bin/"
sudo cp "$OUTDIR/vulos-init" "$ROOTFS/sbin/vulos-init"
sudo chmod +x "$ROOTFS/usr/local/bin/vulos-server" "$ROOTFS/sbin/vulos-init"

# Web root + apps
sudo mkdir -p "$ROOTFS/opt/vulos"
sudo cp -r "$OUTDIR/webroot" "$ROOTFS/opt/vulos/webroot"
sudo cp -r "$OUTDIR/apps" "$ROOTFS/opt/vulos/apps"

# OpenRC services (reuse from Alpine build)
sudo cp "$ROOT_DIR/output/overlay/etc/init.d/vulos" "$ROOTFS/etc/init.d/" 2>/dev/null || true
sudo cp "$ROOT_DIR/output/overlay/etc/init.d/vulos-kiosk" "$ROOTFS/etc/init.d/" 2>/dev/null || true
sudo chmod +x "$ROOTFS/etc/init.d/vulos" "$ROOTFS/etc/init.d/vulos-kiosk" 2>/dev/null || true

# Enable services
pmbootstrap chroot -- rc-update add vulos default 2>/dev/null || true
pmbootstrap chroot -- rc-update add vulos-kiosk default 2>/dev/null || true

# Hostname
echo "vulos" | sudo tee "$ROOTFS/etc/hostname" > /dev/null

# ═══════════════════════════════════
# 5. Build and export image
# ═══════════════════════════════════
echo "▸ Building image..."
pmbootstrap install 2>&1
pmbootstrap export "$OUTDIR" 2>&1

echo ""
echo "═══════════════════════════════════════"
echo "postmarketOS image built!"
echo ""
echo "Flash to device:"
echo "  pmbootstrap flasher flash_rootfs"
echo "  pmbootstrap flasher flash_kernel"
echo ""
echo "Or flash manually:"
echo "  dd if=$OUTDIR/*.img of=/dev/sdX bs=4M status=progress"
echo "═══════════════════════════════════════"
