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
	if err := tmpFile.Close(); err != nil {
		t.Errorf("Failed to close temp file: %v", err)
	}

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

func TestLoadEntriesLineNumbers(t *testing.T) {
	tmpFile := createTempFile(t, strings.Join([]string{
		"2025-01-15 09:00 +0530: arrived**",
		"",
		"2025-01-15 10:00 +0530: Working on task A",
		"2025-01-15 11:00 +0530: Working on task B",
	}, "\n"))

	entries, _, _, err := LoadEntries(tmpFile)
	assert.NoError(t, err)
	assert.Len(t, entries, 3)

	assert.Equal(t, 1, entries[0].LineNumber)
	assert.Equal(t, 3, entries[1].LineNumber) // line 2 is empty
	assert.Equal(t, 4, entries[2].LineNumber)
}

func TestEditEntry(t *testing.T) {
	content := strings.Join([]string{
		"2025-01-15 09:00 +0530: arrived**",
		"2025-01-15 10:00 +0530: Working on task A",
		"2025-01-15 11:00 +0530: Working on task B",
	}, "\n")
	tmpFile := createTempFile(t, content)

	err := EditEntry(tmpFile, 2, "2025-01-15 10:30 +0530", "Updated task A")
	assert.NoError(t, err)

	result, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)

	lines := strings.Split(strings.TrimRight(string(result), "\n"), "\n")
	assert.Len(t, lines, 3)
	assert.Equal(t, "2025-01-15 09:00 +0530: arrived**", lines[0])
	assert.Equal(t, "2025-01-15 10:30 +0530: Updated task A", lines[1])
	assert.Equal(t, "2025-01-15 11:00 +0530: Working on task B", lines[2])
}

func TestDeleteEntry(t *testing.T) {
	content := strings.Join([]string{
		"2025-01-15 09:00 +0530: arrived**",
		"2025-01-15 10:00 +0530: Working on task A",
		"2025-01-15 11:00 +0530: Working on task B",
	}, "\n")
	tmpFile := createTempFile(t, content)

	err := DeleteEntry(tmpFile, 2)
	assert.NoError(t, err)

	result, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)

	lines := strings.Split(strings.TrimRight(string(result), "\n"), "\n")
	assert.Len(t, lines, 2)
	assert.Equal(t, "2025-01-15 09:00 +0530: arrived**", lines[0])
	assert.Equal(t, "2025-01-15 11:00 +0530: Working on task B", lines[1])
}

func TestEditEntryOutOfRange(t *testing.T) {
	tmpFile := createTempFile(t, "2025-01-15 09:00 +0530: arrived**")

	err := EditEntry(tmpFile, 5, "2025-01-15 10:00 +0530", "test")
	assert.Error(t, err)
}

func TestDeleteEntryOutOfRange(t *testing.T) {
	tmpFile := createTempFile(t, "2025-01-15 09:00 +0530: arrived**")

	err := DeleteEntry(tmpFile, 0)
	assert.Error(t, err)
}

// --- Feature 5: Virtual Midnight Tests ---

func TestVirtualDate(t *testing.T) {
	vm := 2 * time.Hour // 02:00

	// 1:30 AM → belongs to previous day
	t130am := time.Date(2026, 3, 8, 1, 30, 0, 0, time.UTC)
	vd := VirtualDate(t130am, vm)
	assert.Equal(t, 7, vd.Day())

	// 2:30 AM → belongs to same day
	t230am := time.Date(2026, 3, 8, 2, 30, 0, 0, time.UTC)
	vd = VirtualDate(t230am, vm)
	assert.Equal(t, 8, vd.Day())

	// 10:00 AM → belongs to same day
	t10am := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	vd = VirtualDate(t10am, vm)
	assert.Equal(t, 8, vd.Day())
}

func TestVirtualDateDefaultMidnight(t *testing.T) {
	vm := time.Duration(0) // 00:00 (disabled)

	t130am := time.Date(2026, 3, 8, 1, 30, 0, 0, time.UTC)
	vd := VirtualDate(t130am, vm)
	assert.Equal(t, 8, vd.Day()) // same day when VM=0
}

func TestParseVirtualMidnight(t *testing.T) {
	d, err := ParseVirtualMidnight("02:00")
	assert.NoError(t, err)
	assert.Equal(t, 2*time.Hour, d)

	d, err = ParseVirtualMidnight("00:00")
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), d)

	d, err = ParseVirtualMidnight("")
	assert.NoError(t, err)
	assert.Equal(t, 2*time.Hour, d) // default

	_, err = ParseVirtualMidnight("invalid")
	assert.Error(t, err)

	_, err = ParseVirtualMidnight("25:00")
	assert.Error(t, err)
}

// --- Feature 1: Date Navigation Tests ---

func TestStatsForDate(t *testing.T) {
	entries := []Entry{
		{EndTime: time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC), Description: "task A", Duration: 1 * time.Hour},
		{EndTime: time.Date(2026, 3, 7, 11, 0, 0, 0, time.UTC), Description: "**slack", Duration: 30 * time.Minute},
		{EndTime: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC), Description: "task B", Duration: 2 * time.Hour},
	}

	stats := StatsForDate(entries, time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC), 0)
	assert.Equal(t, 1*time.Hour, stats.Work)
	assert.Equal(t, 30*time.Minute, stats.Slack)
}

func TestStatsForDateNoEntries(t *testing.T) {
	entries := []Entry{
		{EndTime: time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC), Description: "task A", Duration: 1 * time.Hour},
	}
	stats := StatsForDate(entries, time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC), 0)
	assert.Equal(t, time.Duration(0), stats.Work)
	assert.Equal(t, time.Duration(0), stats.Slack)
}

func TestStatsForWeek(t *testing.T) {
	// 2026-03-02 is Monday, 2026-03-08 is Sunday of the same ISO week
	entries := []Entry{
		{EndTime: time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC), Description: "task A", Duration: 1 * time.Hour},
		{EndTime: time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC), Description: "task B", Duration: 2 * time.Hour},
		{EndTime: time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC), Description: "task C", Duration: 3 * time.Hour}, // next week
	}

	stats := StatsForWeek(entries, time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), 0)
	assert.Equal(t, 3*time.Hour, stats.Work)
}

func TestStatsForMonth(t *testing.T) {
	entries := []Entry{
		{EndTime: time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC), Description: "Feb task", Duration: 1 * time.Hour},
		{EndTime: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), Description: "Mar task A", Duration: 2 * time.Hour},
		{EndTime: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Description: "Mar task B", Duration: 3 * time.Hour},
	}

	stats := StatsForMonth(entries, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), 0)
	assert.Equal(t, 5*time.Hour, stats.Work)
}

func TestFilterEntriesForDate(t *testing.T) {
	entries := []Entry{
		{EndTime: time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC), Description: "task A"},
		{EndTime: time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC), Description: "task B"},
		{EndTime: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC), Description: "task C"},
	}

	filtered := FilterEntriesForDate(entries, time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC), 0)
	assert.Len(t, filtered, 2)
	assert.Equal(t, "task A", filtered[0].Description)
	assert.Equal(t, "task B", filtered[1].Description)
}

func TestFilterEntriesForDateWithVirtualMidnight(t *testing.T) {
	vm := 2 * time.Hour
	entries := []Entry{
		{EndTime: time.Date(2026, 3, 7, 23, 0, 0, 0, time.UTC), Description: "late night work"},
		{EndTime: time.Date(2026, 3, 8, 1, 30, 0, 0, time.UTC), Description: "past midnight work"},  // before VM → belongs to Mar 7
		{EndTime: time.Date(2026, 3, 8, 2, 30, 0, 0, time.UTC), Description: "after VM work"},        // after VM → belongs to Mar 8
	}

	filtered := FilterEntriesForDate(entries, time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC), vm)
	assert.Len(t, filtered, 2)
	assert.Equal(t, "late night work", filtered[0].Description)
	assert.Equal(t, "past midnight work", filtered[1].Description)

	filtered = FilterEntriesForDate(entries, time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC), vm)
	assert.Len(t, filtered, 1)
	assert.Equal(t, "after VM work", filtered[0].Description)
}

// --- Feature 4: Activity History Tests ---

func TestBuildActivityHistory(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{EndTime: now, Description: "task A", Duration: 1 * time.Hour},
		{EndTime: now, Description: "task B", Duration: 1 * time.Hour},
		{EndTime: now, Description: "task A", Duration: 1 * time.Hour},
		{EndTime: now, Description: "task C", Duration: 1 * time.Hour},
		{EndTime: now, Description: "task A", Duration: 1 * time.Hour},
	}
	history := BuildActivityHistory(entries, 10, 90)
	assert.Len(t, history, 3)
	assert.Equal(t, "task A", history[0]) // most frequent
}

func TestBuildActivityHistoryExcludesArrived(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{EndTime: now, Description: "**arrived", Duration: 0},
		{EndTime: now, Description: "arrived**", Duration: 0},
		{EndTime: now, Description: "task A", Duration: 1 * time.Hour},
		{EndTime: now, Description: "**slack break", Duration: 30 * time.Minute},
	}
	history := BuildActivityHistory(entries, 10, 90)
	assert.Len(t, history, 1)
	assert.Equal(t, "task A", history[0])
}

func TestBuildActivityHistoryLimit(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{EndTime: now, Description: "task A", Duration: 1 * time.Hour},
		{EndTime: now, Description: "task B", Duration: 1 * time.Hour},
		{EndTime: now, Description: "task C", Duration: 1 * time.Hour},
	}
	history := BuildActivityHistory(entries, 2, 90)
	assert.Len(t, history, 2)
}

func TestBuildActivityHistoryRecent(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{EndTime: now.AddDate(0, 0, -100), Description: "old task", Duration: 1 * time.Hour},
		{EndTime: now, Description: "recent task", Duration: 1 * time.Hour},
	}
	history := BuildActivityHistory(entries, 10, 90)
	assert.Len(t, history, 1)
	assert.Equal(t, "recent task", history[0])
}

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "ttimelog-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	name := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	return name
}
