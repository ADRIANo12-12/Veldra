#!/usr/bin/env bash
# build/rootfs.sh — build the Veldra rootfs (Arch Linux base).
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Real build, assembled from real layers:
#   1. mount API filesystems into the fresh rootfs (proc/sys/dev/run/tmp)
#   2. pacman --root: install the Arch base (linux, systemd, bash, tools)
#   3. seed the target pacman keyring + mirrorlist
#   4. stage system/ and boot/ layers (identity + live initramfs config)
#   5. create the live user, configure networkd, enable TUI autologin
#   6. build the Veldra live initramfs inside the chroot (mkinitcpio)
#   7. install the built Veldra TUI binary
#   8. build manifest + tar (zstd) -> build/out/veldra-<version>-rootfs.tar.zst
#
# Runs where an Arch build environment exists: on a root Arch host directly,
# or inside the isolated Arch build container (build/container). On any other
# host the script re-executes itself through the container automatically.
#
# Gated: root + Arch. A plan mode (VELDRA_BUILD_DRYRUN=1) prints the steps
# and writes nothing.

set -euo pipefail
# shellcheck source=scripts/common.sh
# shellcheck disable=SC1091
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SELF_DIR}/../scripts/common.sh"

VERSION="$("${VELDRA_PROJECT_ROOT}/scripts/version.sh" show)"
WORK="${VELDRA_BUILD_WORK:-${VELDRA_PROJECT_ROOT}/build/work}"
OUT="${VELDRA_BUILD_OUT:-${VELDRA_PROJECT_ROOT}/build/out}"
RFS="${WORK}/rootfs"
TAR="${OUT}/veldra-${VERSION}-rootfs.tar.zst"
MANIFEST="${OUT}/veldra-${VERSION}-manifest.txt"

DRY=0
[[ "${VELDRA_BUILD_DRYRUN:-0}" == "1" ]] && DRY=1

plan() {
    printf 'Veldra rootfs build (version %s)\n' "$VERSION"
    printf '  rootfs  : %s\n' "$RFS"
    printf '  artifact: %s\n' "$TAR"
    printf '%s\n' \
        "  1. mount API filesystems into the fresh rootfs" \
        "  2. pacman --root: install Arch base (linux, systemd, bash, tools)" \
        "  3. seed pacman keyring + mirrorlist into the target" \
        "  4. stage system/ + boot/ layers (identity, live initramfs config)" \
        "  5. create live user + networkd + TUI autologin" \
        "  6. build Veldra live initramfs (mkinitcpio, chroot)" \
        "  7. install the Veldra TUI binary" \
        "  8. write manifest + tar (zstd) rootfs"
}

if (( DRY )); then
    plan
    vd_ok "plan only — nothing was built"
    exit 0
fi

# On a non-Arch host this re-executes inside the isolated Arch build container.
vd_ensure_arch_build_env "${BASH_SOURCE[0]}" "$@"

vd_require_arch_host
[[ "$(id -u)" == "0" ]] || vd_die 2 \
    "the rootfs build must run as root (pacman + mounts write into the chroot)."

vd_require pacman pacman-key tar zstd mount mountpoint install chroot

vd_info "Veldra rootfs build — $VERSION"
mkdir -p "$WORK" "$OUT"
rm -rf "$RFS" && mkdir -p "$RFS"

# --- API filesystems -------------------------------------------------------
API_MOUNTS=(tmp run dev/shm dev sys proc)

mount_api() {
    local m
    for m in proc sys; do
        mkdir -p "$RFS/$m"
    done
    mkdir -p "$RFS/dev/shm" "$RFS/run" "$RFS/tmp" "$RFS/var/lib/pacman"
    mount -t proc proc "$RFS/proc" -o nosuid,noexec,nodev
    mount -t sysfs sysfs "$RFS/sys" -o nosuid,noexec,nodev,ro
    mount --bind /dev "$RFS/dev"
    mount --bind /dev/shm "$RFS/dev/shm"
    mount -t tmpfs run "$RFS/run" -o nosuid,nodev,mode=0755
    mount -t tmpfs tmp "$RFS/tmp" -o mode=1777,strictatime,nodev,nosuid
}

unmount_api() {
    local m
    for m in "${API_MOUNTS[@]}"; do
        if mountpoint -q "$RFS/$m" 2>/dev/null; then
            umount "$RFS/$m" >/dev/null 2>&1 \
                || vd_warn "could not unmount $RFS/$m (container mount namespace is discarded on exit anyway)"
        fi
    done
}

require_unmounted() {
    local m
    for m in "${API_MOUNTS[@]}"; do
        if mountpoint -q "$RFS/$m" 2>/dev/null; then
            vd_die 2 "API filesystem still mounted at $RFS/$m — refusing to archive a polluted rootfs"
        fi
    done
}

trap 'unmount_api' EXIT

vd_info "mounting API filesystems into $RFS"
mount_api

# --- Arch base ---------------------------------------------------------------
vd_info "installing Arch base system (pacman --root)"
mkdir -p "$WORK/pacman-cache"
pacman -Sy --noconfirm --disable-sandbox \
    --root "$RFS" \
    --dbpath "$RFS/var/lib/pacman" \
    --cachedir "$WORK/pacman-cache" \
    --gpgdir /etc/pacman.d/gnupg \
    base \
    linux linux-firmware \
    systemd systemd-sysvcompat \
    bash less \
    sudo \
    vim \
    procps-ng util-linux \
    e2fsprogs dosfstools \
    grub \
    arch-install-scripts iproute2 \
    parted \
    || vd_die 1 "pacman base install failed"

vd_info "configuring target locale"
install -d -m 0755 "$RFS/etc"
cat >"$RFS/etc/locale.gen" <<'EOF'
en_US.UTF-8 UTF-8
EOF
chroot "$RFS" locale-gen >/dev/null 2>&1 || vd_warn "locale-gen failed"

vd_info "initializing target pacman keyring"
chroot "$RFS" /bin/bash -c '
    pacman-key --init 2>/dev/null || true
    pacman-key --populate archlinux 2>/dev/null || true
' || vd_warn "keyring init failed; the live system can re-initialize on first boot"

vd_info "seeding pacman.conf + mirrorlist into the target"
install -d -m 0755 "$RFS/etc/pacman.d"
[[ -f "$RFS/etc/pacman.d/mirrorlist" ]] || \
    install -m 0644 /etc/pacman.d/mirrorlist "$RFS/etc/pacman.d/mirrorlist"
[[ -f "$RFS/etc/pacman.conf" ]] || \
    install -m 0644 /etc/pacman.conf "$RFS/etc/pacman.conf"

gpgconf --homedir "$RFS/etc/pacman.d/gnupg" --kill all 2>/dev/null || true

vd_info "staging Veldra layers"
"${VELDRA_PROJECT_ROOT}/system/install.sh" "$RFS" --autologin veldra \
    || vd_die 1 "system layer failed"
"${VELDRA_PROJECT_ROOT}/boot/install.sh" "$RFS" \
    || vd_die 1 "boot layer failed"

# Serial consoles (headless QEMU, real serial servers) autologin the same
# live user. systemd-getty-generator spawns getty@ttyS0 for a
# `console=ttyS0` kernel argument; give it the same live autologin.
install -d -m 0755 "$RFS/etc/systemd/system/getty@ttyS0.service.d"
cat >"$RFS/etc/systemd/system/getty@ttyS0.service.d/veldra-autologin.conf" <<'EOF'
[Service]
ExecStart=
ExecStart=-/usr/bin/agetty --autologin veldra --noclear --keep-baud %I 115200 linux
EOF
chmod 0644 "$RFS/etc/systemd/system/getty@ttyS0.service.d/veldra-autologin.conf"
vd_ok "getty@ttyS0 autologin -> veldra"

vd_info "configuring target"
printf '%s\n' "${VELDRA_HOSTNAME:-veldra}" >"$RFS/etc/hostname"

# --- live user --------------------------------------------------------------
vd_info "creating live user 'veldra'"
chroot "$RFS" /bin/bash -c '
    id veldra >/dev/null 2>&1 || useradd -m -s /bin/bash -G wheel veldra
    printf "%s\n%s\n" veldra veldra | chpasswd veldra 2>/dev/null || true
    passwd -l root 2>/dev/null || true
'
install -d -m 0755 "$RFS/etc/sudoers.d"
cat >"$RFS/etc/sudoers.d/10-veldra" <<'EOF'
veldra ALL=(ALL) NOPASSWD: ALL
EOF
chmod 0440 "$RFS/etc/sudoers.d/10-veldra"

# --- networking --------------------------------------------------------------
vd_info "configuring systemd-networkd"
install -d -m 0755 "$RFS/etc/systemd/network"
cat >"$RFS/etc/systemd/network/20-wired.network" <<'EOF'
[Match]
Name=en* eth* wl*

[Network]
DHCP=yes
EOF
systemctl --root="$RFS" enable systemd-networkd 2>/dev/null || vd_warn "could not enable systemd-networkd"
systemctl --root="$RFS" enable systemd-resolved 2>/dev/null || vd_warn "could not enable systemd-resolved"
ln -sf /run/systemd/resolve/stub-resolv.conf "$RFS/etc/resolv.conf" 2>/dev/null || true

# --- live initramfs ----------------------------------------------------------
vd_info "building Veldra live initramfs (mkinitcpio, chroot)"
# mkinitcpio falls back to `uname -r` when no preset is given; inside the
# build container that reports the host kernel, not the packaged one. Pass
# the kernel version that was actually installed into the rootfs.
KVER=""
for d in "$RFS"/usr/lib/modules/*/; do
    if [[ -d "$d/kernel" ]]; then
        KVER="$(basename "$d")"
        break
    fi
done
[[ -n "$KVER" ]] || vd_die 1 "cannot find the installed kernel modules under $RFS/usr/lib/modules"
vd_info "kernel: $KVER"
if chroot "$RFS" mkinitcpio -k "$KVER" -c /etc/mkinitcpio-veldra.conf \
    -g /boot/initramfs-linux-veldra.img 2>&1; then
    vd_ok "live initramfs: /boot/initramfs-linux-veldra.img"
else
    vd_die 1 "mkinitcpio could not build the Veldra live initramfs"
fi

# --- Veldra TUI --------------------------------------------------------------
vd_info "installing the Veldra TUI binary"
TUI_BIN="${VELDRA_PROJECT_ROOT}/tui/bin/veldra-tui"
if [[ -x "$TUI_BIN" ]]; then
    install -d -m 0755 "$RFS/usr/local/bin"
    install -m 0755 "$TUI_BIN" "$RFS/usr/local/bin/veldra-tui"
    vd_ok "usr/local/bin/veldra-tui"
else
    vd_die 1 "TUI binary missing: $TUI_BIN (build it first:  make tui)"
fi

# --- Veldra installer bundle -------------------------------------------------
# The live system carries a self-contained copy of everything the installer
# needs (layers, scripts, config, TUI binary). The single source of truth
# survives inside the running live system, so `veldra-install` works there.
# The bundle mirrors the repo layout exactly, so every relative path inside
# the layer scripts (common.sh, version.sh, boot/initramfs/...) keeps working.
vd_info "staging the Veldra installer bundle"
BUNDLE="$RFS/usr/share/veldra/install-src"
rm -rf "$BUNDLE"
install -d -m 0755 \
    "$BUNDLE/config" \
    "$BUNDLE/scripts" \
    "$BUNDLE/system" \
    "$BUNDLE/boot" \
    "$BUNDLE/installer" \
    "$BUNDLE/tui/bin"
cp -a "${VELDRA_PROJECT_ROOT}/system/install.sh" \
      "${VELDRA_PROJECT_ROOT}/system/os-release.in" \
      "${VELDRA_PROJECT_ROOT}/system/profile.d" \
      "${VELDRA_PROJECT_ROOT}/system/skel" \
      "${VELDRA_PROJECT_ROOT}/system/hostname" \
      "${VELDRA_PROJECT_ROOT}/system/systemd" \
      "$BUNDLE/system/"
cp -a "${VELDRA_PROJECT_ROOT}/boot/install.sh" \
      "${VELDRA_PROJECT_ROOT}/boot/initramfs" \
      "$BUNDLE/boot/"
cp -a "${VELDRA_PROJECT_ROOT}/installer/veldra-install" \
      "${VELDRA_PROJECT_ROOT}/installer/lib-install.sh" \
      "$BUNDLE/installer/"
install -m 0644 "${VELDRA_PROJECT_ROOT}/config/veldra.conf" \
    "$BUNDLE/config/veldra.conf"
install -m 0755 "${VELDRA_PROJECT_ROOT}/scripts/common.sh" \
    "$BUNDLE/scripts/common.sh"
install -m 0755 "${VELDRA_PROJECT_ROOT}/scripts/version.sh" \
    "$BUNDLE/scripts/version.sh"
install -m 0755 "$TUI_BIN" "$BUNDLE/tui/bin/veldra-tui"
ln -sf /usr/share/veldra/install-src/installer/veldra-install \
    "$RFS/usr/local/bin/veldra-install"
vd_ok "installer bundle staged (veldra-install)"

# --- Veldra config copy ------------------------------------------------------
install -d -m 0755 "$RFS/etc/veldra" "$RFS/usr/share/veldra"
install -m 0644 "${VELDRA_PROJECT_ROOT}/config/veldra.conf" \
    "$RFS/etc/veldra/veldra.conf"

unmount_api
require_unmounted

vd_info "archiving rootfs"
if ! tar --zstd -cf "$TAR" -C "$RFS" .; then
    rm -f "$TAR"
    vd_die 1 "tar failed — no partial rootfs archive was left behind"
fi

vd_info "writing build manifest"
{
    printf '# Veldra rootfs manifest\n'
    printf 'version   : %s\n' "$VERSION"
    printf 'built     : %s\n' "$(date '+%Y-%m-%d %H:%M:%S %Z')"
    printf 'packages  :\n'
    pacman --root "$RFS" --dbpath "$RFS/var/lib/pacman" -Q 2>/dev/null || true
    printf 'staged    :\n'
    find "$RFS/etc" "$RFS/usr/share/veldra" -type f 2>/dev/null |
        sed "s#^$RFS##" | sort
} >"$MANIFEST"

chmod -R u+rwX "$RFS" 2>/dev/null || vd_warn "could not make $RFS owner-removable"

vd_ok "rootfs built: $TAR"
vd_ok "manifest:     $MANIFEST"
exit 0