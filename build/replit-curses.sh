#!/usr/bin/env bash
# build/replit-curses.sh — boot Veldra in QEMU curses using the ArchISO kernel directly.
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
KERNEL="$WORK/vmlinuz-linux"
INITRD="$WORK/initramfs-linux.img"

vd_require xorriso mkdir rm test qemu-system-x86_64
[[ -s "$ISO" ]] || vd_die 1 "Replit ISO not found: $ISO"

mkdir -p "$WORK"

vd_info "preparing Veldra for QEMU curses"

# ArchISO places the live kernel and initramfs under arch/boot/x86_64.
# Boot them directly so QEMU can pass nomodeset without relying on a
# particular GRUB/Syslinux layout inside the ISO.
for kernel_path in \
    /arch/boot/x86_64/vmlinuz-linux \
    /arch/boot/x86_64/vmlinuz-linux-lts; do
    if xorriso -osirrox on -indev "$ISO" -extract "$kernel_path" "$KERNEL" >/dev/null 2>&1 && [[ -s "$KERNEL" ]]; then
        vd_ok "using ArchISO kernel: $kernel_path"
        break
    fi
    rm -f "$KERNEL"
done

[[ -s "$KERNEL" ]] || vd_die 1 "could not extract an ArchISO kernel"

for initrd_path in \
    /arch/boot/x86_64/initramfs-linux.img \
    /arch/boot/x86_64/initramfs-linux-lts.img; do
    if xorriso -osirrox on -indev "$ISO" -extract "$initrd_path" "$INITRD" >/dev/null 2>&1 && [[ -s "$INITRD" ]]; then
        vd_ok "using ArchISO initramfs: $initrd_path"
        break
    fi
    rm -f "$INITRD"
done

[[ -s "$INITRD" ]] || vd_die 1 "could not extract an ArchISO initramfs"

# The Arch live system discovers its squashfs on the attached CD using the
# standard archiso parameters. nomodeset keeps the guest on VGA text mode,
# which is exactly what QEMU's curses display can render.
APPEND="archisobasedir=arch archisolabel=ARCH_202608 nomodeset console=tty1"

vd_info "booting Veldra with QEMU curses (VGA text mode)"
exec qemu-system-x86_64 \
    -machine accel=tcg \
    -m 1024 \
    -kernel "$KERNEL" \
    -initrd "$INITRD" \
    -append "$APPEND" \
    -cdrom "$ISO" \
    -display curses \
    -vga std \
    -monitor none
