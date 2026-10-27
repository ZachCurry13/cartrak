package gasprice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAreasInUseAreRefreshedOnASchedule(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Write([]byte(body))
	}))
	defer srv.Close()
	clock := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	kv := memKV{}
	s := &Service{KV: kv, BaseURL: srv.URL, Now: func() time.Time { return clock }}
	mn := ForState("MN")
	if err := s.Refresh(context.Background(), "k", mn); err != nil {
		t.Fatal(err)
	}
	s.At("k", mn, clock.Unix()) // a price was asked for, so the area is in use
	hits = 0

	s.RefreshKnown(context.Background(), "k")
	if hits != 0 {
		t.Fatalf("prices fetched an hour ago are fresh: %d requests", hits)
	}
	clock = clock.Add(13 * time.Hour)
	s2 := &Service{KV: kv, BaseURL: srv.URL, Now: func() time.Time { return clock }} // a restart forgets nothing
	s2.RefreshKnown(context.Background(), "k")
	if hits != 1 {
		t.Fatalf("one request for the one area in use, after 12 hours: %d", hits)
	}
	if st := s2.Status(); st.OK != clock.Unix() || st.Err != "" {
		t.Fatalf("status: %+v", st)
	}
	if strings.Contains(kv[areasKey], ",,") || strings.HasPrefix(kv[areasKey], ",") || kv[areasKey] != "SMN" {
		t.Fatalf("the list of areas in use: %q", kv[areasKey])
	}
}

func TestAFailedFetchIsRememberedWithoutTheKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer srv.Close()
	s := &Service{KV: memKV{}, BaseURL: srv.URL, Now: func() time.Time { return time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC) }}
	if err := s.Refresh(context.Background(), "secret-key", ForState("MN")); err == nil {
		t.Fatal("a refused key is an error")
	}
	st := s.Status()
	if st.OK != 0 || st.Tried == 0 || !strings.Contains(st.Err, "refused") || strings.Contains(st.Err, "secret-key") {
		t.Fatalf("status: %+v", st)
	}
}
