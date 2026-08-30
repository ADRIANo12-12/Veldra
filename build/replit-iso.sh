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

# Replit ships an older SquashFS toolset. Automatically re-exec this builder
# inside the current nixpkgs-unstable squashfsTools package when needed.
if [[ "${VELDRA_REPLIT_SQUASHFS_REEXEC:-0}" != "1" ]] && command -v nix >/dev/null 2>&1 && command -v unsquashfs >/dev/null 2>&1; then
    SQUASHFS_VERSION="$(unsquashfs -version 2>&1 | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -n1 || true)"
    if [[ -z "$SQUASHFS_VERSION" ]] || [[ "$(printf '%s\n' "$SQUASHFS_VERSION" "4.7.6" | sort -V | head -n1)" != "4.7.6" ]]; then
        vd_info "Replit: switching to nixpkgs unstable squashfs-tools"
        exec nix shell github:NixOS/nixpkgs/nixos-unstable#squashfsTools \
            --command env VELDRA_REPLIT_SQUASHFS_REEXEC=1 bash "$0" "$@"
    fi
fi

vd_require curl xorriso unsquashfs mksquashfs sha256sum awk sed install mkdir rm du

vd_info "Replit-compatible Veldra ISO builder — ${VERSION} / ${ARCH}"
vd_warn "Sandbox compatibility path; canonical 'make iso' remains the Arch-container pipeline."

mkdir -p "$WORK" "$OUT"

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

SFS_PATH="/arch/x86_64/airootfs.sfs"
vd_info "Arch live root: $SFS_PATH"
if ! xorriso -indev "$ARCH_ISO" -ls "$SFS_PATH" >/dev/null 2>&1; then
    vd_die 1 "official Arch ISO does not contain expected live root: $SFS_PATH"
fi

rm -f "$SOURCE_SFS" "$NEW_SFS"
rm -rf "$ROOTFS"
mkdir -p "$ROOTFS"

vd_info "extracting Arch live rootfs"
xorriso -osirrox on -indev "$ARCH_ISO" \
    -extract "$SFS_PATH" "$SOURCE_SFS" >/dev/null
[[ -s "$SOURCE_SFS" ]] || vd_die 1 "failed to extract $SFS_PATH"

vd_info "unpacking Arch live rootfs with writable permissions"
if ! unsquashfs -help 2>&1 | grep -q -- '-force-file-mode'; then
    vd_die 2 "SquashFS 4.7.6+ was not activated; run 'nix shell github:NixOS/nixpkgs/nixos-unstable#squashfsTools --command bash build/replit-iso.sh'"
fi
unsquashfs \
    -no-xattrs \
    -force-file-mode 'ugo+rX,u+rw' \
    -force-dir-mode 'ugo+rwx' \
    -d "$ROOTFS" "$SOURCE_SFS" >/dev/null
[[ -d "$ROOTFS/etc" && -d "$ROOTFS/usr" ]] || \
    vd_die 1 "extracted Arch rootfs does not look valid"

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
if [[ -t 1 && "$(tty 2>/dev/null || true)" == "/dev/tty1" && -z "${VELDRA_TUI_ACTIVE:-}" ]]; then
    export VELDRA_TUI_ACTIVE=1
    exec /usr/local/bin/veldra-tui
fi
EOF
chmod 0644 "$ROOTFS/etc/profile.d/veldra.sh"

install -d -m 0755 "$ROOTFS/etc/veldra"
cat > "$ROOTFS/etc/veldra/replit-build" <<EOF
Veldra ${VERSION}
Build backend: replit-iso
Base image: Arch Linux ${ARCH_RELEASE}
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
