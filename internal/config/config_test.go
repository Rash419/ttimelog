package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupTimeLogDirectory(t *testing.T) {
	tempDir := t.TempDir()

	_, err := SetupTimeLogDirectory(tempDir)
	if err != nil {
		t.Fatalf("SetupTimeLogDirectory() failed: %v", err)
	}

	expectedFilePath := filepath.Join(tempDir, TimeLogDirname, TimeLogFilename)
	if _, err := os.Stat(expectedFilePath); err != nil {
		t.Errorf("Expected file to be created, but got error: %v", err)
	}
}

func TestLoadConfig(t *testing.T) {
	testConfig := `
[gtimelog]
task_list_url = https://chronophage/rest-api/proxy/tasks
auth_header = Token ABCDXYZ
`
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "ttimelogrc")
	err := os.WriteFile(tempFile, []byte(testConfig), 0o666)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	appConfig, err := LoadConfig(tempDir)
	if err != nil {
		t.Errorf("Failed to LoadCofig with error: %v", err)
	}

	assert.Equal(t, "https://chronophage/rest-api/proxy/tasks", appConfig.Gtimelog.TaskListURL)
	assert.Equal(t, "Token ABCDXYZ", appConfig.Gtimelog.AuthHeader)
}

func TestLoadConfigDefaultHours(t *testing.T) {
	testConfig := `
[gtimelog]
task_list_url = https://chronophage/rest-api/proxy/tasks
auth_header = Token ABCDXYZ
`
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "ttimelogrc")
	err := os.WriteFile(tempFile, []byte(testConfig), 0o666)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	appConfig, err := LoadConfig(tempDir)
	assert.NoError(t, err)
	assert.Equal(t, 8.0, appConfig.Gtimelog.Hours)
}

func TestLoadConfigCustomHours(t *testing.T) {
	testConfig := `
[gtimelog]
task_list_url = https://chronophage/rest-api/proxy/tasks
auth_header = Token ABCDXYZ
hours = 7.5
`
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "ttimelogrc")
	err := os.WriteFile(tempFile, []byte(testConfig), 0o666)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	appConfig, err := LoadConfig(tempDir)
	assert.NoError(t, err)
	assert.Equal(t, 7.5, appConfig.Gtimelog.Hours)
}

func TestLoadConfigDefaultVirtualMidnight(t *testing.T) {
	testConfig := `
[gtimelog]
task_list_url = https://chronophage/rest-api/proxy/tasks
auth_header = Token ABCDXYZ
`
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "ttimelogrc")
	err := os.WriteFile(tempFile, []byte(testConfig), 0o666)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	appConfig, err := LoadConfig(tempDir)
	assert.NoError(t, err)
	assert.Equal(t, "02:00", appConfig.Gtimelog.VirtualMidnight)
}

func TestLoadConfigCustomVirtualMidnight(t *testing.T) {
	testConfig := `
[gtimelog]
task_list_url = https://chronophage/rest-api/proxy/tasks
auth_header = Token ABCDXYZ
virtual_midnight = 04:00
`
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "ttimelogrc")
	err := os.WriteFile(tempFile, []byte(testConfig), 0o666)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	appConfig, err := LoadConfig(tempDir)
	assert.NoError(t, err)
	assert.Equal(t, "04:00", appConfig.Gtimelog.VirtualMidnight)
}
