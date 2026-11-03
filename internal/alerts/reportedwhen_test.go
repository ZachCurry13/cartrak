package alerts

import (
	"strings"
	"testing"
	"time"
)

func TestReportedWhenUsesAClockTimeForAFreshReportAndAnAgeForAnOldOne(t *testing.T) {
	e, _, _ := setup(t)
	at := func(ago time.Duration) string { return e.reportedWhen(e.now().Add(-ago).UTC().Format(time.RFC3339)) }
	if got := at(30 * time.Minute); !strings.HasPrefix(got, "at ") {
		t.Errorf("30 minutes ago is a clock time: %q", got)
	}
	if got := at(15 * time.Hour); got != "15 hours ago" {
		t.Errorf("15 hours: %q", got)
	}
	if got := at(3 * 24 * time.Hour); got != "3 days ago" {
		t.Errorf("3 days: %q", got)
	}
	if got := e.reportedWhen("not a time"); got != "" {
		t.Errorf("an unreadable time says nothing: %q", got)
	}
}
