package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/kbinani/screenshot"
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
var targetDisplay int
var monitorCombo *widget.Select
var timeInput *widget.Entry
var outInput *widget.Entry
var startstop *widget.Button
var infoText *widget.RichText
var areaX1, areaY1, areaX2, areaY2 *widget.Entry
var frames int
var bytes int64
var recording bool
var stopChan chan struct{}

func main() {
	a = app.NewWithID("net.kettek.gosh")

	normalIcon = fyne.NewStaticResource("gosh", iconBytes)
	recordIcon = fyne.NewStaticResource("gosh", iconRecordBytes)

	window = a.NewWindow("gosh")
	window.Resize(fyne.NewSize(500, 200))
	stopChan = make(chan struct{})

	window.SetCloseIntercept(func() {
		if !recording {
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
				startStop()
			}),
		)
		desk.SetSystemTrayMenu(systrayMenu)
		desk.SetSystemTrayIcon(normalIcon)
	}

	monitorCombo = widget.NewSelect([]string{"aeg"}, func(value string) {
		parts := strings.Split(value, ":")
		targetDisplay, _ = strconv.Atoi(parts[0])
		otherParts := strings.Split(parts[1], "x")
		x1, _ := strconv.Atoi(strings.TrimSpace(otherParts[0]))
		y1, _ := strconv.Atoi(strings.TrimSpace(otherParts[1]))
		x2, _ := strconv.Atoi(strings.TrimSpace(otherParts[2]))
		y2, _ := strconv.Atoi(strings.TrimSpace(otherParts[3]))
		setArea(x1, y1, x2, y2)
		a.Preferences().SetInt("recordDisplay", targetDisplay)
	})

	refreshButton := widget.NewButton("", func() {
		refreshDisplays()
	})
	refreshButton.Icon = theme.ViewRefreshIcon()

	monitorLabel := widget.NewLabel("Monitor")
	monitorLabelContainer := container.NewGridWrap(fyne.NewSize(150, 0), monitorLabel)

	areaLabel := widget.NewLabel("Area")
	areaLabelContainer := container.NewGridWrap(fyne.NewSize(150, 0), areaLabel)
	makeNumberEntry := func() *widget.Entry {
		e := widget.NewEntry()
		e.SetText("0")
		e.Validator = func(s string) error {
			if _, err := strconv.Atoi(s); err != nil {
				return err
			}
			return nil
		}
		return e
	}
	areaX1 = makeNumberEntry()
	areaY1 = makeNumberEntry()
	areaX2 = makeNumberEntry()
	areaY2 = makeNumberEntry()

	refreshDisplays()
	if a.Preferences().Int("recordDisplay") >= len(monitorCombo.Options) {
		monitorCombo.SetSelectedIndex(len(monitorCombo.Options) - 1)
	} else {
		monitorCombo.SetSelectedIndex(a.Preferences().IntWithFallback("recordDisplay", 0))
	}

	timeLabel := widget.NewLabel("Frequency (seconds)")
	timeLabelContainer := container.NewGridWrap(fyne.NewSize(150, 0), timeLabel)
	timeInput = widget.NewEntry()
	timeInput.Validator = func(s string) error {
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return err
		}
		return nil
	}
	timeInput.SetText(a.Preferences().StringWithFallback("recordFrequency", "5"))
	timeInput.OnChanged = func(s string) {
		a.Preferences().SetString("recordFrequency", s)
	}

	outLabel := widget.NewLabel("Output directory")
	outLabelContainer := container.NewGridWrap(fyne.NewSize(150, 0), outLabel)
	outInput = widget.NewEntry()
	outInput.SetText(a.Preferences().StringWithFallback("recordOutput", ""))
	outInput.OnChanged = func(s string) {
		a.Preferences().SetString("recordOutput", s)
	}
	outFolderOpen := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			log.Println("Error opening folder", err)
		} else if uri != nil {
			outInput.SetText(uri.Path())
		}
	}, window)
	outFolderOpen.SetConfirmText("Select")
	outButton := widget.NewButton("", func() {
		outFolderOpen.Show()
	})
	outButton.Icon = theme.FolderOpenIcon()
	openButton := widget.NewButton("", func() {
		p, err := filepath.Abs(outInput.Text)
		if err != nil {
			log.Println("Error getting absolute path", err)
			return
		}
		if err := openPath(p); err != nil {
			log.Println("Error opening path", err)
		}
	})
	openButton.Icon = theme.MailForwardIcon()

	startstop = widget.NewButton("", func() {
		startStop()
	})
	startstop.Icon = theme.MediaRecordIcon()

	infoText = widget.NewRichText()
	refreshInfo()

	recordSection := container.NewVBox(
		container.NewBorder(nil, nil, monitorLabelContainer, nil,
			container.NewBorder(nil, nil, nil, refreshButton, monitorCombo),
		),
		container.NewBorder(nil, nil, areaLabelContainer, nil,
			container.NewAdaptiveGrid(4, areaX1, areaY1, areaX2, areaY2),
		),
		container.NewBorder(nil, nil, timeLabelContainer, nil, timeInput),
		container.NewBorder(nil, nil, outLabelContainer, nil,
			container.NewBorder(nil, nil, nil, container.NewAdaptiveGrid(2, outButton, openButton), outInput),
		),
		container.NewCenter(startstop),
		container.NewCenter(infoText),
	)

	window.SetContent(recordSection)

	window.ShowAndRun()
}

func startStop() {
	if recording {
		stop()
		recording = false
		startstop.SetIcon(theme.MediaRecordIcon())
		if desk, ok := a.(desktop.App); ok {
			desk.SetSystemTrayIcon(normalIcon)
			systrayMenu.Items[1].Label = "Record"
			systrayMenu.Refresh()
		}
	} else {
		recording = true
		startstop.SetIcon(theme.MediaStopIcon())
		if desk, ok := a.(desktop.App); ok {
			desk.SetSystemTrayIcon(recordIcon)
			systrayMenu.Items[1].Label = "Stop"
			systrayMenu.Refresh()
		}
		start()
	}
}

func refreshDisplays() {
	n := screenshot.NumActiveDisplays()

	var displayNames []string
	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		displayNames = append(displayNames, fmt.Sprintf("%d: %dx%dx%dx%d", i, bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy()))
	}
	monitorCombo.Options = displayNames
}

func setArea(x1, y1, x2, y2 int) {
	areaX1.SetText(strconv.Itoa(x1))
	areaY1.SetText(strconv.Itoa(y1))
	areaX2.SetText(strconv.Itoa(x2))
	areaY2.SetText(strconv.Itoa(y2))
}

func start() {
	x1, _ := strconv.ParseInt(areaX1.Text, 10, 64)
	y1, _ := strconv.ParseInt(areaY1.Text, 10, 64)
	x2, _ := strconv.ParseInt(areaX2.Text, 10, 64)
	y2, _ := strconv.ParseInt(areaY2.Text, 10, 64)
	bytes = 0
	frames = 0
	refreshInfo()
	seconds, err := strconv.ParseFloat(timeInput.Text, 64)
	if err != nil {
		log.Println("Error parsing time", err)
		return
	}
	t := time.Duration(seconds * 1000)

	out := outInput.Text

	go func() {
		for {
			select {
			case <-stopChan:
				return
			case <-time.After(time.Millisecond * t):
				img, err := screenshot.CaptureRect(image.Rect(int(x1), int(y1), int(x1+x2), int(y1+y2)))
				if err != nil {
					panic(err)
				}

				p := filepath.Join(out, fmt.Sprintf("%d.png", time.Now().UnixMilli()))
				f, err := os.Create(p)
				if err != nil {
					panic(err)
				}
				png.Encode(f, img)
				s, err := f.Stat()
				if err != nil {
					panic(err)
				}
				f.Close()
				bytes += s.Size()
				frames++
				refreshInfo()
			}
		}
	}()
}

func stop() {
	stopChan <- struct{}{}
}

func refreshInfo() {
	infoText.ParseMarkdown(fmt.Sprintf("**%d** frames\n\n**%.2f** MB", frames, float64(bytes)/1024/1024))
}
