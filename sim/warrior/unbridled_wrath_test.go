package warrior

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
)

// Client 1.60.1.70170 changed the spell-level ProcChance of Unbridled Wrath (12322) from 60 to 100 and the
// notes say "Fixed an issue with Unbridled Rage having a lower proc chance than intended". The talent's
// per-rank numbers are the trait curve on its effect (12/24/36/48/60, the tooltip's "$m1% chance"), and
// the sim rolls that, not the row's ProcChance, so the proc chance is the tooltip's before and after the
// build and a 5/5 warrior rolls 60% on every white hit. The one-Rage payload is the triggered spell's.
func TestUnbridledWrathRollsTheTooltipChanceByRank(t *testing.T) {
	for rank, want := range []float64{0.12, 0.24, 0.36, 0.48, 0.60} {
		got := spellData.UnbridledWrath.FractionAt(int32(rank + 1))
		if !core.WithinToleranceFloat64(want, got, 0.0000001) {
			t.Errorf("Unbridled Wrath rank %d rolls %v, want %v", rank+1, got, want)
		}
	}
	if got := spellData.UnbridledWrathTriggered.Highest().EnergizeEffect().Tenths(); got != 1 {
		t.Errorf("Unbridled Wrath pays %v Rage, want 1", got)
	}
}
