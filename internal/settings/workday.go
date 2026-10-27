package settings

import (
	"fmt"
	"strconv"
	"strings"
)

// The work day rule: the first outing of a work day, from leaving Home until the car is back, is work whatever the clock says.
const (
	WorkDayKey     = "CARTRAK_WORKDAY"      // "on" to use the rule
	WorkDaysKey    = "CARTRAK_WORKDAY_DAYS" // the work days: 1 to 7 for Monday to Sunday, such as "1,2,3,4,5"
	WorkDayStayKey = "CARTRAK_WORKDAY_STAY" // minutes at Home that still count as the same work day
)

const defaultWorkDays = "1,2,3,4,5"

func init() {
	Defs = append(Defs,
		Def{Key: WorkDayKey, Label: "Work days: the first outing is work", Group: "workday", Placeholder: "on",
			Help: "On a work day, everything from the first time the car leaves Home until it is back is work, whatever the time. Later outings that day are personal. Needs a place named Home."},
		Def{Key: WorkDaysKey, Label: "Work days", Group: "workday", Placeholder: defaultWorkDays,
			Help: "The days that are work days, as numbers: 1 Monday to 7 Sunday. Monday to Friday is 1,2,3,4,5."},
		Def{Key: WorkDayStayKey, Label: "Minutes at Home that do not end the work day", Group: "workday", Placeholder: "45",
			Help: "Getting home for lunch and going back out within this many minutes is still the same work day."})
	validators[WorkDaysKey] = func(v string) error {
		_, err := parseWorkDays(v)
		return err
	}
	validators[WorkDayStayKey] = func(v string) error {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); v != "" && (err != nil || n < 0 || n > 600) {
			return fmt.Errorf("use a number of minutes from 0 to 600")
		}
		return nil
	}
}

// parseWorkDays turns "1,2,3,4,5" into the weekday bits of a rule (Sunday = 1).
func parseWorkDays(v string) (int, error) {
	bits := 0
	for _, f := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' }) {
		n, err := strconv.Atoi(f)
		if err != nil || n < 1 || n > 7 {
			return 0, fmt.Errorf("write the days as numbers from 1 (Monday) to 7 (Sunday), such as 1,2,3,4,5")
		}
		bits |= 1 << uint(n%7) // Sunday (7) is bit 0, like time.Weekday
	}
	return bits, nil
}

// WorkDay is the work day rule: whether it is on, the weekday bits (Sunday = 1) and the minutes at Home that do not end the day.
func (s *Settings) WorkDay() (on bool, days int, stayMin int) {
	v := s.Value(WorkDaysKey)
	if v == "" {
		v = defaultWorkDays
	}
	days, _ = parseWorkDays(v)
	stayMin = 45
	if n, err := strconv.Atoi(strings.TrimSpace(s.Value(WorkDayStayKey))); err == nil && n >= 0 {
		stayMin = n
	}
	return s.Value(WorkDayKey) == "on", days, stayMin
}
