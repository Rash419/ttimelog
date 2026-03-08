package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/Rash419/ttimelog/internal/chrono"
	"github.com/Rash419/ttimelog/internal/config"
	"github.com/Rash419/ttimelog/internal/timelog"
	"github.com/Rash419/ttimelog/internal/treeview"
	"github.com/Rash419/ttimelog/internal/watcher"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	statusKind            statusKind
	searchInput           textinput.Model
	recentProjects        []string // paths like "collabora:business-development:demo: "
	recentCursor          int
	inRecents             bool
	appConfig             *config.AppConfig
}

const (
	HeaderHeight = 3
	StatsHeight  = 5
	FooterHeight = 3
)

type (
	errMsg          error
	submitResultMsg struct{ err error }
)

type shutdownCompleteMsg struct{}

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
		appConfig:             appConfig,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("Time log"),
		textinput.Blink,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)
	case watcher.FileChangedMsg:
		m.handleFileChangedMsg()
	case watcher.FileErrorMsg:
		// TODO: handle file watch error
	case submitResultMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Submit failed: %s", msg.err)
			m.statusKind = statusError
		} else {
			m.statusMessage = "Timesheet submitted"
			m.statusKind = statusSuccess
		}
		return m, nil
	case tea.KeyMsg:
		var kr keyResult
		var cmd tea.Cmd
		if m.showDeleteConfirm {
			kr = m.handleDeleteConfirmKeyMsg(msg)
		} else if m.showProjectOverlay {
			kr = m.handleProjectTreeKeyMsg(msg)
		} else {
			kr, cmd = m.handleKeyMsg(msg)
		}
		switch kr {
		case keyHandled:
			return m, cmd
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
		err := watcher.Watch(ctx, wg, p, timeLogFilePath, config.TimeLogFilename)
		if err != nil {
			slog.Error("Failed to start filewatcher", "error", err)
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
