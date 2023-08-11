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
var aRecorder recorder
var aEncoder encoder
var aSettings settings
var tabs *container.AppTabs
var window fyne.Window
var windowHidden bool
var systrayMenu *fyne.Menu

func makeNumberEntry(number int) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(fmt.Sprintf("%d", number))
	e.Validator = func(s string) error {
		if _, err := strconv.Atoi(s); err != nil {
			return err
		}
		return nil
	}
	return e
}

func main() {
	a = app.NewWithID("net.kettek.gosh")

	normalIcon = fyne.NewStaticResource("gosh", iconBytes)
	recordIcon = fyne.NewStaticResource("gosh", iconRecordBytes)

	window = a.NewWindow("gosh")
	window.Resize(fyne.NewSize(500, 200))

	window.SetCloseIntercept(func() {
		if !aRecorder.recording {
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
				aRecorder.startStop()
			}),
		)
		desk.SetSystemTrayMenu(systrayMenu)
		desk.SetSystemTrayIcon(normalIcon)
	}

	aRecorder.setup()
	aSettings.setup()

	tabs = container.NewAppTabs(
		container.NewTabItem("Record", container.NewPadded(aRecorder.container)),
		container.NewTabItem("Encode", container.NewPadded()),
		container.NewTabItem("Settings", container.NewPadded(aSettings.container)),
	)
	window.SetContent(tabs)

	refreshBackend()

	window.ShowAndRun()
}

func refreshBackend() {
	aEncoder = encoder{}

	if a.Preferences().String("backend") != "" && a.Preferences().String("backend") != "auto" {
		switch a.Preferences().String("backend") {
		case "ffmpeg":
			aEncoder.setup(backendFFMPEG)
		case "imagemagick":
			aEncoder.setup(backendImageMagick)
		case "apng":
			aEncoder.setup(backendIntegrated)
		}
	} else {
		if aSettings.discoveredFFMPEGPath != "" {
			aEncoder.setup(backendFFMPEG)
		} else if aSettings.discoveredConvertPath != "" || aSettings.discoveredMagickPath != "" {
			aEncoder.setup(backendImageMagick)
		} else {
			aEncoder.setup(backendIntegrated)
		}
	}
	tabs.Items[1].Content = container.NewPadded(aEncoder.container)
	tabs.Refresh()
}
