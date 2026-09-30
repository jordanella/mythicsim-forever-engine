package druid

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
)

// The Forever tooltip for rank 5 (9850): 45 Energy, 110% normal damage plus 115.
func TestClawMatchesTooltip(t *testing.T) {
	if cost, flat := clawRank.Cost(), clawRank.DamageEffect().Average(core.CharacterLevel); cost != 45 || flat != 115 || clawWeaponMultiplier != 1.1 {
		t.Errorf("Claw: %v Energy, %v flat, x%v, want 45, 115, x1.1", cost, flat, clawWeaponMultiplier)
	}
}
