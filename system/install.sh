#!/usr/bin/env bash
# system/install.sh — stage Veldra system identity into a target root.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Real, incremental, non-destructive: it only writes files under the root
# directory you give it. It never touches anything else. The build/
# pipeline calls this to prepare a rootfs/chroot; it also works for
# installing onto a mounted Veldra target.
#
# Usage:
#   system/install.sh <rootdir> [--autologin <user>]
#   system/install.sh --version        print version and exit
#
# Every generated file is derived from config/veldra.conf via
# scripts/version.sh inject — versions are never hardcoded here.

set -euo pipefail
# shellcheck source=scripts/common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/common.sh"

if [[ "${1:-}" == "--version" ]]; then
    "${VELDRA_PROJECT_ROOT}/scripts/version.sh" show
    exit 0
fi

ROOT="${1:-}"
[[ -n "$ROOT" ]] || vd_die 2 "usage: system/install.sh <rootdir> [--autologin <user>]"

AUTOLOGIN_USER=""
if [[ "${2:-}" == "--autologin" ]]; then
    AUTOLOGIN_USER="${3:-}"
    [[ -n "$AUTOLOGIN_USER" ]] || vd_die 2 "--autologin requires a username"
fi

[[ -d "$ROOT" ]] || vd_die 2 "target root '$ROOT' is not a directory"

SRC="${VELDRA_PROJECT_ROOT}/system"
INJECT="${VELDRA_PROJECT_ROOT}/scripts/version.sh"
VERSION="$("${INJECT}" show)"

vd_info "staging Veldra system identity ($VERSION) into $ROOT"

install -d -m 0755 "${ROOT}/etc"
"${INJECT}" inject "${SRC}/os-release.in" >"${ROOT}/etc/os-release"
vd_ok "etc/os-release"

install -d -m 0755 "${ROOT}/etc/profile.d"
install -m 0644 "${SRC}/profile.d/veldra.sh" "${ROOT}/etc/profile.d/veldra.sh"
vd_ok "etc/profile.d/veldra.sh"

install -d -m 0755 "${ROOT}/etc/skel"
install -m 0644 "${SRC}/skel/.bashrc" "${ROOT}/etc/skel/.bashrc"
vd_ok "etc/skel/.bashrc"

install -m 0644 "${SRC}/hostname" "${ROOT}/etc/hostname"
vd_ok "etc/hostname"

# Veldra user default shell for new users.
install -d -m 0755 "${ROOT}/etc/default"
printf 'SHELL=/bin/bash\n' >"${ROOT}/etc/default/useradd"
vd_ok "etc/default/useradd"

# systemd text-console autologin so the Veldra TUI starts as the first
# interactive environment (only when a live/install user is requested).
if [[ -n "$AUTOLOGIN_USER" ]]; then
    install -d -m 0755 "${ROOT}/etc/systemd/system/getty@tty1.service.d"
    "${INJECT}" inject "${SRC}/systemd/getty-veldra-autologin.conf.in" \
        | sed "s/@USER@/${AUTOLOGIN_USER}/" \
        >"${ROOT}/etc/systemd/system/getty@tty1.service.d/veldra-autologin.conf"
    vd_ok "getty@tty1 autologin -> ${AUTOLOGIN_USER}"
fi

# Veldra branding copy for the live/installed system.
install -d -m 0755 "${ROOT}/usr/share/veldra/branding"
if [[ -f "${VELDRA_PROJECT_ROOT}/branding/logo.txt" ]]; then
    install -m 0644 "${VELDRA_PROJECT_ROOT}/branding/logo.txt" \
        "${ROOT}/usr/share/veldra/branding/logo.txt"
    vd_ok "usr/share/veldra/branding/logo.txt"
fi

vd_ok "Veldra system identity staged"
exit 0
