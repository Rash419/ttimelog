# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
go build ./cmd/ttimelog        # Build the binary
go test -v ./...               # Run all tests
go test ./internal/timelog     # Run timelog package tests
go test ./internal/config      # Run config package tests
```

## Architecture

ttimelog is a terminal-based time tracking app built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea) (TUI framework). It follows Bubble Tea's Model-Update-View pattern.

### Entry Point

`cmd/ttimelog/main.go` — Contains the main Bubble Tea model, all UI rendering, keyboard handling, and file watching logic (~775 lines). This is where most UI changes happen.

### Internal Packages

- **`internal/timelog/`** — Core data: `Entry` struct, file parsing (`LoadEntries`), saving/editing/deleting entries, stats aggregation (`StatsCollection`). File format: `YYYY-MM-DD HH:MM ±HHMM: Description`
- **`internal/config/`** — Reads INI config from `~/.ttimelog/ttimelogrc`. Key fields: `auth_header`, `task_list_url`, `hours` (daily target, default 8.0)
- **`internal/chrono/`** — Chronophage integration: fetches project list via HTTP, parses colon-separated project paths into tree hierarchy
- **`internal/layout/`** — `Pane` component with rounded borders (blue=focused, gray=blurred)
- **`internal/treeview/`** — Hierarchical tree navigation for project selection (expand/collapse nodes)

### UI Structure

4 panes: Header (date/week), Stats (progress bars), Table (today's entries), Footer (time input). Overlay modals for delete confirmation and project tree selection.

### Key Patterns

- **File-based storage**: All data in `~/.ttimelog/` as plain text files — no database
- **File watching**: fsnotify goroutine monitors `ttimelog.txt` for external edits
- **Task markers**: `**arrived` or `arrived**` = work start; `**` prefix = slack/break time
- **Focus system**: `Focus` enum controls which pane receives keyboard input; Tab/Shift+Tab cycles panes
- **Table keys**: `e` (edit), `d` (delete), `p` (reassign project) when table is focused
