package treeview

import (
	"log/slog"
	"strings"
)

type TreeView struct {
	Root   *TreeNode
	Rows   []Row
	Cursor int
}

func NewTreeView(root *TreeNode) *TreeView {
	rows := make([]Row, 0)
	Traverse(root, 0, &rows)
	return &TreeView{
		Root:   root,
		Rows:   rows,
		Cursor: 0,
	}
}

func (t *TreeView) MoveDown() {
	if t.Cursor < len(t.Rows)-1 {
		t.Cursor++
	}
}

func (t *TreeView) MoveUp() {
	if t.Cursor > 0 {
		t.Cursor--
	}
}

func (t *TreeView) rebuild() {
	t.Rows = nil
	Traverse(t.Root, 0, &t.Rows)

	// Clamp cursor (important when collapsing nodes)
	if t.Cursor >= len(t.Rows) {
		t.Cursor = len(t.Rows) - 1
	}
	if t.Cursor < 0 {
		t.Cursor = 0
	}
}

func (t *TreeView) Toggle() {
	node := t.Rows[t.Cursor].TreeNode
	if len(node.Children) == 0 {
		return
	}
	node.Expanded = !node.Expanded
	t.rebuild()
}

func (t *TreeView) View() string {
	var b strings.Builder

	for i, row := range t.Rows {
		cursor := " "
		if i == t.Cursor {
			cursor = ">"
		}

		indent := strings.Repeat("  ", row.Depth)

		icon := " "
		if len(row.TreeNode.Children) > 0 {
			if row.TreeNode.Expanded {
				icon = "▾"
			} else {
				icon = "▸"
			}
		}

		b.WriteString(cursor)
		b.WriteString(" ")
		b.WriteString(indent)
		b.WriteString(icon)
		b.WriteString(" ")
		b.WriteString(row.TreeNode.Label)
		b.WriteString("\n")
	}

	slog.Debug("Treeview", "treeview", b.String())
	return b.String()
}
