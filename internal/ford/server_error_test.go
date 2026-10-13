package ford

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFordServerErrorShowsWhatFordSaid(t *testing.T) {
	f := newFake2(t)
	f.telemetryErr = 500
	c := f.client(memKV{refreshKey: "R0"})

	_, err := c.Fetch(context.Background())
	var se *ServerError
	if !errors.As(err, &se) || se.Status != 500 || se.Path != "/telemetry" {
		t.Fatalf("an HTTP 500 should be a ServerError, got %T %v", err, err)
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "ford /telemetry: HTTP 500") || !strings.Contains(msg, "Internal error talking to the vehicle service") {
		t.Fatalf("the message should keep its familiar start and add what Ford said: %q", msg)
	}
	if strings.Contains(msg, "1FMTK1234ABCD5678") {
		t.Fatalf("a VIN must never appear in an error: %q", msg)
	}
}

func TestClientErrorsAreNotServerErrors(t *testing.T) {
	f := newFake2(t)
	f.telemetryErr = 403
	_, err := f.client(memKV{refreshKey: "R0"}).Fetch(context.Background())
	var se *ServerError
	if err == nil || errors.As(err, &se) || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("an HTTP 403 is not Ford being down: %v", err)
	}
}

func TestErrorDetailIsShortAndSafe(t *testing.T) {
	for in, want := range map[string]string{
		``: ``, `   `: ``,
		`{"message":"Service unavailable","status":503}`:             "message: Service unavailable, status: 503",
		`<html><body>Bad  gateway   1FMTK1234ABCD5678</body></html>`: "<html><body>Bad gateway [vin]</body></html>",
		`{"unknown":"shape"}`: `{"unknown":"shape"}`,
	} {
		if got := errorDetail([]byte(in)); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
	if got := errorDetail([]byte(strings.Repeat("x", 500))); len(got) > 210 {
		t.Errorf("long answers are cut: %d characters", len(got))
	}
}
