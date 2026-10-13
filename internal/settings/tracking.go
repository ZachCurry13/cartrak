package settings

import (
	"fmt"
	"strconv"
)

// KeepDaysKey is how long phone GPS points are kept; empty or 0 keeps them for ever.
const KeepDaysKey = "CARTRAK_TRACK_KEEP_DAYS"

func init() {
	Defs = append(Defs, Def{Key: KeepDaysKey, Label: "Delete phone GPS points after (days)", Placeholder: "empty keeps everything",
		Help: "Optional cleanup for the points phones send. Trips and their summaries are always kept; only the fine route points are removed."})
	validators[KeepDaysKey] = func(val string) error {
		if val == "" {
			return nil
		}
		n, err := strconv.Atoi(val)
		if err != nil || n < 0 || n > 3650 {
			return fmt.Errorf("enter a number of days from 1 to 3650, or leave it empty")
		}
		return nil
	}
}
