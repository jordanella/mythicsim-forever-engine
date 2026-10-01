package druid

import "testing"

// Wowhead Forever 9896: 243 damage over 12 sec at 1 point, 855 at 5, before attack power.
func TestRipReadsItsRow(t *testing.T) {
	for cp, want := range map[float64]float64{1: 243, 5: 855} {
		if got := 6 * ripTickDamage(cp, 0); got != want {
			t.Errorf("Rip at %v points deals %v, want %v", cp, got, want)
		}
	}
}
