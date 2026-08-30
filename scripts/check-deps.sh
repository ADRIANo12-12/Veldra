#!/usr/bin/env bash
# check-deps.sh — verify the Veldra build/dev host prerequisites.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Everything needed to run tests, build the Go TUI, and run checks is
# verified unconditionally. The ISO build itself needs an Arch build
# environment; on non-Arch hosts that environment comes from an isolated
# Arch container (Podman preferred, Docker acceptable), so at least one
# container runtime must exist. QEMU is a host runtime dependency for
# running the built ISO headless (reported as optional warning unless
# make qemu is the goal).
#
# Exit codes: 0 all good, 1 at least one required dependency missing.

set -euo pipefail
# shellcheck source=scripts/common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

# --- dependency catalog -----------------------------------------------------
declare -A DEPS=(
    [bash]=bash
    [coreutils]=true
    [grep]=grep
    [sed]=sed
    [awk]=awk
    [findutils]=find
    [tar]=tar
    [gzip]=gzip
    [zstd]=zstd
    [git]=git
    [make]=make
    [go]=go
)

declare -A DEV_DEPS=(
    [shellcheck]=shellcheck
    [seq]=seq
    [tput]=tput
)

RUNTIME_DEPS=(
    podman
    docker
)

missing_deps=0

check_tool() { # label command
    local label="$1" cmd="$2"
    if command -v "$cmd" >/dev/null 2>&1; then
        printf '  %-14s \033[1;32mok\033[0m\n' "${label}[${cmd}]"
        return 0
    fi
    printf '  %-14s \033[1;31mMISSING\033[0m\n' "${label}[${cmd}]"
    missing_deps=$((missing_deps + 1))
    return 1
}

runtime_hint() {
    printf '%s\n' \
        "  The ISO build provides the Arch build environment (the OS base is Arch" \
        "  Linux) through an isolated container. Install Podman or Docker:" \
        "    Fedora:        sudo dnf install podman" \
        "    Debian/Ubuntu: sudo apt install podman" \
        "    openSUSE:      sudo zypper install podman" \
        "    Arch Linux:    sudo pacman -S podman" \
        "    Alpine:        sudo apk add podman" \
        "    Docker (any distro): install docker — the wrapper falls back to it automatically" \
        "  (On a root Arch host the build can run directly instead.)"
}

qemu_hint() {
    case "$(vd_host_distro)" in
        fedora)          printf '  Fedora:      sudo dnf install qemu-system-x86\n' ;;
        arch)            printf '  Arch Linux:  sudo pacman -S qemu-desktop\n' ;;
        debian|ubuntu)   printf '  Debian/Ubuntu: sudo apt install qemu-system-x86\n' ;;
        opensuse*|suse*) printf '  openSUSE:    sudo zypper install qemu-x86\n' ;;
        *)               printf '  Install the qemu-system-x86 package for %s\n' "$(vd_host_distro)" ;;
    esac
}

vd_info "core build/dev dependencies"
for label in "${!DEPS[@]}"; do
    check_tool "$label" "${DEPS[$label]}" || true
done

vd_info "developer/QoL dependencies (recommended)"
for label in "${!DEV_DEPS[@]}"; do
    check_tool "$label" "${DEV_DEPS[$label]}" || true
done

# --- container runtime (Arch build environment) -----------------------------
vd_info "container runtime (provides the isolated Arch build environment)"
runtime_present=0
for t in "${RUNTIME_DEPS[@]}"; do
    if command -v "$t" >/dev/null 2>&1; then
        printf '  %-14s \033[1;32mok\033[0m\n' "${t}[$t]"
        runtime_present=$((runtime_present + 1))
    else
        printf '  %-14s \033[1;31mMISSING\033[0m\n' "${t}[$t]"
    fi
done
if (( runtime_present == 0 )); then
    if vd_is_arch_host && [[ "$(id -u)" == "0" ]]; then
        vd_warn "no container runtime found — a root Arch host can build directly, but a runtime is recommended"
    else
        vd_error "no container runtime found"
        runtime_hint
        missing_deps=$((missing_deps + 1))
    fi
fi

# --- QEMU (host runtime dependency for make qemu) ---------------------------
vd_info "QEMU (host runtime for make qemu — headless boot of the ISO)"
if command -v qemu-system-x86_64 >/dev/null 2>&1; then
    printf '  %-14s \033[1;32mok\033[0m\n' "qemu[qemu-system-x86_64]"
else
    printf '  %-14s \033[1;33mnot installed\033[0m\n' "qemu[qemu-system-x86_64]"
    printf '%s\n' "  QEMU is only needed to RUN the built ISO; install it:"
    qemu_hint
fi

if ((missing_deps > 0)); then
    vd_error "${missing_deps} required dependency(ies) missing"
    exit 1
fi

vd_ok "all dependencies present"
exit 0
