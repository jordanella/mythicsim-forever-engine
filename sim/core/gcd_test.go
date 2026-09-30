package core

import (
	"testing"
	"time"
)

// EffectiveCastTime reads the spell's own GCD, hasted as Cast hastes it, not a flat 1.5 sec.
func TestEffectiveCastTimeUsesTheSpellsOwnGCD(t *testing.T) {
	haste := 1 / 1.1 // Berserking's 10%
	unit := &Unit{CastSpeed: haste}
	spell := func(gcd, castTime time.Duration, ignoreHaste bool) *Spell {
		return &Spell{Unit: unit, DefaultCast: Cast{GCD: gcd, CastTime: castTime}, CastTimeMultiplier: 1, IgnoreHaste: ignoreHaste}
	}

	for _, c := range []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"1.5 sec GCD instant", spell(GCDDefault, 0, false).EffectiveCastTime(), time.Duration(float64(GCDDefault) * haste)},
		{"1 sec totem GCD stays at the 1 sec floor", spell(time.Second, 0, false).EffectiveCastTime(), time.Second},
		{"off-GCD instant", spell(0, 0, false).EffectiveCastTime(), 0},
		{"cast longer than its GCD", spell(GCDDefault, time.Millisecond*2200, false).EffectiveCastTime(), time.Duration(float64(time.Millisecond*2200) * haste)},
		{"IgnoreHaste", spell(GCDDefault, 0, true).EffectiveCastTime(), GCDDefault},
	} {
		if c.got != c.want {
			t.Errorf("%s: %v, want %v", c.name, c.got, c.want)
		}
	}
}
