package report

import (
	"strings"
	"testing"
	"time"

	"github.com/Rash419/ttimelog/internal/timelog"
	"github.com/stretchr/testify/assert"
)

func TestGenerateDailyReport(t *testing.T) {
	entries := []timelog.Entry{
		{EndTime: time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC), Description: "**arrived", Duration: 0},
		{EndTime: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC), Description: "dev:frontend: Build login page", Duration: 1 * time.Hour},
		{EndTime: time.Date(2026, 3, 8, 11, 0, 0, 0, time.UTC), Description: "dev:frontend: Build login page", Duration: 1 * time.Hour},
		{EndTime: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC), Description: "dev:backend: API design", Duration: 1 * time.Hour},
	}

	r := GenerateDailyReport(entries, time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC), 0)
	assert.Equal(t, "Daily", r.Period)
	assert.Equal(t, 3*time.Hour, r.TotalWork)
	assert.Equal(t, time.Duration(0), r.TotalSlack)
	// Two unique tasks: "dev:frontend: Build login page" (aggregated to 2h) and "dev:backend: API design" (1h)
	assert.Len(t, r.Items, 2)
}

func TestGenerateDailyReportSkipsArrived(t *testing.T) {
	entries := []timelog.Entry{
		{EndTime: time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC), Description: "**arrived", Duration: 0},
		{EndTime: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC), Description: "Working", Duration: 1 * time.Hour},
	}

	r := GenerateDailyReport(entries, time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC), 0)
	assert.Len(t, r.Items, 1)
	assert.Equal(t, "Working", r.Items[0].Description)
}

func TestGenerateDailyReportSlackSeparate(t *testing.T) {
	entries := []timelog.Entry{
		{EndTime: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC), Description: "Working", Duration: 1 * time.Hour},
		{EndTime: time.Date(2026, 3, 8, 10, 30, 0, 0, time.UTC), Description: "**slack break", Duration: 30 * time.Minute},
		{EndTime: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC), Description: "More work", Duration: 90 * time.Minute},
	}

	r := GenerateDailyReport(entries, time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC), 0)
	assert.Equal(t, 150*time.Minute, r.TotalWork)
	assert.Equal(t, 30*time.Minute, r.TotalSlack)
	assert.Len(t, r.Items, 2) // only work items
}

func TestGenerateWeeklyReport(t *testing.T) {
	entries := []timelog.Entry{
		{EndTime: time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC), Description: "Monday work", Duration: 1 * time.Hour},
		{EndTime: time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC), Description: "Wednesday work", Duration: 2 * time.Hour},
		{EndTime: time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC), Description: "Next week work", Duration: 3 * time.Hour},
	}

	r := GenerateWeeklyReport(entries, time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), 0)
	assert.Equal(t, "Weekly", r.Period)
	assert.Equal(t, 3*time.Hour, r.TotalWork)
	assert.Len(t, r.Items, 2)
}

func TestFormatReport(t *testing.T) {
	r := Report{
		Period:     "Daily",
		DateRange:  "2026-03-08",
		Items:      []ReportItem{{Category: "dev", Description: "coding", Duration: 2 * time.Hour}},
		TotalWork:  2 * time.Hour,
		TotalSlack: 30 * time.Minute,
	}

	output := FormatReport(r)
	assert.True(t, strings.Contains(output, "Daily Report: 2026-03-08"))
	assert.True(t, strings.Contains(output, "coding"))
	assert.True(t, strings.Contains(output, "Total work"))
	assert.True(t, strings.Contains(output, "Total slack"))
}
