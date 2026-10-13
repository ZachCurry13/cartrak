package poller

import "github.com/ZachCurry13/cartrak/internal/ford"

// LearnCaps remembers what the car has told us about itself: the engine type from Ford's garage
// record, and which kinds of data it reports (battery, fuel, plug). The data kinds are only ever
// added to, so the vehicle type does not flip when one reading is thin.
func (p *Poller) LearnCaps(st *ford.State) {
	if et := ford.EngineTypeOf(st.Extra); et != "" && et != p.S.GetKV(ford.EngineKV) {
		_ = p.S.SetKV(ford.EngineKV, et)
	}
	seen := ford.ParseCaps(p.S.GetKV(ford.CapsKV))
	now := seen.Or(ford.CapsOf(st.Extra))
	if now != seen {
		_ = p.S.SetKV(ford.CapsKV, now.String())
	}
}
