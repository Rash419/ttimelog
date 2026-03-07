package main

import (
	"log/slog"

	"github.com/Rash419/ttimelog/internal/timelog"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *model) reloadEntries() {
	entries, statsCollections, handledArrivedMessage, err := timelog.LoadEntries(m.timeLogFilePath)
	if err != nil {
		slog.Error("Failed to reload entries", "error", err)
		return
	}
	m.entries = entries
	m.statsCollection = statsCollections
	m.handledArrivedMessage = handledArrivedMessage

	rows, indices := getTableRows(m.entries)
	m.taskTable.SetRows(rows)
	m.entryIndices = indices
}

func (m *model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height

	// -2 for border
	availableWidth := msg.Width - 2
	prefixSpace := lipgloss.Width("15:04 > ")
	m.textInput.Width = availableWidth - prefixSpace - 2 // -2 for safety

	// Update table dimensions
	newCols := getTableCols(availableWidth)
	m.taskTable.SetColumns(newCols)
	fixedHeight := HeaderHeight + StatsHeight + FooterHeight + 2
	bodyHeight := max(msg.Height-fixedHeight, 1)
	m.taskTable.SetHeight(bodyHeight)

	// Update size of projectTree (subtract borders + breadcrumb + hints + search bar + recents)
	overlayWidth := max(30, min(m.width*50/100, 80))
	overlayHeight := max(10, min(m.height*60/100, 30))
	recentsLines := 0
	if len(m.recentProjects) > 0 {
		recentsLines = len(m.recentProjects) + 2 // header + items + blank line
	}
	// -2 borders, -1 hints, -1 search bar, -1 title
	m.projectTree.SetSize(overlayWidth-2, max(overlayHeight-5-recentsLines, 3))
}

func (m *model) updateComponents(msg tea.Msg) []tea.Cmd {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	m.taskTable, cmd = m.taskTable.Update(msg)
	cmds = append(cmds, cmd)

	// Scroll to bottom after table has processed the message
	if m.scrollToBottom {
		rowCount := len(m.taskTable.Rows())
		if rowCount > 0 {
			m.taskTable.SetCursor(rowCount - 1)
		}
		m.scrollToBottom = false
	}
	return cmds
}

func (m *model) handleFileChangedMsg() {
	entries, statsCollections, handledArrivedMessage, err := timelog.LoadEntries(m.timeLogFilePath)
	if err != nil {
		slog.Error("Failed to load entries on reload", "error", err)
		return
	}
	m.entries = entries
	m.statsCollection = statsCollections
	m.handledArrivedMessage = handledArrivedMessage

	rows, indices := getTableRows(m.entries)
	m.taskTable.SetRows(rows)
	m.entryIndices = indices
	m.scrollToBottom = true
}
