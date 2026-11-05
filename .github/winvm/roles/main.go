// Check of #1199: landmarks, live regions, separators and row headers.
package main

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var (
	messages = []string{"Hi"}
	errText  string
)

func view(c *ui.Context) {
	ui.Column(c).Fill().Padding(16).Gap(8).Children(func() {
		ui.Row(c).Role(ui.RoleNavigation).Label("Sections").Gap(6).Children(func() {
			ui.Button(c, "Home")
			ui.Button(c, "Settings")
		})
		ui.Row(c).Role(ui.RoleSearch).Label("Find").Children(func() { ui.Text(c, "search box") })
		ui.Column(c).Role(ui.RoleRegion).Label("Activity").Children(func() { ui.Text(c, "Ann joined") })
		ui.Column(c).Role(ui.RoleRegion).Children(func() { ui.Text(c, "unlabelled region") })
		ui.Column(c).Role(ui.RoleArticle).Label("Post").Children(func() { ui.Text(c, "An article") })
		ui.Column(c).Role(ui.RoleLog).Label("Messages").Children(func() {
			for i, m := range messages {
				ui.Text(c.Key(i), m)
			}
		})
		ui.Row(c).Role(ui.RoleTimer).Label("Time left").Children(func() { ui.Text(c, "5:00") })
		if errText != "" {
			ui.Text(c, errText).Role(ui.RoleAlert)
		}
		ui.Row(c).Role(ui.RoleToolbar).Label("Format").Height(24).Gap(4).Children(func() {
			if ui.Button(c, "Add message").Clicked() {
				messages = append(messages, fmt.Sprintf("Message %d", len(messages)+1))
			}
			ui.Divider(c).Role(ui.RoleSeparator)
			if ui.Button(c, "Show error").Clicked() {
				errText = "Name is required"
				if len(messages) > 3 {
					errText = "Name is too long"
				}
			}
		})
		ui.Column(c).Role(ui.RoleTable).Label("Prices").Children(func() {
			ui.Row(c).Role(ui.RoleRow).Children(func() {
				ui.Row(c).Role(ui.RoleRowHeader).Children(func() {
					ui.Text(c, "Apples")
					ui.Button(c, "Edit")
				})
				ui.Text(c, "3").Role(ui.RoleCell)
			})
		})
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Roles check", Width: 520, Height: 560, Content: ui.View(view)})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
