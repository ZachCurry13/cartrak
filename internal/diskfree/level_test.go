package diskfree

import "testing"

func TestTheDiskLevelWarnsEarlyAndShoutsWhenNearlyFull(t *testing.T) {
	const g = 1 << 30
	for _, c := range []struct {
		free, total uint64
		want        string
	}{{500 << 20, 100 * g, "bad"}, {3 * g, 100 * g, "warn"}, {20 * g, 400 * g, "warn"}, {20 * g, 200 * g, "ok"}, {50 * g, 100 * g, "ok"}} {
		if got := Level(c.free, c.total); got != c.want {
			t.Errorf("%d of %d: %s, want %s", c.free/g, c.total/g, got, c.want)
		}
	}
}
