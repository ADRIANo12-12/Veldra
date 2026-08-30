#!/usr/bin/env bash
# boot/install.sh — stage the Veldra boot/initramfs layer into a target root.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Stages the Veldra live initramfs configuration and the veldra-live
# mkinitcpio hook into a target root directory (non-destructive; writes
# only under the given root). The live initramfs is built from these by the
# ISO build (build/iso.sh). Installed systems ignore the live config and
# keep the stock Arch initramfs configuration.
#
# Usage:
#   boot/install.sh <rootdir>

set -euo pipefail
# shellcheck source=scripts/common.sh
# shellcheck disable=SC1091 # resolved via SELF_DIR at runtime
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SELF_DIR}/../scripts/common.sh"

ROOT="${1:-}"
[[ -n "$ROOT" ]] || vd_die 2 "usage: boot/install.sh <rootdir>"
[[ -d "$ROOT" ]] || vd_die 2 "target root '$ROOT' is not a directory"

SRC="${SELF_DIR}"

vd_info "staging Veldra boot layer into $ROOT"

install -d -m 0755 "${ROOT}/etc/initcpio/hooks" \
                  "${ROOT}/etc/initcpio/install" \
                  "${ROOT}/usr/share/veldra/boot"

install -m 0644 "${SRC}/initramfs/mkinitcpio-veldra.conf" \
    "${ROOT}/etc/mkinitcpio-veldra.conf"
vd_ok "etc/mkinitcpio-veldra.conf"

install -m 0755 "${SRC}/initramfs/hooks/veldra-live" \
    "${ROOT}/etc/initcpio/hooks/veldra-live"
vd_ok "etc/initcpio/hooks/veldra-live"

install -m 0755 "${SRC}/initramfs/install/veldra-live" \
    "${ROOT}/etc/initcpio/install/veldra-live"
vd_ok "etc/initcpio/install/veldra-live"

# A copy of the live initramfs config for the boot build step.
install -m 0644 "${SRC}/initramfs/mkinitcpio-veldra.conf" \
    "${ROOT}/usr/share/veldra/boot/mkinitcpio-veldra.conf"
vd_ok "usr/share/veldra/boot/mkinitcpio-veldra.conf"

vd_ok "Veldra boot layer staged"
exit 0