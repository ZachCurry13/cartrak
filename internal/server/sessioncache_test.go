package server

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestThePricedSessionsAreSharedBrieflyAndOnlyAsCopies(t *testing.T) {
	var c sessionsCache
	if _, _, ok := c.get(100); ok {
		t.Fatal("nothing is kept at first")
	}
	c.put(100, []charging.Session{{Charge: &store.Charge{ID: 1}}, {Charge: &store.Charge{ID: 2}}}, effInfo{Samples: 3})
	got, info, ok := c.get(100)
	if !ok || len(got) != 2 || info.Samples != 3 {
		t.Fatalf("a fresh answer is shared: %v %+v %v", got, info, ok)
	}
	got[0] = got[1] // a caller reorders its copy
	if again, _, _ := c.get(100); again[0].ID != 1 {
		t.Fatalf("the kept answer is not changed by a caller: %+v", again)
	}
	if _, _, ok := c.get(200); ok {
		t.Fatal("another limit is another answer")
	}
	c.mu.Lock()
	e := c.m[100]
	e.at = time.Now().Add(-2 * sessionsKeep)
	c.m[100] = e
	c.mu.Unlock()
	if _, _, ok := c.get(100); ok {
		t.Fatal("an old answer is not shared")
	}
	c.put(100, nil, effInfo{})
	c.drop()
	if _, _, ok := c.get(100); ok {
		t.Fatal("a dropped answer is not shared")
	}
}
