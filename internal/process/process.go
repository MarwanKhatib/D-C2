// Package process contains process-level helpers shared by the message and
// interaction handlers.
package process

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

// Restart re-execs the current binary and returns only if the restart fails.
func Restart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command(exe, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Start(); err != nil {
			return err
		}
		os.Exit(0)
		return nil
	}

	return syscall.Exec(exe, os.Args, os.Environ())
}
