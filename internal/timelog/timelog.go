// Package timelog contains implementation required for timelogging
package timelog

import (
	"fmt"
	"strings"
	"time"
)

type TimeLog struct {
	startTime time.Time
}

// formatDuration formats a time.Duration into "__h __m" format.
func formatDuration(diff time.Duration) string {
	diff = diff.Truncate(time.Minute)

	hours := diff / time.Hour
	diff -= hours * time.Hour
	mins := diff / time.Minute
	return fmt.Sprintf("%d h %d min", hours, mins)
}

func GetTimeDiff(startTime time.Time, endTime time.Time) string {
	diff := endTime.Sub(startTime)
	return formatDuration(diff)
}

func isSlackingTime(input string) bool {
	return strings.Contains(input, "**")
}

func parseTextInput(input string) {

}

func appendFile(input string) {

}
