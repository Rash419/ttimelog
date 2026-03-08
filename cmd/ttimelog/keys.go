package main

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Rash419/ttimelog/internal/chrono"
	"github.com/Rash419/ttimelog/internal/timelog"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) handleInput() {
	val := m.textInput.Value()
	if val == "" {
		return
	}

	if m.editingEntry >= 0 {
		m.handleEditInput(val)
		return
	}

	var lastTaskTime time.Time
	handleArrivedMessage := timelog.IsArrivedMessage(val) && !m.handledArrivedMessage
	if len(m.entries) == 0 || handleArrivedMessage {
		lastTaskTime = time.Now()
		m.handledArrivedMessage = true
	} else {
		lastTaskTime = m.entries[len(m.entries)-1].EndTime
	}

	// update table
	newEntry := timelog.NewEntry(time.Now(), val, time.Since(lastTaskTime))

	m.entries = append(m.entries, newEntry)
	if err := timelog.SaveEntry(newEntry, handleArrivedMessage, m.timeLogFilePath); err != nil {
		slog.Error("Failed to add entry with description", "error", newEntry.Description)
	}

	rows, indices := getTableRows(m.entries, m.viewDate, m.virtualMidnight)
	m.taskTable.SetRows(rows)
	m.entryIndices = indices
	m.scrollToBottom = true

	timelog.UpdateStatsCollection(newEntry, &m.statsCollection)

	m.textInput.Reset()
}

func (m *model) handleEditInput(val string) {
	entry := m.entries[m.editingEntry]

	// Parse "YYYY-MM-DD HH:MM +ZZZZ: description"
	parts := strings.SplitN(val, ": ", 2)
	if len(parts) < 2 {
		m.statusMessage = "Invalid format, expected 'YYYY-MM-DD HH:MM +ZZZZ: description'"
		m.statusKind = statusError
		return
	}

	newTimestamp := parts[0]
	newDescription := parts[1]

	if err := timelog.EditEntry(m.timeLogFilePath, entry.LineNumber, newTimestamp, newDescription); err != nil {
		slog.Error("Failed to edit entry", "error", err)
	}

	m.reloadEntries()
	m.editingEntry = -1
	m.statusMessage = ""
	m.statusKind = statusNone
	m.textInput.Reset()
}

func (m *model) handleProjectTreeKeyMsg(msg tea.KeyMsg) keyResult {
	// When searching, handle search-specific keys first
	if m.projectTree.Searching {
		switch msg.String() {
		case "ctrl+c":
			return keyExit
		case "esc":
			m.projectTree.StopSearch()
			m.searchInput.Reset()
			m.searchInput.Blur()
			return keyHandled
		case "enter":
			m.projectTree.Searching = false
			m.searchInput.Blur()
			projectPath := m.projectTree.GetProjectPath()
			if projectPath != "" {
				m.selectProject(projectPath)
			}
			return keyHandled
		case "up", "down":
			if msg.String() == "up" {
				m.projectTree.MoveUp()
			} else {
				m.projectTree.MoveDown()
			}
			return keyHandled
		default:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			_ = cmd
			m.projectTree.UpdateSearch(m.searchInput.Value())
			return keyHandled
		}
	}

	switch msg.String() {
	case "ctrl+c":
		return keyExit
	case "j", "down":
		m.moveProjectCursorDown()
		return keyHandled
	case "k", "up":
		m.moveProjectCursorUp()
		return keyHandled
	case " ": // space
		if !m.inRecents {
			m.projectTree.Toggle()
		}
		return keyHandled
	case "/":
		m.inRecents = false
		m.projectTree.StartSearch()
		m.searchInput.Reset()
		m.searchInput.Focus()
		return keyHandled
	case "enter":
		if m.inRecents && m.recentCursor < len(m.recentProjects) {
			m.selectProject(m.recentProjects[m.recentCursor])
		} else {
			projectPath := m.projectTree.GetProjectPath()
			if projectPath != "" {
				m.selectProject(projectPath)
			}
		}
		return keyHandled
	case "esc":
		m.closeProjectOverlay()
		return keyHandled
	}
	return keyHandled
}

func (m *model) handleDeleteConfirmKeyMsg(msg tea.KeyMsg) keyResult {
	switch msg.String() {
	case "y":
		if m.deleteTargetEntry >= 0 && m.deleteTargetEntry < len(m.entries) {
			entry := m.entries[m.deleteTargetEntry]
			if err := timelog.DeleteEntry(m.timeLogFilePath, entry.LineNumber); err != nil {
				slog.Error("Failed to delete entry", "error", err)
			}
			m.reloadEntries()
		}
		m.showDeleteConfirm = false
		m.deleteTargetEntry = -1
		m.focus = focusTable
		return keyHandled
	case "n", "esc":
		m.showDeleteConfirm = false
		m.deleteTargetEntry = -1
		m.focus = focusTable
		return keyHandled
	case "ctrl+c":
		return keyExit
	}
	return keyHandled
}

func (m *model) handleTableKeyMsg(msg tea.KeyMsg) keyResult {
	switch msg.String() {
	case "d":
		cursor := m.taskTable.Cursor()
		if cursor < 0 || cursor >= len(m.entryIndices) {
			return keyHandled
		}
		m.deleteTargetEntry = m.entryIndices[cursor]
		m.showDeleteConfirm = true
		m.focus = focusDeleteConfirm
		return keyHandled
	case "e":
		cursor := m.taskTable.Cursor()
		if cursor < 0 || cursor >= len(m.entryIndices) {
			return keyHandled
		}
		entry := m.entries[m.entryIndices[cursor]]
		m.editingEntry = m.entryIndices[cursor]
		m.statusMessage = "Editing entry..."
		m.statusKind = statusInfo
		m.textInput.SetValue(fmt.Sprintf("%s: %s", entry.EndTime.Format(timelog.TimeLayout), entry.Description))
		m.textInput.Focus()
		m.taskTable.Blur()
		m.focus = focusFooter
		return keyHandled
	case "p":
		cursor := m.taskTable.Cursor()
		if cursor < 0 || cursor >= len(m.entryIndices) {
			return keyHandled
		}
		m.reassigningEntry = m.entryIndices[cursor]
		m.showProjectOverlay = true
		m.inRecents = len(m.recentProjects) > 0
		m.recentCursor = 0
		m.focus = focusProjectTree
		return keyHandled
	}
	return keyIgnored
}

func (m *model) handleKeyMsg(msg tea.KeyMsg) (keyResult, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return keyExit, nil
	case "esc":
		if m.editingEntry >= 0 {
			m.editingEntry = -1
			m.statusMessage = ""
			m.statusKind = statusNone
			m.textInput.Reset()
			return keyHandled, nil
		}
		return keyIgnored, nil
	case "enter":
		if m.focus == focusFooter {
			m.handleInput()
			return keyHandled, nil
		}
		return keyIgnored, nil
	case "ctrl+p":
		m.reassigningEntry = -1
		m.showProjectOverlay = true
		m.inRecents = len(m.recentProjects) > 0
		m.recentCursor = 0
		m.focus = focusProjectTree
		return keyHandled, nil
	case "alt+s":
		m.statusMessage = "Submitting..."
		m.statusKind = statusInfo
		entries := m.entries
		appCfg := m.appConfig
		cmd := func() tea.Msg {
			return submitResultMsg{err: chrono.SubmitTimesheet(entries, appCfg)}
		}
		return keyHandled, cmd
	case "tab":
		m.focus = (m.focus + 1) % 4
		m.textInput.Blur()
		m.taskTable.Blur()
		if m.focus == focusFooter {
			m.textInput.Focus()
		} else if m.focus == focusTable {
			m.taskTable.Focus()
		}
		return keyHandled, nil
	case "shift+tab":
		m.focus = (m.focus + 3) % 4
		m.textInput.Blur()
		m.taskTable.Blur()
		if m.focus == focusFooter {
			m.textInput.Focus()
		} else if m.focus == focusTable {
			m.taskTable.Focus()
		}
		return keyHandled, nil
	}

	// Date navigation when header focused
	if m.focus == focusHeader {
		switch msg.String() {
		case "left", "h":
			m.viewDate = m.viewDate.AddDate(0, 0, -1)
			m.refreshViewForDate()
			return keyHandled, nil
		case "right", "l":
			now := time.Now()
			next := m.viewDate.AddDate(0, 0, 1)
			if next.Year() < now.Year() || (next.Year() == now.Year() && next.YearDay() <= now.YearDay()) {
				m.viewDate = next
				m.refreshViewForDate()
			}
			return keyHandled, nil
		case "[":
			m.viewDate = m.viewDate.AddDate(0, 0, -7)
			m.refreshViewForDate()
			return keyHandled, nil
		case "]":
			now := time.Now()
			next := m.viewDate.AddDate(0, 0, 7)
			if next.Year() < now.Year() || (next.Year() == now.Year() && next.YearDay() <= now.YearDay()) {
				m.viewDate = next
			} else {
				m.viewDate = now
			}
			m.refreshViewForDate()
			return keyHandled, nil
		}
	}

	if m.focus == focusTable {
		return m.handleTableKeyMsg(msg), nil
	}

	return keyIgnored, nil
}
