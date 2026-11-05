package googleplaces

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchAsksGoogleWithTheKeyAndTheAreaAndReadsThePlaces(t *testing.T) {
	var gotKey, gotMask string
	var body map[string]any
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotMask = r.Header.Get("X-Goog-Api-Key"), r.Header.Get("X-Goog-FieldMask")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.Write([]byte(`{"places":[{"displayName":{"text":"Acme Widgets"},"formattedAddress":"1 Main St, Townsville, MN 55555, USA","location":{"latitude":44.95,"longitude":-93.25},"primaryTypeDisplayName":{"text":"Wholesaler"}}]}`))
	}))
	defer svc.Close()
	home := [2]float64{44.9, -93.2}
	got, err := Search(context.Background(), nil, svc.URL, "test-key", "Acme Widgets, Inc.", "US", &home)
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	if gotKey != "test-key" || gotMask == "" || body["textQuery"] != "Acme Widgets, Inc." || body["regionCode"] != "US" || body["locationBias"] == nil {
		t.Fatalf("what was asked: key %q mask %q body %v", gotKey, gotMask, body)
	}
	if got[0].Name != "Acme Widgets" || got[0].Lat != 44.95 || got[0].DistKm == nil || *got[0].DistKm > 20 || got[0].Kind != "wholesaler" {
		t.Fatalf("the place: %+v", got[0])
	}
}

func TestAnErrorFromGoogleSaysWhy(t *testing.T) {
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":{"message":"API key not valid"}}`))
	}))
	defer svc.Close()
	if _, err := Search(context.Background(), nil, svc.URL, "bad", "Acme", "", nil); err == nil || err.Error() != "Google answered: API key not valid" {
		t.Fatalf("%v", err)
	}
	if _, err := Search(context.Background(), nil, svc.URL, "", "Acme", "", nil); err == nil {
		t.Fatal("no key is refused before anything is sent")
	}
}
