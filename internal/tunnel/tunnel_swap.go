package tunnel

import "time"

// Program is the connector program that will be started next.
func (s *Supervisor) Program() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Bin
}

// SetProgram chooses the connector program for the next start. It does not restart anything by itself.
func (s *Supervisor) SetProgram(path string) {
	s.mu.Lock()
	s.Bin = path
	s.mu.Unlock()
}

// Running reports whether the connector is switched on (starting, connected or reconnecting).
func (s *Supervisor) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancel != nil
}

// Restart starts the running connector again with the same token and whatever program is now chosen.
// It does nothing when the connector is off.
func (s *Supervisor) Restart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel == nil {
		return
	}
	token := s.token
	s.stopLocked()
	s.startLocked(token)
}

// WaitConnected waits up to d for the connector to report it is connected.
func (s *Supervisor) WaitConnected(d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if s.Status().State == Connected {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return s.Status().State == Connected
}
