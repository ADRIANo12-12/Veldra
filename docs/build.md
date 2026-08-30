# Veldra — Building

Veldra
Copyright (c) 2026 Adrian Sikora
All rights reserved. Proprietary and confidential.

## Prerequisites

- Go 1.21+ for the TUI build
- Podman **or** Docker — provides the isolated Arch Linux build
  environment (Veldra's base is Arch; pacman/grub-mkrescue/kernel
  tooling only exist for Arch)
- `make`, `git`
- `qemu-system-x86_64` for booting the result (`make qemu`)

Check everything with:

    make check-deps

## Quick start

    make check      # compile + vet + tests + shell syntax + template sanity
    make test       # Go unit tests
    make iso        # builds Veldra live ISO (builds the container on first run)
    make qemu       # boots the ISO headless (QEMU -nographic)

Artifacts land in `build/out/`:

- `veldra-<version>-rootfs.tar.zst` — the rootfs archive
- `veldra-<version>-x86_64.iso` — bootable live ISO (BIOS + UEFI)

## How the container works

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