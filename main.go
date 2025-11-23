package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"heckel.io/ntfy/v2/cmd"
	"os"
	"runtime"
)

// version, commit, and date are set during the build process.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// main is the entry point of the ntfy application.
// It sets up the CLI application and executes it based on the provided arguments.
//
// The function handles:
//   - Setting up the help template with version and contact information.
//   - Initializing the CLI application using cmd.New().
//   - Running the application and handling any errors by printing to stderr and exiting with status 1.
func main() {
	cli.AppHelpTemplate += fmt.Sprintf(`
Try 'ntfy COMMAND --help' or https://ntfy.sh/docs/ for more information.

To report a bug, open an issue on GitHub: https://github.com/binwiederhier/ntfy/issues.
If you want to chat, simply join the Discord server (https://discord.gg/cT7ECsZj9w), or
the Matrix room (https://matrix.to/#/#ntfy:matrix.org).

ntfy %s (%s), runtime %s, built at %s
Copyright (C) Philipp C. Heckel, licensed under Apache License 2.0 & GPLv2
`, version, commit[:7], runtime.Version(), date)

	app := cmd.New()
	app.Version = version

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
