package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rash419/ttimelog/internal/chrono"
	"github.com/Rash419/ttimelog/internal/config"
	"github.com/Rash419/ttimelog/internal/layout"
	"github.com/Rash419/ttimelog/internal/timelog"
	"github.com/Rash419/ttimelog/internal/treeview"
	"github.com/charmbracelet/bubbles/key"
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
	dailyTargetHours      float64
	weeklyTargetHours     float64
	entryIndices          []int
	showDeleteConfirm     bool
	deleteTargetEntry     int
	editingEntry          int
	reassigningEntry      int
	statusMessage         string
	searchInput           textinput.Model
	recentProjects        []string // paths like "collabora:business-development:demo: "
	recentCursor          int
	inRecents             bool
}

const (
	HeaderHeight = 3
	StatsHeight  = 5
	FooterHeight = 2
)


type (
	errMsg error
)

func initialModel(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, appConfig *config.AppConfig) model {
	txtInput := textinput.New()
	txtInput.Placeholder = "What are you working on?"
	txtInput.Focus()

	timeLogFilePath := filepath.Join(appConfig.TimeLogDirPath, config.TimeLogFilename)
	entries, statsCollections, handledArrivedMessage, err := timelog.LoadEntries(timeLogFilePath)
	if err != nil {
		slog.Error("Failed to load entries", "error", err)
	}

	taskTable, entryIndices := createBodyContent(0, 0, entries)

	projectListFile := filepath.Join(appConfig.TimeLogDirPath, config.ProjectListFile)
	rootNode, err := chrono.ParseProjectList(projectListFile)
	if err != nil {
		slog.Error("Failed to parse project list", "error", err.Error())
	}
	projectTree := treeview.NewTreeView(rootNode)

	searchInput := textinput.New()
	searchInput.Placeholder = "Search projects..."
	searchInput.CharLimit = 100

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
		dailyTargetHours:      appConfig.Gtimelog.Hours,
		weeklyTargetHours:     appConfig.Gtimelog.Hours * 5,
		entryIndices:          entryIndices,
		editingEntry:          -1,
		deleteTargetEntry:     -1,
		reassigningEntry:      -1,
		searchInput:           searchInput,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("Time log"),
		textinput.Blink,
	)
}

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

	rows, indices := getTableRows(m.entries)
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

	rows, indices := getTableRows(m.entries)
	m.taskTable.SetRows(rows)
	m.entryIndices = indices
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
	focusProjectTree
	focusDeleteConfirm
)

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

func (m *model) handleKeyMsg(msg tea.KeyMsg) keyResult {
	switch msg.String() {
	case "ctrl+c":
		return keyExit
	case "esc":
		if m.editingEntry >= 0 {
			m.editingEntry = -1
			m.statusMessage = ""
			m.textInput.Reset()
			return keyHandled
		}
		return keyIgnored
	case "enter":
		if m.focus == focusFooter {
			m.handleInput()
			return keyHandled
		}
		return keyIgnored
	case "ctrl+p":
		m.reassigningEntry = -1
		m.showProjectOverlay = true
		m.inRecents = len(m.recentProjects) > 0
		m.recentCursor = 0
		m.focus = focusProjectTree
		return keyHandled
	case "tab":
		m.focus = (m.focus + 1) % 4
		m.textInput.Blur()
		m.taskTable.Blur()
		if m.focus == focusFooter {
			m.textInput.Focus()
		} else if m.focus == focusTable {
			m.taskTable.Focus()
		}
		return keyHandled
	case "shift+tab":
		m.focus = (m.focus + 3) % 4
		m.textInput.Blur()
		m.taskTable.Blur()
		if m.focus == focusFooter {
			m.textInput.Focus()
		} else if m.focus == focusTable {
			m.taskTable.Focus()
		}
		return keyHandled
	}

	if m.focus == focusTable {
		return m.handleTableKeyMsg(msg)
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
		var keyResult keyResult
		if m.showDeleteConfirm {
			keyResult = m.handleDeleteConfirmKeyMsg(msg)
		} else if m.showProjectOverlay {
			keyResult = m.handleProjectTreeKeyMsg(msg)
		} else {
			keyResult = m.handleKeyMsg(msg)
		}
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
			parts = append(parts, "")
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

	timeLogDirPath := filepath.Join(userDir, config.TimeLogDirname)
	appConfig, err := config.LoadConfig(timeLogDirPath)
	if err != nil {
		slog.Error("Failed to parse config file", "error", err.Error())
		os.Exit(1)
	}

	err = chrono.FetchProjectList(appConfig)
	if err != nil {
		slog.Error("Faield to fetch project list", "error", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	p := tea.NewProgram(initialModel(ctx, cancel, wg, appConfig), tea.WithAltScreen())

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
