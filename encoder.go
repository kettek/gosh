package main

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/kettek/apng"
)

type backend int

const (
	backendFFMPEG backend = iota
	backendImageMagick
	backendIntegrated
)

type encoder struct {
	container *fyne.Container

	typeCombo     *widget.Select
	fpsInput      *widget.Entry
	inputDirInput *widget.Entry
	outFileInput  *widget.Entry
	toggleButton  *widget.Button
	encodeInfo    *widget.TextGrid

	backend    backend
	outputPath string
}

func (e *encoder) setup(backend backend) {
	e.backend = backend

	var types []string
	switch backend {
	case backendFFMPEG:
		if aSettings.getFFMPEGPath() != "" {
			types = append(types, "webm", "png", "gif", "mp4")
		}
	case backendImageMagick:
		if aSettings.getConvertPath() != "" {
			types = append(types, "gif")
		}
		if aSettings.getMagickPath() != "" {
			types = append(types, "png")
		}
	case backendIntegrated:
		types = append(types, "png")
	}

	// Type
	typeLabel := widget.NewLabel("Type")
	e.typeCombo = widget.NewSelect(types, func(value string) {
		e.outFileInput.SetText(e.outputPath + "." + e.typeCombo.Selected)
	})

	// fps
	fpsLabel := widget.NewLabel("FPS")
	e.fpsInput = widget.NewEntry()
	e.fpsInput.SetText(fmt.Sprintf("%f", 5.0))
	e.fpsInput.Validator = func(s string) error {
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return err
		}
		return nil
	}

	// Input directory
	inputDirLabel := widget.NewLabel("Input directory")
	e.inputDirInput = widget.NewEntry()
	inputDirFolderOpen := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			log.Println("Error opening folder", err)
		} else if uri != nil {
			e.inputDirInput.SetText(uri.Path())
		}
	}, window)
	inputDirFolderOpen.SetConfirmText("Select")
	inputDirFolderButton := widget.NewButton("", func() {
		inputDirFolderOpen.Show()
	})
	inputDirFolderButton.Icon = theme.FolderOpenIcon()
	inputDirOpenButton := widget.NewButton("", func() {
		p, err := filepath.Abs(e.inputDirInput.Text)
		if err != nil {
			log.Println("Error getting absolute path", err)
			return
		}
		if err := openPath(p); err != nil {
			log.Println("Error opening path", err)
		}
	})
	inputDirOpenButton.Icon = theme.MailForwardIcon()

	// Output file
	outFileLabel := widget.NewLabel("Output file")
	e.outFileInput = widget.NewEntry()
	e.outFileInput.Disable()
	outFileSave := dialog.NewFileSave(func(uri fyne.URIWriteCloser, err error) {
		if err != nil {
			log.Println("Error opening file", err)
			return
		} else if uri == nil {
			return
		}
		// This is dumb, but we just want the path, we don't want to write it ourself...
		uri.Close()
		os.Remove(uri.URI().Path())
		e.outputPath = uri.URI().Path()
		// Strip out the extension
		e.outputPath = strings.TrimSuffix(e.outputPath, filepath.Ext(e.outputPath))
		e.outFileInput.SetText(e.outputPath + "." + e.typeCombo.Selected)
	}, window)
	outButton := widget.NewButton("", func() {
		outFileSave.Show()
	})
	outButton.Icon = theme.FileIcon()
	openButton := widget.NewButton("", func() {
		p, err := filepath.Abs(filepath.Dir(e.outFileInput.Text))
		if err != nil {
			log.Println("Error getting absolute path", err)
			return
		}
		if err := openPath(p); err != nil {
			log.Println("Error opening path", err)
		}
	})
	openButton.Icon = theme.MailForwardIcon()

	e.toggleButton = widget.NewButton("", func() {
		e.toggle()
	})
	e.toggleButton.Icon = theme.MediaPlayIcon()

	e.typeCombo.SetSelectedIndex(0)
	e.encodeInfo = widget.NewTextGridFromString("...")

	e.container = container.NewVBox(
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), typeLabel), nil, e.typeCombo),
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), fpsLabel), nil, e.fpsInput),
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), inputDirLabel), nil,
			container.NewBorder(nil, nil, nil, container.NewAdaptiveGrid(2, inputDirFolderButton, inputDirOpenButton), e.inputDirInput),
		),
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), outFileLabel), nil,
			container.NewBorder(nil, nil, nil, container.NewAdaptiveGrid(2, outButton, openButton), e.outFileInput),
		),
		container.NewCenter(e.toggleButton),
		container.NewCenter(e.encodeInfo),
	)
}

func (e *encoder) toggle() {
	inpath := e.inputDirInput.Text
	outpath := e.outputPath
	kind := e.typeCombo.Selected

	e.encodeTo(inpath, outpath, kind)
}

func (e *encoder) encodeTo(inpath, outpath, kind string) {
	e.toggleButton.Icon = theme.MediaStopIcon()
	var args []string
	files, err := getPNGs(inpath)
	if err != nil {
		e.encodeInfo.SetText(err.Error())
		e.toggleButton.Icon = theme.MediaPlayIcon()
		return
	}

	switch e.backend {
	case backendFFMPEG:
		args = append(args, "-y")

		args = append(args, "-framerate", e.fpsInput.Text)

		args = append(args, "-i", "concat:"+strings.Join(files, "|"))

		if kind == "webm" {
			args = append(args, "-c:v", "libvpx")
			args = append(args, "-b:v", "2M")
			args = append(args, "-crf", "10")
			args = append(args, "-f", "webm")
		} else if kind == "gif" {
			args = append(args, "-filter_complex", "split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse")
			args = append(args, "-f", "gif")
		} else if kind == "mp4" {
			args = append(args, "-c:v", "libx264")
			args = append(args, "-crf", "0")
			args = append(args, "-preset", "veryslow")
			args = append(args, "-f", "mp4")
		} else if kind == "png" {
			args = append(args, "-f", "apng")
		}
		args = append(args, outpath+"."+kind)

		e.runCmd(aSettings.getFFMPEGPath(), inpath, args)
	case backendImageMagick:
		cmdPath := aSettings.getConvertPath()
		fr, _ := strconv.ParseFloat(e.fpsInput.Text, 64)

		// convert fps to imagemagick delay:
		args = append(args, "-delay", strconv.Itoa(int(100/fr)))

		args = append(args, "-loop", "0")
		args = append(args, files...)

		if kind == "gif" {
			args = append(args, outpath+"."+kind)
		} else if kind == "png" {
			cmdPath = aSettings.getMagickPath()
			args = append(args, "APNG:"+outpath+"."+kind)
		}

		e.runCmd(cmdPath, inpath, args)
	case backendIntegrated:
		e.encodeInfo.SetText("processing...")
		fr, _ := strconv.ParseFloat(e.fpsInput.Text, 64)
		a := apng.APNG{
			Frames: make([]apng.Frame, len(files)),
		}
		out, err := os.Create(outpath + "." + kind)
		if err != nil {
			e.encodeInfo.SetText(err.Error())
			return
		}
		defer out.Close()
		for i, s := range files {
			in, err := os.Open(s)
			if err != nil {
				e.encodeInfo.SetText(err.Error())
				return
			}
			defer in.Close()
			m, err := png.Decode(in)
			if err != nil {
				e.encodeInfo.SetText(err.Error())
				return
			}
			a.Frames[i].Image = m
			a.Frames[i].DelayDenominator = 100
			a.Frames[i].DelayNumerator = uint16(100 / fr)
		}
		if err := apng.Encode(out, a); err != nil {
			e.encodeInfo.SetText(err.Error())
		}
		e.encodeInfo.SetText("complete")
	}
	e.toggleButton.Icon = theme.MediaPlayIcon()
}

func (e *encoder) runCmd(binPath string, cwd string, args []string) {
	cmd := exec.Command(binPath, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir, _ = filepath.Abs(cwd)

	e.encodeInfo.SetText("processing...")
	if err := cmd.Start(); err != nil {
		e.encodeInfo.SetText(err.Error())
		return
	} else {
		if err := cmd.Wait(); err != nil {
			e.encodeInfo.SetText(err.Error())
			return
		}
	}
	e.encodeInfo.SetText("complete")
}

func getPNGs(p string) (files []string, err error) {
	d, err := os.ReadDir(p)
	if err != nil {
		return files, err
	}

	for _, e := range d {
		if e.IsDir() {
			continue
		}
		if e.Name()[0] == 's' {
			continue
		}
		if strings.HasSuffix(e.Name(), ".png") {
			files = append(files, filepath.Join(p, e.Name()))
		}
	}
	return
}
