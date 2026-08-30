# Veldra — Boot

Veldra
Copyright (c) 2026 Adrian Sikora
All rights reserved. Proprietary and confidential.

## Live media chain

```
GRUB (grub-mkrescue, BIOS + UEFI)
  -> /boot/vmlinuz-linux
  -> /boot/initramfs-linux-veldra.img          (mkinitcpio-veldra.conf)
       base udev modconf block filesystems veldra-live
  -> kernel mounts the initramfs
  -> veldra-live hook:
       1. find the ISO:  filesystem label VELDRA, else scan
          /dev/sr* /loop* /vd* /sd* for a volume containing live.sfs
       2. mount iso9660  (media==root device only; tmpfs overlay otherwise)
       3. loop-mount live.sfs
       4. tmpfs upper + work
       5. mount -t overlay overlay -o
          lowerdir=/run/veldra/lower,upperdir=...,workdir=...
          $1
  -> systemd PID 1 boots the overlay root
  -> getty@tty1 autologin 'veldra'  ->  /etc/profile.d/veldra.sh
  -> Veldra TUI
```

On any failure the hook drops to a recovery shell
(`launch_interactive_shell --exec`) with the Veldra banner, and gives you
shell access to debug.

## Live filesystem

- `live.sfs` is the full rootfs compressed by `mksquashfs` (excluding
  `/boot`, which lives on the ISO directly)
- `CONFIG_OVERLAY_FS` + `CONFIG_SQUASHFS` are in `linux` (stock Arch)
- the squashfs root is `nomosquash`-free, plain zstd

## Installed systems

An installed Veldra boots exactly like stock Arch:

- system-wide `mkinitcpio` preset builds `/boot/initramfs-linux.img`
  (with `autodetect`) at install time
- GRUB is installed by `veldra-install` (BIOS to the disk; UEFI via
  `--removable`), and `grub-mkconfig` generates the boot entries
- the live-only `initramfs-linux-veldra.img` is used only for the live
  ISO; it is never booted from disks

## Rescue

Pick the "Veldra rescue shell" entry (`single`) from the GRUB menu, or,
inside the live system, hold the recovery path: the hook's interactive
shell has the full live rootfs mounted at `/run/veldra/lower`.