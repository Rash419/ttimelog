package timelog

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Entry struct {
	EndTime     time.Time
	Description string
	// Duration is computed on load, not stored
	Duration time.Duration

	Today        bool
	CurrentWeek  bool
	CurrentMonth bool
	LineNumber   int
}

type StatsCollection struct {
	Daily       Stats
	Weekly      Stats
	Monthly     Stats
	ArrivedTime time.Time
}

type Stats struct {
	Work  time.Duration
	Slack time.Duration
}

const TimeLayout = "2006-01-02 15:04 -0700"

// ParseVirtualMidnight parses a "HH:MM" string into a time.Duration from midnight.
func ParseVirtualMidnight(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 2 * time.Hour, nil // default
	}
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid virtual_midnight format: %q (expected HH:MM)", s)
	}
	var h, m int
	if _, err := fmt.Sscanf(parts[0], "%d", &h); err != nil {
		return 0, fmt.Errorf("invalid virtual_midnight hours: %w", err)
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &m); err != nil {
		return 0, fmt.Errorf("invalid virtual_midnight minutes: %w", err)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("virtual_midnight out of range: %02d:%02d", h, m)
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute, nil
}

// VirtualDate returns the "logical" date for a given time, respecting virtual midnight.
// If t is before virtual midnight, it belongs to the previous day.
func VirtualDate(t time.Time, virtualMidnight time.Duration) time.Time {
	midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	vm := midnight.Add(virtualMidnight)
	if virtualMidnight > 0 && t.Before(vm) {
		return midnight.AddDate(0, 0, -1)
	}
	return midnight
}

func NewEntry(endTime time.Time, description string, duration time.Duration) Entry {
	today, currentWeek, currentMonth := GetEntryState(endTime)
	return Entry{
		EndTime:      endTime,
		Description:  description,
		Duration:     duration,
		Today:        today,
		CurrentWeek:  currentWeek,
		CurrentMonth: currentMonth,
	}
}

// TODO: Add test for SaveEntry
// SaveEntry saves the entry in 'YYYY-MM-DD HH:MM +/-0000: Task Description' format
func SaveEntry(entry Entry, addNewLine bool, timeLogFilePath string) error {
	// Open the file in append mode. Create it if it doesn't exist.
	// os.O_APPEND: Open the file for appending.
	// os.O_WRONLY: Open the file for writing only.
	// 0644: File permissions (read/write for owner, read-only for others).
	f, err := os.OpenFile(timeLogFilePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("Failed to close file", "error", err)
		}
	}()

	dateAndTime := entry.EndTime.Format(TimeLayout)

	saveFormat := "%s: %s\n"
	if addNewLine {
		saveFormat = "\n" + saveFormat
	}
	textEntry := fmt.Sprintf(saveFormat, dateAndTime, entry.Description)

	if _, err := f.WriteString(textEntry); err != nil {
		return err
	}

	return nil
}

func GetEntryState(t time.Time, now ...time.Time) (bool, bool, bool) {
	referenceTime := time.Now()
	if len(now) > 0 {
		referenceTime = now[0]
	}
	nowTime := referenceTime
	y1, m1, d1 := t.Date()
	y2, m2, d2 := nowTime.Date()

	_, w1 := t.ISOWeek()
	_, w2 := nowTime.ISOWeek()

	var today, currentWeek, currentMonth bool
	if y1 != y2 {
		return false, false, false
	}

	if w1 == w2 {
		currentWeek = true
	}

	if m1 == m2 {
		currentMonth = true
		if d1 == d2 {
			today = true
		}
	}

	return today, currentWeek, currentMonth
}

// 2025-10-17 13:30 +0530: Working on ttimelog
func parseEntry(line string, firstEntry bool, previousEntry Entry) (Entry, error) {
	// It splits in 3 strings and we merge them later
	tokens := strings.SplitN(line, ":", 3)
	if len(tokens) < 3 {
		return Entry{}, errors.New("invalid format")
	}

	dateAndTime := tokens[0] + ":" + tokens[1]
	dateAndTimeTokens := strings.Split(dateAndTime, " ")
	if len(dateAndTimeTokens) < 3 {
		return Entry{}, errors.New("invalid format")
	}

	parsedDate := dateAndTimeTokens[0]

	endTime, err := time.Parse(TimeLayout, dateAndTime)
	if err != nil {
		return Entry{}, err
	}

	entryDuration := time.Duration(0)
	if !firstEntry {
		prevDate := previousEntry.EndTime.Format("2006-01-02")
		if parsedDate == prevDate {
			entryDuration = endTime.Sub(previousEntry.EndTime)
		}
	}

	entry := NewEntry(endTime, strings.Trim(tokens[2], " "), entryDuration)
	return entry, nil
}

func LoadEntries(filePath string) ([]Entry, StatsCollection, bool, error) {
	statsCollection := StatsCollection{
		Daily:   Stats{},
		Weekly:  Stats{},
		Monthly: Stats{},
	}

	entries := make([]Entry, 0)
	file, err := os.Open(filePath)
	handledArrivedMessage := false
	if err != nil {
		return entries, statsCollection, handledArrivedMessage, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Error("Failed to close file", "error", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++
		if line == "" {
			continue
		}
		var (
			entry Entry
			err   error
		)
		line = strings.Trim(line, " ")
		if len(entries) == 0 {
			entry, err = parseEntry(line, true, Entry{})
		} else {
			entry, err = parseEntry(line, false, entries[len(entries)-1])
		}

		if err != nil {
			return entries, statsCollection, handledArrivedMessage, err
		}

		entry.LineNumber = lineNumber

		if entry.Today && IsArrivedMessage(entry.Description) {
			handledArrivedMessage = true
			statsCollection.ArrivedTime = entry.EndTime
		}

		UpdateStatsCollection(entry, &statsCollection)
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return entries, statsCollection, handledArrivedMessage, err
	}
	return entries, statsCollection, handledArrivedMessage, nil
}

func UpdateStatsCollection(entry Entry, statsCollection *StatsCollection) {
	isSlackTime := strings.Contains(entry.Description, "**")
	if entry.Today {
		if isSlackTime {
			statsCollection.Daily.Slack += entry.Duration
		} else {
			statsCollection.Daily.Work += entry.Duration
		}
	}
	if entry.CurrentWeek {
		if isSlackTime {
			statsCollection.Weekly.Slack += entry.Duration
		} else {
			statsCollection.Weekly.Work += entry.Duration
		}
	}
	if entry.CurrentMonth {
		if isSlackTime {
			statsCollection.Monthly.Slack += entry.Duration
		} else {
			statsCollection.Monthly.Work += entry.Duration
		}
	}
}

// FormatDuration formats a time.Duration into "__h __m" format.
func FormatDuration(diff time.Duration) string {
	diff = diff.Truncate(time.Minute)

	hours := diff / time.Hour
	diff -= hours * time.Hour
	mins := diff / time.Minute
	return fmt.Sprintf("%d h %d min", hours, mins)
}

// FormatDurationShort formats a time.Duration into "H:MM" format (e.g., "1:30" for 1h30m).
func FormatDurationShort(d time.Duration) string {
	d = d.Truncate(time.Minute)
	hours := int(d / time.Hour)
	mins := int((d - time.Duration(hours)*time.Hour) / time.Minute)
	return fmt.Sprintf("%d:%02d", hours, mins)
}

func IsArrivedMessage(val string) bool {
	return val == "**arrived" || val == "arrived**"
}

func FormatTime(t time.Time) string {
	return t.Format("15h04m")
}

func FormatStatDuration(diff time.Duration) string {
	diff = diff.Truncate(time.Minute)

	hours := diff / time.Hour
	diff -= hours * time.Hour
	mins := diff / time.Minute
	return fmt.Sprintf("%dh%dm", hours, mins)
}

func readAllLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Error("Failed to close file", "error", err)
		}
	}()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeAllLines(filePath string, lines []string) error {
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}
	return os.WriteFile(filePath, []byte(content), 0o644)
}

// EditEntry replaces the line at lineNumber (1-indexed) with a new timestamp and description.
func EditEntry(filePath string, lineNumber int, newTimestamp string, newDescription string) error {
	lines, err := readAllLines(filePath)
	if err != nil {
		return err
	}

	idx := lineNumber - 1
	if idx < 0 || idx >= len(lines) {
		return fmt.Errorf("line number %d out of range (file has %d lines)", lineNumber, len(lines))
	}

	lines[idx] = fmt.Sprintf("%s: %s", newTimestamp, newDescription)
	return writeAllLines(filePath, lines)
}

// DeleteEntry removes the line at lineNumber (1-indexed) from the file.
func DeleteEntry(filePath string, lineNumber int) error {
	lines, err := readAllLines(filePath)
	if err != nil {
		return err
	}

	idx := lineNumber - 1
	if idx < 0 || idx >= len(lines) {
		return fmt.Errorf("line number %d out of range (file has %d lines)", lineNumber, len(lines))
	}

	lines = append(lines[:idx], lines[idx+1:]...)
	return writeAllLines(filePath, lines)
}
