#!/usr/bin/env bash
# build/iso.sh — build the Veldra live ISO.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Produces a bootable BIOS+UEFI live ISO via grub-mkrescue (GRUB -> xorriso):
#   1. extract the rootfs archive into a working "airootfs"
#   2. build the live squashfs (the whole rootfs except /boot) — the
#      initramfs's veldra-live hook mounts it + a tmpfs overlay as root
#   3. assemble the ISO root: boot/ (kernel, live initramfs, grub.cfg) +
#      live.sfs
#   4. grub-mkrescue -o build/out/veldra-<version>-x86_64.iso
#
# Gated: root + Arch build env. On a non-Arch host the script re-executes
# itself inside the isolated Arch build container. Plan mode
# (VELDRA_BUILD_DRYRUN=1) prints the plan and writes nothing.

set -euo pipefail
# shellcheck source=scripts/common.sh
# shellcheck disable=SC1091
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SELF_DIR}/../scripts/common.sh"

VERSION="$("${VELDRA_PROJECT_ROOT}/scripts/version.sh" show)"
ARCH="${VELDRA_ARCH_DEFAULT:-x86_64}"
WORK="${VELDRA_BUILD_WORK:-${VELDRA_PROJECT_ROOT}/build/work}"
OUT="${VELDRA_BUILD_OUT:-${VELDRA_PROJECT_ROOT}/build/out}"
TAR="${OUT}/veldra-${VERSION}-rootfs.tar.zst"
ISO="${OUT}/veldra-${VERSION}-${ARCH}.iso"
LIVE="${WORK}/live"
STAGE="${WORK}/iso/${VERSION}"
SFS="${WORK}/veldra-live.sfs"

DRY=0
[[ "${VELDRA_BUILD_DRYRUN:-0}" == "1" ]] && DRY=1

plan() {
    printf 'Veldra ISO build (version %s / %s)\n' "$VERSION" "$ARCH"
    printf '  source  : %s\n' "$TAR"
    printf '  output  : %s\n' "$ISO"
    printf '%s\n' \
        "  1. extract rootfs archive -> $LIVE" \
        "  2. mksquashfs (rootfs excl. boot) -> $SFS" \
        "  3. assemble ISO root: boot/ + live.sfs + grub.cfg" \
        "  4. grub-mkrescue -o $ISO (BIOS + UEFI)"
}

if (( DRY )); then
    plan
    vd_ok "plan only — nothing was built"
    exit 0
fi

[[ -f "$TAR" ]] || vd_die 1 \
    "rootfs archive not found: $TAR (run build/rootfs.sh first, or build/build.sh all)"

# On a non-Arch host this re-executes inside the isolated Arch build container.
vd_ensure_arch_build_env "${BASH_SOURCE[0]}" "$@"

vd_require_arch_host
[[ "$(id -u)" == "0" ]] || vd_die 2 \
    "the ISO build must run as root (mounts/loop devices during grub-mkrescue)."

vd_require xorriso grub-mkrescue tar mksquashfs unsquashfs

vd_info "Veldra ISO build — $VERSION / $ARCH"
rm -rf "$LIVE" "$STAGE"
mkdir -p "$LIVE" "$STAGE" "$STAGE/boot" "$STAGE/boot/grub" "$WORK"

vd_info "extracting rootfs archive"
tar --zstd -xf "$TAR" -C "$LIVE"

# --- live squashfs ------------------------------------------------------------
# The whole rootfs except /boot becomes the (read-only) live root. The
# initramfs mounts it and overlays tmpfs for a writable session.
for f in boot/vmlinuz-linux boot/initramfs-linux-veldra.img; do
    [[ -e "$LIVE/$f" ]] || vd_die 1 "rootfs is missing $f — rebuild the rootfs"
done

vd_info "building live squashfs (live.sfs)"
rm -f "$SFS"
if ! ( cd "$LIVE" && mksquashfs . "$SFS" -e boot -no-progress ); then
    vd_die 1 "mksquashfs failed"
fi
ls -lh "$SFS" | awk '{print "  live.sfs: " $5}'

# --- ISO root -----------------------------------------------------------------
vd_info "assembling ISO root"
cp -a "$LIVE/boot/vmlinuz-linux" "$STAGE/boot/vmlinuz-linux"
cp -a "$LIVE/boot/initramfs-linux-veldra.img" "$STAGE/boot/initramfs-linux-veldra.img"

"${VELDRA_PROJECT_ROOT}/scripts/version.sh" inject \
    "${VELDRA_PROJECT_ROOT}/boot/grub/grub.cfg.in" >"$STAGE/boot/grub/grub.cfg"
vd_ok "boot/grub/grub.cfg"

cp -a "$SFS" "$STAGE/live.sfs"
vd_ok "live.sfs"

# --- build ISO ----------------------------------------------------------------
vd_info "grub-mkrescue (BIOS + UEFI)"
grub-mkrescue -o "$ISO" "$STAGE" \
    -x "$LIVE" \
    -- -volid VELDRA 2>/dev/null || \
    grub-mkrescue -o "$ISO" "$STAGE" || vd_die 1 "grub-mkrescue failed"

# The staging dirs are 0555 by pacman's convention; relax the WORK copy so
# the host user can remove and rebuild. The archive itself keeps the modes.
chmod -R u+rwX "$STAGE" "$LIVE" 2>/dev/null || true

vd_ok "ISO built: $ISO"
exit 0