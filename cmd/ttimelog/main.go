package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Rash419/ttimelog/internal/config"
	"github.com/Rash419/ttimelog/internal/layout"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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

func (m model) View() string {
	border := layout.GetBorderStyle()

	textinputBox := border.Width(m.width - 2).Render(
		fmt.Sprintf("%v %s", time.Now().Format("15:04:05"), m.textInput.View()),
	)
	textInputHeight := lipgloss.Height(ansi.Strip(textinputBox))

	listBoxHeight := max(m.height-textInputHeight, 1)

	var listBoxData string
	for _, item := range m.list {
		listBoxData += fmt.Sprintf("* %s\n", item)
	}
	listBox := border.Width(m.width - 2).Height(listBoxHeight - 2).Render(listBoxData)

	return lipgloss.JoinVertical(lipgloss.Left, listBox, textinputBox)
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
