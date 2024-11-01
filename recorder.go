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

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/kbinani/screenshot"
)

type recorder struct {
	container                      *fyne.Container
	displaysCombo                  *widget.Select
	frequencyInput                 *widget.Entry
	outInput                       *widget.Entry
	toggleButton                   *widget.Button
	infoText                       *widget.RichText
	areaX1, areaY1, areaX2, areaY2 *widget.Entry

	targetDisplay int

	recording bool
	stopChan  chan struct{}

	writtenFrames int
	writtenBytes  int64
}

func (r *recorder) setup() {
	r.stopChan = make(chan struct{})

	// Displays
	displaysLabel := widget.NewLabel("Display")

	r.displaysCombo = widget.NewSelect([]string{"aeg"}, func(value string) {
		parts := strings.Split(value, ":")
		r.targetDisplay, _ = strconv.Atoi(parts[0])
		otherParts := strings.Split(parts[1], "x")
		x1, _ := strconv.Atoi(strings.TrimSpace(otherParts[0]))
		y1, _ := strconv.Atoi(strings.TrimSpace(otherParts[1]))
		x2, _ := strconv.Atoi(strings.TrimSpace(otherParts[2]))
		y2, _ := strconv.Atoi(strings.TrimSpace(otherParts[3]))
		r.setArea(x1, y1, x2, y2)
		a.Preferences().SetInt("recordDisplay", r.targetDisplay)
	})

	refreshDisplaysButton := widget.NewButton("", func() {
		r.refreshDisplays()
	})
	refreshDisplaysButton.Icon = theme.ViewRefreshIcon()

	// Area
	areaLabel := widget.NewLabel("Area")

	r.areaX1 = makeNumberEntry(0)
	r.areaY1 = makeNumberEntry(0)
	r.areaX2 = makeNumberEntry(0)
	r.areaY2 = makeNumberEntry(0)

	// Frequency
	frequencyLabel := widget.NewLabel("Frequency (seconds)")

	r.frequencyInput = widget.NewEntry()
	r.frequencyInput.Validator = func(s string) error {
		_, err := strconv.ParseFloat(s, 64)
		return err
	}
	r.frequencyInput.SetText(a.Preferences().StringWithFallback("recordFrequency", "5"))
	r.frequencyInput.OnChanged = func(s string) {
		a.Preferences().SetString("recordFrequency", s)
	}

	// Output
	outLabel := widget.NewLabel("Output directory")

	r.outInput = widget.NewEntry()
	r.outInput.SetText(a.Preferences().StringWithFallback("recordOutput", os.TempDir()))
	r.outInput.OnChanged = func(s string) {
		a.Preferences().SetString("recordOutput", s)
	}

	outFolderOpen := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			log.Println(err)
			return
		}
		if uri == nil {
			return
		}
		r.outInput.SetText(uri.Path())
	}, window)
	outFolderOpen.SetConfirmText("Select")

	outButton := widget.NewButton("", func() {
		outFolderOpen.Show()
	})
	outButton.Icon = theme.FolderOpenIcon()
	revealButton := widget.NewButton("", func() {
		p, err := filepath.Abs(r.outInput.Text)
		if err != nil {
			log.Println("Error getting absolute path", err)
			return
		}
		if err := openPath(p); err != nil {
			log.Println("Error opening path", err)
		}
	})
	revealButton.Icon = theme.MailForwardIcon()

	// Start/Stop
	r.toggleButton = widget.NewButton("", func() {
		r.startStop()
	})
	r.toggleButton.Icon = theme.MediaRecordIcon()

	r.infoText = widget.NewRichText()

	// Refresh/Sync state
	r.refreshDisplays()
	if a.Preferences().Int("recordDisplay") >= len(r.displaysCombo.Options) {
		r.displaysCombo.SetSelectedIndex(len(r.displaysCombo.Options) - 1)
	} else {
		r.displaysCombo.SetSelectedIndex(a.Preferences().IntWithFallback("recordDisplay", 0))
	}
	r.refreshInfo()

	// Setup container
	r.container = container.NewVBox(
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), displaysLabel), nil,
			container.NewBorder(nil, nil, nil, refreshDisplaysButton, r.displaysCombo),
		),
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), areaLabel), nil,
			container.NewAdaptiveGrid(4, r.areaX1, r.areaY1, r.areaX2, r.areaY2),
		),
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), frequencyLabel), nil, r.frequencyInput),
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), outLabel), nil,
			container.NewBorder(nil, nil, nil, container.NewAdaptiveGrid(2, outButton, revealButton), r.outInput),
		),
		container.NewCenter(r.toggleButton),
		container.NewCenter(r.infoText),
	)

	// Setup shortcuts.
	toggleShortcut := &desktop.CustomShortcut{KeyName: fyne.KeySpace, Modifier: fyne.KeyModifierControl}
	window.Canvas().AddShortcut(toggleShortcut, func(_ fyne.Shortcut) {
		r.startStop()
	})
}

func (r *recorder) refreshDisplays() {
	n := screenshot.NumActiveDisplays()

	var displayNames []string
	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		displayNames = append(displayNames, fmt.Sprintf("%d: %dx%dx%dx%d", i, bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy()))
	}
	r.displaysCombo.Options = displayNames
}

func (r *recorder) refreshInfo() {
	r.infoText.ParseMarkdown(fmt.Sprintf("**%d** frames\n\n**%.2f** MB", r.writtenFrames, float64(r.writtenBytes)/1024/1024))
}

func (r *recorder) setArea(x1, y1, x2, y2 int) {
	r.areaX1.SetText(strconv.Itoa(x1))
	r.areaY1.SetText(strconv.Itoa(y1))
	r.areaX2.SetText(strconv.Itoa(x2))
	r.areaY2.SetText(strconv.Itoa(y2))
}

func (r *recorder) startStop() {
	if r.recording {
		r.stop()
	} else {
		r.start()
	}
}

func (r *recorder) start() {
	r.recording = true
	r.toggleButton.SetIcon(theme.MediaStopIcon())
	if desk, ok := a.(desktop.App); ok {
		desk.SetSystemTrayIcon(recordIcon)
		systrayMenu.Items[1].Label = "Stop"
		systrayMenu.Refresh()
	}

	x1, _ := strconv.ParseInt(r.areaX1.Text, 10, 64)
	y1, _ := strconv.ParseInt(r.areaY1.Text, 10, 64)
	x2, _ := strconv.ParseInt(r.areaX2.Text, 10, 64)
	y2, _ := strconv.ParseInt(r.areaY2.Text, 10, 64)
	r.writtenBytes = 0
	r.writtenFrames = 0
	r.refreshInfo()
	seconds, err := strconv.ParseFloat(r.frequencyInput.Text, 64)
	if err != nil {
		log.Println("Error parsing time", err)
		return
	}
	t := time.Duration(seconds * 1000)

	out := r.outInput.Text

	go func() {
		for {
			select {
			case <-r.stopChan:
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
				r.writtenBytes += s.Size()
				r.writtenFrames++
				r.refreshInfo()
			}
		}
	}()
}

func (r *recorder) stop() {
	r.stopChan <- struct{}{}
	r.recording = false
	r.toggleButton.SetIcon(theme.MediaRecordIcon())
	if desk, ok := a.(desktop.App); ok {
		desk.SetSystemTrayIcon(normalIcon)
		systrayMenu.Items[1].Label = "Record"
		systrayMenu.Refresh()
	}
}
