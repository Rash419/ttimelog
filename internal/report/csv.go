package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rash419/ttimelog/internal/timelog"
)

func csvHeader() []string {
	return []string{"Date", "Start", "End", "Duration", "Category", "Description"}
}

func entriesToCSVRows(entries []timelog.Entry) [][]string {
	var rows [][]string
	var lastEndTime time.Time
	for i, e := range entries {
		if timelog.IsArrivedMessage(e.Description) {
			lastEndTime = e.EndTime
			continue
		}
		startTime := lastEndTime
		if i == 0 {
			startTime = e.EndTime
		}

		cat, desc := categorize(e.Description)
		rows = append(rows, []string{
			e.EndTime.Format("2006-01-02"),
			startTime.Format("15:04"),
			e.EndTime.Format("15:04"),
			timelog.FormatDurationShort(e.Duration),
			cat,
			desc,
		})
		lastEndTime = e.EndTime
	}
	return rows
}

// ExportDailyCSV generates CSV content for a single day.
func ExportDailyCSV(entries []timelog.Entry, date time.Time, virtualMidnight time.Duration) string {
	filtered := timelog.FilterEntriesForDate(entries, date, virtualMidnight)
	return formatCSV(filtered)
}

// ExportWeeklyCSV generates CSV content for the ISO week containing date.
func ExportWeeklyCSV(entries []timelog.Entry, date time.Time, virtualMidnight time.Duration) string {
	targetYear, targetWeek := date.ISOWeek()
	var filtered []timelog.Entry
	for _, e := range entries {
		vd := timelog.VirtualDate(e.EndTime, virtualMidnight)
		y, w := vd.ISOWeek()
		if y == targetYear && w == targetWeek {
			filtered = append(filtered, e)
		}
	}
	return formatCSV(filtered)
}

// ExportMonthlyCSV generates CSV content for the month containing date.
func ExportMonthlyCSV(entries []timelog.Entry, date time.Time, virtualMidnight time.Duration) string {
	var filtered []timelog.Entry
	for _, e := range entries {
		vd := timelog.VirtualDate(e.EndTime, virtualMidnight)
		if vd.Year() == date.Year() && vd.Month() == date.Month() {
			filtered = append(filtered, e)
		}
	}
	return formatCSV(filtered)
}

func formatCSV(entries []timelog.Entry) string {
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write(csvHeader())
	rows := entriesToCSVRows(entries)
	_ = w.WriteAll(rows)
	w.Flush()
	return sb.String()
}

// WriteCSV writes CSV content to a file, creating parent directories if needed.
func WriteCSV(content string, filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return os.WriteFile(filePath, []byte(content), 0o644)
}
