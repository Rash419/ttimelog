// Package config implements application's configuraiton related function
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	TimeLogDirname  = ".ttimelog"
	TimeLogFilename = "ttimelog.txt"
	TimeLogFile     = "ttimelog.log"
)

func GetSlogger(logFile *os.File) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := slog.NewTextHandler(logFile, opts)
	return slog.New(handler)
}

func SetupTimeLogDirectory(userDir string) (string, error) {
	fullDirPath := filepath.Join(userDir, TimeLogDirname)
	err := os.MkdirAll(fullDirPath, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create directory[%s] with error[%v]", fullDirPath, err)
	}

	timeLogFilePath := filepath.Join(fullDirPath, TimeLogFilename)

	_, err = os.Stat(timeLogFilePath)
	if errors.Is(err, os.ErrNotExist) {
		timeLogFile, err := os.Create(timeLogFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to create timeLogFile[%s] with error[%v]", timeLogFilePath, err)
		}
		defer func() {
			if err := timeLogFile.Close(); err != nil {
				slog.Error("Failed to close time log file", "error", err)
			}
		}()
		slog.Info("Successfully created", "file", timeLogFilePath)
	} else if err != nil {
		return "", fmt.Errorf("failed to open timeLogFile[%s] with error[%v]", timeLogFilePath, err)
	}

	return timeLogFilePath, nil
}
