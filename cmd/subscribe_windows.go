package cmd

const (
	scriptExt                      = "bat"
	scriptHeader                   = ""
	clientCommandDescriptionSuffix = `The default config file for all client commands is %AppData%\ntfy\client.yml.`
)

var (
	scriptLauncher = []string{"cmd.exe", "/Q", "/C"}
)

// defaultClientConfigFile determines the default configuration file path for Windows.
//
// Returns:
//   - The path to the config file or an error.
func defaultClientConfigFile() (string, error) {
	return defaultClientConfigFileWindows()
}
