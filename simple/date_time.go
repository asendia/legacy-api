package simple

import "time"

func TimeTodayUTC() time.Time {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return today
}

func DaysToDuration(days int) time.Duration {
	return time.Hour * time.Duration(24*days)
}

func MonthsToDuration(months int) time.Duration {
	return time.Hour * time.Duration(30*24*months)
}

func DurationToDays(dur time.Duration) float64 {
	nanoSeconds := float64(dur / 24 / 3600)
	days := nanoSeconds / 1000000000
	return days
}
