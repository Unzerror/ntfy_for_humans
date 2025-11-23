// Package log provides a structured logging interface with support for multiple formats and levels.
package log

import (
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// Defaults for package level variables
var (
	DefaultLevel  = InfoLevel
	DefaultFormat = TextFormat
	DefaultOutput = &peekLogWriter{os.Stderr}
)

var (
	level               = DefaultLevel
	format              = DefaultFormat
	overrides           = make(map[string][]*levelOverride)
	output    io.Writer = DefaultOutput
	filename            = ""
	mu                  = &sync.RWMutex{}
)

// init sets the default log output (including log.SetOutput)
//
// This has to be explicitly called, because DefaultOutput is a peekLogWriter,
// which wraps os.Stderr.
func init() {
	SetOutput(DefaultOutput)
}

// Fatal prints the given message, and exits the program.
//
// Parameters:
//   - message: The message to log.
//   - v: Optional arguments for formatting the message.
func Fatal(message string, v ...any) {
	newEvent().Fatal(message, v...)
}

// Error prints the given message, if the current log level is ERROR or lower.
//
// Parameters:
//   - message: The message to log.
//   - v: Optional arguments for formatting the message.
func Error(message string, v ...any) {
	newEvent().Error(message, v...)
}

// Warn prints the given message, if the current log level is WARN or lower.
//
// Parameters:
//   - message: The message to log.
//   - v: Optional arguments for formatting the message.
func Warn(message string, v ...any) {
	newEvent().Warn(message, v...)
}

// Info prints the given message, if the current log level is INFO or lower.
//
// Parameters:
//   - message: The message to log.
//   - v: Optional arguments for formatting the message.
func Info(message string, v ...any) {
	newEvent().Info(message, v...)
}

// Debug prints the given message, if the current log level is DEBUG or lower.
//
// Parameters:
//   - message: The message to log.
//   - v: Optional arguments for formatting the message.
func Debug(message string, v ...any) {
	newEvent().Debug(message, v...)
}

// Trace prints the given message, if the current log level is TRACE.
//
// Parameters:
//   - message: The message to log.
//   - v: Optional arguments for formatting the message.
func Trace(message string, v ...any) {
	newEvent().Trace(message, v...)
}

// With creates a new log event and adds the fields of the given Contexter structs.
//
// Parameters:
//   - contexts: A list of objects implementing the Contexter interface.
//
// Returns:
//   - A new Event pointer.
func With(contexts ...Contexter) *Event {
	return newEvent().With(contexts...)
}

// Field creates a new log event and adds a custom field and value to it.
//
// Parameters:
//   - key: The field name.
//   - value: The field value.
//
// Returns:
//   - A new Event pointer.
func Field(key string, value any) *Event {
	return newEvent().Field(key, value)
}

// Fields creates a new log event and adds a map of fields to it.
//
// Parameters:
//   - fields: A map of fields to add.
//
// Returns:
//   - A new Event pointer.
func Fields(fields Context) *Event {
	return newEvent().Fields(fields)
}

// Tag creates a new log event and adds a "tag" field to it.
//
// Parameters:
//   - tag: The tag string.
//
// Returns:
//   - A new Event pointer.
func Tag(tag string) *Event {
	return newEvent().Tag(tag)
}

// Time creates a new log event and sets the time field.
//
// Parameters:
//   - time: The timestamp for the event.
//
// Returns:
//   - A new Event pointer.
func Time(time time.Time) *Event {
	return newEvent().Time(time)
}

// Timing runs f and records the time if took to execute it in "time_taken_ms".
//
// Parameters:
//   - f: The function to time.
//
// Returns:
//   - A new Event pointer.
func Timing(f func()) *Event {
	return newEvent().Timing(f)
}

// CurrentLevel returns the current log level.
//
// Returns:
//   - The current log level.
func CurrentLevel() Level {
	mu.RLock()
	defer mu.RUnlock()
	return level
}

// SetLevel sets a new log level.
//
// Parameters:
//   - newLevel: The new log level to set.
func SetLevel(newLevel Level) {
	mu.Lock()
	defer mu.Unlock()
	level = newLevel
}

// SetLevelOverride adds a log override for the given field.
//
// Parameters:
//   - field: The field name to match.
//   - value: The value to match.
//   - level: The log level to apply if the match succeeds.
func SetLevelOverride(field string, value string, level Level) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := overrides[field]; !ok {
		overrides[field] = make([]*levelOverride, 0)
	}
	overrides[field] = append(overrides[field], &levelOverride{value: value, level: level})
}

// ResetLevelOverrides removes all log level overrides.
func ResetLevelOverrides() {
	mu.Lock()
	defer mu.Unlock()
	overrides = make(map[string][]*levelOverride)
}

// CurrentFormat returns the current log format.
//
// Returns:
//   - The current log format.
func CurrentFormat() Format {
	mu.RLock()
	defer mu.RUnlock()
	return format
}

// SetFormat sets a new log format.
//
// Parameters:
//   - newFormat: The new format to set.
func SetFormat(newFormat Format) {
	mu.Lock()
	defer mu.Unlock()
	format = newFormat
	if newFormat == JSONFormat {
		DisableDates()
	}
}

// SetOutput sets the log output writer.
//
// Parameters:
//   - w: The io.Writer to write logs to.
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	output = &peekLogWriter{w}
	if f, ok := w.(*os.File); ok {
		filename = f.Name()
	} else {
		filename = ""
	}
	log.SetOutput(output)
}

// File returns the log file, if any, or an empty string otherwise.
//
// Returns:
//   - The filename of the log output, or empty string.
func File() string {
	mu.RLock()
	defer mu.RUnlock()
	return filename
}

// IsFile returns true if the output is a non-default file.
//
// Returns:
//   - True if logging to a file, false otherwise.
func IsFile() bool {
	mu.RLock()
	defer mu.RUnlock()
	return filename != ""
}

// DisableDates disables the date/time prefix.
func DisableDates() {
	log.SetFlags(0)
}

// Loggable returns true if the given log level is lower or equal to the current log level.
//
// Parameters:
//   - l: The level to check.
//
// Returns:
//   - True if the level is loggable.
func Loggable(l Level) bool {
	return CurrentLevel() <= l
}

// IsTrace returns true if the current log level is TraceLevel.
//
// Returns:
//   - True if logging trace level messages.
func IsTrace() bool {
	return Loggable(TraceLevel)
}

// IsDebug returns true if the current log level is DebugLevel or below.
//
// Returns:
//   - True if logging debug level messages.
func IsDebug() bool {
	return Loggable(DebugLevel)
}

// peekLogWriter is an io.Writer which will peek at the rendered log event,
// and ensure that the rendered output is valid JSON. This is a hack!
type peekLogWriter struct {
	w io.Writer
}

func (w *peekLogWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 || p[0] == '{' || CurrentFormat() == TextFormat {
		return w.w.Write(p)
	}
	m := newEvent().Tag(tagStdLog).Render(InfoLevel, "%s", strings.TrimSpace(string(p)))
	if m == "" {
		return 0, nil
	}
	return w.w.Write([]byte(m + "\n"))
}
