package layout

import (
	"github.com/charmbracelet/lipgloss"
)

func GetBorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
}


