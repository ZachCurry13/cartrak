package diskfree

const (
	gb       = 1 << 30
	badBelow = 1 * gb
	lowBelow = 5 * gb
	lowShare = 0.08
)

// Level is "ok", "warn" (getting low) or "bad" (nearly full) for a disk with this much free of this much.
func Level(free, total uint64) string {
	switch {
	case free < badBelow:
		return "bad"
	case free < lowBelow || (total > 0 && float64(free)/float64(total) < lowShare):
		return "warn"
	}
	return "ok"
}
