//go:build darwin || linux || dragonfly || freebsd || netbsd || openbsd
// +build darwin linux dragonfly freebsd netbsd openbsd

package cmd

import "syscall"

// processExists checks if a process with the given PID exists.
// It sends a signal 0 to the process, which does not affect the process
// but allows checking for its existence.
//
// Parameters:
//   - pid: The PID to check.
//
// Returns:
//   - True if the process exists, false otherwise.
func processExists(pid int) bool {
	err := syscall.Kill(pid, syscall.Signal(0))
	return err == nil
}
