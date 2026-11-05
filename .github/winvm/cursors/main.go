// Check of #1204: the new cursors, pointer type and pressure, and the
// clipboard write result.
package main

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var cursors = []struct {
	name string
	c    ui.Cursor
}{
	{"progress", ui.CursorProgress}, {"wait", ui.CursorWait}, {"help", ui.CursorHelp}, {"cell", ui.CursorCell},
	{"zoom-in", ui.CursorZoomIn}, {"zoom-out", ui.CursorZoomOut}, {"all-scroll", ui.CursorAllScroll}, {"no-drop", ui.CursorNoDrop},
	{"pointer", ui.CursorPointer}, {"crosshair", ui.CursorCrosshair}, {"not-allowed", ui.CursorNotAllowed}, {"move", ui.CursorMove},
}

var copied = "not yet"

func view(c *ui.Context) {
	ui.Column(c).Fill().Padding(10).Gap(6).Children(func() {
		for row := 0; row < 3; row++ {
			ui.Row(c.Key(row)).Gap(6).Children(func() {
				for _, k := range cursors[row*4 : row*4+4] {
					ui.Box(c.Key(k.name)).Width(110).Height(56).Center().Background(ui.RGB(0xe8, 0xe8, 0xf0)).Cursor(k.c).Label("cursor " + k.name).Children(func() {
						ui.Text(c, k.name)
					})
				}
			})
		}
		pad := ui.Box(c).Width(462).Height(120).Background(ui.RGB(0xd0, 0xe8, 0xd0)).Label("Pad").Center().Children(func() { ui.Text(c, "pad") })
		pad.HandleInput(func(ev ui.InputEvent) bool {
			switch ev.Kind {
			case ui.InputPointerDown, ui.InputPointerMove, ui.InputPointerUp:
				fmt.Printf("input kind=%v pointer=%v pressure=%.3f x=%.0f y=%.0f button=%d\n", ev.Kind, ev.Pointer, ev.Pressure, ev.X, ev.Y, ev.Button)
			}
			return false
		})
		if t, p, ok := pad.PressedPointer(); ok {
			fmt.Printf("pressed pointer=%v pressure=%.3f\n", t, p)
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "Copy").Clicked() {
				c.WriteClipboardThen("MyGo winvm", func(err error) {
					copied = fmt.Sprintf("result err=%v", err)
					fmt.Println("clipboard", copied)
				})
			}
			ui.Text(c, "Copy: "+copied)
		})
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Cursors check", Width: 500, Height: 420, Content: ui.View(view)})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
