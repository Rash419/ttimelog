package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Rash419/ttimelog/internal/config"
	"github.com/Rash419/ttimelog/internal/timelog"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	textInput             textinput.Model
	taskTable             table.Model
	err                   error
	width                 int
	height                int
	entries               []timelog.Entry
	statsCollection       timelog.StatsCollection
	scrollToBottom        bool
	handledArrivedMessage bool
}

const (
	HeaderHeight  = 1
	StatsHeight   = 2
	FooterHeight  = 1
	DividerHeight = 1
	NumDividers   = 3
	BorderHeight  = 2 // Top + Bottom
)

// TODO: update dynamically using config
const (
	targetDailyHours  = 8.0
	targetWeeklyHours = 40.0
)

type (
	errMsg error
)

func initialModel() model {
	txtInput := textinput.New()
	txtInput.Placeholder = "What are you working on?"
	txtInput.Focus()

	filename := "/home/rashesh/.ttimelog/ttimelog.txt"
	entries, statsCollections, handledArrivedMessage, err := timelog.LoadEntries(filename)
	if err != nil {
		slog.Error("Failed to load entries", "error", err)
	}

	// TODO: maybe not the best to use "0" width values
	taskTable := createBodyContent(0, 0, entries)

	return model{
		textInput:             txtInput,
		err:                   nil,
		entries:               entries,
		taskTable:             taskTable,
		statsCollection:       statsCollections,
		scrollToBottom:        true,
		handledArrivedMessage: handledArrivedMessage,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("Time log"),
		textinput.Blink,
	)
}

func (m *model) handleInput() {
	val := m.textInput.Value()
	if val == "" {
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
	if err := timelog.SaveEntry(newEntry, handleArrivedMessage); err != nil {
		slog.Error("Failed to add entry with description", "error", newEntry.Description)
	}

	rows := getTableRows(m.entries)
	m.taskTable.SetRows(rows)
	m.scrollToBottom = true

	timelog.UpdateStatsCollection(newEntry, &m.statsCollection)

	m.textInput.Reset()
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
	fixedHeight := HeaderHeight + StatsHeight + FooterHeight + (DividerHeight * NumDividers) + BorderHeight
	bodyHeight := max(msg.Height-fixedHeight, 1)
	m.taskTable.SetHeight(bodyHeight)
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			m.handleInput()
		case tea.KeyEsc:
			if m.taskTable.Focused() {
				m.taskTable.Blur()
			} else {
				m.taskTable.Focus()
			}
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	cmds := m.updateComponents(msg)
	return m, tea.Batch(cmds...)
}

func createHeaderContent() string {
	timeNow := time.Now()
	_, week := timeNow.ISOWeek()
	dateAndDay := timeNow.Format("January, 02-01-2006")
	return fmt.Sprintf("%s (Week %d)", dateAndDay, week)
}

func createStatsContent(width int, m model) string {
	colWidth := (width - 4) / 3
	progressBarWidth := colWidth - 14

	colStyle := lipgloss.NewStyle().Width(colWidth).Align(lipgloss.Left)

	dailyPercent := m.statsCollection.Daily.Work.Hours() / targetDailyHours
	weeklyPercent := m.statsCollection.Weekly.Work.Hours() / targetWeeklyHours

	dailyBar := progress.New(progress.WithoutPercentage(), progress.WithWidth(progressBarWidth))
	weeklyBar := progress.New(progress.WithoutPercentage(), progress.WithWidth(progressBarWidth))

	arrivedTime := timelog.FormatTime(m.statsCollection.ArrivedTime)

	timeRemaining := targetDailyHours - m.statsCollection.Daily.Work.Hours()
	timeRemainingDuration := time.Duration(timeRemaining * float64(time.Hour))

	dailyStat := colStyle.Render("TODAY " + dailyBar.ViewAs(dailyPercent) + " " + timelog.FormatStatDuration(m.statsCollection.Daily.Work) + "\nLeft: " + arrivedTime + " → " + timelog.FormatStatDuration(timeRemainingDuration))
	weeklyStat := colStyle.Render("WEEK " + weeklyBar.ViewAs(weeklyPercent) + " " + timelog.FormatStatDuration(m.statsCollection.Weekly.Work) + "\nSlack: 0h0m")
	monthlyStat := colStyle.Render("MONTH " + timelog.FormatStatDuration(m.statsCollection.Monthly.Work) + "\nLast: 0h0m/22d")

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).PaddingRight(1).
		Render(strings.TrimRight(strings.Repeat("│\n", 2), "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, dailyStat, divider, weeklyStat, divider, monthlyStat)
}

func createFooterContent(m model) string {
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

func getTableRows(entries []timelog.Entry) []table.Row {
	rows := make([]table.Row, 0)

	var lastEndTime time.Time
	for i, entry := range entries {
		startTime := lastEndTime
		entryDate := entry.EndTime.Format("2006-01-02")
		currentDate := time.Now().Format("2006-01-02")

		// only show entries for today
		if entryDate != currentDate {
			continue
		}

		if i == 0 || lastEndTime.Format("2006-01-02") != entryDate {
			startTime = entry.EndTime
		}

		timeRange := fmt.Sprintf("%s - %s", startTime.Format("15:04"), entry.EndTime.Format("15:04"))
		lastEndTime = entry.EndTime
		rows = append(rows, table.Row{timelog.FormatDuration(entry.Duration), timeRange, entry.Description})
	}

	return rows
}

func createBodyContent(width, height int, entries []timelog.Entry) table.Model {
	cols := getTableCols(width)
	rows := getTableRows(entries)
	taskTable := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)
	return taskTable
}

func (m model) View() string {
	// make sure width is not negative
	availableWidth := max(m.width-2, 1)

	headerContent := createHeaderContent()
	statsContent := createStatsContent(availableWidth, m)
	footerContent := createFooterContent(m)

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(strings.Repeat("─", availableWidth))

	innerView := lipgloss.JoinVertical(lipgloss.Left,
		headerContent,
		divider,
		statsContent,
		divider,
		m.taskTable.View(),
		divider,
		footerContent,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Render(innerView)
}

func main() {
	slogger := config.GetSlogger()
	slog.SetDefault(slogger)

	userDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Failed to get user home directory", "error", err.Error())
		os.Exit(1)
	}
	err = config.SetupTimeLogDirectory(userDir)
	if err != nil {
		slog.Error("Setup failed", "error", err.Error())
		os.Exit(1)
	}
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
