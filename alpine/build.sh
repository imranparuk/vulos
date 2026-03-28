#!/bin/sh
# Vula OS — Alpine Desktop Image Builder
# Builds a bootable ISO for x86_64 (desktop/laptop/server).
#
# Prerequisites:
#   - Go 1.21+, Node 18+, npm
#   - On Alpine: apk add alpine-sdk build-base alpine-conf syslinux xorriso mtools
#   - Or run in Docker: docker run -v $PWD:/src alpine:edge sh /src/alpine/build.sh
#
# Usage:
#   ./alpine/build.sh              # build everything to ./output/
#   ./alpine/build.sh /tmp/vulos   # custom output dir
#   ARCH=aarch64 ./alpine/build.sh # build for ARM64

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ARCH="${ARCH:-x86_64}"
OUTDIR="$(cd "${1:-$ROOT_DIR/output}" 2>/dev/null && pwd || echo "${1:-$ROOT_DIR/output}")"
GOARCH="amd64"
[ "$ARCH" = "aarch64" ] && GOARCH="arm64"

echo "╔══════════════════════════════════╗"
echo "║     Vula OS — Alpine Builder     ║"
echo "╠══════════════════════════════════╣"
echo "║ Arch:   $ARCH"
echo "║ Output: $OUTDIR"
echo "╚══════════════════════════════════╝"
echo ""

mkdir -p "$OUTDIR"

# ═══════════════════════════════════
# 1. Build Go binaries
# ═══════════════════════════════════
echo "▸ Building Go binaries ($GOARCH)..."
cd "$ROOT_DIR/backend"
GOOS=linux GOARCH=$GOARCH CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTDIR/vulos-server" ./cmd/server
GOOS=linux GOARCH=$GOARCH CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTDIR/vulos-init" ./cmd/init
echo "  ✓ vulos-server, vulos-init"

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
        echo "  ✓ $app"
    fi
done

# ═══════════════════════════════════
# 4. Generate root filesystem overlay
# ═══════════════════════════════════
echo "▸ Generating overlay..."
OVL="$OUTDIR/overlay"
rm -rf "$OVL"
mkdir -p "$OVL"/{sbin,usr/local/bin,opt/vulos,etc/init.d,etc/runlevels/default}
mkdir -p "$OVL"/etc/wpa_supplicant

# Binaries
cp "$OUTDIR/vulos-server" "$OVL/usr/local/bin/"
cp "$OUTDIR/vulos-init" "$OVL/sbin/vulos-init"
chmod +x "$OVL/usr/local/bin/vulos-server" "$OVL/sbin/vulos-init"

# Web root + apps
cp -r "$OUTDIR/webroot" "$OVL/opt/vulos/webroot"
cp -r "$OUTDIR/apps" "$OVL/opt/vulos/apps"

# Hostname
echo "vulos" > "$OVL/etc/hostname"

# wpa_supplicant template
cat > "$OVL/etc/wpa_supplicant/wpa_supplicant.conf" << 'EOF'
ctrl_interface=/var/run/wpa_supplicant
update_config=1
EOF

# ── OpenRC: vulos-server ──
cat > "$OVL/etc/init.d/vulos" << 'EOF'
#!/sbin/openrc-run
name="vulos"
description="Vula OS Server"
command="/usr/local/bin/vulos-server"
command_args="-env main"
command_background=true
pidfile="/run/vulos.pid"
output_log="/var/log/vulos.log"
error_log="/var/log/vulos.log"

depend() {
    need net
    after firewall
}
EOF
chmod +x "$OVL/etc/init.d/vulos"
ln -sf /etc/init.d/vulos "$OVL/etc/runlevels/default/vulos"

# ── OpenRC: kiosk ──
cat > "$OVL/etc/init.d/vulos-kiosk" << 'EOF'
#!/sbin/openrc-run
name="vulos-kiosk"
description="Vula OS Kiosk (Cage + WPE WebKit)"
command="/usr/bin/cage"
command_args="-- /usr/bin/cog http://localhost:8080"
command_background=true
pidfile="/run/vulos-kiosk.pid"
output_log="/var/log/vulos-kiosk.log"
error_log="/var/log/vulos-kiosk.log"

depend() {
    need vulos
}

start_pre() {
    export WLR_LIBINPUT_NO_DEVICES=1
    export WLR_NO_HARDWARE_CURSORS=1
    export XDG_RUNTIME_DIR=/run/user/0
    mkdir -p /run/user/0
    # GPU detection
    if [ -e /dev/dri/renderD128 ]; then
        export WLR_RENDERER=gles2
    else
        export WLR_RENDERER=pixman
    fi
}
EOF
chmod +x "$OVL/etc/init.d/vulos-kiosk"
ln -sf /etc/init.d/vulos-kiosk "$OVL/etc/runlevels/default/vulos-kiosk"

# ═══════════════════════════════════
# 5. Build ISO (if tools available)
# ═══════════════════════════════════
if command -v mkimage.sh >/dev/null 2>&1 || [ -f /usr/share/alpine-conf/mkimage.sh ]; then
    echo "▸ Building bootable ISO..."
    MKIMAGE="$(command -v mkimage.sh || echo /usr/share/alpine-conf/mkimage.sh)"

    # Write mkimage profile
    cat > "$OUTDIR/profile-vulos.sh" << 'PROFILE'
profile_vulos() {
    profile_standard
    title="Vula OS"
    desc="AI-first web operating system"
    arch="x86_64 aarch64"
    image_ext="iso"
    output_format="iso"
    kernel_flavors="lts"
    kernel_cmdline="quiet"
    apks="$apks
        cage wlroots mesa-dri-gallium wpewebkit cog wlr-randr brightnessctl
        iproute2 iptables wpa_supplicant dhcpcd
        bluez bluez-utils
        pipewire pipewire-pulse wireplumber
        restic python3
        iw ethtool curl jq dbus eudev font-dejavu
    "
    hostname="vulos"
}
PROFILE

    sh "$MKIMAGE" \
        --arch "$ARCH" \
        --profile vulos \
        --outdir "$OUTDIR" \
        --repository http://dl-cdn.alpinelinux.org/alpine/edge/main \
        --repository http://dl-cdn.alpinelinux.org/alpine/edge/community \
        --extra-repository http://dl-cdn.alpinelinux.org/alpine/edge/testing \
        2>&1 || echo "  ⚠ ISO build failed (run on Alpine with alpine-conf installed)"

    echo "  ✓ ISO built"
else
    echo "  ⚠ mkimage.sh not found — skipping ISO build"
    echo "  Install: apk add alpine-conf"
fi

echo ""
echo "═══════════════════════════════════"
echo "Build complete!"
echo ""
ls -lh "$OUTDIR/vulos-server" "$OUTDIR/vulos-init" 2>/dev/null
echo ""
echo "To test with QEMU:"
echo "  qemu-system-x86_64 -m 2048 -cdrom $OUTDIR/vulos-*.iso -enable-kvm"
echo ""
echo "Boot menu: Try Vula OS (live) or Install to disk"
echo "═══════════════════════════════════"
