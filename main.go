package main

import (
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	flex := tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(tview.NewBox().SetBorder(true).SetTitle("Time log"), 0, 3, false).
				AddItem(tview.NewBox().SetBorder(true).SetTitle("Task"), 0, 1, false), 0, 100, false).
			AddItem(tview.NewInputField().SetLabel("22:11"), 0, 1, false), 0, 1, false)

	if err := app.SetRoot(flex, true).SetFocus(flex).Run(); err != nil {
		panic(err)
	}
}
