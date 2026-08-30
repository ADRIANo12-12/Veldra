#!/usr/bin/env bash
# run.sh — run a Veldra build command inside the isolated Arch Linux build
# container.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Veldra is Arch-based, so ISO/rootfs build steps need the Arch toolchain
# (pacman, arch-chroot, grub-mkrescue, xorriso). This wrapper makes that
# toolchain available on ANY host by running the commands inside a
# self-built Arch container. The developer never opens a shell or manually
# enters the container — `make iso` handles it.
#
#   - Podman is preferred, Docker is the fallback (VELDRA_CONTAINER_RUNTIME
#     overrides the choice; "none" forces the not-found error path).
#   - The repository is bind-mounted read/write so build/out lands directly
#     in the host tree.
#   - The container runs with exactly one extra capability, SYS_ADMIN, which
#     the Arch pacman post-transaction hooks need when they mount proc/sys
#     into the fresh rootfs inside a user namespace. No --privileged.
#   - After the job, build/work and build/out are chowned back to the
#     invoking user so artifacts are never stranded as root-owned.
#
# Usage:
#   run.sh <command...>      run the command inside the Arch build container
#   run.sh --init            build/refresh the builder image, then exit
#   run.sh --rebuild         force an image rebuild, then run <command...>
#   run.sh --no-build        never auto-build the image (fail if missing)
#   run.sh --check           verify runtime + image; exit 0 when usable
#   run.sh --dry-run         print the exact commands without running them
#   run.sh --help            this text

set -euo pipefail
# shellcheck source=scripts/common.sh
# shellcheck disable=SC1091 # resolved via SELF_DIR at runtime
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SELF_DIR}/../../scripts/common.sh"

# --- configuration ----------------------------------------------------------
RUNTIME_MODE="${VELDRA_CONTAINER_RUNTIME:-}"       # podman | docker | none | ""
IMAGE="${VELDRA_ARCH_BUILDER_IMAGE:-veldra-arch-builder:latest}"
BASE_IMAGE="${VELDRA_ARCH_BASE_IMAGE:-archlinux:latest}"
MOUNT_SRC="${VELDRA_PROJECT_ROOT}"
MOUNT_DST="/veldra"

# --- runtime detection ------------------------------------------------------
runtime_cmd() { # -> prints podman|docker and returns 0, 1 when unavailable
    if [[ -n "$RUNTIME_MODE" ]]; then
        case "$RUNTIME_MODE" in
            none) return 1 ;;
            podman|docker)
                command -v "$RUNTIME_MODE" >/dev/null 2>&1 \
                    && { printf '%s\n' "$RUNTIME_MODE"; return 0; }
                return 1 ;;
            *) vd_die 2 \
                "VELDRA_CONTAINER_RUNTIME='$RUNTIME_MODE' is invalid (podman|docker|none)" ;;
        esac
    fi
    if command -v podman >/dev/null 2>&1; then printf 'podman\n'; return 0; fi
    if command -v docker >/dev/null 2>&1; then printf 'docker\n'; return 0; fi
    return 1
}

runtime_rootless() { # runtime -> 0 when rootless, 1 when rootful/unknown
    local runtime="$1"
    if [[ "$runtime" == "podman" ]]; then
        podman info --format '{{.Host.Security.Rootless}}' 2>/dev/null | grep -q 'true' && return 0
        return 1
    fi
    docker info --format '{{.SecurityOptions}}' 2>/dev/null | grep -q 'rootless' && return 0
    return 1
}

runtime_hint() {
    vd_error "no usable container runtime found (podman/docker not installed)"
    printf '%s\n' \
        "  The Veldra build provides the Arch build environment (the OS base" \
        "  is Arch Linux) through an isolated container. Install Podman or Docker:" \
        "    Fedora:        sudo dnf install podman" \
        "    Debian/Ubuntu: sudo apt install podman" \
        "    openSUSE:      sudo zypper install podman" \
        "    Arch Linux:    sudo pacman -S podman" \
        "    Docker (any distro): install docker — the wrapper falls back to it automatically" \
        "  See docs/build.md for the full picture."
}

# --- container helpers ------------------------------------------------------
image_present() { # runtime -> 0 when the builder image already exists
    local runtime="$1"
    if [[ "$runtime" == "podman" ]]; then
        podman image exists "$IMAGE" >/dev/null 2>&1
    else
        docker image inspect "$IMAGE" >/dev/null 2>&1
    fi
}

build_image() { # runtime
    local runtime="$1"
    vd_info "building the Veldra Arch build container ($IMAGE)"
    if [[ "$runtime" == "podman" ]]; then
        podman build -t "$IMAGE" --build-arg VELDRA_ARCH_BASE="$BASE_IMAGE" \
            -f "${SELF_DIR}/Containerfile" "$SELF_DIR"
    else
        docker build -t "$IMAGE" --build-arg VELDRA_ARCH_BASE="$BASE_IMAGE" \
            -f "$SELF_DIR/Containerfile" "$SELF_DIR"
    fi
}

selinux_suffix() { # -> ":z" on SELinux-enforcing hosts (Fedora, etc.)
    if command -v getenforce >/dev/null 2>&1 \
        && [[ "$(getenforce 2>/dev/null)" == "Enforcing" ]]; then
        printf ':z'
    fi
    printf '\n'
}

workdir_for() { # -> container working directory mapped from $PWD
    if [[ "$PWD" == "$MOUNT_SRC" || "$PWD" == "$MOUNT_SRC/"* ]]; then
        printf '%s%s\n' "$MOUNT_DST" "${PWD#"$MOUNT_SRC"}"
        return 0
    fi
    printf '%s\n' "$MOUNT_DST"
}

rewrite_path() {
    local p="$1"
    case "$p" in
        "$MOUNT_SRC") printf '%s\n' "$MOUNT_DST" ;;
        "$MOUNT_SRC/"*) printf '%s\n' "$MOUNT_DST/${p#"$MOUNT_SRC/"}" ;;
        *) printf '%s\n' "$p" ;;
    esac
}

emit_run_tokens() { # runtime command...
    local runtime="$1"
    shift
    printf '%s\n' "$runtime" "run" "--rm"
    local suf
    suf="$(selinux_suffix)"
    printf '%s\n' "-v" "${MOUNT_SRC}:${MOUNT_DST}${suf}"
    printf '%s\n' "-w" "$(workdir_for)"
    printf '%s\n' "--cap-add" "SYS_ADMIN"
    printf '%s\n' "-e" "VELDRA_IN_ARCH_CONTAINER=1"
    local line
    while IFS= read -r line; do
        case "$line" in
            VELDRA*=*|PHANTOM*=*) printf '%s\n' "-e" "$line" ;;
        esac
    done < <(env)
    printf '%s\n' "$IMAGE"
    local arg
    for arg in "$@"; do
        printf '%s\n' "$(rewrite_path "$arg")"
    done
}

own_paths() {
    local -a paths=()
    [[ -d "$MOUNT_SRC/build/work" ]] && paths+=("$MOUNT_DST/build/work")
    [[ -d "$MOUNT_SRC/build/out" ]] && paths+=("$MOUNT_DST/build/out")
    printf '%s\n' "${paths[@]:-}"
}

usage() {
    sed 's/^ *//' <<'HELP'
usage: build/container/run.sh [--init|--rebuild|--no-build|--check|--dry-run|--help] [command...]
HELP
}

# --- main -------------------------------------------------------------------
[[ "${1:-}" == "--help" || "${1:-}" == "-h" ]] && { usage; exit 0; }

if [[ "${VELDRA_CONTAINER_DRYRUN:-0}" == "1" && "${1:-}" != -* ]]; then
    set -- --dry-run "$@"
fi

runtime=""
runtime="$(runtime_cmd)" || runtime=""
if [[ -z "$runtime" ]]; then
    runtime_hint
    exit 2
fi

case "${1:-}" in
    --check)
        vd_info "runtime: $runtime"
        if image_present "$runtime"; then
            vd_ok "builder image $IMAGE present"
        else
            vd_warn "builder image $IMAGE missing (built automatically on first make iso)"
        fi
        vd_ok "Arch build container ready"
        exit 0 ;;
    --init)
        build_image "$runtime"
        vd_ok "Arch build container ready: $IMAGE"
        exit 0 ;;
    --rebuild)
        build_image "$runtime"
        ;;
    --no-build)
        if ! image_present "$runtime"; then
            vd_die 2 "builder image '$IMAGE' is missing and --no-build was given (run: build/container/run.sh --init)"
        fi
        ;;
    --dry-run)
        vd_info "Veldra build container (host: $(vd_host_distro))"
        vd_info "runtime: $runtime"
        vd_info "image:   $IMAGE"
        vd_info "mount:   $MOUNT_SRC -> $MOUNT_DST"
        vd_info "workdir: $(workdir_for)"
        if ! image_present "$runtime"; then
            vd_info "image missing -> it will be built automatically, via:"
            if [[ "$runtime" == "podman" ]]; then
                printf '  podman build -t %s --build-arg VELDRA_ARCH_BASE=%s -f %s/Containerfile %s\n' \
                    "$IMAGE" "$BASE_IMAGE" "$SELF_DIR" "$SELF_DIR"
            else
                printf '  docker build -t %s --build-arg VELDRA_ARCH_BASE=%s -f %s/Containerfile %s\n' \
                    "$IMAGE" "$BASE_IMAGE" "$SELF_DIR" "$SELF_DIR"
            fi
        fi
        shift
        mapfile -t CMD < <(emit_run_tokens "$runtime" "$@")
        printf 'command: '
        printf '%q ' "${CMD[@]}"
        printf '\n'
        exit 0 ;;
esac

[[ $# -gt 0 ]] || { usage; exit 2; }

if ! image_present "$runtime"; then
    build_image "$runtime"
fi

mapfile -t CMD < <(emit_run_tokens "$runtime" "$@")
set +e
"${CMD[@]}"
rc=$?
set -e

if [[ "$rc" -eq 0 ]] && [[ "$(id -u)" != "0" ]]; then
    if runtime_rootless "$runtime"; then
        vd_info "rootless $runtime — build outputs already carry the invoking user's ownership"
    else
        own=()
        mapfile -t own < <(own_paths)
        if ((${#own[@]} > 0)); then
            vd_info "fixing ownership of build outputs (uid:gid $(id -u):$(id -g))"
            set +e
            "$runtime" run --rm \
                -v "${MOUNT_SRC}:${MOUNT_DST}$(selinux_suffix)" \
                -w "$MOUNT_DST" \
                --cap-add SYS_ADMIN \
                "$IMAGE" \
                chown -R "$(id -u):$(id -g)" "${own[@]}"
            own_rc=$?
            set -e
            if [[ "$own_rc" -ne 0 ]]; then
                vd_error "could not normalize artifact ownership; outputs under build/work, build/out may be root-owned."
                vd_error "fix manually with:  sudo chown -R $(id -u):$(id -g) build/work build/out"
                exit "$own_rc"
            fi
        fi
    fi
fi

exit "$rc"
