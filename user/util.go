package user

import (
	"golang.org/x/crypto/bcrypt"
	"heckel.io/ntfy/v2/util"
	"regexp"
	"strings"
)

var (
	allowedUsernameRegex     = regexp.MustCompile(`^[-_.+@a-zA-Z0-9]+$`)    // Does not include Everyone (*)
	allowedTopicRegex        = regexp.MustCompile(`^[-_A-Za-z0-9]{1,64}$`)  // No '*'
	allowedTopicPatternRegex = regexp.MustCompile(`^[-_*A-Za-z0-9]{1,64}$`) // Adds '*' for wildcards!
	allowedTierRegex         = regexp.MustCompile(`^[-_A-Za-z0-9]{1,64}$`)
	allowedTokenRegex        = regexp.MustCompile(`^tk_[-_A-Za-z0-9]{29}$`) // Must be tokenLength-len(tokenPrefix)
)

// AllowedRole returns true if the given role can be used for new users.
//
// Parameters:
//   - role: The role to check.
//
// Returns:
//   - True if the role is valid.
func AllowedRole(role Role) bool {
	return role == RoleUser || role == RoleAdmin
}

// AllowedUsername returns true if the given username is valid.
//
// Parameters:
//   - username: The username to check.
//
// Returns:
//   - True if the username is valid.
func AllowedUsername(username string) bool {
	return allowedUsernameRegex.MatchString(username)
}

// AllowedTopic returns true if the given topic name is valid.
//
// Parameters:
//   - topic: The topic to check.
//
// Returns:
//   - True if the topic is valid.
func AllowedTopic(topic string) bool {
	return allowedTopicRegex.MatchString(topic)
}

// AllowedTopicPattern returns true if the given topic pattern is valid; this includes the wildcard character (*).
//
// Parameters:
//   - topic: The topic pattern to check.
//
// Returns:
//   - True if the topic pattern is valid.
func AllowedTopicPattern(topic string) bool {
	return allowedTopicPatternRegex.MatchString(topic)
}

// AllowedTier returns true if the given tier name is valid.
//
// Parameters:
//   - tier: The tier to check.
//
// Returns:
//   - True if the tier is valid.
func AllowedTier(tier string) bool {
	return allowedTierRegex.MatchString(tier)
}

// ValidPasswordHash checks if the given password hash is a valid bcrypt hash.
//
// Parameters:
//   - hash: The hash string to check.
//   - minCost: The minimum bcrypt cost.
//
// Returns:
//   - An error if the hash is invalid or too weak.
func ValidPasswordHash(hash string, minCost int) error {
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
		return ErrPasswordHashInvalid
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil { // Check if the hash is valid (length, format, etc.)
		return err
	} else if cost < minCost {
		return ErrPasswordHashWeak
	}
	return nil
}

// ValidToken returns true if the given token matches the naming convention.
//
// Parameters:
//   - token: The token string to check.
//
// Returns:
//   - True if the token is valid.
func ValidToken(token string) bool {
	return allowedTokenRegex.MatchString(token)
}

// GenerateToken generates a new token with a prefix and a fixed length.
// Lowercase only to support "<topic>+<token>@<domain>" email addresses.
//
// Returns:
//   - A new random token string.
func GenerateToken() string {
	return util.RandomLowerStringPrefix(tokenPrefix, tokenLength)
}

// HashPassword hashes the given password using bcrypt with the configured cost.
//
// Parameters:
//   - password: The password to hash.
//
// Returns:
//   - The hashed password or an error.
func HashPassword(password string) (string, error) {
	return hashPassword(password, DefaultUserPasswordBcryptCost)
}

func hashPassword(password string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
