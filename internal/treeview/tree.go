package treeview

import "strings"

type TreeNode struct {
	Label    string
	Children []*TreeNode
	Expanded bool
	Path     string
}

type Row struct {
	TreeNode *TreeNode
	Depth    int
}

func Traverse(node *TreeNode, depth int, rows *[]Row) {
	if node == nil {
		return
	}

	*rows = append(*rows, Row{
		TreeNode: node,
		Depth:    depth,
	})

	if !node.Expanded {
		return
	}

	for _, child := range node.Children {
		Traverse(child, depth+1, rows)
	}
}

// nodeMatches returns true if the node or any descendant matches the query (case-insensitive).
func nodeMatches(node *TreeNode, query string) bool {
	if strings.Contains(strings.ToLower(node.Label), query) {
		return true
	}
	for _, child := range node.Children {
		if nodeMatches(child, query) {
			return true
		}
	}
	return false
}

// TraverseFiltered performs a depth-first traversal including only nodes that match
// the query or are ancestors of matching nodes. Matching subtrees are shown expanded.
func TraverseFiltered(node *TreeNode, depth int, rows *[]Row, query string) {
	if node == nil {
		return
	}

	lowerQuery := strings.ToLower(query)

	if !nodeMatches(node, lowerQuery) {
		return
	}

	*rows = append(*rows, Row{
		TreeNode: node,
		Depth:    depth,
	})

	for _, child := range node.Children {
		TraverseFiltered(child, depth+1, rows, query)
	}
}

func AppendPath(rootNode *TreeNode, path []string, index int) {
	// Base case: no more labels to consume
	if len(path) == index {
		return
	}

	currentLabel := path[index]

	for _, child := range rootNode.Children {
		if child.Label == currentLabel {
			AppendPath(child, path, index+1)
			return
		}
	}

	// TODO: improve this to only save "Path" for leaf nodes
	newChild := &TreeNode{Label: currentLabel, Path: strings.Join(path, ":")}
	rootNode.Children = append(rootNode.Children, newChild)

	AppendPath(newChild, path, index+1)
}
