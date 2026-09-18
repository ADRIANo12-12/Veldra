// /etc/profile.d/veldra.sh — Veldra session environment.
//
// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// The fullscreen shell is started by systemd on the primary console.
// Keeping profile.d passive avoids recursive TUI launches and keeps SSH,
// tty2+, scripts and nested terminals fast and predictable.

export VELDRA_SESSION=1
export EDITOR="${EDITOR:-veldra-tui --editor}"
export VISUAL="${VISUAL:-$EDITOR}"

alias ls='ls --color=auto'
alias ll='ls -l'
alias grep='grep --color=auto'
alias veldra='veldra-tui'
