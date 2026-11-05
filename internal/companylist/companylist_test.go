package companylist

import "testing"

func TestParse(t *testing.T) {
	got, more := Parse("Acme Widgets-1234567-U.S.A.\r\n\n- Beta Corp - 88112 - USA\nAcme Widgets-1234567-U.S.A.\nJust A Name\n")
	want := []Entry{{"Acme Widgets-1234567-U.S.A.", "Acme Widgets", "1234567"}, {"Beta Corp - 88112 - USA", "Beta Corp", "88112"}, {"Just A Name", "Just A Name", ""}}
	if more != 0 || len(got) != len(want) {
		t.Fatalf("got %v more %d", got, more)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%d: got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestParseLimit(t *testing.T) {
	text := ""
	for i := 0; i < 13; i++ {
		text += "Company " + string(rune('A'+i)) + "-100" + string(rune('0'+i%10)) + "-USA\n"
	}
	got, more := Parse(text)
	if len(got) != MaxLines || more != 3 {
		t.Fatalf("got %d more %d", len(got), more)
	}
}

func TestSame(t *testing.T) {
	yes := [][2]string{{"Acme Widgets", "ACME WIDGETS, INC."}, {"Acme", "Acme Widgets Corporation"}}
	no := [][2]string{{"Acme Widgets", "Beta Widgets"}, {"Co", "Co Inc"}, {"", "Acme"}}
	for _, p := range yes {
		if !Same(p[0], p[1]) {
			t.Errorf("%v should match", p)
		}
	}
	for _, p := range no {
		if Same(p[0], p[1]) {
			t.Errorf("%v should not match", p)
		}
	}
}

func TestTheActivityLineUnderEachCompanyIsNotACompany(t *testing.T) {
	list := "Acme Widgets, Inc.-111111111-U.S.A.\n12 - Visit\nBlue-Sky Supplies, LLC-222222222-U.S.A.\n007 - Phone call\nNORTH STAR GOODS-1234567890-U.S.A.\n5 - Visit\n"
	got, more := Parse(list)
	if len(got) != 3 || more != 0 {
		t.Fatalf("three companies and nothing left out: %d, %d: %+v", len(got), more, got)
	}
	if got[1].Name != "Blue-Sky Supplies, LLC" || got[1].Ref != "222222222" || got[2].Name != "NORTH STAR GOODS" || got[2].Ref != "1234567890" {
		t.Fatalf("names and numbers: %+v", got)
	}
	if got, _ := Parse("Fifth Street Cafe\n2 - Visit"); len(got) != 1 || got[0].Name != "Fifth Street Cafe" {
		t.Fatalf("a plain name stays: %+v", got)
	}
}
