# ttimelog

> **Warning**
> This project is a Work in Progress (WIP). Features and UX may change.

A terminal-based time tracking application written in Go.
Inspired [Collabora's gtimelog fork](https://gitlab.collabora.com/collabora/gtimelog), built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Motivation

[gtimelog](https://gtimelog.org/) is a GNOME-based time tracking app written in Python.
[Collabora](https://www.collabora.com/) maintains a fork with Chronophage integration.

ttimelog is a Go-based terminal rewrite aiming to provide the same functionality:

- Terminal-native UI
- Just for fun!

## Installation

```bash
go install github.com/Rash419/ttimelog/cmd/ttimelog@latest
```

## Usage

### Keybindings

#### General

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle focus between panes |
| `Ctrl+P` | Open project tree |
| `Alt+S` | Submit timesheet to Chronophage |
| `Ctrl+C` | Quit |

#### Input Pane

| Key | Action |
|-----|--------|
| `Enter` | Submit task |
| `Esc` | Cancel editing |

#### Table Pane

| Key | Action |
|-----|--------|
| `e` | Edit selected entry |
| `d` | Delete selected entry |
| `p` | Reassign project for selected entry |

#### Project Tree

| Key | Action |
|-----|--------|
| `j` / `k` or `↑` / `↓` | Navigate |
| `Space` | Expand/collapse node |
| `/` | Search projects |
| `Enter` | Select project |
| `Esc` | Close overlay |

### Task Markers

- `**arrived`: Mark work start time
- `**task description`: Mark as slack/break time

### Status Bar

A bottom status bar (lazygit-style) shows context-sensitive keyboard shortcuts on the left and transient status messages on the right, color-coded by type (yellow = in progress, green = success, red = error).

## Data Location

- `~/.ttimelog/ttimelog.txt`: Timelog entries
- `~/.ttimelog/ttimelogrc`: Configuration file (INI format)
- `~/.ttimelog/ttimelog.log`: Application logs

## Todo

### Core Features

- [x] Configurable target hours (daily/weekly)
- [x] Edit/delete existing entries
- [x] Keyboard navigation in table
- [x] Project tree with search and recent projects
- [x] Bottom status bar with shortcut hints
- [ ] Reports/export functionality
- [ ] Theme support

### Chronophage Integration

- [x] Submit timesheet via `Alt+S`
- [ ] Submit weekly status reports with email

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
