// Check of #1178: tray, context, menu-bar and submenu windows on Windows 11.
//
//	winvm-menus light|dark|system
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func icon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			if dx, dy := x-16, y-16; dx*dx+dy*dy <= 12*12 {
				img.Set(x, y, color.NRGBA{0x20, 0x80, 0xe0, 0xff})
			}
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func items() []*mygo.MenuItem {
	return []*mygo.MenuItem{
		{Label: "New Window", Accelerator: "CmdOrCtrl+N"},
		{Label: "Open…", Accelerator: "CmdOrCtrl+Shift+O"},
		{Label: "Show Status", Type: mygo.MenuItemCheckbox, Checked: true},
		mygo.Separator(),
		{Label: "Recent", Submenu: []*mygo.MenuItem{
			{Label: "notes.txt", Accelerator: "CmdOrCtrl+1"},
			{Label: "todo.md"},
		}},
		{Label: "Disabled item", Disabled: true},
		mygo.Separator(),
		{Label: "Quit", Accelerator: "CmdOrCtrl+Q", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
	}
}

var win *mygo.Window

func main() {
	switch os.Args[1] {
	case "dark":
		mygo.Theme.SetSource(mygo.ThemeDark)
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	}
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Label: "&File", Submenu: items()},
			{Label: "&Edit", Submenu: []*mygo.MenuItem{{Role: mygo.RoleCopy}, {Role: mygo.RolePaste}, {Label: "Find", Accelerator: "CmdOrCtrl+F"}}},
		}))
		win = mygo.NewWindow(mygo.WindowOptions{
			Title: "Menus check", Width: 480, Height: 320,
			Content: ui.View(func(c *ui.Context) {
				ui.Column(c).Fill().Center().Gap(12).Children(func() {
					ui.Text(c, "Menus check: "+os.Args[1])
					if ui.Button(c, "Context menu").Clicked() {
						mygo.NewMenu(items()).Popup(win)
					}
				})
			}),
		})
		tray, err := mygo.NewTray(mygo.TrayOptions{ID: "winvm", Icon: icon(), ToolTip: "Menus check", Menu: mygo.NewMenu(items())})
		if err != nil {
			fmt.Println("tray:", err)
			return
		}
		go func() {
			b := tray.Bounds()
			fmt.Printf("tray %d %d %d %d\n", b.X, b.Y, b.Width, b.Height)
		}()
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
