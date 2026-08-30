#!/usr/bin/env bash
# build/replit-curses.sh — make the Replit ISO stay in VGA text mode for QEMU curses.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$ROOT/scripts/common.sh"

VERSION="$("$ROOT/scripts/version.sh" show)"
ARCH="${VELDRA_ARCH_DEFAULT:-x86_64}"
ISO="$ROOT/build/out/veldra-${VERSION}-${ARCH}.iso"
WORK="$ROOT/build/work/replit/curses-${EPOCHSECONDS:-$(date +%s)}-${RANDOM}"
PATCHED="$WORK/patched.iso"

vd_require xorriso mkdir rm cp sed grep test
[[ -s "$ISO" ]] || vd_die 1 "Replit ISO not found: $ISO"

mkdir -p "$WORK"

PATCHES=()

patch_file() {
    local iso_path="$1"
    local out_file="$WORK/$(echo "$iso_path" | tr '/:' '__')"

    if ! xorriso -indev "$ISO" -osirrox on -extract "$iso_path" "$out_file" >/dev/null 2>&1; then
        return 1
    fi

    [[ -s "$out_file" ]] || return 1

    case "$iso_path" in
        */grub.cfg)
            sed -E -i \
                '/^[[:space:]]*(linux|linuxefi)([[:space:]]|$)/ {
                    /nomodeset/! s/[[:space:]]*$/ nomodeset/;
                }' "$out_file"
            ;;
        */syslinux*.cfg)
            sed -E -i \
                '/^[[:space:]]*APPEND([[:space:]]|$)/ {
                    /nomodeset/! s/[[:space:]]*$/ nomodeset/;
                }' "$out_file"
            ;;
        *)
            return 1
            ;;
    esac

    PATCHES+=("$out_file|$iso_path")
    vd_ok "patched boot config: $iso_path (nomodeset)"
    return 0
}

vd_info "preparing Replit ISO for QEMU curses"

# The BIOS GRUB config is the important path for the x86_64 QEMU invocation.
patch_file "/boot/grub/grub.cfg" || true

# Cover common ArchISO Syslinux layouts as a fallback/secondary BIOS path.
patch_file "/syslinux/syslinux.cfg" || true
patch_file "/syslinux/archiso_sys.cfg" || true
patch_file "/syslinux/archiso_sys-linux.cfg" || true

if [[ ${#PATCHES[@]} -eq 0 ]]; then
    vd_die 1 "could not locate a patchable BIOS boot configuration in the Replit ISO"
fi

rm -f "$PATCHED"

XORRISO_ARGS=(
    -indev "$ISO"
    -outdev "$PATCHED"
    -boot_image any replay
)
for patch in "${PATCHES[@]}"; do
    src="${patch%%|*}"
    dst="${patch#*|}"
    XORRISO_ARGS+=( -map "$src" "$dst" )
done
XORRISO_ARGS+=( -commit )

xorriso "${XORRISO_ARGS[@]}" >/dev/null
[[ -s "$PATCHED" ]] || vd_die 1 "failed to create curses-compatible ISO"

mv -f "$PATCHED" "$ISO"
vd_ok "curses-compatible Veldra ISO ready: $ISO"
