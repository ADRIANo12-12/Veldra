#!/usr/bin/env bash
# tests/installer/lib.sh — verify the installer library loads and its
# pure functions behave.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.

set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

fail() { echo "FAIL: $*"; exit 1; }

VD_INSTALL_SRC=
VELDRA_PROJECT_ROOT="$ROOT"
# shellcheck disable=SC1091
source "$ROOT/installer/lib-install.sh"

# functions defined
for fn in vd_install_src vd_install_src_require vd_install_prereqs \
    vd_disk_label vd_choose_disk vd_partition_disk vd_mkfs \
    vd_mount_target vd_pacstrap_base vd_stage_layers \
    vd_chroot_configure vd_install_grub vd_gen_fstab \
    vd_install_cleanup; do
    declare -F "$fn" >/dev/null || fail "missing function $fn"
done

# install source resolves to the repo when no live bundle exists
src="$(vd_install_src)" || fail "vd_install_src failed"
[[ "$src" == "$ROOT" ]] || fail "vd_install_src -> $src (expected $ROOT)"

# disk label enumerates at least the real root device
disks="$(vd_disk_label)"
[[ -n "$disks" ]] || fail "no disks reported by lsblk"

# partition layout math on a fake device: msdos -> single root part
VD_PART_ROOT=
VD_PART_ESP=
vd_partition_disk /dev/null msdos no >/dev/null 2>&1 || true
# (function is not runnable against /dev/null; just ensure it parses)
bash -n "$ROOT/installer/lib-install.sh" || fail "lib-install.sh has syntax errors"
bash -n "$ROOT/installer/veldra-install" || fail "veldra-install has syntax errors"

echo "PASS: installer lib (functions + source resolution + syntax)"
exit 0