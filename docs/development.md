# Veldra — Development

Veldra
Copyright (c) 2026 Adrian Sikora
All rights reserved. Proprietary and confidential.

## Rules

- **Version and identity come only from `config/veldra.conf`.** Nothing
  else hardcodes versions; templates use `@VERSION@` etc. and are
  resolved with `scripts/version.sh inject`.
- **Everything must be real.** No stubbed internals, no mock data in the
  TUI, no fake ISO. If a feature is not implemented yet, it must not
  pretend to work.
- **Shell scripts** are strict: `set -euo pipefail`, source
  `scripts/common.sh` for logging and helpers, pass `bash -n` and
  `shellcheck`.
- **Go code** is formatted with `gofmt`, vetted, and tested
  (`tui/` uses Go package tests, not asserts).
- **Licensing header** in every source file:
  project name, `Copyright (c) 2026 Adrian Sikora`,
  `All rights reserved.`, `Proprietary and confidential.`

## Layout

    config/        identity/version single source of truth
    system/        runtime identity layer (os-release, autologin, profile.d)
    boot/          live initramfs hook + config + grub.cfg.in
    tui/           Go TUI (Bubble Tea + Lip Gloss), package-tested
    installer/     veldra-install + lib-install.sh
    build/         container env + rootfs/iso scripts (outputs are git-ignored)
    scripts/       common.sh, version.sh, build-tui.sh, check-deps.sh
    tests/         shell layer tests; tests/run-all.sh runs them

## Common tasks

    make check          # compile + vet + go tests + shell syntax + templates
    make test           # Go unit tests only
    make lint           # shellcheck (requires shellcheck)
    make fmt            # gofmt the TUI source (also prints diffs)
    make build
    make iso
    make qemu           # headless boot of the last built ISO
    make plan           # show the end-to-end build plan

## TUI

- `tui/cmd/veldra/main.go` — entry, `--version` / `--help`
- `tui/apps/` — dock model: Terminal, Files, Editor, Settings,
  Task Manager
- `tui/system/info.go` — real system facts; version vars are set via
  `-ldflags` by `scripts/build-tui.sh`
- tests live next to the packages (`*_test.go`), headless-runnable

To smoke-test the TUI without a terminal, use the pty harnesses in
`/tmp/opencode/` (smartpty.py) to spawn it on a pseudo-terminal and read
the rendered frame.

## Versioning

    scripts/version.sh bump patch     # 0.0.1-pre-alpha -> 0.0.2-pre-alpha
    scripts/version.sh prerelease ""  # drop the prerelease for a release

The `Makefile` and all build scripts read the version from
`config/veldra.conf`; the ISO filename and every stamp follow it.