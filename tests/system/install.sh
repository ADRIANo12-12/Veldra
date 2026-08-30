#!/usr/bin/env bash
# tests/system/install.sh — verify the system layer staging.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.

set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "$ROOT/scripts/common.sh"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

"$ROOT/system/install.sh" "$TMP" --autologin veldra >/dev/null

pass=0
fail() { echo "FAIL: $*"; exit 1; }

[[ -f "$TMP/etc/os-release" ]] || fail "etc/os-release missing"
grep -q "^NAME=\"Veldra\"" "$TMP/etc/os-release" || fail "os-release lacks NAME"
grep -q "^VERSION_ID=" "$TMP/etc/os-release" || fail "os-release lacks VERSION_ID"
grep -q "@VERSION@" "$TMP/etc/os-release" && fail "os-release has unexpanded token"

[[ -f "$TMP/etc/profile.d/veldra.sh" ]] || fail "profile.d script missing"
[[ -f "$TMP/etc/skel/.bashrc" ]] || fail "skel .bashrc missing"
[[ -f "$TMP/etc/hostname" ]] || fail "hostname missing"
[[ -f "$TMP/etc/systemd/system/getty@tty1.service.d/veldra-autologin.conf" ]] \
    || fail "getty autologin drop-in missing"
grep -q -- "--autologin veldra" \
    "$TMP/etc/systemd/system/getty@tty1.service.d/veldra-autologin.conf" \
    || fail "autologin user not set"

# version comes from the single source of truth
EXPECTED="$("$ROOT/scripts/version.sh" show)"
grep -q "VERSION_ID=\"$EXPECTED\"" "$TMP/etc/os-release" \
    || fail "os-release VERSION_ID != $EXPECTED"

echo "PASS: system layer (os-release/profile.d/skel/hostname/autologin/version)"
exit 0