package server

import (
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/smooth"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/tripwhy"
)

// carCaches are answers worked out from a car's history and kept for a while. They live on the server of one car, never
// in the package, so two cars can never be given each other's numbers.
type carCaches struct {
	truthMu    sync.Mutex
	truthAt    time.Time
	truthCache []store.Day
	rangeCache []store.Day // the range the car expects on a full charge, the same days

	smoothMu   sync.Mutex
	smoothAt   time.Time
	smoothBase smooth.Baseline

	whyMu   sync.Mutex
	whyAt   time.Time
	whyBase tripwhy.Baseline

	reasonsMu   sync.Mutex
	reasonsAt   time.Time
	reasonsDays int
	reasonsOut  map[string]any
}
