package pocketbook

import (
	"os"
	"os/exec"
	"syscall"
)

// None - No icon
const None = "0"

// Info Icon
const Info = "1"

// Question Icon
const Question = "2"

// Attention Icon
const Attention = "3"

// X Icon
const X = "4"

// WLAN Icon
const WLAN = "5"

const dialogPath = "/ebrmain/bin/dialog"

// Dialog - Call PocketBooks's dialog utility
// Parameters: icon, text, buttons (0..3)
func Dialog(icon string, text string, buttons ...string) int {
	cmd := &exec.Cmd{
		Path: dialogPath,
		Args: append(
			[]string{
				dialogPath,
				icon,
				"",
				text},
			buttons...,
		),
		Stdout: os.Stdout,
		Stdin:  os.Stdin,
	}
	return displayDialog(cmd)
}

// Fatal should be used in log.Fatal:
// log.Fatal( Fatal(text) )
// will display an alert and then return the text for logging
func Fatal(text string) string {
	cmd := &exec.Cmd{
		Path: dialogPath,
		Args: []string{
			dialogPath,
			Attention,
			"",
			text,
			"OK",
		},
		Stdout: os.Stdout,
		Stdin:  os.Stdin,
	}
	_ = displayDialog(cmd)
	return text
}

func displayDialog(cmd *exec.Cmd) int {
	var waitStatus syscall.WaitStatus
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			return waitStatus.ExitStatus()
		}
	}
	return 0
}
