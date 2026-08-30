#!/usr/bin/env bash
# common.sh — shared library for all Veldra scripts.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Source this file first in every script:
#     source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"
#
# Provides: strict shell discipline, consistent logging, environment
# detection, and the central config loader. Pure bash + coreutils, so it
# works on a build host, inside a chroot, and in the live installer.

set -euo pipefail

# Resolve the Veldra project root regardless of where scripts are invoked
# from (repo scripts, installed system tools, etc.).
# shellcheck disable=SC2128
VELDRA_PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export VELDRA_PROJECT_ROOT

# --- Base ANSI constants --------------------------------------------------
VELDRA_ESC=$'\033'

# --- Logging --------------------------------------------------------------
vd_info()  { printf '%s[INFO]%s %s\n' "${VELDRA_ESC}[1;36m" "${VELDRA_ESC}[0m" "$*"; }
vd_ok()    { printf '%s[ OK ]%s %s\n' "${VELDRA_ESC}[1;32m" "${VELDRA_ESC}[0m" "$*"; }
vd_warn()  { printf '%s[WARN]%s %s\n' "${VELDRA_ESC}[1;33m" "${VELDRA_ESC}[0m" "$*" >&2; }
vd_error() { printf '%s[ERROR]%s %s\n' "${VELDRA_ESC}[1;31m" "${VELDRA_ESC}[0m" "$*" >&2; }
vd_debug() { [[ "${VELDRA_DEBUG:-0}" == "1" ]] && printf '%s[DEBUG]%s %s\n' "${VELDRA_ESC}[1;90m" "${VELDRA_ESC}[0m" "$*" >&2; return 0; }

# Die: print an error and exit. Accepts an optional exit code as first arg.
vd_die() {
    local code="${1:-1}"
    shift || true
    vd_error "$*"
    exit "$code"
}

# Fail clearly when a prerequisite tool is missing.
vd_require() {
    local missing=()
    local tool
    for tool in "$@"; do
        command -v "$tool" >/dev/null 2>&1 || missing+=("$tool")
    done
    if ((${#missing[@]} > 0)); then
        vd_die 2 "missing required tool(s): ${missing[*]}"
    fi
}

# Confirmation prompt. Returns 0 when the user explicitly confirms.
# Set VELDRA_AUTOPROCEED=1 to auto-confirm (headless testing/infra).
# Usage: vd_confirm "question" ; then safe to proceed.
vd_confirm() {
    local answer
    [[ "${VELDRA_AUTOPROCEED:-0}" == "1" ]] && return 0
    printf '%s [y/N] ' "$*"
    read -r answer </dev/tty 2>/dev/null || read -r answer
    case "${answer:-n}" in
        [yY]|[yY][eE][sS]) return 0 ;;
        *) return 1 ;;
    esac
}

# --- Environment detection ------------------------------------------------
vd_host_distro() {
    if [[ -r /etc/os-release ]]; then
        awk -F= '/^ID=/{gsub(/["\r]/,"",$2); print $2}' /etc/os-release
    elif [[ -r /etc/arch-release ]]; then
        printf 'arch\n'
    else
        printf 'unknown\n'
    fi
}

# True when the current host looks like a usable Arch Linux build host.
vd_is_arch_host() { [[ "$(vd_host_distro)" == "arch" ]]; }

# The Veldra base is Arch Linux. Non-Arch hosts can still run every test,
# TUI build and documentation check; only full ISO builds need Arch.
vd_require_arch_host() {
    vd_is_arch_host || vd_die 2 \
        "Veldra must be built on Arch Linux (found: $(vd_host_distro))." \
        "See docs/build.md — real builds run in the isolated Arch build container."
}

# --- Isolated Arch build container ------------------------------------------
# Veldra is Arch-based, so real ISO/rootfs builds need Arch tooling
# (pacman, arch-chroot, grub-mkrescue, xorriso). On any host that is not
# "root on Arch" (and not already inside the Arch build container), build
# scripts re-execute themselves inside an isolated Arch container via
# build/container/run.sh. Podman is preferred, Docker is the fallback.

vd_container_runtime() {
    if [[ -n "${VELDRA_CONTAINER_RUNTIME:-}" ]]; then
        case "$VELDRA_CONTAINER_RUNTIME" in
            none) return 1 ;;
            podman|docker)
                command -v "$VELDRA_CONTAINER_RUNTIME" >/dev/null 2>&1 \
                    && { printf '%s\n' "$VELDRA_CONTAINER_RUNTIME"; return 0; }
                return 1 ;;
            *)
                vd_die 2 "VELDRA_CONTAINER_RUNTIME='$VELDRA_CONTAINER_RUNTIME' is invalid (podman|docker|none)" ;;
        esac
    fi
    command -v podman >/dev/null 2>&1 && { printf 'podman\n'; return 0; }
    command -v docker >/dev/null 2>&1 && { printf 'docker\n'; return 0; }
    return 1
}

vd_needs_arch_container() {
    if [[ "${VELDRA_IN_ARCH_CONTAINER:-0}" == "1" ]]; then return 1; fi
    if vd_is_arch_host && [[ "$(id -u)" == "0" ]]; then return 1; fi
    return 0
}

vd_ensure_arch_build_env() {
    local script="${1:-}"
    shift || true
    vd_needs_arch_container || return 0
    local runtime
    runtime="$(vd_container_runtime)" || runtime=""
    if [[ -z "$runtime" ]]; then
        vd_error "Veldra ISO builds need the Arch Linux build environment, which"
        vd_error "is provided through an isolated container on non-Arch hosts."
        vd_error "No container runtime was found. Install Podman or Docker:"
        printf '%s\n' "    Fedora:        sudo dnf install podman" \
            "    Debian/Ubuntu: sudo apt install podman" \
            "    openSUSE:      sudo zypper install podman" \
            "    Arch Linux:    sudo pacman -S podman" \
            "    Docker (any distro): install docker — the wrapper falls back to it automatically" >&2
        vd_die 2 "see docs/build.md for the portable build workflow"
    fi
    vd_info "host is $(vd_host_distro) — running '$(basename "$script")' inside the isolated Arch build container ($runtime)"
    exec "${VELDRA_PROJECT_ROOT}/build/container/run.sh" "$script" "$@"
}

# --- Config loading -------------------------------------------------------
vd_load_config() {
    local cfg="${VELDRA_PROJECT_ROOT}/config/veldra.conf"
    [[ -f "$cfg" ]] || cfg="/etc/veldra/veldra.conf"
    [[ -f "$cfg" ]] || vd_die 2 "cannot find config/veldra.conf"
    # shellcheck disable=SC1090
    source "$cfg"
}
vd_load_config

# --- Misc helpers ----------------------------------------------------------
vd_is_tty() { [[ -t 1 ]]; }
