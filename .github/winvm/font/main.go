// Check of #1232: Selawik's faces registered under families the app gives.
//
//	winvm-font <fonts dir> <order: semi|light> [window]
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const sample = "The quick brown fox jumps over the lazy dog 0123456789"

func main() {
	dir, order := os.Args[1], os.Args[2]
	files := []struct{ file, family string }{
		{"selawksl.ttf", "Selawik Semilight"},
		{"selawkl.ttf", "Selawik"},
		{"selawk.ttf", "Selawik"},
	}
	if order == "light" {
		files[0], files[1], files[2] = files[1], files[2], files[0]
	}
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f.file))
		if err != nil {
			panic(err)
		}
		if err := ui.RegisterFont(data, f.family); err != nil {
			panic(err)
		}
		fmt.Printf("registered %s as %q\n", f.file, f.family)
	}
	width := func(family string, weight int) float32 {
		w, _ := ui.ShapeText(sample, ui.Font{Family: family, Size: 40, Weight: weight}, 0).Size()
		return w
	}
	for _, fam := range []string{"Selawik Semilight", "Selawik"} {
		for _, wt := range []int{300, 350, 400} {
			fmt.Printf("width %q %d = %.2f\n", fam, wt, width(fam, wt))
		}
	}
	for _, f := range ui.FontFamilies() {
		if f.Registered {
			fmt.Printf("family %q registered\n", f.Name)
		}
	}
	if len(os.Args) < 4 {
		return
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title: "Font check", Width: 900, Height: 520,
			Content: ui.View(func(c *ui.Context) {
				ui.Column(c).Fill().Padding(16).Gap(8).Children(func() {
					ui.Text(c, "Selawik Semilight 300: "+sample).Font("Selawik Semilight").FontWeight(300).FontSize(20)
					ui.Text(c, "Selawik Semilight 400: "+sample).Font("Selawik Semilight").FontWeight(400).FontSize(20)
					ui.Text(c, "Selawik 300 (Light): "+sample).Font("Selawik").FontWeight(300).FontSize(20)
					ui.Text(c, "Selawik 400 (Regular): "+sample).Font("Selawik").FontWeight(400).FontSize(20)
					ui.Text(c, "Segoe UI 400: "+sample).Font("Segoe UI").FontSize(20)
					ui.Text(c, "Fallback list: Semilight Ωλ 中文 → ✓ Ж").Font("Selawik Semilight, Segoe UI").FontSize(28)
					ui.Text(c, "Semilight alone: Ωλ 中文 → ✓ Ж").Font("Selawik Semilight").FontSize(28)
				})
			}),
		})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
