package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

const (
	// ./bart/logs/bart.log
	LOG_FILE_PATH = "./.bart/logs/bart.log"
)

func InitLogging() (*log.Logger, func() error, error) {
	if err := os.MkdirAll(filepath.Dir(LOG_FILE_PATH), 0755); err != nil {
		fmt.Println("Error creating log directory:", err)
		return nil, nil, err
	}

	logFile, err := os.OpenFile(LOG_FILE_PATH, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return nil, nil, err
	}
	// defer logFile.Close() // Only Relying on the cleanup function to close the file

	logger := log.New(io.MultiWriter(logFile, os.Stdout), "", log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	cleanupFunc := func() error { 
		fmt.Println("Closing log file...")
		return logFile.Close() 
	}

	return logger, cleanupFunc, nil
}
