package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Rash419/ttimelog/internal/timelog"
)

// Report represents a generated time report.
type Report struct {
	Period     string
	DateRange  string
	Items      []ReportItem
	TotalWork  time.Duration
	TotalSlack time.Duration
}

// ReportItem is a single line in a report.
type ReportItem struct {
	Category    string
	Description string
	Duration    time.Duration
}

func categorize(description string) (string, string) {
	if idx := strings.Index(description, ": "); idx >= 0 {
		return description[:idx], description[idx+2:]
	}
	return "", description
}

func generateReport(entries []timelog.Entry, period, dateRange string) Report {
	type taskKey struct {
		category    string
		description string
	}

	aggregated := make(map[taskKey]time.Duration)
	var totalWork, totalSlack time.Duration

	for _, e := range entries {
		if timelog.IsArrivedMessage(e.Description) {
			continue
		}
		if strings.Contains(e.Description, "**") {
			totalSlack += e.Duration
			continue
		}
		totalWork += e.Duration
		cat, desc := categorize(e.Description)
		k := taskKey{cat, desc}
		aggregated[k] += e.Duration
	}

	items := make([]ReportItem, 0, len(aggregated))
	for k, d := range aggregated {
		items = append(items, ReportItem{
			Category:    k.category,
			Description: k.description,
			Duration:    d,
		})
	}

	// Sort by category then description
	sort.Slice(items, func(i, j int) bool {
		if items[i].Category != items[j].Category {
			return items[i].Category < items[j].Category
		}
		return items[i].Description < items[j].Description
	})

	return Report{
		Period:     period,
		DateRange:  dateRange,
		Items:      items,
		TotalWork:  totalWork,
		TotalSlack: totalSlack,
	}
}

// GenerateDailyReport creates a report for a single day.
func GenerateDailyReport(entries []timelog.Entry, date time.Time, virtualMidnight time.Duration) Report {
	filtered := timelog.FilterEntriesForDate(entries, date, virtualMidnight)
	dateRange := date.Format("2006-01-02")
	return generateReport(filtered, "Daily", dateRange)
}

// GenerateWeeklyReport creates a report for the ISO week containing date.
func GenerateWeeklyReport(entries []timelog.Entry, date time.Time, virtualMidnight time.Duration) Report {
	// Find Monday of the ISO week
	weekday := date.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	monday := date.AddDate(0, 0, -int(weekday-time.Monday))
	sunday := monday.AddDate(0, 0, 6)

	var filtered []timelog.Entry
	targetYear, targetWeek := date.ISOWeek()
	for _, e := range entries {
		vd := timelog.VirtualDate(e.EndTime, virtualMidnight)
		y, w := vd.ISOWeek()
		if y == targetYear && w == targetWeek {
			filtered = append(filtered, e)
		}
	}

	dateRange := fmt.Sprintf("%s to %s", monday.Format("2006-01-02"), sunday.Format("2006-01-02"))
	return generateReport(filtered, "Weekly", dateRange)
}

// GenerateMonthlyReport creates a report for the month containing date.
func GenerateMonthlyReport(entries []timelog.Entry, date time.Time, virtualMidnight time.Duration) Report {
	var filtered []timelog.Entry
	for _, e := range entries {
		vd := timelog.VirtualDate(e.EndTime, virtualMidnight)
		if vd.Year() == date.Year() && vd.Month() == date.Month() {
			filtered = append(filtered, e)
		}
	}

	first := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	last := first.AddDate(0, 1, -1)
	dateRange := fmt.Sprintf("%s to %s", first.Format("2006-01-02"), last.Format("2006-01-02"))
	return generateReport(filtered, "Monthly", dateRange)
}

// FormatReport formats a report as human-readable text.
func FormatReport(r Report) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s Report: %s\n", r.Period, r.DateRange))
	sb.WriteString(strings.Repeat("─", 60) + "\n\n")

	currentCategory := ""
	for _, item := range r.Items {
		if item.Category != currentCategory {
			if currentCategory != "" {
				sb.WriteString("\n")
			}
			if item.Category != "" {
				sb.WriteString(fmt.Sprintf("[%s]\n", item.Category))
			}
			currentCategory = item.Category
		}
		sb.WriteString(fmt.Sprintf("  %-8s %s\n", timelog.FormatDurationShort(item.Duration), item.Description))
	}

	sb.WriteString("\n" + strings.Repeat("─", 60) + "\n")
	sb.WriteString(fmt.Sprintf("Total work:  %s\n", timelog.FormatDurationShort(r.TotalWork)))
	sb.WriteString(fmt.Sprintf("Total slack: %s\n", timelog.FormatDurationShort(r.TotalSlack)))

	return sb.String()
}
