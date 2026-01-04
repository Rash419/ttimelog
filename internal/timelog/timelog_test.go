package timelog

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEntryState(t *testing.T) {
	baseDate := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	today, currentWeek, currentMonth := GetEntryState(baseDate, baseDate)
	assert.Equal(t, true, today)
	assert.Equal(t, true, currentWeek)
	assert.Equal(t, true, currentMonth)

	nextDay := baseDate.AddDate(0, 0, 1)
	today, currentWeek, currentMonth = GetEntryState(nextDay, baseDate)
	assert.Equal(t, false, today)
	assert.Equal(t, true, currentWeek)
	assert.Equal(t, true, currentMonth)

	nextWeek := baseDate.AddDate(0, 0, 7)
	today, currentWeek, currentMonth = GetEntryState(nextWeek, baseDate)
	assert.Equal(t, false, today)
	assert.Equal(t, false, currentWeek)
	assert.Equal(t, true, currentMonth)

	nextMonth := baseDate.AddDate(0, 1, 0)
	today, currentWeek, currentMonth = GetEntryState(nextMonth, baseDate)
	assert.Equal(t, false, today)
	assert.Equal(t, false, currentWeek)
	assert.Equal(t, false, currentMonth)

	nextYear := baseDate.AddDate(1, 0, 0)
	today, currentWeek, currentMonth = GetEntryState(nextYear, baseDate)
	assert.Equal(t, false, today)
	assert.Equal(t, false, currentWeek)
	assert.Equal(t, false, currentMonth)
}

func TestLoadEntries(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "ttimelog.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")

	lines := []string{
		fmt.Sprintf("%s 22:00 +0530: Yesterday task", yesterday),
		// Yesterday's last task
		fmt.Sprintf("%s 23:00 +0530: End of yesterday", yesterday),
		// Today's first task (Gap should be ignored)
		fmt.Sprintf("%s 09:00 +0530: Start of today", today),
		fmt.Sprintf("%s 10:00 +0530: Working", today),
	}

	result := strings.Join(lines, "\n")
	result = strings.TrimRight(result, "\n")

	if _, err := tmpFile.WriteString(result); err != nil {
		t.Fatalf("Failed write content to temp file with error[%v]", err)
	}

	tmpFilename := tmpFile.Name()
	tmpFile.Close()

	entries, _, _, err := LoadEntries(tmpFilename)

	assert.NoError(t, err)
	assert.Len(t, entries, 4)

	// Assertions
	// Entry 0 -> Duration 0 (first entry in timelog)
	assert.Equal(t, time.Duration(0), entries[0].Duration)
	// Entry 1 -> Duration 1 h (Yesterday's last task)
	assert.Equal(t, 1*time.Hour, entries[1].Duration)
	// Entry 2 (Today 09:00) -> Duration 0 (Reset! Not 10 hours)
	assert.Equal(t, time.Duration(0), entries[2].Duration)
	// Entry 3 (Today 10:00) -> Duration 1h
	assert.Equal(t, 1*time.Hour, entries[3].Duration)
}
