package buffs

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/stats"
)

// Forever's Faerie Fire and Curse of Recklessness take the same 505 armor and do not stack it: with
// both up the target loses 505, not 1010. The curse keeps everything else it does.
func TestFaerieFireAndCurseOfRecklessnessShareTheirArmor(t *testing.T) {
	type debuff func(*core.Unit, bool, int32) *core.Aura
	measure := func(apply ...debuff) stats.Stats {
		target := shapeTarget()
		sim := &core.Simulation{}
		before := target.GetStats()
		for _, a := range apply {
			a(target, false, 0).Activate(sim)
		}
		after := target.GetStats()
		return after.Subtract(before)
	}

	ff := measure(FaerieFireAura)
	cor := measure(CurseOfRecklessnessAura)
	both := measure(FaerieFireAura, CurseOfRecklessnessAura)
	if ff[stats.Armor] != -505 || cor[stats.Armor] != -505 {
		t.Fatalf("each alone: Faerie Fire %v armor, Curse of Recklessness %v, want -505", ff[stats.Armor], cor[stats.Armor])
	}
	if both[stats.Armor] != -505 {
		t.Errorf("together they take %v armor, want -505", both[stats.Armor])
	}
	for stat := range both {
		if stats.Stat(stat) == stats.Armor {
			continue
		}
		if want := ff[stat] + cor[stat]; both[stat] != want {
			t.Errorf("together %v is %v, want %v", stats.Stat(stat), both[stat], want)
		}
	}
}
