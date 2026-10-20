package server

import (
	"fmt"
	"testing"
	"time"
)

func TestTheLockoutBlocksAfterFiveAndForgetsQuietAddresses(t *testing.T) {
	var l limiter
	l.fail("1.2.3.4") // a failure before any check must not crash
	for i := 0; i < 4; i++ {
		l.fail("1.2.3.4")
	}
	if !l.blocked("1.2.3.4") {
		t.Fatal("five failures block the address")
	}
	if l.blocked("5.6.7.8") {
		t.Fatal("another address is not blocked")
	}
	if _, kept := l.fails["5.6.7.8"]; kept {
		t.Fatal("an address with no failures is not remembered")
	}
	// many addresses that failed long ago are dropped once the list is large
	old := time.Now().Add(-2 * time.Minute)
	for i := 0; i < 1100; i++ {
		l.fails[fmt.Sprintf("10.0.%d.%d", i/250, i%250)] = []time.Time{old}
	}
	l.blocked("1.2.3.4")
	if len(l.fails) > 5 {
		t.Fatalf("quiet addresses are forgotten: %d left", len(l.fails))
	}
}
