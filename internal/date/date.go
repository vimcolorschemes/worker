package date

import "time"

func Today() time.Time {
	return RoundTimeToDate(time.Now().UTC())
}

func RoundTimeToDate(datetime time.Time) time.Time {
	return time.Date(datetime.Year(), datetime.Month(), datetime.Day(), 0, 0, 0, 0, time.UTC)
}

func IsSameDay(time1 time.Time, time2 time.Time) bool {
	return RoundTimeToDate(time1) == RoundTimeToDate(time2)
}
