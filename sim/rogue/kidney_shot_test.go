package rogue

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// The Assassination build with Improved Kidney Shot 2/2 (15th Assassination talent).
func kidneyShotSim(talents string) (*core.Simulation, *Rogue) {
	player := core.WithSpec(&proto.Player{
		Race:          proto.Race_RaceHuman,
		Class:         proto.Class_ClassRogue,
		Equipment:     daggersOnly(),
		Consumables:   &proto.ConsumesSpec{},
		TalentsString: talents,
		Rotation:      &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, DefaultOptions)
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 100},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: &proto.Encounter{Duration: 180, Targets: []*proto.Target{
			{Name: "humanoid", Level: 63, MobType: proto.MobType_MobTypeHumanoid},
		}},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(RogueAgent).GetRogue()
}

// Client 8643: 1 sec plus 1 sec a combo point, and damage taken from the rogue only with the talent.
func TestKidneyShotStunAndImprovedKidneyShot(t *testing.T) {
	for _, tc := range []struct {
		talents string
		want    float64
	}{
		{AssassinationTalents, 1},
		{"00530310551021251-302303202004", 1.1},
	} {
		sim, rogue := kidneyShotSim(tc.talents)
		target := rogue.CurrentTarget
		kidneyShot := rogue.GetSpell(core.ActionID{SpellID: 8643})
		stun := target.GetAura("Kidney Shot - " + rogue.Label)
		metrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: 8643, Tag: 1})

		for i := 0; i < 20 && !stun.IsActive(); i++ {
			rogue.AddComboPoints(sim, 5, metrics)
			kidneyShot.SkipCastAndApplyEffects(sim, target)
		}
		if !stun.IsActive() {
			t.Fatalf("%s: Kidney Shot never landed", tc.talents)
		}

		if got := stun.RemainingDuration(sim); got != 6*time.Second {
			t.Errorf("%s: 5 point stun lasts %v, want 6s", tc.talents, got)
		}
		if !target.PseudoStats.Stunned {
			t.Errorf("%s: target not stunned", tc.talents)
		}
		// Ruthlessness (3/3 in both builds) may hand one back.
		if rogue.ComboPoints() > 1 {
			t.Errorf("%s: %d combo points left after the finisher", tc.talents, rogue.ComboPoints())
		}
		table := rogue.AttackTables[target.UnitIndex]
		if !core.WithinToleranceFloat64(tc.want, table.DamageTakenMultiplier, 1e-9) {
			t.Errorf("%s: damage taken from the rogue x%v while stunned, want x%v", tc.talents, table.DamageTakenMultiplier, tc.want)
		}

		stun.Deactivate(sim)
		if table.DamageTakenMultiplier != 1 || target.PseudoStats.Stunned {
			t.Errorf("%s: after the stun x%v, stunned %v; want x1, false", tc.talents, table.DamageTakenMultiplier, target.PseudoStats.Stunned)
		}
	}
}
