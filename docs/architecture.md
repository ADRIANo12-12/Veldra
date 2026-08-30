# Veldra — Architecture

Veldra
Copyright (c) 2026 Adrian Sikora
All rights reserved. Proprietary and confidential.

## Overview

Veldra is a terminal-native Linux distribution. It is Arch Linux under a
single identity: one bootable live system, one installer, one TUI that is
the primary user interface. Everything is generated from one source of
truth (`config/veldra.conf`) and every subsystem is real and bootable.

## Layers

| Layer      | Path       | What it is                                             |
|------------|------------|--------------------------------------------------------|
| identity   | config/    | single source of truth: name, version, channel, arch   |
| system     | system/    | os-release, profile.d, skel, hostname, autologin getty |
| boot       | boot/      | live initramfs hook + mkinitcpio config + grub.cfg     |
| tui        | tui/       | Bubble Tea + Lip Gloss terminal UI (Go)                |
| installer  | installer/ | real Arch installer (partition/pacstrap/grub)          |
| build      | build/     | container env + rootfs + ISO pipeline                  |
| tests      | tests/     | layer verification scripts                             |

## Boot model

The live ISO carries:

- `boot/vmlinuz-linux` and `boot/initramfs-linux-veldra.img`
- `live.sfs` — the whole rootfs squashed (excluding `/boot`)
- `boot/grub/grub.cfg` — auto-boots Veldra after a 2s timeout

The initramfs is built from `mkinitcpio-veldra.conf` with
`HOOKS=(base udev modconf block filesystems veldra-live)` and **no
autodetect**, so it carries the full fs/block driver set and boots on
arbitrary hardware. The `veldra-live` hook locates the media (label
`VELDRA`, falling back to scanning `/dev/sr*`, `/loop*`, `/vd*`, `/sd*`),
mounts the ISO, loop-mounts `live.sfs`, then forms an OverlayFS
(`live.sfs` lower, tmpfs upper/work) mounted as `/`. Veldra boots into a
writable session.

Installed systems ignore the live initramfs entirely and use the stock
Arch initramfs built by `mkinitcpio -P` from the system-wide preset.

## The TUI

`tui/cmd/veldra` renders a full-screen docked UI (apps: 1 Terminal,
2 Files, 3 Editor, 4 Settings, 5 Task Manager). All information shown
is read at runtime from `/proc`, `/sys`, `/etc` and system utilities.
The TUI is auto-started on tty1 through a `getty@tty1` autologin drop-in
plus `/etc/profile.d/veldra.sh`. Version/channel/status/arch are injected
at build time from `config/veldra.conf` via `-ldflags`.

## Installer

`veldra-install` follows the Arch install sequence: disk selection,
partitioning (GPT/MBR, optional ESP), `mkfs.ext4`/`mkfs.fat`,
mounting, `pacstrap` of the Arch base, `genfstab`, staging the Veldra
layers, then `arch-chroot` configuration (locale, timezone, users,
networkd, `mkinitcpio -P`, GRUB). The live system carries a
self-contained copy of the install sources under
`/usr/share/veldra/install-src`, so the installer works without network
beyond the Arch mirrors.

## Build pipeline

`make iso` runs, on any host:

1. `scripts/build-tui.sh` — static Go build with version injection
2. `build/rootfs.sh` — Arch base rootfs + live initramfs, inside the
   isolated Arch build container (Podman; Docker fallback)
3. `build/iso.sh` — squashfs + grub-mkrescue ISO (BIOS + UEFI)