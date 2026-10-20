package fleet

// Front is the first car's runtime, which answers for the whole installation (sign-in, people, settings of the site).
func (f *Fleet) Front() *Runtime {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var best *Runtime
	for _, r := range f.cars {
		if best == nil || r.Car.ID < best.Car.ID {
			best = r
		}
	}
	return best
}

// Runtime returns one car's runtime, or nil.
func (f *Fleet) Runtime(id int64) *Runtime {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.cars[id]
}

// Close stops the shared services and closes every database.
func (f *Fleet) Close() {
	f.Tun.Stop()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.cars {
		if r.cancel != nil {
			r.cancel()
		}
		_ = r.S.Close()
	}
	_ = f.Core.Close()
}
