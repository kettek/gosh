package main

import (
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/kbinani/screenshot"
)

var targetDisplay int
var combo *widget.Select
var timeInput *widget.Entry
var outInput *widget.Entry
var startstop *widget.Button
var infoText *widget.RichText
var frames int
var bytes int64
var recording bool
var stopChan chan struct{}

func main() {
	a := app.New()
	w := a.NewWindow("gosh")
	w.Resize(fyne.NewSize(600, 330))
	stopChan = make(chan struct{})

	combo = widget.NewSelect([]string{"aeg"}, func(value string) {
		parts := strings.Split(value, ":")
		targetDisplay, _ = strconv.Atoi(parts[0])
	})

	refreshButton := widget.NewButton("Refresh", func() {
		refreshDisplays()
	})

	refreshDisplays()
	combo.SetSelectedIndex(0)
	label := widget.NewLabel("Select a Monitor")

	timeLabel := widget.NewLabel("Time in seconds")
	timeInput = widget.NewEntry()
	timeInput.SetText("5.0")
	timeInput.Validator = func(s string) error {
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return err
		}
		return nil
	}

	outLabel := widget.NewLabel("Output directory")
	outInput = widget.NewEntry()
	outFolderOpen := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			log.Println("Error opening folder", err)
		} else if uri != nil {
			outInput.SetText(uri.Path())
		}
	}, w)
	outFolderOpen.SetConfirmText("Select")
	outButton := widget.NewButton("...", func() {
		outFolderOpen.Show()
	})
	openButton := widget.NewButton("Open", func() {
		p, err := filepath.Abs(outInput.Text)
		if err != nil {
			log.Println("Error getting absolute path", err)
			return
		}
		if err := openPath(p); err != nil {
			log.Println("Error opening path", err)
		}
	})

	startstop = widget.NewButton("Start", func() {
		if recording {
			stop()
			recording = false
			startstop.SetText("Start")
		} else {
			recording = true
			startstop.SetText("Stop")
			start()
		}
	})

	infoText = widget.NewRichText()
	refreshInfo()

	w.SetContent(container.NewVBox(
		label,
		container.NewBorder(nil, nil, nil, refreshButton, combo),
		timeLabel,
		timeInput,
		outLabel,
		container.NewBorder(nil, nil, nil, container.NewAdaptiveGrid(2, outButton, openButton), outInput),
		startstop,
		container.NewCenter(infoText),
	))

	w.ShowAndRun()
}

func refreshDisplays() {
	n := screenshot.NumActiveDisplays()

	var displayNames []string
	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		displayNames = append(displayNames, fmt.Sprintf("%d: %dx%dx%dx%d", i, bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy()))
	}
	combo.Options = displayNames
}

func start() {
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
				img, err := screenshot.CaptureRect(screenshot.GetDisplayBounds(targetDisplay))
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
