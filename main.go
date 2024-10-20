package main

// A simple example illustrating how to set a window title.

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	taskList  []string
	textinput textinput.Model
	err       error
}

type (
	errMsg error
)

func initialModel() model {
	textInput := textinput.New()
	textInput.Focus()
	textInput.Width = 25

	return model{
		taskList:  make([]string, 0),
		textinput: textInput,
		err:       nil,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
    case tea.KeyEnter:
      task := strings.TrimSpace(m.textinput.Value())
      m.taskList = append(m.taskList, task)
      m.textinput.SetValue("")
		}
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textinput, cmd = m.textinput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	var view string

	for _, task := range m.taskList {
		view += fmt.Sprintf("%s - %s\n", "*", task)
	}

	view += fmt.Sprintf(
		"Arrival Message%s\n\n%s",
		m.textinput.View(),
		"(esc to quit)",
	) + "\n"

	return view
}

func main() {
	if _, err := tea.NewProgram(initialModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Uh oh:", err)
		os.Exit(1)
	}
}
