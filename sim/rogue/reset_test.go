package rogue

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// A Slice and Dice cast sets the aura's duration from its combo points. The next iteration must
// start from the default again: an aura activated without the cast (an APL Activate Aura) reads
// whatever duration it holds.
func TestSliceAndDiceDurationResetsBetweenIterations(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Rogue", Class: proto.Class_ClassRogue, Race: proto.Race_RaceHuman, TalentsString: CombatTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{}, Spec: DefaultOptions,
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()

	rogue := sim.Raid.Parties[0].Players[0].(RogueAgent).GetRogue()
	defaultDuration := rogue.SliceAndDiceAura.Duration

	rogue.AddComboPoints(sim, 1, rogue.NewComboPointMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionMove}))
	if !rogue.SliceAndDice.Cast(sim, rogue.CurrentTarget) {
		t.Fatal("Slice and Dice did not cast")
	}
	if rogue.SliceAndDiceAura.Duration == defaultDuration {
		t.Fatalf("a 1 combo point Slice and Dice kept the default duration %v; the test no longer exercises the leak", defaultDuration)
	}

	sim.Cleanup()
	sim.Reset()
	if got := rogue.SliceAndDiceAura.Duration; got != defaultDuration {
		t.Errorf("Slice and Dice starts the next iteration at %v, want the default %v", got, defaultDuration)
	}
}
