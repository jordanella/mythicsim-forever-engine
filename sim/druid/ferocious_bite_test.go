package druid

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core"
)

// The Forever tooltip for rank 5 reads 199-259 damage at one combo point and 787-847 at five.
func TestFerociousBiteRangeMatchesTooltip(t *testing.T) {
	e := ferociousBiteRank.DamageEffect()
	for cp, want := range map[float64][2]float64{1: {199, 259}, 5: {787, 847}} {
		lo := ferociousBiteDamage(e.Min(core.CharacterLevel), cp, 0)
		hi := ferociousBiteDamage(e.Max(core.CharacterLevel), cp, 0)
		if math.Abs(lo-want[0]) > 0.5 || math.Abs(hi-want[1]) > 0.5 {
			t.Errorf("%v combo points: %.1f-%.1f, want %v-%v", cp, lo, hi, want[0], want[1])
		}
	}
}
