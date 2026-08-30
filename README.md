# Veldra

**Terminal-native Linux.**

Veldra is an Arch Linux–based, terminal-only distribution whose primary
interface is a full-screen TUI written in Go (Bubble Tea + Lip Gloss).
This repository contains the complete, real system: the bootable live
ISO, the rootfs, the installer, and the TUI itself — all built from one
source of truth (`config/veldra.conf`).

> **Status: 0.0.1-pre-alpha — UNSTABLE • BOOTABLE • x86_64**

## What ships

| Artifact | Where |
|----------|-------|
| Veldra TUI (static binary) | `tui/bin/veldra-tui` (built via `make tui`) |
| Live rootfs archive        | `build/out/veldra-<version>-rootfs.tar.zst` |
| Bootable live ISO (BIOS+UEFI) | `build/out/veldra-<version>-x86_64.iso` |
| Installer                  | `veldra-install` (live session) / `installer/` |
| Veldra system layer        | `system/` (os-release, autologin, profile.d) |
| Live boot hook             | `boot/initramfs/` (squashfs + overlay live root) |

## Quick start (Fedora/Ubuntu/any x86_64 Linux)

    sudo dnf install podman qemu-system-x86             # or the equivalent
    make check-deps
    make iso          # builds the Arch build container once, then the ISO
    make qemu         # boots Veldra headless

You do **not** need Arch Linux to build it — the ISO build runs inside an
isolated Arch container (Podman, Docker fallback). See `docs/build.md`.

Inside the live system, `veldra-install` installs Veldra to a real disk
(partition, pacstrap, GRUB).

## The TUI apps

45-second tour: `veldra-tui` (or just boot the ISO — it auto-starts).

- **1 Terminal** — system facts, read live from `/proc`, `/sys`,
  `/etc` (kernel, hostname, uptime, CPU, RAM, network)
- **2 Files** — real directory browser
- **3 Editor** — real file editor
- **4 Settings** — network interfaces
- **5 Task Manager** — real `/proc` processes

## Docs

- `docs/architecture.md` — how the pieces fit
- `docs/build.md` — build workflow
- `docs/boot.md` — live boot chain
- `docs/installer.md` — installing to disk
- `docs/development.md` — rules for contributing

## Identity

Veldra is a proprietary project by Adrian Sikora (Phantom Systems (TM)).
See `LICENSE`. All source files carry the license header.

- email: adriangithub@proton.me
- issues: https://github.com/ADRIANo12-12/Veldra/issues