#!/usr/bin/env bash
# tests/boot/install.sh — verify the boot layer staging.
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

"$ROOT/boot/install.sh" "$TMP" >/dev/null

fail() { echo "FAIL: $*"; exit 1; }

[[ -f "$TMP/etc/mkinitcpio-veldra.conf" ]] || fail "mkinitcpio-veldra.conf missing"
[[ -f "$TMP/etc/initcpio/hooks/veldra-live" ]] || fail "hook missing"
[[ -f "$TMP/etc/initcpio/install/veldra-live" ]] || fail "install function missing"
[[ -x "$TMP/etc/initcpio/hooks/veldra-live" ]] || fail "hook not executable"

bash -n "$TMP/etc/initcpio/hooks/veldra-live" || fail "hook has syntax errors"
bash -n "$TMP/etc/initcpio/install/veldra-live" || fail "install function has syntax errors"

grep -q "veldra-live" "$TMP/etc/mkinitcpio-veldra.conf" || fail "HOOKS missing veldra-live"
grep -E "^HOOKS=" "$TMP/etc/mkinitcpio-veldra.conf" | grep -q "autodetect" \
    && fail "live conf must NOT use autodetect"

echo "PASS: boot layer (mkinitcpio-veldra.conf + veldra-live hook)"
exit 0