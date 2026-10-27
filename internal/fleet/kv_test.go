package fleet

import (
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheEIAKeyEnteredOnTheFirstCarIsUsedByTheOthers(t *testing.T) {
	core, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first, _ := store.Open(t.TempDir())
	second, _ := store.Open(t.TempDir())
	t.Cleanup(func() { core.Close(); first.Close(); second.Close() })
	a := &CarKV{Car: first, Core: core, Primary: true}
	b := &CarKV{Car: second, Core: core}

	if err := a.SetKV("setting:CARTRAK_EIA_KEY", "one-key"); err != nil {
		t.Fatal(err)
	}
	if got := b.GetKV("setting:CARTRAK_EIA_KEY"); got != "one-key" {
		t.Fatalf("the second car starts with the installation's key: %q", got)
	}
	if err := b.SetKV("setting:CARTRAK_EIA_KEY", "own-key"); err != nil {
		t.Fatal(err)
	}
	if b.GetKV("setting:CARTRAK_EIA_KEY") != "own-key" || a.GetKV("setting:CARTRAK_EIA_KEY") != "one-key" {
		t.Fatal("a car can set its own without changing the other")
	}
	// the gas price and the economy are about each car's own comparison, so they are not shared
	_ = a.SetKV("setting:CARTRAK_GAS_PRICE", "3.50")
	if b.GetKV("setting:CARTRAK_GAS_PRICE") != "" {
		t.Fatal("the gas price stays with its car")
	}
}

func TestAKeySavedOnTheFirstCarBeforeItWasSharedReachesTheInstallationAtStart(t *testing.T) {
	core, _ := store.Open(t.TempDir())
	first, _ := store.Open(t.TempDir())
	second, _ := store.Open(t.TempDir())
	t.Cleanup(func() { core.Close(); first.Close(); second.Close() })
	_ = first.SetKV("setting:CARTRAK_EIA_KEY", "old-key") // saved by a version where the key was not shared
	b := &CarKV{Car: second, Core: core}
	if b.GetKV("setting:CARTRAK_EIA_KEY") != "" {
		t.Fatal("the second car has nothing yet")
	}
	ShareInherited(first, core)
	if got := b.GetKV("setting:CARTRAK_EIA_KEY"); got != "old-key" {
		t.Fatalf("the second car now uses it: %q", got)
	}
	_ = core.SetKV("setting:CARTRAK_EIA_KEY", "newer")
	ShareInherited(first, core)
	if core.GetKV("setting:CARTRAK_EIA_KEY") != "newer" {
		t.Fatal("a value already in the installation is never overwritten")
	}
}
