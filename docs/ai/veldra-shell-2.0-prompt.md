# Veldra Shell 2.0 — implementation prompt

Veldra
Copyright (c) 2026 Adrian Sikora
All rights reserved.
Proprietary and confidential.

## Prompt

You are a senior Linux desktop engineer, Go engineer, TUI UX designer, systems programmer, and performance engineer.

Transform the existing Veldra TUI into **Veldra Shell 2.0**: a highly polished terminal-native desktop shell inspired by the ergonomic ideas of macOS and the keyboard-driven workflow of Omarchy, but with an original Veldra identity. Do not copy proprietary artwork, logos, assets, or source code from macOS or Omarchy.

### Non-negotiable goals

1. Make the shell genuinely useful, not a decorative mockup.
2. Keep the application terminal-first and fully usable over a normal PTY, QEMU curses, and serial console.
3. Preserve fast startup and low idle CPU usage.
4. Make keyboard navigation excellent while supporting mouse input where the terminal supports it.
5. Work well at 80x24 and scale cleanly to large terminals.
6. Keep all functionality offline-first. Do not require network access for the core desktop shell.
7. Use the existing Go module and existing packages first. The current project already uses Bubble Tea and Lip Gloss; expand the architecture around them instead of replacing them with an unrelated framework.
8. Use the Go standard library aggressively for OS integration, process execution, filesystem operations, terminal detection, signals, concurrency, and system information.

### Visual language

Create an original Veldra visual system combining:

- macOS-like spatial hierarchy and calm panel composition
- Omarchy-like keyboard-first launcher and workflow
- dense but readable terminal information
- crisp borders and layered cards
- subtle separators and restrained animation
- strong focus states
- consistent typography using terminal-safe glyphs
- adaptive layout with graceful degradation in small terminals

The shell should feel like a complete terminal desktop environment rather than a menu program.

### Main shell layout

Implement a persistent shell layout with:

- top status bar: Veldra logo/name, current workspace, active application, network state, CPU/RAM summary, current time
- central workspace area with application windows/panels
- bottom dock with the major Veldra applications
- contextual footer with shortcuts and current state
- optional notification/toast area

Use reusable components instead of one giant View function.

### Core applications

Turn the current applications into first-class usable tools:

- Terminal: real shell execution through PTY, scrollback, resize handling, ANSI colors, Ctrl-C/Ctrl-D, command history, working directory tracking
- Files: directory tree, preview panel, file metadata, navigation, create/delete/rename/copy/move, open-with routing to Editor/Terminal
- Editor: practical lightweight terminal editor with cursor navigation, insert mode, save, line numbers, status bar, unsaved-state indicator, search, file type indicator
- Settings: real settings backed by a small config file; theme, UI density, clock, mouse support, shell, startup behavior
- Task Manager: process list with CPU/RAM, sorting, filtering, refresh rate, process details, safe terminate action
- Launcher: fuzzy application/command launcher with keyboard-first navigation
- System Center: CPU, memory, storage, uptime, kernel, hostname, terminal size, load average

### Workflow

Design a coherent keyboard workflow:

- number keys or configurable shortcuts for primary apps
- arrows/tab for navigation
- Enter to activate
- Escape to back out of transient views
- a command/launcher overlay
- global help overlay
- quick terminal toggle
- workspace switching
- application focus switching
- safe confirmation overlays for destructive actions

Never trap the user in a modal without an obvious exit.

### Architecture

Refactor toward a component architecture such as:

- `tui/app/` shell state and orchestration
- `tui/components/` reusable panels, bars, lists, overlays, dialogs
- `tui/services/` OS/process/filesystem/system services
- `tui/apps/` application models and views
- `tui/theme/` theme tokens and style definitions
- `tui/input/` keymaps and mouse handling

Keep packages small, testable, and idiomatic.

### Charm / Go stack

Use the existing Charm stack deeply where useful:

- Bubble Tea for the event loop, models, commands, messages, resizing and input
- Lip Gloss for all styling, layout and borders
- terminal-aware packages already present in `go.mod` where appropriate for ANSI, terminal dimensions and color capability
- standard Go concurrency primitives for background refreshes

Do not add dependencies just to imitate a visual effect. Every dependency must have a clear functional reason.

### Performance

Treat performance as a feature:

- avoid doing filesystem scans in every `View()` call
- cache expensive system metrics
- update dynamic metrics on timers with sane intervals
- do not spawn a process for every redraw
- avoid full-screen recomputation when a component has not changed
- keep idle CPU usage extremely low
- avoid excessive terminal escape sequences
- gracefully reduce animation and refresh frequency on slow terminals

### QEMU / Replit compatibility

The shell must work with:

- normal Linux terminal
- `TERM=xterm-256color`
- `TERM=linux`
- QEMU `-display curses`
- QEMU `-nographic` with serial console

Do not assume a graphical framebuffer exists.

Detect terminal capabilities at runtime and degrade gracefully. The shell must remain usable when mouse support is unavailable.

### Error handling

Every OS-facing action must fail gracefully:

- show an in-shell error panel instead of crashing
- include the operation that failed
- include the underlying error
- keep the shell running when an individual application fails

### Deliverables

Produce production-quality code, not pseudocode.

1. Refactor the current Veldra TUI into the architecture above.
2. Implement the new shell and launcher.
3. Improve the Terminal, Files, Editor, Settings and Task Manager applications.
4. Add responsive layouts for small and large terminals.
5. Add unit tests for core state transitions and services.
6. Add integration-safe tests that do not require a real interactive terminal.
7. Keep the binary statically buildable for the Veldra live ISO.
8. Update documentation with keybindings and architecture.
9. Preserve existing CLI flags such as `--dir` and `--editor` unless there is a strong reason to extend them.
10. Verify `make tui` and the Replit build path still work.

### Quality bar

The finished result should feel like a real terminal-native operating environment:

- coherent
- fast
- discoverable
- keyboard-first
- visually sophisticated
- robust on weak terminals
- useful immediately after boot

Do not stop after making the interface look good. Every visible control must perform a real operation or clearly state that the feature is unavailable.
