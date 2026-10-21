package push

import "time"

// Holiday touches: around a holiday the friendly alerts (a trip ended, charging done, a badge, the monthly Wrapped) start with
// the holiday's emoji, and on the day itself they also wish a happy holiday. On a person's birthday it is a cake and a happy
// birthday instead. Warnings stay plain. The dates match the app's seasonal themes (js/seasons.js), without the seasons.

// friendly are the alerts that may carry a holiday touch.
var friendly = map[string]bool{ChargeComplete: true, PlugIn: true, PlugOut: true, TripStart: true, TripEnd: true, ArrivePlace: true,
	LeavePlace: true, WeeklySummary: true, MonthlyRecap: true, Achievement: true, ServiceVisit: true}

type holiday struct {
	Emoji    string
	Greeting string // said on the day itself; empty outside it
}

func md(m time.Month, d int) int { return int(m)*100 + d }

// easterMD is Easter Sunday (Gregorian) as month*100 + day.
func easterMD(y int) int {
	a, b, c := y%19, y/100, y%100
	d, e, f := b/4, b%4, (b+8)/25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i, k := c/4, c%4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	return ((h+l-7*m+114)/31)*100 + (h+l-7*m+114)%31 + 1
}

// thanksgivingMD is the fourth Thursday of November.
func thanksgivingMD(y int) int {
	first := int(time.Date(y, time.November, 1, 0, 0, 0, 0, time.UTC).Weekday())
	return md(time.November, 1+(4-first+7)%7+21)
}

// holidayOn is the holiday around a date, if any.
func holidayOn(t time.Time) (holiday, bool) {
	y, now := t.Year(), md(t.Month(), t.Day())
	e := easterMD(y)
	eDay := time.Date(y, time.Month(e/100), e%100, 12, 0, 0, 0, time.UTC).AddDate(0, 0, -14)
	eFrom := md(eDay.Month(), eDay.Day())
	tg := thanksgivingMD(y)
	on := func(emoji string, day int, greeting string) (holiday, bool) {
		if now != day {
			greeting = ""
		}
		return holiday{emoji, greeting}, true
	}
	switch {
	case now == 1231:
		return on("🎉", 1231, "Happy New Year's Eve!")
	case now == 101:
		return on("🎉", 101, "Happy New Year!")
	case now >= 201 && now <= 214:
		return on("💕", 214, "Happy Valentine's Day!")
	case now >= 301 && now <= 317:
		return on("☘️", 317, "Happy St. Patrick's Day!")
	case now >= eFrom && now <= e:
		return on("🐣", e, "Happy Easter!")
	case now >= 701 && now <= 704:
		return on("🎆", 704, "Happy Fourth of July!")
	case now >= 1001 && now <= 1031:
		return on("🎃", 1031, "Happy Halloween!")
	case now >= 1101 && now <= tg:
		return on("🦃", tg, "Happy Thanksgiving!")
	case now > tg && now <= 1226:
		return on("🎄", 1225, "Merry Christmas!")
	}
	return holiday{}, false
}

// IsBirthday says whether "MM-DD" is the date (29 February is kept on the 28th in other years).
func IsBirthday(birthday string, t time.Time) bool {
	if birthday == "" {
		return false
	}
	if birthday == "02-29" && t.Month() == time.February && t.Day() == 28 && time.Date(t.Year(), 3, 0, 0, 0, 0, 0, time.UTC).Day() == 28 {
		return true
	}
	return birthday == t.Format("01-02")
}

// Touch gives an alert its holiday or birthday touch for one person, when it is a friendly one.
func Touch(a Alert, birthday string, now time.Time) Alert {
	if !friendly[a.Kind] {
		return a
	}
	if IsBirthday(birthday, now) {
		a.Title = "🎂 " + a.Title
		a.Body = joinLine(a.Body, "Happy birthday!")
		return a
	}
	if h, ok := holidayOn(now); ok {
		a.Title = h.Emoji + " " + a.Title
		a.Body = joinLine(a.Body, h.Greeting)
	}
	return a
}

func joinLine(body, extra string) string {
	if extra == "" {
		return body
	}
	if body == "" {
		return extra
	}
	return body + "\n" + extra
}
