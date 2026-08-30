#!/usr/bin/env bash
# build/replit-iso.sh — build a Replit-compatible Veldra ISO without Docker.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.

set -euo pipefail

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SELF_DIR}/../scripts/common.sh"

VERSION="$("${VELDRA_PROJECT_ROOT}/scripts/version.sh" show)"
ARCH="${VELDRA_ARCH_DEFAULT:-x86_64}"
WORK="${VELDRA_BUILD_WORK:-${VELDRA_PROJECT_ROOT}/build/work/replit}"
OUT="${VELDRA_BUILD_OUT:-${VELDRA_PROJECT_ROOT}/build/out}"
ISO="${OUT}/veldra-${VERSION}-${ARCH}.iso"

ARCH_RELEASE="2026.08.01"
ARCH_ISO_NAME="archlinux-${ARCH_RELEASE}-x86_64.iso"
ARCH_ISO_URL="https://geo.mirror.pkgbuild.com/iso/${ARCH_RELEASE}/${ARCH_ISO_NAME}"
ARCH_ISO_SHA256="4e82dced1c4fd3e498b22a853f8db2a4d262d32b97e7e07d97390d9e425ffe5e"
ARCH_ISO="${WORK}/${ARCH_ISO_NAME}"
SOURCE_SFS="${WORK}/airootfs.sfs"
RUN_ID="${EPOCHSECONDS:-$(date +%s)}-${RANDOM}"
ROOTFS="${WORK}/rootfs-${RUN_ID}"
NEW_SFS="${WORK}/veldra-airootfs-${RUN_ID}.sfs"

vd_require curl xorriso unsquashfs mksquashfs fakeroot sha256sum awk sed install mkdir rm du mv cat grep chmod ln

vd_info "Replit-compatible Veldra ISO builder — ${VERSION} / ${ARCH}"
vd_warn "Sandbox compatibility path; canonical 'make iso' remains the Arch-container pipeline."

mkdir -p "$WORK" "$OUT"

if [[ ! -f "$ARCH_ISO" ]]; then
    vd_info "downloading official Arch Linux ${ARCH_RELEASE} ISO (~1.5 GB)"
    curl -fL --retry 3 --retry-delay 2 --progress-bar -o "$ARCH_ISO.part" "$ARCH_ISO_URL"
    mv "$ARCH_ISO.part" "$ARCH_ISO"
else
    vd_info "using cached Arch ISO: $ARCH_ISO"
fi

printf '%s  %s\n' "$ARCH_ISO_SHA256" "$ARCH_ISO" | sha256sum -c -
vd_ok "Arch ISO checksum verified"

SFS_PATH="/arch/x86_64/airootfs.sfs"
vd_info "Arch live root: $SFS_PATH"
if ! xorriso -indev "$ARCH_ISO" -ls "$SFS_PATH" >/dev/null 2>&1; then
    vd_die 1 "official Arch ISO does not contain expected live root: $SFS_PATH"
fi

rm -f "$SOURCE_SFS" "$NEW_SFS"
rm -rf "$ROOTFS"
mkdir -p "$ROOTFS"

vd_info "extracting Arch live rootfs"
xorriso -osirrox on -indev "$ARCH_ISO" -extract "$SFS_PATH" "$SOURCE_SFS" >/dev/null
[[ -s "$SOURCE_SFS" ]] || vd_die 1 "failed to extract $SFS_PATH"

vd_info "unpacking Arch live rootfs with fakeroot"
fakeroot -- unsquashfs -no-xattrs -d "$ROOTFS" "$SOURCE_SFS" >/dev/null
[[ -d "$ROOTFS/etc" && -d "$ROOTFS/usr" ]] || vd_die 1 "extracted Arch rootfs does not look valid"

TUI_BIN="${VELDRA_PROJECT_ROOT}/tui/bin/veldra-tui"
if [[ ! -x "$TUI_BIN" ]]; then
    vd_info "building Veldra TUI"
    "${VELDRA_PROJECT_ROOT}/scripts/build-tui.sh"
fi

vd_info "installing Veldra TUI into the live rootfs"
install -D -m 0755 "$TUI_BIN" "$ROOTFS/usr/local/bin/veldra-tui"

vd_info "configuring Veldra live user"
install -d -m 0755 "$ROOTFS/home/veldra"

if ! grep -q '^veldra:' "$ROOTFS/etc/passwd"; then
    PASSWD_TMP="$ROOTFS/etc/.passwd.veldra.tmp"
    cat "$ROOTFS/etc/passwd" > "$PASSWD_TMP"
    printf '%s\n' 'veldra:x:1000:1000:Veldra Live User:/home/veldra:/bin/bash' >> "$PASSWD_TMP"
    chmod 0644 "$PASSWD_TMP"
    mv -f "$PASSWD_TMP" "$ROOTFS/etc/passwd"
fi

if ! grep -q '^veldra:' "$ROOTFS/etc/group"; then
    GROUP_TMP="$ROOTFS/etc/.group.veldra.tmp"
    cat "$ROOTFS/etc/group" > "$GROUP_TMP"
    printf '%s\n' 'veldra:x:1000:' >> "$GROUP_TMP"
    chmod 0644 "$GROUP_TMP"
    mv -f "$GROUP_TMP" "$ROOTFS/etc/group"
fi

if [[ -f "$ROOTFS/etc/shadow" ]] && ! grep -q '^veldra:' "$ROOTFS/etc/shadow"; then
    SHADOW_TMP="$ROOTFS/etc/.shadow.veldra.tmp"
    cat "$ROOTFS/etc/shadow" > "$SHADOW_TMP"
    printf '%s\n' 'veldra:!:19700:0:99999:7:::' >> "$SHADOW_TMP"
    chmod 0640 "$SHADOW_TMP"
    mv -f "$SHADOW_TMP" "$ROOTFS/etc/shadow"
fi

vd_info "configuring Veldra console services"
install -d -m 0755 "$ROOTFS/etc/systemd/system"

cat > "$ROOTFS/etc/systemd/system/veldra-tui-tty1.service" <<'EOF'
[Unit]
Description=Veldra TUI on VGA tty1
After=systemd-user-sessions.service
Conflicts=getty@tty1.service

[Service]
Type=simple
TTYPath=/dev/tty1
StandardInput=tty-force
StandardOutput=tty
StandardError=tty
TTYReset=yes
TTYVHangup=yes
TTYVTDisallocate=yes
ExecStart=/usr/local/bin/veldra-tui
Restart=on-failure
RestartSec=1

[Install]
WantedBy=multi-user.target
EOF

cat > "$ROOTFS/etc/systemd/system/veldra-tui-serial.service" <<'EOF'
[Unit]
Description=Veldra TUI on serial console
After=systemd-user-sessions.service
Conflicts=serial-getty@ttyS0.service

[Service]
Type=simple
TTYPath=/dev/ttyS0
StandardInput=tty-force
StandardOutput=tty
StandardError=tty
TTYReset=yes
TTYVHangup=yes
ExecStart=/usr/local/bin/veldra-tui
Restart=on-failure
RestartSec=1

[Install]
WantedBy=multi-user.target
EOF

ln -sfn /dev/null "$ROOTFS/etc/systemd/system/getty@tty1.service"
ln -sfn /dev/null "$ROOTFS/etc/systemd/system/serial-getty@ttyS0.service"
install -d -m 0755 "$ROOTFS/etc/systemd/system/multi-user.target.wants"
ln -sfn ../veldra-tui-tty1.service "$ROOTFS/etc/systemd/system/multi-user.target.wants/veldra-tui-tty1.service"
ln -sfn ../veldra-tui-serial.service "$ROOTFS/etc/systemd/system/multi-user.target.wants/veldra-tui-serial.service"
rm -f "$ROOTFS/etc/profile.d/veldra.sh"

# systemd must not repaint the terminal over the fullscreen Bubble Tea UI.
install -d -m 0755 "$ROOTFS/etc/systemd/system.conf.d"
cat > "$ROOTFS/etc/systemd/system.conf.d/90-veldra-tui.conf" <<'EOF'
[Manager]
ShowStatus=no
EOF

install -d -m 0755 "$ROOTFS/etc/veldra"
cat > "$ROOTFS/etc/veldra/replit-build" <<EOF
Veldra ${VERSION}
Build backend: replit-iso
Base image: Arch Linux ${ARCH_RELEASE}
Console backend: systemd tty1 + ttyS0
UI: Veldra Shell fullscreen
EOF
chmod 0644 "$ROOTFS/etc/veldra/replit-build"

vd_info "building modified airootfs.sfs"
mksquashfs "$ROOTFS" "$NEW_SFS" \
    -comp zstd \
    -noappend \
    -all-root \
    -no-xattrs \
    -no-progress
[[ -s "$NEW_SFS" ]] || vd_die 1 "failed to build modified airootfs.sfs"

rm -f "$ISO"
vd_info "rewriting Arch ISO boot image with Veldra live root"
xorriso \
    -indev "$ARCH_ISO" \
    -outdev "$ISO" \
    -boot_image any replay \
    -map "$NEW_SFS" "$SFS_PATH" \
    -commit >/dev/null

[[ -s "$ISO" ]] || vd_die 1 "xorriso did not produce the Veldra ISO"

vd_ok "Replit ISO built: $ISO"
printf '  source: %s\n' "$ARCH_ISO"
printf '  size:   %s\n' "$(du -h "$ISO" | cut -f1)"
printf '  boot:   official Arch BIOS+UEFI metadata preserved\n'
exit 0
