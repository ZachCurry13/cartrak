package ford

import (
	"encoding/json"
	"fmt"
	"strings"
)

// When the Ford account behind a car's link authorized more than one vehicle, Ford's answer holds all of them. A car follows
// one: the one chosen for it (its VIN), or the first. The others are left out before anything is read, so two vehicles are
// never mixed, and they are listed so the owner can choose.

// VehicleChoice is a vehicle on the linked Ford account.
type VehicleChoice struct {
	VIN   string `json:"vin"`
	Label string `json:"label"` // its nickname or model when Ford says it, else "Vehicle ending 5678"
}

// SeenVINKey is where the car remembers which vehicle it follows, so the other cars can tell it is taken.
const SeenVINKey = "ford_vin_seen"

func vinTail(vin string) string {
	if len(vin) > 6 {
		return vin[len(vin)-6:]
	}
	return vin
}

func labelOf(v map[string]any) string {
	for _, k := range []string{"nickName", "nickname", "vehicleNickname", "modelName", "model"} {
		if s, ok := v[k].(string); ok && strings.TrimSpace(s) != "" {
			name := strings.TrimSpace(s)
			if y, ok := v["modelYear"]; ok {
				name = fmt.Sprintf("%v %s", y, name)
			}
			return name
		}
	}
	return ""
}

// pickVehicle keeps one vehicle of a telemetry answer: the one with VIN want, or the first. It returns the answer to read, every
// vehicle in it, and the VIN followed.
func pickVehicle(tel []byte, want string) ([]byte, []VehicleChoice, string) {
	var doc map[string]any
	if json.Unmarshal(tel, &doc) != nil {
		return tel, nil, ""
	}
	vs, _ := doc["vehicles"].([]any)
	if len(vs) == 0 {
		return tel, nil, ""
	}
	list := make([]VehicleChoice, 0, len(vs))
	pick := 0
	for i, x := range vs {
		v, _ := x.(map[string]any)
		vin, _ := v["vin"].(string)
		label := labelOf(v)
		if label == "" {
			label = "Vehicle ending " + vinTail(vin)
		}
		list = append(list, VehicleChoice{VIN: vin, Label: label})
		if want != "" && strings.EqualFold(vin, want) {
			pick = i
		}
	}
	if len(vs) == 1 {
		return tel, list, list[0].VIN
	}
	doc["vehicles"] = []any{vs[pick]}
	out, err := json.Marshal(doc)
	if err != nil {
		return tel, list, list[pick].VIN
	}
	return out, list, list[pick].VIN
}

// Vehicles is every vehicle the linked account authorized, as last seen.
func (c *Connect2) Vehicles() []VehicleChoice {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]VehicleChoice(nil), c.vehicles...)
}

// noteVehicles remembers the vehicles of the latest answer and which one this car follows.
func (c *Connect2) noteVehicles(list []VehicleChoice, chosen string) {
	c.mu.Lock()
	c.vehicles = list
	c.mu.Unlock()
	if chosen != "" && c.store.GetKV(SeenVINKey) != chosen {
		_ = c.store.SetKV(SeenVINKey, chosen)
	}
}
