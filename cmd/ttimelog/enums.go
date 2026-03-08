package main

type keyResult int

const (
	keyIgnored keyResult = iota
	keyHandled
	keyExit
)

type statusKind int

const (
	statusNone statusKind = iota
	statusInfo
	statusSuccess
	statusError
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
