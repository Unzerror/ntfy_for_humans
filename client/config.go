package client

import (
	"gopkg.in/yaml.v2"
	"heckel.io/ntfy/v2/log"
	"os"
)

const (
	// DefaultBaseURL is the base URL used to expand short topic names
	DefaultBaseURL = "https://ntfy.sh"
)

// Config is the config struct for a Client.
type Config struct {
	// DefaultHost is the default ntfy server to use.
	DefaultHost     string      `yaml:"default-host"`
	// DefaultUser is the default username for authentication.
	DefaultUser     string      `yaml:"default-user"`
	// DefaultPassword is the default password for authentication.
	DefaultPassword *string     `yaml:"default-password"`
	// DefaultToken is the default access token for authentication.
	DefaultToken    string      `yaml:"default-token"`
	// DefaultCommand is the default command to execute when a message is received.
	DefaultCommand  string      `yaml:"default-command"`
	// Subscribe is a list of topics to subscribe to.
	Subscribe       []Subscribe `yaml:"subscribe"`
}

// Subscribe is the struct for a Subscription within Config.
type Subscribe struct {
	// Topic is the topic to subscribe to.
	Topic    string            `yaml:"topic"`
	// User is the username for authentication for this specific topic.
	User     *string           `yaml:"user"`
	// Password is the password for authentication for this specific topic.
	Password *string           `yaml:"password"`
	// Token is the access token for authentication for this specific topic.
	Token    *string           `yaml:"token"`
	// Command is the command to execute when a message is received on this topic.
	Command  string            `yaml:"command"`
	// If is a map of conditions that must be met for the command to execute (not fully implemented in this struct definition but implied).
	If       map[string]string `yaml:"if"`
}

// NewConfig creates a new Config struct for a Client with default values.
//
// Returns:
//   - A new Config instance.
func NewConfig() *Config {
	return &Config{
		DefaultHost:     DefaultBaseURL,
		DefaultUser:     "",
		DefaultPassword: nil,
		DefaultToken:    "",
		DefaultCommand:  "",
		Subscribe:       nil,
	}
}

// LoadConfig loads the Client config from a yaml file.
//
// Parameters:
//   - filename: The path to the YAML configuration file.
//
// Returns:
//   - A Config instance populated from the file, or an error if loading failed.
func LoadConfig(filename string) (*Config, error) {
	log.Debug("Loading client config from %s", filename)
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	c := NewConfig()
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}
	return c, nil
}
