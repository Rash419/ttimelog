package timelog

import (
	"bufio"
	"errors"
	"fmt"
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
}

type StatsCollection struct {
	Daily   Stats
	Weekly  Stats
	Monthly Stats
}

type Stats struct {
	Work  time.Duration
	Slack time.Duration
}

const timeLayout = "2006-01-02 15:04 -0700"

// TODO: Add test for SaveEntry

// SaveEntry saves the entry in 'YYYY-MM-DD HH:MM +/-0000: Task Description' format
func SaveEntry(entry Entry) error {
	// TODO: make it dynamic
	// we already create the file if didn't exist in config.go when we create
	// we can save the path of the file in config struct or make a global variable.
	// I am not sure.
	filename := "/home/rashesh/.ttimelog/ttimelog.txt"

	// Open the file in append mode. Create it if it doesn't exist.
	// os.O_APPEND: Open the file for appending.
	// os.O_WRONLY: Open the file for writing only.
	// 0644: File permissions (read/write for owner, read-only for others).
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	dateAndTime := entry.EndTime.Format(timeLayout)
	textEntry := fmt.Sprintf("%s: %s\n", dateAndTime, entry.Description)

	if _, err := f.WriteString(textEntry); err != nil {
		return err
	}

	return nil
}

func GetEntryState(t time.Time) (bool, bool, bool) {
	now := time.Now()
	y1, m1, d1 := t.Date()
	y2, m2, d2 := now.Date()

	_, w1 := t.ISOWeek()
	_, w2 := now.ISOWeek()

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
func parseEntry(line string, previousEntry *Entry) (*Entry, error) {
	// It splits in 3 strings and we merge them later
	tokens := strings.Split(line, ":")
	if len(tokens) < 3 {
		return nil, errors.New("invalid format")
	}

	dateAndTime := tokens[0] + ":" + tokens[1]
	dateAndTimeTokens := strings.Split(dateAndTime, " ")
	if len(dateAndTimeTokens) < 3 {
		return nil, errors.New("invalid format")
	}

	parsedDate := dateAndTimeTokens[0]

	endTime, err := time.Parse(timeLayout, dateAndTime)
	if err != nil {
		return nil, err
	}

	entryDuration := time.Duration(0)
	if previousEntry != nil {
		prevDate := previousEntry.EndTime.Format("2006-01-02")
		if parsedDate == prevDate {
			entryDuration = endTime.Sub(previousEntry.EndTime)
		}
	}

	today, currentWeek, currentMonth := GetEntryState(endTime)
	return &Entry{
		EndTime:      endTime,
		Description:  strings.Trim(tokens[2], " "),
		Duration:     entryDuration,
		Today:        today,
		CurrentWeek:  currentWeek,
		CurrentMonth: currentMonth,
	}, nil
}

func LoadEntries(filePath string) ([]Entry, StatsCollection, error) {
	statsCollection := StatsCollection{
		Daily: Stats{},
		Weekly: Stats{},
		Monthly: Stats{},
	}

	entries := make([]Entry, 0)
	file, err := os.Open(filePath)
	if err != nil {
		return entries, statsCollection, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var (
			entry *Entry
			err   error
		)
		line = strings.Trim(line, " ")
		if len(entries) == 0 {
			entry, err = parseEntry(line, nil)
		} else {
			entry, err = parseEntry(line, &entries[len(entries)-1])
		}
		if err != nil {
			return entries, statsCollection, err
		}

		if entry.Today {
			statsCollection.Daily.Work += entry.Duration
		}
		if entry.CurrentWeek {
			statsCollection.Weekly.Work += entry.Duration
		}
		if entry.CurrentMonth {
			statsCollection.Monthly.Work += entry.Duration
		}
		entries = append(entries, *entry)
	}

	if err := scanner.Err(); err != nil {
		return entries, statsCollection, err
	}
	return entries, statsCollection, nil
}

// FormatDuration formats a time.Duration into "__h __m" format.
func FormatDuration(diff time.Duration) string {
	diff = diff.Truncate(time.Minute)

	hours := diff / time.Hour
	diff -= hours * time.Hour
	mins := diff / time.Minute
	return fmt.Sprintf("%d h %d min", hours, mins)
}

func FormatStatDuration(diff time.Duration) string {
	diff = diff.Truncate(time.Minute)

	hours := diff / time.Hour
	diff -= hours * time.Hour
	mins := diff / time.Minute
	return fmt.Sprintf("%dh%dm", hours, mins)
}
