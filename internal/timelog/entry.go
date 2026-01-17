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

const timeLayout = "2006-01-02 15:04 -0700"

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

	dateAndTime := entry.EndTime.Format(timeLayout)

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

	endTime, err := time.Parse(timeLayout, dateAndTime)
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
	for scanner.Scan() {
		line := scanner.Text()
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
