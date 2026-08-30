#!/usr/bin/env bash
# build/replit-iso.sh — build a Replit-compatible Veldra ISO without Docker.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Replit does not expose Docker/Podman daemons or root privileges. Instead of
# building an Arch rootfs from scratch, this backend takes the current
# official Arch Linux installation ISO, replaces its live airootfs.sfs with a
# SquashFS containing the Veldra TUI and autologin setup, then rewrites the ISO
# while preserving the original BIOS/UEFI boot metadata.
#
# This is a compatibility build path for hosted sandboxes. The normal
# `make iso` path remains the canonical Veldra build and still uses the
# isolated Arch container.

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
ROOTFS="${WORK}/rootfs"
WRITABLE_ROOTFS="${WORK}/rootfs-writable"
NEW_SFS="${WORK}/veldra-airootfs.sfs"

vd_require curl xorriso unsquashfs mksquashfs sha256sum awk sed install mkdir rm cp

vd_info "Replit-compatible Veldra ISO builder — ${VERSION} / ${ARCH}"
vd_warn "This is a sandbox compatibility build; canonical 'make iso' remains the normal Arch-container pipeline."

mkdir -p "$WORK" "$OUT"

# --- official Arch source ISO -----------------------------------------------
if [[ ! -f "$ARCH_ISO" ]]; then
    vd_info "downloading official Arch Linux ${ARCH_RELEASE} ISO (~1.5 GB)"
    curl -fL --retry 3 --retry-delay 2 --progress-bar \
        -o "$ARCH_ISO.part" "$ARCH_ISO_URL"
    mv "$ARCH_ISO.part" "$ARCH_ISO"
else
    vd_info "using cached Arch ISO: $ARCH_ISO"
fi

printf '%s  %s\n' "$ARCH_ISO_SHA256" "$ARCH_ISO" | sha256sum -c -
vd_ok "Arch ISO checksum verified"

# Archiso stores the live x86_64 filesystem at this stable path.
SFS_PATH="/arch/x86_64/airootfs.sfs"
vd_info "Arch live root: $SFS_PATH"

# Verify that the expected ISO member exists before extracting it.
if ! xorriso -indev "$ARCH_ISO" -ls "$SFS_PATH" >/dev/null 2>&1; then
    vd_die 1 "official Arch ISO does not contain expected live root: $SFS_PATH"
fi

# --- extract and modify live rootfs -----------------------------------------
rm -f "$SOURCE_SFS" "$NEW_SFS"
rm -rf "$ROOTFS" "$WRITABLE_ROOTFS"
mkdir -p "$ROOTFS" "$WRITABLE_ROOTFS"

vd_info "extracting Arch live rootfs"
xorriso -osirrox on -indev "$ARCH_ISO" \
    -extract "$SFS_PATH" "$SOURCE_SFS" >/dev/null
[[ -s "$SOURCE_SFS" ]] || vd_die 1 "failed to extract $SFS_PATH"

vd_info "unpacking Arch live rootfs without root"
unsquashfs -no-xattrs -d "$ROOTFS" "$SOURCE_SFS" >/dev/null
[[ -d "$ROOTFS/etc" && -d "$ROOTFS/usr" ]] || \
    vd_die 1 "extracted Arch rootfs does not look valid"

# SquashFS stores the original ownership/mode bits. Rebuild a writable working
# tree owned by the current Replit user, without requiring root/chown. GNU cp
# can copy the readable tree while intentionally not preserving ownership.
vd_info "creating writable rootfs working tree"
cp -a --no-preserve=ownership "$ROOTFS/." "$WRITABLE_ROOTFS/"
rm -rf "$ROOTFS"
ROOTFS="$WRITABLE_ROOTFS"

# Build the TUI locally if the caller did not already do so.
TUI_BIN="${VELDRA_PROJECT_ROOT}/tui/bin/veldra-tui"
if [[ ! -x "$TUI_BIN" ]]; then
    vd_info "building Veldra TUI"
    "${VELDRA_PROJECT_ROOT}/scripts/build-tui.sh"
fi

vd_info "installing Veldra TUI into the live rootfs"
install -D -m 0755 "$TUI_BIN" "$ROOTFS/usr/local/bin/veldra-tui"

# The Arch installation ISO already provides systemd. Add a dedicated live
# user and autologin without invoking useradd/chroot or needing root.
vd_info "configuring unprivileged Veldra live user"
install -d -m 0755 "$ROOTFS/home/veldra"

if ! grep -q '^veldra:' "$ROOTFS/etc/passwd"; then
    printf '%s\n' 'veldra:x:1000:1000:Veldra Live User:/home/veldra:/bin/bash' >> "$ROOTFS/etc/passwd"
fi
if ! grep -q '^veldra:' "$ROOTFS/etc/group"; then
    printf '%s\n' 'veldra:x:1000:' >> "$ROOTFS/etc/group"
fi
if [[ -f "$ROOTFS/etc/shadow" ]] && ! grep -q '^veldra:' "$ROOTFS/etc/shadow"; then
    printf '%s\n' 'veldra:!:19700:0:99999:7:::' >> "$ROOTFS/etc/shadow"
fi

install -d -m 0755 "$ROOTFS/etc/systemd/system/getty@tty1.service.d"
cat > "$ROOTFS/etc/systemd/system/getty@tty1.service.d/veldra-autologin.conf" <<'EOF'
[Service]
ExecStart=
ExecStart=-/usr/bin/agetty --autologin veldra --noclear %I $TERM
EOF

install -d -m 0755 "$ROOTFS/etc/systemd/system/serial-getty@ttyS0.service.d"
cat > "$ROOTFS/etc/systemd/system/serial-getty@ttyS0.service.d/veldra-autologin.conf" <<'EOF'
[Service]
ExecStart=
ExecStart=-/usr/bin/agetty --autologin veldra --noclear --keep-baud %I 115200 linux
EOF

install -d -m 0755 "$ROOTFS/etc/profile.d"
cat > "$ROOTFS/etc/profile.d/veldra.sh" <<'EOF'
# Start the Veldra TUI automatically on the first local virtual terminal.
# Do not interfere with SSH, serial consoles, or non-interactive shells.
if [[ -t 1 && "$(tty 2>/dev/null || true)" == "/dev/tty1" && -z "${VELDRA_TUI_ACTIVE:-}" ]]; then
    export VELDRA_TUI_ACTIVE=1
    exec /usr/local/bin/veldra-tui
fi
EOF
chmod 0644 "$ROOTFS/etc/profile.d/veldra.sh"

# Keep an explicit identity marker for diagnostics inside this compatibility ISO.
install -d -m 0755 "$ROOTFS/etc/veldra"
cat > "$ROOTFS/etc/veldra/replit-build" <<EOF
Veldra ${VERSION}
Build backend: replit-iso
Base image: Arch Linux ${ARCH_RELEASE}
EOF
chmod 0644 "$ROOTFS/etc/veldra/replit-build"

# Repack the rootfs as root-owned entries for a normal Arch live filesystem.
# Ownership in the archive is metadata; the build itself remains unprivileged.
vd_info "building modified airootfs.sfs"
mksquashfs "$ROOTFS" "$NEW_SFS" -comp zstd -noappend -all-root -no-xattrs -no-progress
[[ -s "$NEW_SFS" ]] || vd_die 1 "failed to build modified airootfs.sfs"

# --- rewrite only the live filesystem inside the official ISO ---------------
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
