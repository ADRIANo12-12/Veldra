#!/usr/bin/env bash
# scripts/build-tui.sh — build the Veldra TUI binary.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Builds the static Veldra TUI (Go) with the real version, channel, build
# status and arch injected from config/veldra.conf (single source of
# truth). CGO is disabled so the binary is a static, self-contained
# linux/amd64 executable that runs identically on the build host and on the
# (Arch) live/installed systems.
#
# Usage:
#   scripts/build-tui.sh                 build for linux/amd64
#   scripts/build-tui.sh --check         verify the source tree compiles
#
# Output: tui/bin/veldra-tui

set -euo pipefail
# shellcheck source=scripts/common.sh
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SELF_DIR}/common.sh"

vd_require go

MODE=build
[[ "${1:-}" == "--check" ]] && MODE=check

VERSION="$("${SELF_DIR}/version.sh" show)"
CHANNEL="$("${SELF_DIR}/version.sh" channel)"
STATUS="$("${SELF_DIR}/version.sh" status)"
ARCH="${VELDRA_ARCH_DEFAULT}"
GO_ARCH="amd64"
case "$ARCH" in
    x86_64)  GO_ARCH="amd64" ;;
    aarch64) GO_ARCH="arm64" ;;
    armv7l)  GO_ARCH="arm" ;;
    riscv64) GO_ARCH="riscv64" ;;
    *)       GO_ARCH="${GOARCH:-$ARCH}" ;;
esac

PKG="veldra/tui/system"
LDFLAGS="-s -w \
    -X ${PKG}.Version=${VERSION} \
    -X ${PKG}.Channel=${CHANNEL} \
    -X ${PKG}.BuildStatus=${STATUS} \
    -X ${PKG}.ReleaseArch=${ARCH}"

OUT="${VELDRA_PROJECT_ROOT}/tui/bin/veldra-tui"

if [[ "$MODE" == "check" ]]; then
    vd_info "checking TUI source (compile + vet)"
    (cd "$VELDRA_PROJECT_ROOT" && go build ./cmd/... ./tui/... && go vet ./cmd/... ./tui/...)
    vd_ok "TUI source compiles cleanly"
    exit 0
fi

vd_info "building Veldra TUI — $VERSION / $ARCH"
mkdir -p "$(dirname "$OUT")"
# shellcheck disable=SC2086
GOOS=linux GOARCH="$GO_ARCH" CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "$LDFLAGS" \
    -o "$OUT" \
    "${VELDRA_PROJECT_ROOT}/tui/cmd/veldra"

chmod 0755 "$OUT"

SIZE="$(du -h "$OUT" | cut -f1)"
FILE="$(file -b "$OUT" 2>/dev/null || true)"
vd_ok "TUI built: $OUT ($SIZE, ${FILE:-})"

"$OUT" --version 2>/dev/null || true
exit 0