package ford

import (
	"os"
	"testing"
	"time"
)

func TestLastReportedIsTheNewestUpdateTimeOfAnyValue(t *testing.T) {
	raw, err := os.ReadFile("testdata/bev_response.json")
	if err != nil {
		t.Fatal(err)
	}
	got := LastReported(Flatten(raw))
	if want := time.Date(2026, 10, 9, 14, 17, 32, 0, time.UTC).Unix(); got < want {
		t.Fatalf("the real reading's newest update is on or after %d, got %d", want, got)
	}
	f := map[string]any{"a.updateTime": "2026-01-01T00:00:00Z", "b.updateTime": "2026-03-05T10:00:00Z", "c.value": "2027-01-01T00:00:00Z", "d.updateTime": 5.0, "e.updateTime": "2999-01-01T00:00:00Z"}
	if LastReported(f) != time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC).Unix() {
		t.Fatalf("only update times count, garbage and far-future times are ignored: %d", LastReported(f))
	}
	if LastReported(map[string]any{"x": 1}) != 0 {
		t.Fatal("none reported")
	}
}
