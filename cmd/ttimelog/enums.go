package main

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
