// A focused SearchField and nothing else: the Japanese IME there, on a PR
// and on main.
package main

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var query, last string

func view(c *ui.Context) {
	if query != last {
		last = query
		fmt.Printf("query=%q\n", query)
	}
	ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
		ui.SearchField(c, &query).Label("Search shortcuts").AutoFocus()
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "IME check", Width: 480, Height: 200, Content: ui.View(view)})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
