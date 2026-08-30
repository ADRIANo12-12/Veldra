# Veldra — Installer

Veldra
Copyright (c) 2026 Adrian Sikora
All rights reserved. Proprietary and confidential.

## What it is

`veldra-install` is Veldra's real installer. It follows the standard Arch
install sequence — no hand-waving, no simulated steps:

1. pick a target disk (interactive; the wipe requires an explicit
   confirmation)
2. partitioning: GPT or MBR, optional EFI System Partition
3. filesystems: `mkfs.ext4` root, `mkfs.fat -F32` ESP
4. mount, `pacstrap` a clean Arch base
5. `genfstab`
6. stage the Veldra system + boot layers from the single source of truth
7. `arch-chroot` configuration: locale, timezone, hostname, users,
   sudo, systemd-networkd, `mkinitcpio -P`
8. GRUB: BIOS to the target disk, UEFI removable for EFI systems
9. `grub-mkconfig`

It is available in the live session as `veldra-install`, and lives in
`installer/` (with `lib-install.sh`) in the source tree.

## Sources of truth

The installer runs the same layer scripts the build uses. On the live
ISO they are bundled self-contained at:

    /usr/share/veldra/install-src/
      config/veldra.conf
      system/install.sh, boot/install.sh, scripts/, boot/initramfs/
      installer/, tui/bin/veldra-tui

`veldra-install` resolves the install source as (in order):

    $VELDRA_INSTALL_SRC
    $VELDRA_PROJECT_ROOT          (repo checkout)
    /usr/share/veldra/install-src (live bundle)

## Usage

    veldra-install --help
    veldra-install --disk /dev/sda --user veldra

Interactive prompts ask along the way. For hands-free automation set the
`VD_*` variables:

    VD_DISK=/dev/vda VD_USER=veldra VD_USER_PASSWD=secret \
    VD_ROOT_PASSWD=secret VD_PART_MODE=gpt-efi \
    VELDRA_AUTOPROCEED=1 veldra-install

| Variable          | Meaning                                       |
|-------------------|-----------------------------------------------|
| `VD_DISK`         | target block device (skips the interactive list) |
| `VD_USER`         | primary user to create                        |
| `VD_USER_PASSWD`  | user password (`LOCKED` = no auth)            |
| `VD_ROOT_PASSWD`  | root password (`LOCKED` = locked)             |
| `VD_HOSTNAME`     | hostname (default `veldra`)                   |
| `VD_TIMEZONE`     | timezone (default `UTC`)                      |
| `VD_LOCALE`       | locale (default `en_US.UTF-8`)                |
| `VD_PART_MODE`    | `auto` `msdos` `gpt-efi` `gpt-only`           |
| `VD_SKIP_CHROOT`  | skip initramfs/bootloader/users (staging only) |
| `VELDRA_AUTOPROCEED` | auto-confirm every question (automation)  |

## Limitation

UEFI firmwares boot Veldra through the GRUB *removable* path
(`grub-install --removable --efi-directory=/boot`), which avoids
dependency on an NVRAM entry (`efibootmgr`). BIOS boot installs the boot
code directly into the disk's MBR. Secure Boot is not supported at this
version.