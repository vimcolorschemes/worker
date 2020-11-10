package date

import (
	"testing"
	"time"
)

func TestToday(t *testing.T) {
	t.Run("should returns today's date", func(t *testing.T) {
		today := time.Now().UTC()
		result := Today()

		hourDiff := today.Sub(result).Hours()

		// Since Today returns no hours, there should be a diff, but it should be
		// less than 24 hours
		if hourDiff > 24 {
			t.Errorf("Incorrect result for Today, got diff: %f, want diff: <%d", hourDiff, 24)
		}
	})

	t.Run("should be formatted with no hours", func(t *testing.T) {
		result := Today()

		if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 || result.Nanosecond() != 0 {
			t.Errorf("Incorrect result for Today, got time")
		}
	})
}

func TestRoundTimeToDate(t *testing.T) {
	t.Run("should be formatted with no hours", func(t *testing.T) {
		datetime := time.Date(2020, time.January, 01, 10, 30, 50, 500, time.UTC)
		result := RoundTimeToDate(datetime)

		if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 || result.Nanosecond() != 0 {
			t.Errorf("Incorrect result for RoundTimeToDate, got time")
		}
	})
}
