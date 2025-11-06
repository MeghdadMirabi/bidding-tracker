package utils

import (
	"os"

	log "github.com/sirupsen/logrus"
)

// init initializes the global logger configuration when the package is imported.
func init() {
	//set log formatter to JSON with ISO 8601 timestamps
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05Z07:00",
	})

	// Output to stdout
	log.SetOutput(os.Stdout)

	// Set default log level
	log.SetLevel(log.InfoLevel)
}

// Info logs a message at info level with optional fields
func Info(message string, fields map[string]any) {
	log.WithFields(fields).Info(message)
}

// Warn logs a message at warning level with optional fields
func Warn(message string, fields map[string]any) {
	log.WithFields(fields).Warn(message)
}

// Error logs a message at error level with optional fields
func Error(message string, fields map[string]any) {
	log.WithFields(fields).Error(message)
}

// Fatal logs a message at fatal level and exits the application
func Fatal(message string, fields map[string]any) {
	log.WithFields(fields).Fatal(message)
}
