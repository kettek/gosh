package main

import (
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type settings struct {
	container             *fyne.Container
	backendsCombo         *widget.Select
	discoveredFFMPEGPath  string
	discoveredConvertPath string
	discoveredMagickPath  string
	currentFFMPEGPath     string
	currentConvertPath    string
	currentMagickPath     string
}

func (s *settings) setup() {
	setup := false
	backendsLabel := widget.NewLabel("Backend")
	s.backendsCombo = widget.NewSelect([]string{"auto", "ffmpeg", "imagemagick", "apng"}, func(value string) {
		a.Preferences().SetString("backend", value)
		if setup {
			refreshBackend()
		}
	})
	s.backendsCombo.SetSelected(a.Preferences().StringWithFallback("backend", "auto"))

	ffmpegPathLabel := widget.NewLabel("ffmpeg path")
	ffmpegPathInput := widget.NewEntry()
	s.currentFFMPEGPath = a.Preferences().String("ffmpegPath")
	ffmpegPathInput.SetText(s.currentFFMPEGPath)
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		s.discoveredFFMPEGPath = p
		ffmpegPathInput.SetPlaceHolder(p)
	}
	ffmpegPathInput.OnChanged = func(value string) {
		a.Preferences().SetString("ffmpegPath", value)
		s.currentFFMPEGPath = value
		refreshBackend()
	}
	ffmpegPathFileOpen := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
		if err != nil {
			return
		} else if uc == nil {
			return
		}
		uc.Close()
		ffmpegPathInput.SetText(uc.URI().Path())
	}, window)
	ffmpegPathButton := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		ffmpegPathFileOpen.Show()
	})
	ffmpegRefreshButton := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if p, err := exec.LookPath("ffmpeg"); err == nil {
			s.discoveredFFMPEGPath = p
			ffmpegPathInput.SetPlaceHolder(p)
		}
	})

	convertPathLabel := widget.NewLabel("convert path")
	convertPathInput := widget.NewEntry()
	s.currentConvertPath = a.Preferences().String("convertPath")
	convertPathInput.SetText(a.Preferences().StringWithFallback("convertPath", ""))
	if p, err := exec.LookPath("convert"); err == nil {
		s.discoveredConvertPath = p
		convertPathInput.SetPlaceHolder(p)
	}
	convertPathInput.OnChanged = func(value string) {
		a.Preferences().SetString("convertPath", value)
		s.currentConvertPath = value
		refreshBackend()
	}
	convertPathFileOpen := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
		if err != nil {
			return
		} else if uc == nil {
			return
		}
		uc.Close()
		convertPathInput.SetText(uc.URI().Path())
	}, window)
	convertPathButton := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		convertPathFileOpen.Show()
	})
	convertRefreshButton := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if p, err := exec.LookPath("convert"); err == nil {
			s.discoveredConvertPath = p
			convertPathInput.SetPlaceHolder(p)
		}
	})

	magickPathLabel := widget.NewLabel("magick path")
	magickPathInput := widget.NewEntry()
	s.currentMagickPath = a.Preferences().String("magickPath")
	magickPathInput.SetText(a.Preferences().StringWithFallback("magickPath", ""))
	if p, err := exec.LookPath("magick"); err == nil {
		s.discoveredMagickPath = p
		magickPathInput.SetPlaceHolder(p)
	}
	magickPathInput.OnChanged = func(value string) {
		a.Preferences().SetString("magickPath", value)
		s.currentMagickPath = value
		refreshBackend()
	}
	magickPathFileOpen := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
		if err != nil {
			return
		} else if uc == nil {
			return
		}
		uc.Close()
		magickPathInput.SetText(uc.URI().Path())
	}, window)
	magickPathButton := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		magickPathFileOpen.Show()
	})
	magickRefreshButton := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if p, err := exec.LookPath("magick"); err == nil {
			s.discoveredMagickPath = p
			magickPathInput.SetPlaceHolder(p)
		}
	})

	s.container = container.NewVBox(
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), backendsLabel), nil,
			container.NewBorder(nil, nil, nil, nil, s.backendsCombo),
		),
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), ffmpegPathLabel), nil,
			container.NewBorder(nil, nil, nil, container.NewAdaptiveGrid(2, ffmpegPathButton, ffmpegRefreshButton), ffmpegPathInput),
		),
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), convertPathLabel), nil,
			container.NewBorder(nil, nil, nil, container.NewAdaptiveGrid(2, convertPathButton, convertRefreshButton), convertPathInput),
		),
		container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(150, 0), magickPathLabel), nil,
			container.NewBorder(nil, nil, nil, container.NewAdaptiveGrid(2, magickPathButton, magickRefreshButton), magickPathInput),
		),
	)
	setup = true
}

func (s *settings) getFFMPEGPath() string {
	if s.currentFFMPEGPath == "" {
		return s.discoveredFFMPEGPath
	}
	return s.currentFFMPEGPath
}

func (s *settings) getConvertPath() string {
	if s.currentConvertPath == "" {
		return s.discoveredConvertPath
	}
	return s.currentConvertPath
}

func (s *settings) getMagickPath() string {
	if s.currentMagickPath == "" {
		return s.discoveredMagickPath
	}
	return s.currentMagickPath
}
