// Check of #1209: role descriptions, controls, popups, sort and chosen cells,
// and the live regions of #1199 it contains.
package main

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var (
	open     bool
	picked   = 1
	sortDir  = ui.SortAscending
	messages = []string{"Hi"}
	errText  string
)

func view(c *ui.Context) {
	ui.Column(c).Fill().Padding(16).Gap(8).Children(func() {
		ui.Column(c).Role(ui.RoleGroup).Label("Highlights").RoleDescription("carousel").Children(func() {
			ui.Column(c).Role(ui.RoleGroup).Label("1 of 2").RoleDescription("slide").Children(func() { ui.Text(c, "Slide one") })
		})
		toggle := ui.Button(c, "Details").Expanded(open)
		if toggle.Clicked() {
			open = !open
		}
		if open {
			toggle.Controls(ui.Column(c).Role(ui.RoleRegion).Label("Details panel").Children(func() { ui.Text(c, "More") }))
		}
		ui.ButtonBase(c).Role(ui.RolePopUpButton).Label("Fruit").Value("Apple").HasPopup(ui.PopupListBox).Expanded(false).Children(func() { ui.Text(c, "Fruit: Apple") })
		ui.Button(c, "Actions").HasPopup(ui.PopupMenu)
		ui.Column(c).Role(ui.RoleTable).Label("Days").Children(func() {
			ui.Row(c).Role(ui.RoleRow).Children(func() {
				h := ui.Button(c, "Day").Role(ui.RoleColumnHeader).Sort(sortDir)
				if h.Clicked() {
					if sortDir == ui.SortAscending {
						sortDir = ui.SortDescending
					} else {
						sortDir = ui.SortAscending
					}
				}
			})
			for i := 0; i < 3; i++ {
				ui.Row(c.Key(i)).Role(ui.RoleRow).Children(func() {
					cell := ui.Box(c).Role(ui.RoleCell).Label(fmt.Sprintf("November %d", i+1)).Checked(i == picked).Padding(4)
					if cell.Clicked() {
						picked = i
					}
					cell.Children(func() { ui.Text(c, fmt.Sprintf("Nov %d", i+1)) })
				})
			}
		})
		ui.Column(c).Role(ui.RoleLog).Label("Messages").Children(func() {
			for i, m := range messages {
				ui.Text(c.Key(i), m)
			}
		})
		if errText != "" {
			ui.Text(c, errText).Role(ui.RoleAlert)
		}
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, "Add message").Clicked() {
				messages = append(messages, fmt.Sprintf("Message %d", len(messages)+1))
			}
			if ui.Button(c, "Show error").Clicked() {
				errText = "Name is required"
				if len(messages) > 3 {
					errText = "Name is too long"
				}
			}
			if ui.Button(c, "Next day").Clicked() {
				picked = (picked + 1) % 3
			}
		})
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "States check", Width: 520, Height: 640, Content: ui.View(view)})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
