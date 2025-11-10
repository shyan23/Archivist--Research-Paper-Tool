package app

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// InitLogger initializes logging based on config
// Returns a cleanup function that should be deferred to close log files
func InitLogger(config *Config) (func(), error) {
	var writers []io.Writer
	var logFile *os.File

	// Console output
	if config.Logging.Console {
		writers = append(writers, os.Stdout)
	}

	// File output
	if config.Logging.File != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(config.Logging.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		var err error
		logFile, err = os.OpenFile(config.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		writers = append(writers, logFile)
	}

	if len(writers) > 0 {
		multiWriter := io.MultiWriter(writers...)
		log.SetOutput(multiWriter)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Return cleanup function
	cleanup := func() {
		if logFile != nil {
			logFile.Close()
		}
	}

	return cleanup, nil
}
