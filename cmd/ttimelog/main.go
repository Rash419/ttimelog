package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Rash419/ttimelog/internal/config"
	"github.com/Rash419/ttimelog/internal/timelog"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	textInput textinput.Model
	err       error
	width     int
	height    int
	entries   []timelog.Entry
}

type (
	errMsg error
)

func initialModel() model {
	txtInput := textinput.New()
	txtInput.Placeholder = "What are you working on?"
	txtInput.Focus()

	filename := "/home/rashesh/.ttimelog/ttimelog.txt"
	entries, err := timelog.LoadEntries(filename)
	if err != nil {
		slog.Error("Failed to load entries", "error", err)
	}

	return model{
		textInput: txtInput,
		err:       nil,
		entries:   entries,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("Time log"),
		textinput.Blink,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = m.width - 8
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			val := m.textInput.Value()
			if val != "" {
				var lastTaskTime time.Time
				if len(m.entries) == 0 {
					lastTaskTime = time.Now()
				} else {
					lastTaskTime = m.entries[len(m.entries)-1].EndTime
				}

				newEntry := timelog.Entry{
					EndTime:     time.Now(),
					Description: val,
					Duration:    time.Since(lastTaskTime),
				}
				m.entries = append(m.entries, newEntry)

				if err := timelog.SaveEntry(newEntry); err != nil {
					slog.Error("Failed to add entry with description", "error", newEntry.Description)
				}
				m.textInput.Reset()
			}
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func createHeaderContent() string {
	timeNow := time.Now()
	_, week := timeNow.ISOWeek()
	dateAndDay := timeNow.Format("January, 02-01-2006")
	return fmt.Sprintf("%s (Week %d)", dateAndDay, week)
}

func createStatsContent(width int) string {
	colWidth := (width - 4) / 3

	colStyle := lipgloss.NewStyle().Width(colWidth).Align(lipgloss.Left)

	// TODO: mock data
	dailyStat := colStyle.Render("TODAY  ████░░░ 1h6m \nLeft: 6h52m → 05:13")
	weeklyStat := colStyle.Render("WEEK  █░░░░░░ 1h6m \nSlack: 0h0m")
	monthlyStat := colStyle.Render("MONTH  1h6m/23d \nLast: 0h0m/22d")

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

func createBodyContent(width int, height int, entries []timelog.Entry) string {
	tableHeaders := getTableHeaders()

	durationColWidth := lipgloss.Width("0 h 00 min")
	timeRangeColWidth := lipgloss.Width("00:00 - 00:00")
	taskColWidth := width - durationColWidth - timeRangeColWidth - len(tableHeaders)*2 // adjust width according to default padding added by the table component

	columns := []table.Column{
		{Title: tableHeaders[0], Width: durationColWidth},
		{Title: tableHeaders[1], Width: timeRangeColWidth},
		{Title: tableHeaders[2], Width: taskColWidth},
	}

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

	taskTable := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	return taskTable.View()
}

func (m model) View() string {
	// make sure width is not negative
	// model.width/height - 2 (border width)
	availableWidth := max(m.width-2, 1)
	availableHeight := max(m.height-2, 1)

	headerContent := createHeaderContent()
	statsContent := createStatsContent(availableWidth)
	footerContent := createFooterContent(m)

	headerHeight := lipgloss.Height(headerContent)
	statsHeight := lipgloss.Height(statsContent)
	footerHeight := lipgloss.Height(footerContent)

	const numOfDividers = 3
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(strings.Repeat("─", availableWidth))

	totalDividerHeight := numOfDividers * lipgloss.Height(divider)
	bodyHeigth := availableHeight - headerHeight - statsHeight - footerHeight - totalDividerHeight

	bodyContent := createBodyContent(availableWidth, bodyHeigth, m.entries)

	innerView := lipgloss.JoinVertical(lipgloss.Left,
		headerContent,
		divider,
		statsContent,
		divider,
		bodyContent,
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
