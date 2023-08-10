package main

import (
	"fmt"
	"strconv"

	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

//go:embed assets/gosh.png
var iconBytes []byte
var normalIcon *fyne.StaticResource

//go:embed assets/gosh-record.png
var iconRecordBytes []byte
var recordIcon *fyne.StaticResource

var a fyne.App
var window fyne.Window
var windowHidden bool
var systrayMenu *fyne.Menu

func main() {
	var recorder recorder

	a = app.NewWithID("net.kettek.gosh")

	normalIcon = fyne.NewStaticResource("gosh", iconBytes)
	recordIcon = fyne.NewStaticResource("gosh", iconRecordBytes)

	window = a.NewWindow("gosh")
	window.Resize(fyne.NewSize(500, 200))

	window.SetCloseIntercept(func() {
		if !recorder.recording {
			a.Quit()
			return
		}
		windowHidden = true
		window.Hide()
		systrayMenu.Items[0].Label = "Show"
		systrayMenu.Refresh()
	})

	if desk, ok := a.(desktop.App); ok {
		systrayMenu = fyne.NewMenu("gosh",
			fyne.NewMenuItem("Hide", func() {
				if windowHidden {
					window.Show()
					systrayMenu.Items[0].Label = "Hide"
				} else {
					window.Hide()
					systrayMenu.Items[0].Label = "Show"
				}
				systrayMenu.Refresh()
				windowHidden = !windowHidden
			}),
			fyne.NewMenuItem("Record", func() {
				recorder.startStop()
			}),
		)
		desk.SetSystemTrayMenu(systrayMenu)
		desk.SetSystemTrayIcon(normalIcon)
	}

	recorder.setup()

	window.SetContent(container.NewPadded(recorder.container))

	window.ShowAndRun()
}
