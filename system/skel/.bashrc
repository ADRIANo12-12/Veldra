# /etc/skel/.bashrc — default shell for new Veldra users.
#
# Veldra
# Copyright (c) 2026 Adrian Sikora
# All rights reserved.
# Proprietary and confidential.

# If not running interactively, don't do anything
case $- in
    *i*) ;;
    *) return ;;
esac

# A minimal, calm prompt.
PS1='\[\e[1;36m\]\u\[\e[0m\]@\[\e[1;36m\]\h\[\e[0m\]:\[\e[1;34m\]\w\[\e[0m\]\$ '

# Basic shell conveniences.
alias ls='ls --color=auto'
alias ll='ls -l'
alias grep='grep --color=auto'

# Restart the Veldra TUI from a plain shell.
alias veldra='veldra-tui'

export EDITOR='veldra-tui --editor'
