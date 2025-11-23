package cmd

const (
	scriptExt                      = "sh"
	scriptHeader                   = "#!/bin/sh\n"
	clientCommandDescriptionSuffix = `The default config file for all client commands is /etc/ntfy/client.yml (if root user),
or "~/Library/Application Support/ntfy/client.yml" for all other users.`
)

var (
	scriptLauncher = []string{"sh", "-c"}
)

// defaultClientConfigFile determines the default configuration file path for Darwin (macOS).
//
// Returns:
//   - The path to the config file or an error.
func defaultClientConfigFile() (string, error) {
	return defaultClientConfigFileUnix()
}
