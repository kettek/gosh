package main

import (
	"os/exec"
)

func openPath(path string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", path)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return nil
}
