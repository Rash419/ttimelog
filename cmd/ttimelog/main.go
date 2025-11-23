package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Rash419/ttimelog/internal/config"
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
	list      []string
}

type (
	errMsg error
)

func initialModel() model {
	txtInput := textinput.New()
	txtInput.Placeholder = "What are you working on?"
	txtInput.Focus()

	return model{
		textInput: txtInput,
		err:       nil,
		list:      make([]string, 0),
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
				m.list = append(m.list, val)
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

func createBodyContent(width int, height int) string {
	tableHeaders := []string{"Duration", "Time Range", "Task"}

	durationColWidth := max(lipgloss.Width("00h 00m"), lipgloss.Width(tableHeaders[0]))
	timeRangeColWidth := max(lipgloss.Width("00:00 - 00:00"), lipgloss.Width(tableHeaders[1]))
	taskColWidth := width - durationColWidth - timeRangeColWidth - len(tableHeaders)*2 // adjust width according to default padding added by the table component

	columns := []table.Column{
		{Title: tableHeaders[0], Width: durationColWidth},
		{Title: tableHeaders[1], Width: timeRangeColWidth},
		{Title: tableHeaders[2], Width: taskColWidth},
	}

	// TODO: mock data
	rows := []table.Row{
		{"0h 0m", "21:13 - 21:13", "**arrived"},
		{"1h 2m", "21:35 - 21:37", "productivity: r&d-productivity: product: collabora-online-25-04: working on online"},
		{"2h 44m", "21:37 - 22:19", "working"},
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

	bodyContent := createBodyContent(availableWidth, bodyHeigth)

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
