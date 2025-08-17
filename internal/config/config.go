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
	timeLogDirname  = ".ttimelog"
	timeLogFilename = "ttimelog.txt"
)

func GetSlogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	return slog.New(handler)
}

func SetupTimeLogDirectory(userDir string) error {
	fullDirPath := filepath.Join(userDir, timeLogDirname)
	err := os.MkdirAll(fullDirPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory[%s] with error[%v]", fullDirPath, err)
	}

	timeLogFilePath := filepath.Join(fullDirPath, timeLogFilename)

	_, err = os.Stat(timeLogFilePath)
	if errors.Is(err, os.ErrNotExist) {
		timeLogFile, err := os.Create(timeLogFilePath)
		if err != nil {
			return fmt.Errorf("failed to create timeLogFile[%s] with error[%v]", timeLogFilePath, err)
		}
		defer timeLogFile.Close()
		slog.Info("Successfully created", "file", timeLogFilePath)
	} else if err != nil {
		return fmt.Errorf("failed to open timeLogFile[%s] with error[%v]", timeLogFilePath, err)
	}

	return nil
}
