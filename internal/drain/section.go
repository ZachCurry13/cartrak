package drain

import (
	"fmt"
	"math"
	"time"

	"github.com/ZachCurry13/cartrak/internal/vhealth"
)

func rateRow(label string, r Rate, capacityKwh *float64) vhealth.Row {
	row := vhealth.Row{Label: label, Kind: "text"}
	if !r.Enough {
		row.Text, row.State = "Not enough parked time yet", "info"
		row.Note = "CarTrak needs about half a day of parked readings in this period."
		return row
	}
	row.Text = fmt.Sprintf("%.1f%% a day", r.PctPerDay)
	if capacityKwh != nil && *capacityKwh > 0 {
		row.Text += fmt.Sprintf(" (about %.1f kWh)", r.PctPerDay/100**capacityKwh)
	}
	switch {
	case r.PctPerDay >= 4:
		row.State, row.Note = "bad", "Much higher than most electric cars. Things that keep the car awake include Sentry or similar modes, scheduled preconditioning, a weak 12 V battery and frequent app wake-ups."
	case r.PctPerDay >= 2:
		row.State, row.Note = "warn", "A little higher than usual. Many electric cars lose about 1% a day or less when left alone."
	default:
		row.State, row.Note = "ok", "Normal. Many electric cars lose about 1% a day or less when left alone."
	}
	return row
}

func span(h float64) string {
	if h >= 48 {
		return fmt.Sprintf("%d days %d h", int(h)/24, int(math.Round(h))%24)
	}
	return fmt.Sprintf("%d h", int(math.Round(h)))
}

// Section is the Parked drain block of the Health page.
func Section(rep Report, capacityKwh *float64, loc *time.Location) vhealth.Section {
	sec := vhealth.Section{ID: "drain", Title: "Parked drain",
		Intro: "How much charge the car loses while it sits, not driving and not charging. A little is normal, because the car keeps its computers and 12 V battery alive. Ford updates the battery level only when the car wakes, so treat this as a guide."}
	sec.Rows = append(sec.Rows, rateRow("Last 7 days", rep.Week, capacityKwh), rateRow("Last 30 days", rep.Month, capacityKwh))
	for _, s := range rep.Stretches {
		sec.Rows = append(sec.Rows, vhealth.Row{
			Label: time.Unix(s.From, 0).In(loc).Format("Mon Jan 2, 3:04 PM"), Kind: "text",
			Text: fmt.Sprintf("parked %s, lost %.1f%%", span(s.Hours), s.DropPct), State: "info"})
	}
	return sec
}
