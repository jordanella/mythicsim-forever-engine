package rogue

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
)

// Wowhead Forever 31016: 54-162 plus 170 a combo point.
func TestEviscerateReadsItsRow(t *testing.T) {
	damage := eviscerateRank.DamageEffect()
	if lo, hi, step := damage.Min(core.CharacterLevel), damage.Max(core.CharacterLevel), damage.PointsPerResource; lo != 54 || hi != 162 || step != 170 {
		t.Fatalf("Eviscerate rolls %v-%v plus %v a point, want 54-162 plus 170", lo, hi, step)
	}
}
