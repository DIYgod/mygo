// What a view gets of pointer input, with only what main has: a baseline
// for #1204's pen and touch checks.
package main

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func view(c *ui.Context) {
	ui.Column(c).Fill().Padding(10).Gap(6).Children(func() {
		pad := ui.Box(c).Width(462).Height(300).Background(ui.RGB(0xd0, 0xe8, 0xd0)).Label("Pad").Center().Children(func() { ui.Text(c, "pad") })
		pad.HandleInput(func(ev ui.InputEvent) bool {
			switch ev.Kind {
			case ui.InputPointerDown, ui.InputPointerMove, ui.InputPointerUp:
				fmt.Printf("input kind=%v x=%.0f y=%.0f button=%d\n", ev.Kind, ev.X, ev.Y, ev.Button)
			}
			return false
		})
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Pointer check", Width: 500, Height: 360, Content: ui.View(view)})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
