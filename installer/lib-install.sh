#!/usr/bin/env bash
# installer/lib-install.sh — shared functions for the Veldra installer.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Real installer machinery (mirrors the Arch install sequence):
#   - detect the Veldra install source (live bundle or a repo checkout)
#   - pick and partition a target disk (GPT BIOS+UEFI layout)
#   - format (ext4 root, FAT ESP) and mount the target
#   - pacstrap a clean Arch base
#   - stage the Veldra system/boot layers from the single source of truth
#   - arch-chroot configuration: locale, timezone, hostname, users,
#     keyring, networkd, initramfs, and GRUB bootloader
#
# Everything here is strictly question-and-confirm, with an explicit
# one-time "wipe this disk" confirmation. Loading this file only defines
# functions; nothing is executed until the caller invokes them.

# Source priority: repo checkout -> live bundle (/usr/share/veldra).
if ! command -v vd_die >/dev/null 2>&1; then
    # shellcheck source=scripts/common.sh
    source "${VELDRA_INSTALL_SRC:-}/scripts/common.sh" 2>/dev/null \
        || source "${VELDRA_PROJECT_ROOT:-}/scripts/common.sh" 2>/dev/null \
        || source "/usr/share/veldra/install-src/scripts/common.sh" 2>/dev/null \
        || { echo "[ERROR] cannot find scripts/common.sh" >&2; exit 2; }
fi

# --- install source resolution ---------------------------------------------

# vd_install_src -> prints the directory that looks like a Veldra project
# (has config/veldra.conf + scripts/). Preference: explicit env, live
# bundle, then repo.
vd_install_src() {
    local d
    for d in "${VELDRA_INSTALL_SRC:-}" \
        "${VELDRA_PROJECT_ROOT:-}" \
        "/usr/share/veldra/install-src"; do
        if [[ -n "$d" && -f "$d/config/veldra.conf" && -x "$d/scripts/version.sh" ]]; then
            printf '%s\n' "$d"
            return 0
        fi
    done
    return 1
}

# vd_install_src_require (var) -> resolve into a global var or die.
vd_install_src_require() {
    local src
    src="$(vd_install_src)" || vd_die 1 \
        "cannot find Veldra install source (config/veldra.conf missing)"
    printf -v "$1" '%s' "$src"
}

# --- prereqs ---------------------------------------------------------------
vd_install_prereqs() {
    vd_require lsblk parted mkfs.ext4 mount umount arch-chroot pacstrap \
        genfstab grub-install grub-mkconfig mkinitcpio systemctl tar zstd
    [[ "$(id -u)" == "0" ]] || vd_die 2 "the Veldra installer must run as root"
}

# --- disk selection --------------------------------------------------------
# vd_disk_label -> prints candidate disks (whole devices, not partitions).
vd_disk_label() {
    lsblk -dnbo NAME,TYPE,SIZE 2>/dev/null \
        | awk '$2=="disk" {printf "%s %s\n", $1, $3}' \
        | sort -k2 -n
}

# vd_choose_disk -> interactive/argument based disk selection.
# Sets $VD_TARGET_DISK + related partition variable names via nameref-free
# globals to keep bash 3/4 compatible.
vd_choose_disk() {
    local arg="${1:-}"
    if [[ -n "$arg" ]]; then
        if [[ -b "$arg" ]]; then
            VD_TARGET_DISK="$arg"
        else
            vd_die 1 "not a block device: $arg"
        fi
    else
        local disks line
        echo "Available disks:"
        disks="$(vd_disk_label)"
        [[ -n "$disks" ]] || vd_die 1 "no writable block devices found"
        local i=0
        while IFS= read -r line; do
            i=$((i + 1))
            set -- $line
            printf '  %d) %s  %s\n' "$i" "$1" "$2"
            VD_DISK_LIST+=("$1")
        done <<<"$disks"
        printf 'Select the disk to install onto [1]: '
        read -r sel
        sel="${sel:-1}"
        [[ "$sel" =~ ^[0-9]+$ && "$sel" -ge 1 && "$sel" -le "$i" ]] \
            || vd_die 1 "invalid selection: $sel"
        VD_TARGET_DISK="${VD_DISK_LIST[$((sel - 1))]}"
    fi

    echo "Target disk: $VD_TARGET_DISK"
    lsblk "$VD_TARGET_DISK" 2>/dev/null | sed 's/^/  /'
    if ! vd_confirm "THIS WILL ERASE EVERYTHING ON DISK $VD_TARGET_DISK. Continue?"; then
        vd_die 0 "aborted."
    fi
}

# --- partitioning ----------------------------------------------------------
# vd_partition_disk: interactive/env driven layout.
#   GPT, single / (plus optional ESP when EFI requested).
vd_partition_disk() {
    local disk="$1" mode="${2:-auto}" efi_req="${3:-auto}"

    local type="gpt"
    if ! vd_confirm "Use GPT partitioning? [Y/n]"; then
        type="msdos"
    fi

    vd_info "partitioning $disk ($type)"
    parted -s "$disk" mklabel "$type"
    if [[ "$type" == "msdos" ]]; then
        parted -s "$disk" mkpart primary ext4 1MiB 100%
        parted -s "$disk" set 1 boot on
        VD_PART_ROOT="$disk"1
        VD_PART_ESP=""
    else
        local efi="no"
        if [[ "$efi_req" == "yes" ]]; then efi="yes"; fi
        if [[ "$efi_req" == "auto" ]] && vd_confirm "Create an EFI System Partition? (recommended) [Y/n]"; then
            efi="yes"
        fi
        if [[ "$efi" == "yes" ]]; then
            parted -s "$disk" mkpart ESP fat32 1MiB 513MiB
            parted -s "$disk" set 1 esp on
            parted -s "$disk" mkpart velex ext4 513MiB 100%
            VD_PART_ESP="$disk"1
            VD_PART_ROOT="$disk"2
        else
            parted -s "$disk" mkpart velex ext4 1MiB 100%
            VD_PART_ROOT="$disk"1
            VD_PART_ESP=""
        fi
    fi
}

# --- filesystems -----------------------------------------------------------
vd_mkfs() {
    vd_info "formatting filesystems"
    if [[ -n "${VD_PART_ESP:-}" ]]; then
        mkfs.fat -F32 "${VD_PART_ESP}" >/dev/null
        vd_ok "ESP: $VD_PART_ESP (FAT32)"
    fi
    mkfs.ext4 -F -L VELDRA "${VD_PART_ROOT}" >/dev/null
    vd_ok "root: $VD_PART_ROOT (ext4)"
}

vd_mount_target() {
    local mnt="$1"
    mkdir -p "$mnt"
    mount "${VD_PART_ROOT}" "$mnt"
    if [[ -n "${VD_PART_ESP:-}" ]]; then
        mkdir -p "$mnt/boot"
        mount "${VD_PART_ESP}" "$mnt/boot"
    fi
    vd_ok "mounted target at $mnt"
}

# --- base install ----------------------------------------------------------
vd_pacstrap_base() {
    local mnt="$1"
    local pkgs=(
        base linux linux-firmware
        systemd systemd-sysvcompat
        bash less sudo vim
        procps-ng util-linux
        e2fsprogs dosfstools parted
        grub arch-install-scripts iproute2
    )
    echo "Packages to install:"
    printf '  %s\n' "${pkgs[@]}"
    if ! vd_confirm "Proceed with pacstrap? [Y/n]"; then
        vd_die 0 "aborted."
    fi
    pacstrap -K "$mnt" "${pkgs[@]}"
    vd_ok "Arch base installed into $mnt"
}

# --- layer staging (single source of truth) --------------------------------
vd_stage_layers() {
    local mnt="$1" src="$2" user="${3:-}"
    local args=()
    [[ -n "$user" ]] && args+=(--autologin "$user")
    "${src}/system/install.sh" "$mnt" "${args[@]}"
    "${src}/boot/install.sh" "$mnt"
    install -d -m 0755 "$mnt/etc/veldra"
    install -m 0644 "$src/config/veldra.conf" "$mnt/etc/veldra/veldra.conf"
    # the TUI binary from the live bundle (or a repo-local build)
    local tui=""
    for t in "${src}/tui/bin/veldra-tui" "$src/veldra-tui"; do
        if [[ -x "$t" ]]; then tui="$t"; break; fi
    done
    if [[ -n "$tui" ]]; then
        install -d -m 0755 "$mnt/usr/local/bin"
        install -m 0755 "$tui" "$mnt/usr/local/bin/veldra-tui"
    fi
}

# --- chroot configuration --------------------------------------------------
vd_chroot_configure() {
    local mnt="$1"
    local host="${VD_HOSTNAME:-veldra}"
    local zone="${VD_TIMEZONE:-UTC}"
    local locale="${VD_LOCALE:-en_US.UTF-8}"

    vd_info "configuring the installed system (arch-chroot)"
    printf '%s\n' "$host" >"$mnt/etc/hostname"

    printf '%s UTF-8\n' "$locale" >"$mnt/etc/locale.gen"
    arch-chroot "$mnt" locale-gen >/dev/null 2>&1 || vd_warn "locale-gen failed"

    printf 'LANG=%s\n' "$locale" >"$mnt/etc/locale.conf"

    rm -f "$mnt/etc/localtime"
    ln -sf "/usr/share/zoneinfo/$zone" "$mnt/etc/localtime"

    local user="${VD_USER:-}"
    if [[ -n "$user" ]]; then
        arch-chroot "$mnt" /bin/bash -c \
            "useradd -m -s /bin/bash -G wheel '$user'"
        if [[ "${VD_USER_PASSWD:-}" != "LOCKED" ]]; then
            printf '%s:%s\n' "$user" "${VD_USER_PASSWD}" \
                | arch-chroot "$mnt" chpasswd
        else
            arch-chroot "$mnt" usermod -p '!' "$user" || true
        fi
        install -d -m 0755 "$mnt/etc/sudoers.d"
        printf '%s ALL=(ALL) ALL\n' "$user" \
            >"$mnt/etc/sudoers.d/10-$user"
        chmod 0440 "$mnt/etc/sudoers.d/10-$user"
    fi

    if [[ "${VD_ROOT_PASSWD:-}" != "LOCKED" ]]; then
        printf 'root:%s\n' "${VD_ROOT_PASSWD}" | arch-chroot "$mnt" chpasswd
    fi

    vd_info "building initramfs (mkinitcpio -P)"
    arch-chroot "$mnt" mkinitcpio -P >/dev/null 2>&1 \
        || vd_warn "mkinitcpio -P reported issues"

    vd_info "installing the Veldra TUI autostart"
    # profile.d already staged by system/install.sh --autologin

    vd_info "enabling systemd-networkd"
    arch-chroot "$mnt" systemctl enable systemd-networkd systemd-resolved \
        >/dev/null 2>&1 || vd_warn "could not enable systemd-networkd"

    cat >"$mnt/etc/systemd/network/20-wired.network" <<'EOF'
[Match]
Name=en* eth* wl*

[Network]
DHCP=yes
EOF
}

# --- bootloader ------------------------------------------------------------
vd_install_grub() {
    local mnt="$1"
    vd_info "installing GRUB"
    if [[ -n "${VD_PART_ESP:-}" ]]; then
        arch-chroot "$mnt" grub-install \
            --target=x86_64-efi --efi-directory=/boot \
            --removable --recheck >/dev/null 2>&1 \
            && vd_ok "GRUB (UEFI removable)" \
            || vd_warn "UEFI GRUB install failed; trying BIOS"
    fi
    arch-chroot "$mnt" grub-install "${VD_TARGET_DISK}" >/dev/null 2>&1 \
        && vd_ok "GRUB (BIOS, $VD_TARGET_DISK)" \
        || vd_warn "BIOS GRUB install failed"
    arch-chroot "$mnt" grub-mkconfig -o /boot/grub/grub.cfg >/dev/null 2>&1 \
        || vd_warn "grub-mkconfig failed"
    vd_ok "bootloader configured"
}

# --- fstab -----------------------------------------------------------------
vd_gen_fstab() {
    local mnt="$1"
    genfstab -U "$mnt" >>"$mnt/etc/fstab"
    vd_ok "fstab written"
}

# --- cleanup ---------------------------------------------------------------
vd_install_cleanup() {
    local mnt="$1"
    umount -R "$mnt" 2>/dev/null || true
}