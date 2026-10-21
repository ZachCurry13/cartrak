package fleet

import (
	"context"
	"fmt"
	"time"

	"github.com/ZachCurry13/cartrak/internal/push"
)

// A happy birthday alert: from 9 in the morning on a person's birthday, once, to each of their own devices outside quiet
// hours. It runs here, not per car, so someone with several cars hears it once. Turning off holiday touches turns it off.

const birthdayHour = 9

func (f *Fleet) birthdayLoop(ctx context.Context) {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		f.birthdays(ctx, time.Now())
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (f *Fleet) birthdays(ctx context.Context, now time.Time) {
	front := f.Front()
	if front == nil || front.Srv.Push == nil {
		return
	}
	now = now.In(front.Cfgs.Location())
	if now.Hour() < birthdayHour {
		return
	}
	users, err := f.Core.ListUsers()
	if err != nil {
		return
	}
	today := now.Format("2006-01-02")
	for _, u := range users {
		p := f.Core.Prefs(u.ID)
		key := fmt.Sprintf("birthday_sent:%d", u.ID)
		if p.HolidayOff || !push.IsBirthday(p.Birthday, now) || f.Core.GetKV(key) == today {
			continue
		}
		a := push.Alert{Kind: "birthday", Title: "🎂 Happy birthday!", Body: "Have a great day. CarTrak dressed up for it.", URL: "#/"}
		if _, waiting := front.Srv.Push.SendToUser(ctx, u.ID, a); !waiting {
			_ = f.Core.SetKV(key, today)
		}
	}
}
