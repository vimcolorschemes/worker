package date

import "time"

// Today returns the UTC date without time
func Today() time.Time {
	return RoundTimeToDate(time.Now().UTC())
}

// RoundTimeToDate removes time from a datetime
func RoundTimeToDate(datetime time.Time) time.Time {
	return time.Date(datetime.Year(), datetime.Month(), datetime.Day(), 0, 0, 0, 0, time.UTC)
}

// IsSameDay returns true if 2 dates are on the same specific day
func IsSameDay(time1 time.Time, time2 time.Time) bool {
	return RoundTimeToDate(time1) == RoundTimeToDate(time2)
}
