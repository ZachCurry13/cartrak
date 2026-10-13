package mileage

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseWeeks reads "26", "26-28", "26 – 28" or "26 to 28" as a range of ISO calendar weeks.
func ParseWeeks(s string) (from, to int, err error) {
	t := strings.NewReplacer("–", "-", "—", "-", " to ", "-", " ", "").Replace(strings.TrimSpace(strings.ToLower(s)))
	a, b, hasRange := strings.Cut(t, "-")
	if from, err = strconv.Atoi(a); err != nil {
		return 0, 0, fmt.Errorf("weeks must be like 26 or 26-28")
	}
	to = from
	if hasRange {
		if to, err = strconv.Atoi(b); err != nil {
			return 0, 0, fmt.Errorf("weeks must be like 26 or 26-28")
		}
	}
	if from < 1 || to < from || to > 53 {
		return 0, 0, fmt.Errorf("weeks must run from 1 to 53, first week first")
	}
	return from, to, nil
}

// weekOneMonday is the Monday starting ISO week 1 of the year (the week that holds January 4th).
func weekOneMonday(year int, loc *time.Location) time.Time {
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, loc)
	return jan4.AddDate(0, 0, -((int(jan4.Weekday()) + 6) % 7))
}

// WeekRange returns the start of the first week (a Monday) and the start of the day after the last
// week (the next Monday), so the period is [start, end).
func WeekRange(year, from, to int, loc *time.Location) (start, end time.Time, err error) {
	if year < 2000 || year > 2100 {
		return start, end, fmt.Errorf("year %d is out of range", year)
	}
	w1 := weekOneMonday(year, loc)
	end = w1.AddDate(0, 0, 7*to)
	if end.After(weekOneMonday(year+1, loc)) {
		return start, end, fmt.Errorf("%d has fewer than %d weeks", year, to)
	}
	return w1.AddDate(0, 0, 7*(from-1)), end, nil
}
