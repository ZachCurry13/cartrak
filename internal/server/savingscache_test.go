package server

import (
	"testing"
	"time"
)

func TestTheSavingsAnswerIsKeptBrieflyAndDroppedOnDemand(t *testing.T) {
	var c savingsCache
	if _, hit := c.get(); hit {
		t.Fatal("nothing is kept at first")
	}
	c.put(map[string]any{"a": 1})
	if b, hit := c.get(); !hit || b["a"] != 1 {
		t.Fatalf("a fresh answer is reused: %v %v", b, hit)
	}
	c.at = time.Now().Add(-2 * savingsKeep)
	if _, hit := c.get(); hit {
		t.Fatal("an old answer is not reused")
	}
	c.put(map[string]any{"a": 2})
	c.drop()
	if _, hit := c.get(); hit {
		t.Fatal("a dropped answer is not reused")
	}
}
