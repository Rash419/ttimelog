package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rash419/ttimelog/internal/timelog"
	"github.com/stretchr/testify/assert"
)

func TestExportDailyCSV(t *testing.T) {
	entries := []timelog.Entry{
		{EndTime: time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC), Description: "**arrived", Duration: 0},
		{EndTime: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC), Description: "dev:frontend: Build login page", Duration: 1 * time.Hour},
		{EndTime: time.Date(2026, 3, 8, 11, 0, 0, 0, time.UTC), Description: "dev:backend: API design", Duration: 1 * time.Hour},
	}

	csv := ExportDailyCSV(entries, time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC), 0)
	lines := strings.Split(strings.TrimSpace(csv), "\n")

	// Header + 2 data rows (arrived is skipped)
	assert.Len(t, lines, 3)
	assert.True(t, strings.HasPrefix(lines[0], "Date,"))
	assert.True(t, strings.Contains(lines[1], "Build login page"))
}

func TestExportDailyCSVEmpty(t *testing.T) {
	csv := ExportDailyCSV(nil, time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC), 0)
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	assert.Len(t, lines, 1) // header only
}

func TestWriteCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reports", "test.csv")
	content := "Date,Start,End,Duration,Category,Description\n2026-03-08,09:00,10:00,1:00,dev,coding\n"

	err := WriteCSV(content, path)
	assert.NoError(t, err)

	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))
}
