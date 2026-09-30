package warrior

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core"
)

// Revenge and Shield Slam roll the range their client rows state: the tooltips' 138-168 and 640-670.
func TestRevengeAndShieldSlamRollTheirRows(t *testing.T) {
	for _, c := range []struct {
		name   string
		lo, hi float64
	}{{"Revenge", 138, 168}, {"Shield Slam", 640, 670}} {
		rank := spellData.Revenge.Highest()
		if c.name == "Shield Slam" {
			rank = spellData.ShieldSlam.Highest()
		}
		effect := rank.DamageEffect()
		lo, hi := math.Round(effect.Min(core.CharacterLevel)), math.Round(effect.Max(core.CharacterLevel))
		if lo != c.lo || hi != c.hi {
			t.Errorf("%s rolls %v-%v, want %v-%v", c.name, lo, hi, c.lo, c.hi)
		}
	}
}
