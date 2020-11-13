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

		// since Today returns no hours, there should be a diff, but it should be
		// less than 24 hours
		if hourDiff > 24 {
			t.Errorf("Incorrect result for Today, got diff: %f, want diff: <%d", hourDiff, 24)
		}
	})

	t.Run("should be formatted with no hours", func(t *testing.T) {
		result := Today()

		if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 || result.Nanosecond() != 0 {
			t.Error("Incorrect result for Today, got time")
		}
	})
}

func TestRoundTimeToDate(t *testing.T) {
	t.Run("should be formatted with no hours", func(t *testing.T) {
		datetime := time.Date(2020, time.January, 01, 10, 30, 50, 500, time.UTC)
		result := RoundTimeToDate(datetime)

		if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 || result.Nanosecond() != 0 {
			t.Error("Incorrect result for RoundTimeToDate, got time")
		}
	})
}

func TestIsSameDay(t *testing.T) {
	t.Run("should return true when comparing the same time", func(t *testing.T) {
		date := time.Now()

		if !IsSameDay(date, date) {
			t.Errorf("Incorrect result for IsSameDay, returned false for %s and %s", date, date)
		}
	})

	t.Run("should return true when comparing 2 equal times", func(t *testing.T) {
		date1 := time.Now()
		date2 := time.Now()

		if !IsSameDay(date1, date2) {
			t.Errorf("Incorrect result for IsSameDay, returned false for %s and %s", date1, date2)
		}
	})

	t.Run("should return true when comparing 2 times of same day", func(t *testing.T) {
		date1 := time.Date(2020, time.January, 01, 23, 59, 59, 999, time.UTC)
		date2 := time.Date(2020, time.January, 01, 00, 00, 00, 000, time.UTC)

		if !IsSameDay(date1, date2) {
			t.Errorf("Incorrect result for IsSameDay, returned false for %s and %s", date1, date2)
		}
	})

	t.Run("should return false when comparing 2 different days", func(t *testing.T) {
		date1 := time.Date(2020, time.January, 01, 23, 59, 59, 999, time.UTC)
		date2 := time.Date(2019, time.January, 01, 00, 00, 00, 000, time.UTC)

		if IsSameDay(date1, date2) {
			t.Errorf("Incorrect result for IsSameDay, returned true for %s and %s", date1, date2)
		}
	})
}
