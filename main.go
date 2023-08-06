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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/kbinani/screenshot"
)

var targetDisplay int
var combo *widget.Select
var timeInput *widget.Entry
var outInput *widget.Entry
var startstop *widget.Button
var recording bool
var stopChan chan struct{}

func main() {
	a := app.New()
	w := a.NewWindow("gosh")
	stopChan = make(chan struct{})

	combo = widget.NewSelect([]string{"aeg"}, func(value string) {
		parts := strings.Split(value, ":")
		targetDisplay, _ = strconv.Atoi(parts[0])
		log.Println("Select set to", targetDisplay)
	})

	refreshDisplays()
	label := widget.NewLabel("Select a Monitorus")

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
		outInput.SetText(uri.Path())
	}, w)
	outButton := widget.NewButton("...", func() {
		outFolderOpen.Show()
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

	w.SetContent(container.NewVBox(
		label,
		combo,
		timeLabel,
		timeInput,
		outLabel,
		container.New(layout.NewGridLayout(2), outInput, outButton),
		startstop,
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
				f.Close()
			}
		}
	}()
}

func stop() {
	stopChan <- struct{}{}
}
