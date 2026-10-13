package ford

import (
	"sort"
	"strconv"
	"strings"
)

// listItems returns the items of a list metric (for example "doorLockStatus" or
// "tirePressure") from a flattened document. Each item is a map of its fields,
// with field names lower-cased. Only the first vehicle's list is read.
func listItems(f map[string]any, metric string) []map[string]any {
	needle := norm(metric) + "["
	byIdx := map[int]map[string]any{}
	for p, v := range f {
		lp := norm(p)
		i := strings.Index(lp, needle)
		if i < 0 || (i > 0 && lp[i-1] != '.') {
			continue
		}
		rest := lp[i+len(needle):]
		end := strings.Index(rest, "]")
		if end < 0 {
			continue
		}
		idx, err := strconv.Atoi(rest[:end])
		if err != nil {
			continue
		}
		field := strings.TrimPrefix(rest[end+1:], ".")
		if field == "" {
			continue
		}
		if byIdx[idx] == nil {
			byIdx[idx] = map[string]any{}
		}
		byIdx[idx][field] = v
	}
	keys := make([]int, 0, len(byIdx))
	for k := range byIdx {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, byIdx[k])
	}
	return out
}

// tireSlot maps a wheel name such as FRONT_LEFT to 0..3 (FL, FR, RL, RR), or -1.
func tireSlot(wheel string) int {
	w := strings.ToUpper(wheel)
	front, rear := strings.Contains(w, "FRONT"), strings.Contains(w, "REAR")
	left, right := strings.Contains(w, "LEFT"), strings.Contains(w, "RIGHT")
	switch {
	case front && left:
		return 0
	case front && right:
		return 1
	case rear && left:
		return 2
	case rear && right:
		return 3
	}
	return -1
}

// parseTires fills the four tire pressures (kPa) from the tirePressure list.
func parseTires(f map[string]any, s *State) {
	for _, it := range listItems(f, "tirePressure") {
		wheel, _ := it["vehiclewheel"].(string)
		v, ok := num(it["value"])
		if slot := tireSlot(wheel); ok && slot >= 0 {
			s.Tires[slot] = ptr(v)
		}
	}
}

// parseLocks reads the doorLockStatus list: locked when every known door is locked
// (or ALL_DOORS is), unlocked when any door is unlocked. Unknown doors are ignored.
func parseLocks(f map[string]any, s *State) {
	items := listItems(f, "doorLockStatus")
	if len(items) == 0 {
		return
	}
	locked, unlocked := 0, 0
	for _, it := range items {
		val, _ := it["value"].(string)
		door, _ := it["vehicledoor"].(string)
		switch strings.ToUpper(val) {
		case "LOCKED", "DOUBLE_LOCKED":
			if strings.EqualFold(door, "ALL_DOORS") {
				s.Locked = ptr(true)
				return
			}
			locked++
		case "UNLOCKED":
			unlocked++
		}
	}
	switch {
	case unlocked > 0:
		s.Locked = ptr(false)
	case locked > 0:
		s.Locked = ptr(true)
	}
}
