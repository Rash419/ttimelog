package treeview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#3b4261")).
			Foreground(lipgloss.Color("#c0caf5")).
			Bold(true)
	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a9b1d6"))
	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")).
			Italic(true)
	hintsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3b4261"))
)

type TreeView struct {
	Root         *TreeNode
	Rows         []Row
	Cursor       int
	Viewport     viewport.Model
	SearchQuery  string
	Searching    bool
}

func traverseChildren(root *TreeNode, rows *[]Row) {
	for _, child := range root.Children {
		Traverse(child, 0, rows)
	}
}

func traverseChildrenFiltered(root *TreeNode, rows *[]Row, query string) {
	for _, child := range root.Children {
		TraverseFiltered(child, 0, rows, query)
	}
}

func NewTreeView(root *TreeNode) *TreeView {
	rows := make([]Row, 0)
	traverseChildren(root, &rows)
	return &TreeView{
		Root:     root,
		Rows:     rows,
		Viewport: viewport.New(10, 20),
		Cursor:   0,
	}
}

func (t *TreeView) MoveDown() {
	if t.Cursor < len(t.Rows)-1 {
		t.Cursor++

		if t.Cursor >= t.Viewport.YOffset+t.Viewport.Height {
			t.Viewport.ScrollDown(1)
		}
	}
}

func (t *TreeView) MoveUp() {
	if t.Cursor > 0 {
		t.Cursor--
	}

	if t.Cursor < t.Viewport.YOffset {
		t.Viewport.ScrollUp(1)
	}
}

func (t *TreeView) rebuild() {
	t.Rows = nil
	traverseChildren(t.Root, &t.Rows)

	// Clamp cursor (important when collapsing nodes)
	if t.Cursor >= len(t.Rows) {
		t.Cursor = len(t.Rows) - 1
	}
	if t.Cursor < 0 {
		t.Cursor = 0
	}
}

func (t *TreeView) rebuildFiltered() {
	t.Rows = nil
	if t.SearchQuery == "" {
		traverseChildren(t.Root, &t.Rows)
	} else {
		traverseChildrenFiltered(t.Root, &t.Rows, t.SearchQuery)
	}

	if t.Cursor >= len(t.Rows) {
		t.Cursor = len(t.Rows) - 1
	}
	if t.Cursor < 0 {
		t.Cursor = 0
	}
}

func (t *TreeView) StartSearch() {
	t.Searching = true
	t.SearchQuery = ""
}

func (t *TreeView) StopSearch() {
	t.Searching = false
	t.SearchQuery = ""
	t.rebuild()
}

func (t *TreeView) UpdateSearch(query string) {
	t.SearchQuery = query
	t.rebuildFiltered()
}

func (t *TreeView) Toggle() {
	if len(t.Rows) == 0 {
		return
	}
	node := t.Rows[t.Cursor].TreeNode
	if len(node.Children) == 0 {
		return
	}
	node.Expanded = !node.Expanded
	if t.SearchQuery != "" {
		t.rebuildFiltered()
	} else {
		t.rebuild()
	}
}

func (t *TreeView) View() string {
	var b strings.Builder

	for i, row := range t.Rows {
		indent := strings.Repeat("  ", row.Depth)

		icon := " "
		if len(row.TreeNode.Children) > 0 {
			if row.TreeNode.Expanded {
				icon = "▾"
			} else {
				icon = "▸"
			}
		}

		line := fmt.Sprintf(" %s%s %s", indent, icon, row.TreeNode.Label)

		// Pad the line to viewport width for full-width highlighting
		if len(line) < t.Viewport.Width {
			line += strings.Repeat(" ", t.Viewport.Width-len(line))
		}

		if i == t.Cursor {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(normalStyle.Render(line))
		}
		b.WriteString("\n")
	}

	t.Viewport.SetContent(b.String())
	return t.Viewport.View()
}

func (t *TreeView) GetBreadcrumb() string {
	if len(t.Rows) == 0 {
		return ""
	}
	node := t.Rows[t.Cursor].TreeNode
	if node.Path != "" {
		return breadcrumbStyle.Render(node.Path)
	}
	return breadcrumbStyle.Render(node.Label)
}

func (t *TreeView) GetHints() string {
	return hintsStyle.Render("space:expand  enter:select  /:search  esc:close")
}

func (t *TreeView) SetSize(width, height int) {
	t.Viewport.Width = width
	t.Viewport.Height = height
}

func (t *TreeView) GetProjectPath() string {
	if len(t.Rows) == 0 {
		return ""
	}
	node := t.Rows[t.Cursor].TreeNode
	if node.Children != nil {
		return ""
	}
	return node.Path + ": "
}
