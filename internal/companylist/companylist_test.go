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
