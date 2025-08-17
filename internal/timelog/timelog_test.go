package timelog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeLog(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(2*time.Hour + 1*time.Minute)
	timeDiff := GetTimeDiff(startTime, endTime)
	assert.Equal(t, "2 h 1 min", timeDiff)
}
