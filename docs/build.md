# Veldra — Building

Veldra
Copyright (c) 2026 Adrian Sikora
All rights reserved. Proprietary and confidential.

## Prerequisites

- Go 1.21+ for the TUI build
- Podman **or** Docker — provides the isolated Arch Linux build
  environment for the canonical ISO build
- `make`, `git`
- `qemu-system-x86_64` for booting the result (`make qemu`)

For the canonical build, check everything with:

    make check-deps

## Quick start

    make check      # compile + vet + tests + shell syntax + template sanity
    make test       # Go unit tests
    make iso        # canonical Veldra live ISO (Arch container required)
    make qemu       # boots the canonical ISO headless (QEMU -nographic)

Artifacts land in `build/out/`:

- `veldra-<version>-rootfs.tar.zst` — the canonical Veldra rootfs archive
- `veldra-<version>-x86_64.iso` — bootable live ISO (BIOS + UEFI)

## Replit / unprivileged hosted environments

Some hosted development environments provide a Docker client but no Docker
engine, and do not grant root privileges. In that case the canonical
container build cannot run. Veldra therefore includes a separate compatibility
backend:

    make replit-iso
    make replit-qemu

`build/replit-iso.sh` downloads the current official Arch Linux release ISO,
verifies its SHA256 checksum, extracts only the live `airootfs.sfs`, installs
the Veldra TUI and autologin configuration without `pacman`, `mount`, `chroot`,
or root, repacks the SquashFS, and replaces that file inside the original ISO
while preserving the official BIOS/UEFI boot metadata.

The Replit backend is intentionally separate from `make iso`. It is a
sandbox-compatibility path, not a replacement for the canonical Arch build.
The current pinned source release is Arch Linux 2026.08.01 (kernel 7.1.5). See
the official Arch release page for the release and checksum information.

Replit also needs these host tools available in PATH:

    curl
    xorriso
    unsquashfs
    mksquashfs
    sha256sum

The Replit QEMU target forces TCG and disables graphical display:

    qemu-system-x86_64 -machine accel=tcg -nographic -display none

## How the canonical container works

`build/container/run.sh` builds `veldra-arch-builder:latest` from
`archlinux:latest` (base-devel, arch-install-scripts, grub, libisoburn,
mtools, squashfs-tools, go, zstd). Build steps that need Arch tooling
re-execute themselves inside the container automatically:

- the repository is bind-mounted to `/veldra`
- the container adds exactly one capability: `SYS_ADMIN` (pacman
  transaction hooks mount proc/sys into the fresh rootfs)
- after the job, `build/work` and `build/out` are chowned back to the
  invoking user

You can also drive the stages directly:

    build/build.sh plan     # show the plan
    build/build.sh tui      # TUI binary only (host)
    build/build.sh rootfs   # rootfs + live initramfs (container)
    build/build.sh iso      # ISO (container)
    make container          # build/refresh the container image

## Plans and dry runs

Every build script honors `VELDRA_BUILD_DRYRUN=1` to print its plan and
write nothing:

    VELDRA_BUILD_DRYRUN=1 build/rootfs.sh
    VELDRA_CONTAINER_DRYRUN=1 build/container/run.sh --check

## Versioning

The version is managed from a single source of truth:

    scripts/version.sh show            # 0.0.1-pre-alpha
    scripts/version.sh identity
    scripts/version.sh bump patch
    scripts/version.sh prerelease stable

Releases should clear the prerelease (`stable`) so the live system and
the ISO name match the tagged version.
