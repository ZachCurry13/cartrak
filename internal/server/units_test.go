package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMessagesFollowTheUnitsTheDeviceSends(t *testing.T) {
	imperial := httptest.NewRequest("GET", "/", nil)
	metric := httptest.NewRequest("GET", "/", nil)
	metric.Header.Set("X-Units", "metric")
	if metricReq(imperial) || !metricReq(metric) {
		t.Fatal("miles are the default, kilometres only when asked")
	}
	f := routeFailure{RoadKm: 18, OdoKm: 9}
	if got := f.text(false); got != "the road found was 11.2 mi but the odometer says 5.6 mi" {
		t.Fatal(got)
	}
	if got := f.text(true); got != "the road found was 18.0 km but the odometer says 9.0 km" {
		t.Fatal(got)
	}
	if speedText(100, false) != "62 mph" || distText(160.9344, false, 0) != "100 mi" {
		t.Fatal("conversion")
	}
	in := maintIn{EveryKm: new(float64)}
	*in.EveryKm = 5
	if msg := in.valid(false); !strings.Contains(msg, "miles") || strings.Contains(msg, " km") {
		t.Fatal(msg)
	}
}
