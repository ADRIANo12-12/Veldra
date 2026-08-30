#!/usr/bin/env bash
# build/build.sh — Veldra build orchestrator.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Builds the Veldra distribution end to end:
#
#   tui     -> build the TUI (static binary, version injected)
#   rootfs  -> build the Arch rootfs + live initramfs (in the Arch container)
#   iso     -> assemble + burn the bootable live ISO (in the Arch container)
#   all     -> tui + rootfs + iso
#   plan    -> print the build plan (nothing is built)
#   clean   -> remove build artifacts and work products
#
# The TUI builds on the host; rootfs and ISO each self-schedule inside the
# isolated Arch build container on demand.
#
# Usage:
#   build/build.sh [all|tui|rootfs|iso|plan|clean]

set -euo pipefail
# shellcheck source=scripts/common.sh
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SELF_DIR}/../scripts/common.sh"

VERSION="$("${VELDRA_PROJECT_ROOT}/scripts/version.sh" show)"
OUT="${VELDRA_BUILD_OUT:-${VELDRA_PROJECT_ROOT}/build/out}"
WORK="${VELDRA_BUILD_WORK:-${VELDRA_PROJECT_ROOT}/build/work}"

cmd="${1:-all}"

cmd_tui() {
    vd_info "step 1/3: build the Veldra TUI"
    "${VELDRA_PROJECT_ROOT}/scripts/build-tui.sh"
}

cmd_rootfs() {
    vd_info "step 2/3: build the Veldra rootfs (Arch base + live initramfs)"
    "${SELF_DIR}/rootfs.sh"
}

cmd_iso() {
    vd_info "step 3/3: build the Veldra live ISO"
    "${SELF_DIR}/iso.sh"
}

cmd_plan() {
    printf 'Veldra build plan (version %s)\n' "$VERSION"
    printf '  output  : %s/veldra-%s-x86_64.iso\n' "$OUT" "$VERSION"
    printf '%s\n' \
        "  [ tui ]     build the static TUI binary (host)" \
        "  [ rootfs ]  Arch rootfs + live initramfs (isolated Arch container)" \
        "  [ iso ]     bootable BIOS/UEFI live ISO (isolated Arch container)" \
        "  [ all ]     tui -> rootfs -> iso"
    vd_ok "plan only — nothing was built"
    exit 0
}

cmd_clean() {
    vd_info "cleaning build artifacts"
    rm -rf "$OUT" "$WORK"
    vd_ok "removed $OUT and $WORK"
    exit 0
}

case "$cmd" in
    all)   cmd_tui; vd_ok "TUI done"; cmd_rootfs; cmd_iso ;;
    tui)   cmd_tui ;;
    rootfs) cmd_rootfs ;;
    iso)   cmd_iso ;;
    plan)  cmd_plan ;;
    clean) cmd_clean ;;
    *)
        vd_die 1 "unknown command '$cmd' (all|tui|rootfs|iso|plan|clean)" ;;
esac

vd_ok "Veldra $VERSION build complete"
printf '%s\n' \
    "  ISO:    ${OUT}/veldra-${VERSION}-x86_64.iso" \
    "  Rootfs: ${OUT}/veldra-${VERSION}-rootfs.tar.zst" \
    "  Boot:   qemu-system-x86_64 -cdrom ${OUT}/veldra-${VERSION}-x86_64.iso -nographic"
exit 0