package push

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 12, 0, 0, 0, time.UTC) }

func TestHolidayCalendar(t *testing.T) {
	cases := []struct {
		at              time.Time
		emoji, greeting string
		none            bool
	}{
		{at: day(2026, time.October, 21), emoji: "🎃"},
		{at: day(2026, time.October, 31), emoji: "🎃", greeting: "Happy Halloween!"},
		{at: day(2026, time.November, 26), emoji: "🦃", greeting: "Happy Thanksgiving!"}, // fourth Thursday
		{at: day(2026, time.November, 27), emoji: "🎄"},
		{at: day(2026, time.December, 25), emoji: "🎄", greeting: "Merry Christmas!"},
		{at: day(2026, time.April, 5), emoji: "🐣", greeting: "Happy Easter!"},
		{at: day(2027, time.January, 1), emoji: "🎉", greeting: "Happy New Year!"},
		{at: day(2026, time.August, 10), none: true}, // a season, not a holiday
		{at: day(2026, time.December, 28), none: true},
	}
	for _, c := range cases {
		h, ok := holidayOn(c.at)
		if ok == c.none || h.Emoji != c.emoji || h.Greeting != c.greeting {
			t.Errorf("%s: got %+v %v", c.at.Format("2006-01-02"), h, ok)
		}
	}
}

func TestTouch(t *testing.T) {
	trip := Alert{Kind: TripEnd, Title: "Trip ended", Body: "12 mi"}
	if a := Touch(trip, "", day(2026, time.October, 31)); a.Title != "🎃 Trip ended" || a.Body != "12 mi\nHappy Halloween!" {
		t.Fatalf("halloween: %+v", a)
	}
	if a := Touch(trip, "10-31", day(2026, time.October, 31)); a.Title != "🎂 Trip ended" || a.Body != "12 mi\nHappy birthday!" {
		t.Fatalf("birthday wins: %+v", a)
	}
	warn := Alert{Kind: LowTire, Title: "Low tire pressure"}
	if a := Touch(warn, "10-31", day(2026, time.October, 31)); a != warn {
		t.Fatalf("warnings stay plain: %+v", a)
	}
	if a := Touch(trip, "", day(2026, time.August, 10)); a != trip {
		t.Fatalf("no holiday: %+v", a)
	}
	if !IsBirthday("02-29", day(2027, time.February, 28)) || IsBirthday("02-29", day(2028, time.February, 28)) {
		t.Fatal("29 February is kept on the 28th only in other years")
	}
}
