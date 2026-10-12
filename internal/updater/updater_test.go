package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		l, c string
		want bool
	}{
		{"0.2.0", "0.1.0", true}, {"0.1.1", "0.1.0", true}, {"1.0.0", "0.9.9", true},
		{"0.1.0", "0.1.0", false}, {"0.1.0", "0.2.0", false}, {"0.10.0", "0.9.0", true},
		{"0.2.0", "dev", false}, {"0.2.0", "main", false}, {"garbage", "0.1.0", false},
		{"v0.2.0", "0.1.0", true}, {"0.2.0", "0.2.0-rc1", false},
	}
	for _, c := range cases {
		if got := Newer(c.l, c.c); got != c.want {
			t.Errorf("Newer(%q,%q)=%v want %v", c.l, c.c, got, c.want)
		}
	}
}

func TestCheck(t *testing.T) {
	status := 200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(`{"tag_name":"v0.3.0","html_url":"https://example.test/r"}`))
	}))
	defer srv.Close()
	c := New("0.2.0")
	c.URL = srv.URL
	if err := c.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if i := c.Info(); !i.Available || i.Latest != "0.3.0" || i.URL != "https://example.test/r" || i.Error != "" {
		t.Fatalf("%+v", i)
	}
	status = 403 // rate limited: keep the last good answer, record the error
	if err := c.Check(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if i := c.Info(); i.Error == "" || i.Latest != "0.3.0" {
		t.Fatalf("%+v", i)
	}
}
