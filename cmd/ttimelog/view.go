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

func createHeaderContent() string {
	timeNow := time.Now()
	_, week := timeNow.ISOWeek()
	dateAndDay := timeNow.Format("January, 02-01-2006")
	return fmt.Sprintf("%s (Week %d)", dateAndDay, week)
}

func (m model) createStatsContent() string {
	colWidth := max((m.width-4)/3, 1)
	progressBarWidth := colWidth - 14

	colStyle := lipgloss.NewStyle().Width(colWidth).Align(lipgloss.Left)

	dailyPercent := m.statsCollection.Daily.Work.Hours() / m.dailyTargetHours
	weeklyPercent := m.statsCollection.Weekly.Work.Hours() / m.weeklyTargetHours

	dailyBar := progress.New(progress.WithoutPercentage(), progress.WithWidth(progressBarWidth))
	weeklyBar := progress.New(progress.WithoutPercentage(), progress.WithWidth(progressBarWidth))

	leaveTime := timelog.FormatTime(m.statsCollection.ArrivedTime.Add(time.Duration(m.dailyTargetHours * float64(time.Hour))))

	timeRemaining := m.dailyTargetHours - m.statsCollection.Daily.Work.Hours()
	timeRemainingDuration := time.Duration(timeRemaining * float64(time.Hour))

	dailyStat := colStyle.Render("TODAY " + dailyBar.ViewAs(dailyPercent) + " " + timelog.FormatStatDuration(m.statsCollection.Daily.Work) + "\nLeft: " + leaveTime + " → " + timelog.FormatStatDuration(timeRemainingDuration) + ", Slack: " + timelog.FormatStatDuration(m.statsCollection.Daily.Slack))
	weeklyStat := colStyle.Render("WEEK " + weeklyBar.ViewAs(weeklyPercent) + " " + timelog.FormatStatDuration(m.statsCollection.Weekly.Work) + "\nSlack: " + timelog.FormatStatDuration(m.statsCollection.Weekly.Slack))
	monthlyStat := colStyle.Render("MONTH " + timelog.FormatStatDuration(m.statsCollection.Monthly.Work))

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).PaddingRight(1).
		Render(strings.TrimRight(strings.Repeat("│\n", 2), "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, dailyStat, divider, weeklyStat, divider, monthlyStat)
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

func getTableRows(entries []timelog.Entry) ([]table.Row, []int) {
	rows := make([]table.Row, 0)
	indices := make([]int, 0)

	var lastEndTime time.Time
	for i, entry := range entries {
		startTime := lastEndTime
		entryDate := entry.EndTime.Format("2006-01-02")
		currentDate := time.Now().Format("2006-01-02")

		// only show entries for today
		if entryDate != currentDate {
			lastEndTime = entry.EndTime
			continue
		}

		if i == 0 || lastEndTime.Format("2006-01-02") != entryDate {
			startTime = entry.EndTime
		}

		timeRange := fmt.Sprintf("%s - %s", startTime.Format("15:04"), entry.EndTime.Format("15:04"))
		lastEndTime = entry.EndTime
		rows = append(rows, table.Row{timelog.FormatDuration(entry.Duration), timeRange, entry.Description})
		indices = append(indices, i)
	}

	return rows, indices
}

func createBodyContent(width, height int, entries []timelog.Entry) (table.Model, []int) {
	cols := getTableCols(width)
	rows, indices := getTableRows(entries)

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
		View:    createHeaderContent,
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

	footerTitle := "Input"
	if m.statusMessage != "" {
		footerTitle = "Input" + " " + m.statusMessage
	}

	footerPane := layout.Pane{
		Width:   availableWidth,
		Title:   footerTitle,
		View:    m.createFooterContent,
		Focused: m.focus == focusFooter,
	}

	mainView := lipgloss.JoinVertical(lipgloss.Left,
		headerPane.Render(),
		statsPane.Render(),
		bodyPane.Render(),
		footerPane.Render(),
	)

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
