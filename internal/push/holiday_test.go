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

func TestHolidayVoicesChangeTheWordsNotTheFacts(t *testing.T) {
	done := Alert{Kind: ChargeComplete, Title: "Charging complete", Body: "Battery at 80%."}
	a := TouchWith(done, "", day(2026, time.December, 24), true)
	if a.Title != "🎄 Ho ho ho! Fully charged" || a.Body != "Battery at 80%." {
		t.Fatalf("Santa words the title, the facts stay: %+v", a)
	}
	other := TouchWith(Alert{Kind: Achievement, Title: "Achievement: Hypermiler", Body: "A good drive."}, "", day(2026, time.October, 31), true)
	if other.Title != "🎃 Achievement: Hypermiler" || other.Body != "Boo! (a friendly one)\nA good drive.\nHappy Halloween!" {
		t.Fatalf("an alert with no voice line of its own gets an opener: %+v", other)
	}
	if plain := TouchWith(done, "", day(2026, time.December, 24), false); plain.Title != "🎄 Charging complete" {
		t.Fatalf("voices are off unless asked for: %+v", plain)
	}
	if warn := TouchWith(Alert{Kind: LowTire, Title: "Low tire pressure"}, "", day(2026, time.December, 24), true); warn.Title != "Low tire pressure" {
		t.Fatalf("warnings never use a voice: %+v", warn)
	}
	if b := TouchWith(done, "12-24", day(2026, time.December, 24), true); b.Title != "🎂 Charging complete" {
		t.Fatalf("a birthday beats the holiday: %+v", b)
	}
}

func TestTheShamrockVoiceIsFound(t *testing.T) {
	a := TouchWith(Alert{Kind: TripEnd, Title: "Trip ended", Body: "12 mi"}, "", day(2026, time.March, 17), true)
	if a.Title != "☘️ Home safe, bless ye: trip ended" {
		t.Fatalf("a leprechaun: %+v", a)
	}
}
