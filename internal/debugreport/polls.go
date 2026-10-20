package debugreport

import (
	"fmt"
	"sort"
	"strings"
)

// pollSummary says how the polls went: the pace CarTrak actually kept, any gaps, and the errors.
func pollSummary(ps []PollInfo, at func(int64) string) string {
	if len(ps) == 0 {
		return "(no polls recorded yet since CarTrak started)\n"
	}
	var b strings.Builder
	ok := 0
	byWait := map[int]int{}
	var gaps []int64
	var prevOK int64
	var errs []PollInfo
	for _, p := range ps {
		if p.Result == "ok" {
			ok++
			byWait[p.NextSec]++
			if prevOK > 0 {
				gaps = append(gaps, p.TS-prevOK)
			}
			prevOK = p.TS
		} else {
			errs = append(errs, p)
		}
	}
	fmt.Fprintf(&b, "%d polls from %s to %s: %d answered, %d did not\n", len(ps), at(ps[0].TS), at(ps[len(ps)-1].TS), ok, len(errs))
	var waits []int
	for w := range byWait {
		waits = append(waits, w)
	}
	sort.Ints(waits)
	for _, w := range waits {
		fmt.Fprintf(&b, "    next poll planned %d s after: %d times\n", w, byWait[w])
	}
	if len(gaps) > 0 {
		sort.Slice(gaps, func(i, j int) bool { return gaps[i] < gaps[j] })
		fmt.Fprintf(&b, "    actual time between answered polls: shortest %d s, middle %d s, longest %d s\n", gaps[0], gaps[len(gaps)/2], gaps[len(gaps)-1])
	}
	if len(errs) > 12 {
		errs = errs[len(errs)-12:]
	}
	for _, e := range errs {
		fmt.Fprintf(&b, "    %s | %s | next try in %d s | %s\n", at(e.TS), e.Result, e.NextSec, e.Detail)
	}
	return b.String()
}
