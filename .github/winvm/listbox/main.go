// Check of #1179: an open Select's popup reads as a list box of options.
package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var plan = "Pro"

func view(c *ui.Context) {
	ui.Column(c).Fill().Padding(24).Gap(12).Children(func() {
		ui.Text(c, "Plan: "+plan)
		ui.Select(c, &plan, []string{"Free", "Pro", "Team", "Enterprise"}).Label("Plan")
		ui.Button(c, "Other")
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Listbox check", Width: 420, Height: 360, Content: ui.View(view)})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
