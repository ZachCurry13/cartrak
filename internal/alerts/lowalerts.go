package alerts

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/push"
)

const (
	rearmPoints  = 5.0
	kvBudgetSent = "alert_budget_month"
)

// lowBattery warns once when the battery falls to the chosen level while the car is not charging. It resets
// when the battery is five points higher or the car starts charging. A restart while already low stays quiet.
func (e *Engine) lowBattery(st *ford.State, first bool) {
	if e.LowBatteryLevel == nil || st.SoCPct == nil {
		return
	}
	level := e.LowBatteryLevel()
	if level <= 0 {
		e.lowSent = false
		return
	}
	soc := *st.SoCPct
	switch {
	case charging(st) || soc >= level+rearmPoints:
		e.lowSent = false
	case soc <= level && !e.lowSent:
		e.lowSent = true
		if first {
			return
		}
		e.send(push.Alert{Kind: push.LowBattery, Title: "Low battery", Body: fmt.Sprintf("The battery is at %.0f%%.", soc), URL: "#/"})
	}
}

// budget warns once a month when the month's charging cost passes the budget.
func (e *Engine) budget() {
	if e.ChargeBudget == nil || e.MonthCost == nil {
		return
	}
	limit := e.ChargeBudget()
	if limit <= 0 {
		return
	}
	cost, ok := e.MonthCost()
	if !ok || cost < limit {
		return
	}
	month := e.now().In(e.zone()).Format("2006-01")
	if e.S.GetKV(kvBudgetSent) == month {
		return
	}
	_ = e.S.SetKV(kvBudgetSent, month)
	e.send(push.Alert{Kind: push.ChargeBudget, Title: "Charging budget reached", Body: "This month's charging has cost " + strconv.FormatFloat(cost, 'f', 2, 64) + ", over your budget of " + strconv.FormatFloat(limit, 'f', 2, 64) + ".", URL: "#/charging"})
}

func (e *Engine) zone() *time.Location {
	if e.Loc != nil {
		return e.Loc()
	}
	return time.UTC
}
