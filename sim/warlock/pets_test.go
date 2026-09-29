package warlock

import (
	"testing"

	"github.com/wowsims/forever/sim/core/stats"
)

func TestPetStatInheritance(t *testing.T) {
	owner := stats.Stats{}
	owner[stats.SpellHitPercent] = 12
	owner[stats.SpellCritPercent] = 9.5
	owner[stats.SpellDamage] = 400
	owner[stats.PhysicalHitPercent] = 3
	owner[stats.Stamina] = 500

	got := petStatInheritance(owner)

	want := stats.Stats{}
	want[stats.SpellHitPercent] = 12
	want[stats.PhysicalHitPercent] = 12
	want[stats.SpellCritPercent] = 9.5
	want[stats.PhysicalCritPercent] = 9.5
	want[stats.SpellDamage] = 40
	want[stats.AttackPower] = 68
	for stat := range got {
		if diff := got[stat] - want[stat]; diff > 1e-9 || diff < -1e-9 {
			t.Errorf("%s = %v, want %v", stats.Stat(stat).StatName(), got[stat], want[stat])
		}
	}

	// The dynamic inheritance feeds this function deltas, so it must be linear.
	half := stats.Stats{}
	half[stats.SpellDamage] = 200
	if petStatInheritance(half)[stats.AttackPower]*2 != got[stats.AttackPower] {
		t.Error("inheritance is not linear in spell damage")
	}
}
