package googleplaces

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChargersAsksNearbyForChargersAndReadsFreePlugs(t *testing.T) {
	var mask, key string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mask, key = r.Header.Get("X-Goog-FieldMask"), r.Header.Get("X-Goog-Api-Key")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.Write([]byte(`{"places":[
 {"displayName":{"text":"Far Supercharger"},"formattedAddress":"2 Road","location":{"latitude":45.05,"longitude":-93.2},
  "evChargeOptions":{"connectorCount":8,"connectorAggregation":[{"type":"EV_CONNECTOR_TYPE_TESLA","maxChargeRateKw":250,"count":8,"availableCount":3,"outOfServiceCount":1,"availabilityLastUpdateTime":"2026-11-16T15:00:00Z"}]}},
 {"displayName":{"text":"Near Library"},"formattedAddress":"1 Main","location":{"latitude":45.001,"longitude":-93.2},
  "evChargeOptions":{"connectorCount":2,"connectorAggregation":[{"type":"EV_CONNECTOR_TYPE_J1772","maxChargeRateKw":7.2,"count":2},{"type":"EV_CONNECTOR_TYPE_CCS_COMBO_1","maxChargeRateKw":50,"count":1}]}}]}`))
	}))
	defer srv.Close()
	got, err := Chargers(context.Background(), nil, srv.URL, "k", 45, -93.2, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if key != "k" || !strings.Contains(mask, "evChargeOptions") || body["includedTypes"].([]any)[0] != "electric_vehicle_charging_station" {
		t.Fatalf("asks for chargers with their plugs: %q %v", mask, body)
	}
	if len(got) != 2 || got[0].Name != "Near Library" || got[0].Connectors[0].Type != "CCS" || got[0].Connectors[0].Available != nil {
		t.Fatalf("nearest first, fastest plug first, no count when none is reported: %+v", got)
	}
	c := got[1].Connectors[0]
	if c.Type != "Tesla (NACS)" || *c.Available != 3 || *c.OutOfService != 1 || c.Updated == 0 {
		t.Fatalf("free plugs read: %+v", c)
	}
}

func TestChargersSaysWhatGoogleSaid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":{"message":"API key not valid"}}`))
	}))
	defer srv.Close()
	if _, err := Chargers(context.Background(), nil, srv.URL, "k", 45, -93, 1000); err == nil || !strings.Contains(err.Error(), "API key not valid") {
		t.Fatalf("%v", err)
	}
}
