# /etc/profile.d/veldra.sh — Veldra session environment.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.
#
# Sourced for login shells on a real Veldra system. Configures the shell
# so that the Veldra TUI — the primary interface — starts automatically
# when a user logs in on a text console, while leaving a normal Linux
# shell fully accessible beneath it.

# Only auto-launch the Veldra TUI on an interactive text login on a console
# (tty), never when we are already inside a nested terminal or a real shell
# that already spawned the TUI.
veldra_maybe_start_tui() {

    # 1. Only auto-start for an interactive login shell.
    case $- in
        *i*) ;;
        *) return 0 ;;
    esac

    # 2. Only on a real TTY (text console / serial), not a pty on the host.
    if [[ ! -t 0 ]]; then
        return 0
    fi

    # 3. Only auto-start once per session.
    if [[ -n "${VELDRA_TUI_STARTED:-}" ]]; then
        return 0
    fi

    # 4. Only if the TUI binary actually exists on the system.
    local tui="${VELDRA_TUI_BIN:-/usr/local/bin/veldra-tui}"
    if [[ ! -x "$tui" ]]; then
        return 0
    fi

    # 5. Do not auto-start over an explicit shell already running the TUI.
    if pgrep -f 'veldra-tui' >/dev/null 2>&1; then
        return 0
    fi

    VELDRA_TUI_STARTED=1
    export VELDRA_TUI_STARTED

    # Welcome line, then hand the terminal to the Veldra TUI. When the user
    # quits the TUI (Q / Ctrl+C) they are returned to this shell.
    printf '\n\033[1;36mVeldra\033[0m — terminal session. The Veldra TUI is starting...\n'
    "$tui"
    printf '\n\033[1;32mReturned to shell.\033[0m Type \033[1;36mveldra\033[0m to restart the TUI.\n'
}

veldra_maybe_start_tui
unset -f veldra_maybe_start_tui
