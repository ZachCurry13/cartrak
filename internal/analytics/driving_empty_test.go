package analytics

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// The Insights page reads these as lists, so a report with no trips must send empty lists, never null: a null here was the cause of
// "Cannot read properties of null (reading 'reduce')" on the page.
func TestAnEmptyDrivingReportHasEmptyListsNotNull(t *testing.T) {
	b, err := json.Marshal(Driving(nil, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{`"temp":null`, `"speed":null`, `"lengths":null`, `"days":null`} {
		if strings.Contains(string(b), f) {
			t.Fatalf("%s in %s", f, b)
		}
	}
}
