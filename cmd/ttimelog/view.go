package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/Rash419/ttimelog/internal/layout"
	"github.com/Rash419/ttimelog/internal/timelog"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

func (m model) createHeaderContent() string {
	_, week := m.viewDate.ISOWeek()
	dateAndDay := m.viewDate.Format("January, 02-01-2006")
	today := time.Now()
	isToday := m.viewDate.Year() == today.Year() && m.viewDate.YearDay() == today.YearDay()
	nav := "◀ h/l ▶"
	if isToday {
		nav = "◀ h"
	}
	return fmt.Sprintf("%s (Week %d)  %s", dateAndDay, week, nav)
}

func (m model) createStatsContent() string {
	colWidth := max((m.width-4)/3, 1)
	progressBarWidth := colWidth - 14

	colStyle := lipgloss.NewStyle().Width(colWidth).Align(lipgloss.Left)

	stats := m.viewDateStats()

	dailyPercent := stats.Daily.Work.Hours() / m.dailyTargetHours
	weeklyPercent := stats.Weekly.Work.Hours() / m.weeklyTargetHours

	dailyBar := progress.New(progress.WithoutPercentage(), progress.WithWidth(progressBarWidth))
	weeklyBar := progress.New(progress.WithoutPercentage(), progress.WithWidth(progressBarWidth))

	leaveTime := timelog.FormatTime(stats.ArrivedTime.Add(time.Duration(m.dailyTargetHours * float64(time.Hour))))

	timeRemaining := m.dailyTargetHours - stats.Daily.Work.Hours()
	timeRemainingDuration := time.Duration(timeRemaining * float64(time.Hour))

	dailyLabel := "TODAY"
	if !m.isViewingToday() {
		dailyLabel = "DAY"
	}

	dailyStat := colStyle.Render(dailyLabel + " " + dailyBar.ViewAs(dailyPercent) + " " + timelog.FormatStatDuration(stats.Daily.Work) + "\nLeft: " + leaveTime + " → " + timelog.FormatStatDuration(timeRemainingDuration) + ", Slack: " + timelog.FormatStatDuration(stats.Daily.Slack))
	weeklyStat := colStyle.Render("WEEK " + weeklyBar.ViewAs(weeklyPercent) + " " + timelog.FormatStatDuration(stats.Weekly.Work) + "\nSlack: " + timelog.FormatStatDuration(stats.Weekly.Slack))
	monthlyStat := colStyle.Render("MONTH " + timelog.FormatStatDuration(stats.Monthly.Work))

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).PaddingRight(1).
		Render(strings.TrimRight(strings.Repeat("│\n", 2), "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, dailyStat, divider, weeklyStat, divider, monthlyStat)
}

func (m model) isViewingToday() bool {
	now := time.Now()
	return m.viewDate.Year() == now.Year() && m.viewDate.YearDay() == now.YearDay()
}

func (m model) viewDateStats() timelog.StatsCollection {
	if m.isViewingToday() {
		return m.statsCollection
	}
	return timelog.StatsCollectionForDate(m.entries, m.viewDate, m.virtualMidnight)
}

func (m model) createFooterContent() string {
	return fmt.Sprintf("%v %s", time.Now().Format("15:04"), m.textInput.View())
}

// best way to get const slice/maps in go
func getTableHeaders() []string {
	return []string{"Duration", "Time Range", "Task"}
}

func getTableCols(width int) []table.Column {
	tableHeaders := getTableHeaders()

	durationColWidth := lipgloss.Width("00 h 00 min")
	timeRangeColWidth := lipgloss.Width("00:00 - 00:00")
	// adjust width according to default padding added by the table component
	taskColWidth := max(0, width-durationColWidth-timeRangeColWidth-len(tableHeaders)*2)

	columns := []table.Column{
		{Title: tableHeaders[0], Width: durationColWidth},
		{Title: tableHeaders[1], Width: timeRangeColWidth},
		{Title: tableHeaders[2], Width: taskColWidth},
	}

	return columns
}

func getTableRows(entries []timelog.Entry, viewDate time.Time, virtualMidnight time.Duration) ([]table.Row, []int) {
	rows := make([]table.Row, 0)
	indices := make([]int, 0)

	targetDate := time.Date(viewDate.Year(), viewDate.Month(), viewDate.Day(), 0, 0, 0, 0, viewDate.Location())

	var lastEndTime time.Time
	for i, entry := range entries {
		startTime := lastEndTime

		vd := timelog.VirtualDate(entry.EndTime, virtualMidnight)
		if vd.Year() != targetDate.Year() || vd.Month() != targetDate.Month() || vd.Day() != targetDate.Day() {
			lastEndTime = entry.EndTime
			continue
		}

		if i == 0 || !timelog.VirtualDate(lastEndTime, virtualMidnight).Equal(vd) {
			startTime = entry.EndTime
		}

		timeRange := fmt.Sprintf("%s - %s", startTime.Format("15:04"), entry.EndTime.Format("15:04"))
		lastEndTime = entry.EndTime
		rows = append(rows, table.Row{timelog.FormatDuration(entry.Duration), timeRange, entry.Description})
		indices = append(indices, i)
	}

	return rows, indices
}

func createBodyContent(width, height int, entries []timelog.Entry, viewDate time.Time, virtualMidnight time.Duration) (table.Model, []int) {
	cols := getTableCols(width)
	rows, indices := getTableRows(entries, viewDate, virtualMidnight)

	km := table.DefaultKeyMap()
	km.HalfPageDown = key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "½ page down"),
	)

	taskTable := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
		table.WithKeyMap(km),
	)
	return taskTable, indices
}

func (m model) View() string {
	// make sure width is not negative
	availableWidth := max(m.width-2, 1)

	headerPane := layout.Pane{
		Width:   availableWidth,
		Title:   "Date",
		View:    m.createHeaderContent,
		Focused: m.focus == focusHeader,
	}

	statsPane := layout.Pane{
		Width:   availableWidth,
		Title:   "Stats",
		View:    m.createStatsContent,
		Focused: m.focus == focusStats,
	}

	fixedHeight := HeaderHeight + StatsHeight + FooterHeight + 2
	bodyHeight := max(m.height-fixedHeight, 1)

	bodyPane := layout.Pane{
		Width:   availableWidth,
		Title:   "Entries",
		View:    m.taskTable.View,
		Height:  bodyHeight,
		Focused: m.focus == focusTable,
	}

	footerPane := layout.Pane{
		Width:   availableWidth,
		Title:   "Input",
		View:    m.createFooterContent,
		Focused: m.focus == focusFooter,
	}

	mainView := lipgloss.JoinVertical(lipgloss.Left,
		headerPane.Render(),
		statsPane.Render(),
		bodyPane.Render(),
		footerPane.Render(),
		m.createStatusBar(availableWidth),
	)

	if m.showReportOverlay {
		reportPane := layout.Pane{
			Title:   "Report",
			Width:   max(60, min(m.width*70/100, 100)),
			Height:  max(15, min(m.height*70/100, 40)),
			View:    func() string { return m.reportViewport.View() },
			Focused: true,
		}
		return overlay.Composite(reportPane.Render(), mainView, overlay.Center, overlay.Center, 0, 0)
	}

	if m.showDeleteConfirm && m.deleteTargetEntry >= 0 && m.deleteTargetEntry < len(m.entries) {
		entry := m.entries[m.deleteTargetEntry]
		content := fmt.Sprintf(
			"%s\n%s\n\nPress y to confirm, n or esc to cancel",
			entry.EndTime.Format(timelog.TimeLayout),
			entry.Description,
		)
		deletePane := layout.Pane{
			Title:   "Confirm Delete",
			Width:   50,
			Height:  8,
			View:    func() string { return content },
			Focused: true,
		}
		return overlay.Composite(deletePane.Render(), mainView, overlay.Center, overlay.Center, 0, 0)
	}

	if !m.showProjectOverlay {
		return mainView
	}

	overlayWidth := max(30, min(m.width*50/100, 80))
	overlayHeight := max(10, min(m.height*60/100, 30))

	recentSelectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#3b4261")).
		Foreground(lipgloss.Color("#c0caf5")).
		Bold(true)
	recentNormalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a9b1d6"))
	recentHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#565f89")).
		Bold(true)

	projectContent := func() string {
		var parts []string
		if m.projectTree.Searching {
			parts = append(parts, "/ "+m.searchInput.View())
		}
		// Recent projects section
		if len(m.recentProjects) > 0 && !m.projectTree.Searching {
			parts = append(parts, recentHeaderStyle.Render("Recent"))
			for i, path := range m.recentProjects {
				// Display the path without trailing ": "
				label := strings.TrimSuffix(path, ": ")
				line := " " + label
				if len(line) < overlayWidth-2 {
					line += strings.Repeat(" ", overlayWidth-2-len(line))
				}
				if m.inRecents && i == m.recentCursor {
					parts = append(parts, recentSelectedStyle.Render(line))
				} else {
					parts = append(parts, recentNormalStyle.Render(line))
				}
			}
			divider := lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Render(strings.Repeat("─", overlayWidth-2))
			parts = append(parts, divider)
		}
		parts = append(parts, m.projectTree.View())
		parts = append(parts, m.projectTree.GetHints())
		return strings.Join(parts, "\n")
	}

	projectPane := layout.Pane{
		Title:   "Projects",
		Width:   overlayWidth,
		Height:  overlayHeight,
		View:    projectContent,
		Focused: true,
	}

	return overlay.Composite(projectPane.Render(), mainView, overlay.Center, overlay.Center, 0, 0)
}

func (m model) createStatusBar(width int) string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89"))

	var hints string
	switch m.focus {
	case focusHeader:
		hints = "h/l: Prev/Next day | [/]: Prev/Next week | alt+o: Editor | alt+r: Report | alt+e: CSV | tab: Switch"
	case focusFooter:
		hints = "↑/↓: History | alt+s: Submit | ctrl+p: Projects | tab: Switch"
	case focusTable:
		hints = "e: Edit | d: Delete | p: Project | alt+o: Editor | tab: Switch"
	default:
		hints = "tab: Switch"
	}

	left := dimStyle.Render(" " + hints)

	var right string
	if m.statusMessage != "" {
		var color lipgloss.Color
		switch m.statusKind {
		case statusSuccess:
			color = lipgloss.Color("#9ece6a")
		case statusError:
			color = lipgloss.Color("#f7768e")
		case statusInfo:
			color = lipgloss.Color("#e0af68")
		default:
			color = lipgloss.Color("#a9b1d6")
		}
		right = lipgloss.NewStyle().Foreground(color).Render(m.statusMessage + " ")
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := max(width-leftWidth-rightWidth, 0)

	return left + strings.Repeat(" ", gap) + right
}
