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

| Key | Action |
|-----|--------|
| `Enter` | Submit task |
| `Esc` | Toggle focus |
| `Ctrl+C` | Quit |

### Task Markers

- `**arrived`: Mark work start time
- `**task description`: Mark as slack/break time

## Data Location

- `~/.ttimelog/ttimelog.txt`: Timelog entries
- `~/.ttimelog/ttimelog.log`: Application logs

## Todo

### Core Features

- [ ] Configurable target hours (daily/weekly)
- [ ] Edit/delete existing entries
- [ ] Reports/export functionality
- [ ] Keyboard navigation in table
- [ ] Theme support

### Chronophage Integration

- [ ] Import/export timesheets in chronophage-compatible format
- [ ] Submit weekly status reports with email

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
