package main

import (
	"log/slog"
	"strings"

	"github.com/Rash419/ttimelog/internal/timelog"
)

const maxRecentProjects = 3

func (m *model) addRecentProject(path string) {
	// Remove duplicate if exists
	filtered := make([]string, 0, len(m.recentProjects))
	for _, p := range m.recentProjects {
		if p != path {
			filtered = append(filtered, p)
		}
	}
	// Prepend
	m.recentProjects = append([]string{path}, filtered...)
	if len(m.recentProjects) > maxRecentProjects {
		m.recentProjects = m.recentProjects[:maxRecentProjects]
	}
}

func (m *model) selectProject(projectPath string) {
	m.addRecentProject(projectPath)
	if m.reassigningEntry >= 0 {
		entry := m.entries[m.reassigningEntry]
		desc := entry.Description
		if parts := strings.SplitN(desc, ": ", 2); len(parts) == 2 {
			desc = parts[1]
		}
		newDescription := projectPath + desc
		if err := timelog.EditEntry(m.timeLogFilePath, entry.LineNumber, entry.EndTime.Format(timelog.TimeLayout), newDescription); err != nil {
			slog.Error("Failed to reassign project", "error", err)
		}
		m.reloadEntries()
		m.reassigningEntry = -1
		m.focus = focusTable
	} else {
		m.textInput.SetValue(projectPath)
		m.focus = focusFooter
	}
	m.showProjectOverlay = false
	m.projectTree.StopSearch()
	m.searchInput.Reset()
	m.searchInput.Blur()
	m.inRecents = false
}

func (m *model) closeProjectOverlay() {
	m.showProjectOverlay = false
	m.projectTree.StopSearch()
	m.searchInput.Reset()
	m.searchInput.Blur()
	m.inRecents = false
	if m.reassigningEntry >= 0 {
		m.reassigningEntry = -1
		m.focus = focusTable
	}
}

func (m *model) moveProjectCursorDown() {
	if m.inRecents {
		if m.recentCursor < len(m.recentProjects)-1 {
			m.recentCursor++
		} else {
			// Move from recents into tree
			m.inRecents = false
		}
	} else {
		m.projectTree.MoveDown()
	}
}

func (m *model) moveProjectCursorUp() {
	if m.inRecents {
		if m.recentCursor > 0 {
			m.recentCursor--
		}
	} else if m.projectTree.Cursor == 0 && len(m.recentProjects) > 0 {
		// Move from tree into recents
		m.inRecents = true
		m.recentCursor = len(m.recentProjects) - 1
	} else {
		m.projectTree.MoveUp()
	}
}
