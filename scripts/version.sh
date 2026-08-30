#!/usr/bin/env bash
# version.sh — Veldra version management.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# All version data lives in config/veldra.conf. This tool reads, validates,
# bumps, sets, and injects the version into templates — the only sanctioned
# way to change the version.
#
# Usage:
#   version.sh show                 print the full version string
#   version.sh identity             print: Veldra <ver> <channel> <status> <arch>
#   version.sh fields               print MAJOR MINOR PATCH PRERELEASE CODENAME
#   version.sh bump [major|minor|patch]   increment and drop prerelease
#   version.sh set 1.2.3            set and drop prerelease
#   version.sh prerelease rc1       set prerelease tag ("stable" clears it)
#   version.sh inject <file>        substitute @VERSION@ @NAME@ ... tokens
#
# Exit code 0 on success, non-zero on any failure (invalid version etc).

set -euo pipefail
# shellcheck source=scripts/common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

VER_FILE="${VELDRA_PROJECT_ROOT}/config/veldra.conf"

# --- semantic version validation -------------------------------------------
valid_version() {
    local v="$1"
    [[ "$v" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

# --- field helpers ---------------------------------------------------------
vd_major() { grep -E '^VELDRA_VERSION_MAJOR=' "$VER_FILE" | cut -d'"' -f2; }
vd_minor() { grep -E '^VELDRA_VERSION_MINOR=' "$VER_FILE" | cut -d'"' -f2; }
vd_patch() { grep -E '^VELDRA_VERSION_PATCH=' "$VER_FILE" | cut -d'"' -f2; }
vd_pre()   { grep -E '^VELDRA_PRERELEASE=' "$VER_FILE" | cut -d'"' -f2; }
vd_codename() { grep -E '^VELDRA_CODENAME=' "$VER_FILE" | cut -d'"' -f2; }
vd_channel()  { grep -E '^VELDRA_RELEASE_CHANNEL=' "$VER_FILE" | cut -d'"' -f2; }
vd_status()   { grep -E '^VELDRA_BUILD_STATUS=' "$VER_FILE" | cut -d'"' -f2; }
vd_arch()     { grep -E '^VELDRA_ARCH_DEFAULT=' "$VER_FILE" | cut -d'"' -f2; }

set_field() {
    local field="$1" value="$2"
    sed -i "s|^${field}=.*|${field}=\"${value}\"|" "$VER_FILE"
}

version_string() {
    local pre
    pre="$(vd_pre)"
    if [[ "$pre" == "stable" || -z "$pre" ]]; then
        printf '%s.%s.%s\n' "$(vd_major)" "$(vd_minor)" "$(vd_patch)"
    else
        printf '%s.%s.%s-%s\n' "$(vd_major)" "$(vd_minor)" "$(vd_patch)" "$pre"
    fi
}

# --- actions ----------------------------------------------------------------
cmd_show() { version_string; }

cmd_identity() {
    printf 'Veldra %s  %s • %s • %s\n' \
        "$(version_string)" "$(vd_channel)" "$(vd_status)" "$(vd_arch)"
}

cmd_fields() {
    printf '%s %s %s %s %s\n' \
        "$(vd_major)" "$(vd_minor)" "$(vd_patch)" "$(vd_pre)" "$(vd_codename)"
}

cmd_bump() {
    local part="${1:-patch}"
    local m n p
    m="$(vd_major)"; n="$(vd_minor)"; p="$(vd_patch)"
    case "$part" in
        major) m=$((m + 1)); n=0; p=0 ;;
        minor) n=$((n + 1)); p=0 ;;
        patch) p=$((p + 1)) ;;
        *) vd_die 1 "bump expects major|minor|patch (got '$part')" ;;
    esac
    set_field VELDRA_VERSION_MAJOR "$m"
    set_field VELDRA_VERSION_MINOR "$n"
    set_field VELDRA_VERSION_PATCH "$p"
    set_field VELDRA_PRERELEASE pre-alpha
    set_field VELDRA_VERSION "$(version_string)"
    vd_ok "version bumped: $(version_string)"
}

cmd_set() {
    local v="${1:-}"
    valid_version "$v" || vd_die 1 "invalid version '$v' (expected MAJOR.MINOR.PATCH)"
    local m n p
    m="${v%%.*}"; n="${v#*.}"; n="${n%%.*}"; p="${v##*.}"
    set_field VELDRA_VERSION_MAJOR "$m"
    set_field VELDRA_VERSION_MINOR "$n"
    set_field VELDRA_VERSION_PATCH "$p"
    set_field VELDRA_PRERELEASE pre-alpha
    set_field VELDRA_VERSION "$(version_string)"
    vd_ok "version set: $(version_string)"
}

cmd_prerelease() {
    local tag="${1:-}"
    set_field VELDRA_PRERELEASE "${tag:-stable}"
    set_field VELDRA_VERSION "$(version_string)"
    vd_ok "prerelease '$(vd_pre)': $(version_string)"
}

# Substitute @TOKEN@ placeholders from the central config into a template.
# Line-based; safe for text configs and boot/readme material.
cmd_inject() {
    local file="${1:--}"
    local tmp
    tmp="$(mktemp)"
    if [[ "$file" == "-" ]]; then
        cat - >"$tmp"
    else
        [[ -f "$file" ]] || vd_die 1 "inject: no such file '$file'"
        cp "$file" "$tmp"
    fi
    sed -e "s|@ID@|${VELDRA_ID}|g" \
        -e "s|@NAME@|${VELDRA_NAME}|g" \
        -e "s|@VERSION@|$(version_string)|g" \
        -e "s|@VERSION_FULL@|${VELDRA_VERSION}|g" \
        -e "s|@CODENAME@|$(vd_codename)|g" \
        -e "s|@CHANNEL@|$(vd_channel)|g" \
        -e "s|@STATUS@|$(vd_status)|g" \
        -e "s|@EDITION@|${VELDRA_EDITION}|g" \
        -e "s|@CREATOR@|${VELDRA_CREATOR}|g" \
        -e "s|@OWNER@|${VELDRA_OWNER}|g" \
        -e "s|@OWNER_CONTEXT@|${VELDRA_OWNER_CONTEXT}|g" \
        -e "s|@EMAIL@|${VELDRA_EMAIL}|g" \
        -e "s|@HOME@|${VELDRA_HOME}|g" \
        -e "s|@ISSUES@|${VELDRA_ISSUES}|g" \
        -e "s|@BASE@|${VELDRA_BASE}|g" \
        -e "s|@KERNEL@|${VELDRA_KERNEL_PACKAGE}|g" \
        -e "s|@ARCH@|$(vd_arch)|g" \
        -e "s|@THEME@|${VELDRA_THEME_DEFAULT}|g" \
        -e "s|@SHELL@|${VELDRA_SHELL}|g" \
        "$tmp" >"${tmp}.out" && cat "${tmp}.out"
    rm -f "$tmp" "${tmp}.out"
}

# --- arg handling ------------------------------------------------------------
cmd="${1:-show}"
case "$cmd" in
    show)       cmd_show ;;
    identity)   cmd_identity ;;
    fields)     cmd_fields ;;
    codename)   vd_codename ;;
    channel)    vd_channel ;;
    status)     vd_status ;;
    arch)       vd_arch ;;
    bump)       shift; cmd_bump "${1:-patch}" ;;
    set)        shift; cmd_set "${1:-}" ;;
    prerelease) shift; cmd_prerelease "${1:-}" ;;
    inject)     shift; cmd_inject "${1:-}" ;;
    *)          vd_die 1 "unknown command '$cmd' (show|identity|fields|codename|channel|status|arch|bump|set|prerelease|inject)" ;;
esac
