#!/usr/bin/env bash
# tests/run-all.sh — run every Veldra shell test.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.

set -euo pipefail
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
rc=0

tests=(
    "$TEST_DIR/system/install.sh"
    "$TEST_DIR/boot/install.sh"
    "$TEST_DIR/installer/lib.sh"
)

for t in "${tests[@]}"; do
    echo "== $(basename "$(dirname "$t")")/$(basename "$t") =="
    if bash "$t"; then
        echo
    else
        echo "FAILED: $t"
        rc=1
    fi
done

exit "$rc"