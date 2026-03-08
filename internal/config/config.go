// Package config implements application's configuraiton related function
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

const (
	TimeLogDirname  = ".ttimelog"
	TimeLogFilename = "ttimelog.txt"
	TimeLogFile     = "ttimelog.log"
	TimeConfigFile  = "ttimelogrc"
	ProjectListFile = "project-list.txt"
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
	err := os.MkdirAll(fullDirPath, 0o755)
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

type AppConfig struct {
	Gtimelog struct {
		AuthHeader  string  `ini:"auth_header"`
		TaskListURL string  `ini:"task_list_url"`
		ReportToURL string  `ini:"report_to_url"`
		Hours       float64 `ini:"hours"`
	} `ini:"gtimelog"`
	TimeLogDirPath string
}

func LoadConfig(timeLogDir string) (*AppConfig, error) {
	configPath := filepath.Join(timeLogDir, TimeConfigFile)
	iniCfg, err := ini.Load(configPath)
	if err != nil {
		return nil, err
	}

	var cfg AppConfig
	if err := iniCfg.MapTo(&cfg); err != nil {
		return nil, err
	}
	cfg.TimeLogDirPath = timeLogDir
	if cfg.Gtimelog.Hours == 0 {
		cfg.Gtimelog.Hours = 8.0
	}
	return &cfg, nil
}
