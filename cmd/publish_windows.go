package cmd

import (
	"os"
)

// processExists checks if a process with the given PID exists.
// It uses os.FindProcess to check for the process.
//
// Parameters:
//   - pid: The PID to check.
//
// Returns:
//   - True if the process exists, false otherwise.
func processExists(pid int) bool {
	_, err := os.FindProcess(pid)
	return err == nil
}
