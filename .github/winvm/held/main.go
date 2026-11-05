// Check of #1177: the keys held in a window, next to a focused SearchField.
package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var (
	query string
	last  string
	start = time.Now()
)

func view(c *ui.Context) {
	held, mods := c.KeysDown(), c.Modifiers()
	s := fmt.Sprintf("held=%v mods=%v query=%q", held, mods, query)
	if s != last {
		last = s
		fmt.Printf("%6dms %s\n", time.Since(start).Milliseconds(), s)
	}
	ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
		ui.SearchField(c, &query).Label("Search shortcuts").AutoFocus()
		ui.Text(c, fmt.Sprintf("Held: %v", held)).FontSize(20)
		ui.Text(c, fmt.Sprintf("Modifiers: %v", mods))
		ui.Text(c, fmt.Sprintf("F down: %v", c.KeyDown(ui.KeyF)))
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Label: "&File", Submenu: []*mygo.MenuItem{{Label: "New", Accelerator: "CmdOrCtrl+N"}, {Label: "Open"}}},
			{Label: "&View", Submenu: []*mygo.MenuItem{{Label: "Zoom"}}},
		}))
		mygo.NewWindow(mygo.WindowOptions{Title: "Held keys check", Width: 480, Height: 300, Content: ui.View(view)})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
