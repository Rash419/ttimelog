package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rash419/ttimelog/internal/config"
	"github.com/Rash419/ttimelog/internal/layout"
	"github.com/Rash419/ttimelog/internal/timelog"
	"github.com/Rash419/ttimelog/internal/treeview"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	overlay "github.com/rmhubbert/bubbletea-overlay"
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
	ctx                   context.Context
	cancel                context.CancelFunc
	wg                    *sync.WaitGroup
	timeLogFilePath       string
	focus                 Focus
	showProjectOverlay    bool
	projectTree           *treeview.TreeView
}

const (
	HeaderHeight = 3
	StatsHeight  = 5
	FooterHeight = 2
)

// TODO: update dynamically using config
const (
	targetDailyHours  = 8.0
	targetWeeklyHours = 40.0
)

type (
	errMsg error
)

func initialModel(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, timeLogFilePath string) model {
	txtInput := textinput.New()
	txtInput.Placeholder = "What are you working on?"
	txtInput.Focus()

	entries, statsCollections, handledArrivedMessage, err := timelog.LoadEntries(timeLogFilePath)
	if err != nil {
		slog.Error("Failed to load entries", "error", err)
	}

	taskTable := createBodyContent(0, 0, entries)

	nodeA := treeview.TreeNode{
		Label:    "A",
		Expanded: true,
	}

	nodeB := treeview.TreeNode{
		Label:    "B",
		Expanded: true,
	}

	nodeC := treeview.TreeNode{
		Label:    "C",
		Expanded: true,
	}

	nodeD := treeview.TreeNode{
		Label:    "D",
		Expanded: true,
	}

	nodeC.Children = append(nodeC.Children, &nodeD)
	nodeA.Children = append(nodeA.Children, &nodeB, &nodeC)

	projectTree := treeview.NewTreeView(&nodeA)

	return model{
		textInput:             txtInput,
		err:                   nil,
		entries:               entries,
		taskTable:             taskTable,
		statsCollection:       statsCollections,
		scrollToBottom:        true,
		handledArrivedMessage: handledArrivedMessage,
		ctx:                   ctx,
		cancel:                cancel,
		wg:                    wg,
		timeLogFilePath:       timeLogFilePath,
		focus:                 focusFooter,
		projectTree:           projectTree,
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
	if err := timelog.SaveEntry(newEntry, handleArrivedMessage, m.timeLogFilePath); err != nil {
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
	newCols := getTableCols(int(math.Round(float64(availableWidth) / 1.3)))
	m.taskTable.SetColumns(newCols)
	fixedHeight := HeaderHeight + StatsHeight + FooterHeight + 2
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

type shutdownCompleteMsg struct{}

func (m *model) handleFileChangedMsg() {
	entries, statsCollections, handledArrivedMessage, err := timelog.LoadEntries(m.timeLogFilePath)
	if err != nil {
		slog.Error("Failed to load entries on reload", "error", err)
		return
	}
	m.entries = entries
	m.statsCollection = statsCollections
	m.handledArrivedMessage = handledArrivedMessage

	rows := getTableRows(m.entries)
	m.taskTable.SetRows(rows)
	m.scrollToBottom = true
}

type keyResult int

const (
	keyIgnored keyResult = iota
	keyHandled
	keyExit
)

type Focus int

const (
	focusHeader Focus = iota
	focusStats
	focusTable
	focusFooter
)

func (m *model) handleKeyMsg(msg tea.KeyMsg) keyResult {
	switch msg.String() {
	case "ctrl+c":
		return keyExit
	case "enter":
		m.handleInput()
		return keyHandled
	case "ctrl+p":
		if m.showProjectOverlay {
			m.showProjectOverlay = false
		} else {
			m.showProjectOverlay = true
		}
		return keyHandled
	case "esc":
		m.showProjectOverlay = false
		return keyHandled
	case "1":
		m.focus = focusHeader
		m.textInput.Blur()
		m.taskTable.Blur()
		return keyHandled
	case "2":
		m.focus = focusStats
		m.textInput.Blur()
		m.taskTable.Blur()
		return keyHandled
	case "3":
		m.focus = focusTable
		m.textInput.Blur()
		m.taskTable.Focus()
		return keyHandled
	case "4":
		m.focus = focusFooter
		m.taskTable.Blur()
		m.textInput.Focus()
		return keyHandled
	}
	return keyIgnored
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)
	case fileChangedMsg:
		m.handleFileChangedMsg()
	case fileErrorMsg:
	// TODO: handle file watch error
	case tea.KeyMsg:
		keyResult := m.handleKeyMsg(msg)
		switch keyResult {
		case keyHandled:
			return m, nil
		case keyExit:
			m.cancel()
			return m, func() tea.Msg {
				m.wg.Wait()
				return shutdownCompleteMsg{}
			}
		}
	case shutdownCompleteMsg:
		return m, tea.Quit

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

func (m model) createStatsContent() string {
	colWidth := max((m.width-4)/3, 1)
	progressBarWidth := colWidth - 14

	colStyle := lipgloss.NewStyle().Width(colWidth).Align(lipgloss.Left)

	dailyPercent := m.statsCollection.Daily.Work.Hours() / targetDailyHours
	weeklyPercent := m.statsCollection.Weekly.Work.Hours() / targetWeeklyHours

	dailyBar := progress.New(progress.WithoutPercentage(), progress.WithWidth(progressBarWidth))
	weeklyBar := progress.New(progress.WithoutPercentage(), progress.WithWidth(progressBarWidth))

	leaveTime := timelog.FormatTime(m.statsCollection.ArrivedTime.Add(time.Duration(targetDailyHours * float64(time.Hour))))

	timeRemaining := targetDailyHours - m.statsCollection.Daily.Work.Hours()
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

func (m model) createProjectTree() string {
	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(40).
		Height(15).
		Background(lipgloss.Color("235"))

	return overlayStyle.Render(m.projectTree.View())
}

func (m model) View() string {
	// make sure width is not negative
	availableWidth := max(m.width-2, 1)

	headerPane := layout.Pane{
		Width:   availableWidth,
		Title:   "[1]",
		View:    createHeaderContent,
		Focused: m.focus == focusHeader,
	}

	statsPane := layout.Pane{
		Width:   availableWidth,
		Title:   "[2]",
		View:    m.createStatsContent,
		Focused: m.focus == focusStats,
	}

	fixedHeight := HeaderHeight + StatsHeight + FooterHeight + 2
	bodyHeight := max(m.height-fixedHeight, 1)

	bodyPane := layout.Pane{
		Width:   availableWidth,
		Title:   "[3]",
		View:    m.taskTable.View,
		Height:  bodyHeight,
		Focused: m.focus == focusTable,
	}

	footerPane := layout.Pane{
		Width:   availableWidth,
		Title:   "[4]",
		View:    m.createFooterContent,
		Focused: m.focus == focusFooter,
	}

	mainView := lipgloss.JoinVertical(lipgloss.Left,
		headerPane.Render(),
		statsPane.Render(),
		bodyPane.Render(),
		footerPane.Render(),
	)

	if !m.showProjectOverlay {
		return mainView
	}

	return overlay.Composite(m.createProjectTree(), mainView, overlay.Center, overlay.Center, 0, 0)
}

type fileChangedMsg struct{}

type fileErrorMsg struct {
	err error
}

// watch modification in ".ttimelog.txt"
func fileWatcher(ctx context.Context, wg *sync.WaitGroup, program *tea.Program, timeLogFilePath string) error {
	defer wg.Done()

	slog.Debug("Starting filewatcher on", "filePath", timeLogFilePath)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	defer func() {
		if err := watcher.Close(); err != nil {
			slog.Error("Failed to close watcher", "error", err.Error())
		}
	}()

	err = watcher.Add(filepath.Dir(timeLogFilePath))
	if err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Write|
				fsnotify.Create|
				fsnotify.Rename) != 0 && filepath.Base(event.Name) == config.TimeLogFilename {
				program.Send(fileChangedMsg{})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			program.Send(fileErrorMsg{
				err: err,
			})
		case <-ctx.Done():
			return nil
		}
	}
}

func main() {
	userDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Failed to get user home directory", "error", err.Error())
		os.Exit(1)
	}

	logFilePath := filepath.Join(userDir, config.TimeLogDirname, "ttimelog.log")
	logFile, err := os.OpenFile(
		logFilePath,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0o644,
	)
	if err != nil {
		log.Fatalf("Failed to create logFile with error[%v]", err.Error())
	}

	defer func() {
		if err := logFile.Close(); err != nil {
			slog.Error("Failed to close log file", "error", err)
		}
	}()

	slogger := config.GetSlogger(logFile)
	slog.SetDefault(slogger)

	timeLogFilePath, err := config.SetupTimeLogDirectory(userDir)
	if err != nil {
		slog.Error("Setting up timelog file", "error", err.Error())
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	p := tea.NewProgram(initialModel(ctx, cancel, wg, timeLogFilePath), tea.WithAltScreen())

	wg.Add(1)
	go func() {
		err := fileWatcher(ctx, wg, p, timeLogFilePath)
		if err != nil {
			slog.Error("Failed to start filewatcher", "error", err)
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
