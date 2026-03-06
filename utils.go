package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// messages
func showError(message ...string) {
	fmt.Println(CliTheme.Focused.ErrorMessage.Render("✕ " + strings.Join(message, " ")))
}

func showTitle(message ...string) {
	fmt.Println(CliTheme.Focused.Title.Render(strings.Join(message, " ")))
}

func showSuccess(message ...string) {
	// what is go even about :(
	showTitle(append([]string{"✓"}, message...)...)
}

// os/exec
func hasExecutable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func execAtCwd(dir string, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Run()
}
