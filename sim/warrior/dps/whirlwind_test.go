package dps

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// Build 70170 + hotfix 112347: Whirlwind strikes with both weapons on its own, and Raging Blows
// takes 3 Rage off Whirlwind as well as Cleave.
func TestWhirlwindOffHandAndRagingBlows(t *testing.T) {
	for _, c := range []struct {
		talents  string
		gear     *proto.EquipmentSpec
		wantCost float64
		wantOH   bool
	}{
		{DpsTalents, DualWieldGear.GearSet, 22, true},
		{FuryTalents, DualWieldGear.GearSet, 25, true}, // no Raging Blows, the off hand still swings
		{ArmsTalents, TwoHandGear.GearSet, 25, false},
	} {
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid: core.SinglePlayerRaidProto(&proto.Player{
				Race: proto.Race_RaceOrc, Class: proto.Class_ClassWarrior,
				Equipment: c.gear, TalentsString: c.talents, Spec: DefaultOptions,
			}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{RandomSeed: 1},
		}, simsignals.CreateSignals())
		sim.Reset()
		warrior := sim.Raid.Parties[0].Players[0].(*DpsWarrior)
		if got := warrior.Talents.RagingBlows; got != (c.wantCost == 22) {
			t.Fatalf("%s: Raging Blows taken = %v", c.talents, got)
		}
		ww := warrior.GetSpell(core.ActionID{SpellID: 1680}.WithTag(1))
		if got := ww.Cost.GetCurrentCost(); got != c.wantCost {
			t.Errorf("%s: Whirlwind costs %v Rage, want %v", c.talents, got, c.wantCost)
		}
		if got := warrior.GetSpell(core.ActionID{SpellID: 1680}.WithTag(2)) != nil; got != c.wantOH {
			t.Errorf("%s: off-hand Whirlwind registered = %v, want %v", c.talents, got, c.wantOH)
		}
	}
}

// Build 70170: Booming Voice's second effect takes 5% a point off the shouts' Rage cost.
func TestBoomingVoiceCutsShoutCost(t *testing.T) {
	for talents, want := range map[string]float64{
		FuryTalents: 7.5, // Booming Voice 5/5
		DpsTalents:  10,  // untaken
	} {
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid: core.SinglePlayerRaidProto(&proto.Player{
				Race: proto.Race_RaceOrc, Class: proto.Class_ClassWarrior,
				Equipment: &proto.EquipmentSpec{}, TalentsString: talents, Spec: DefaultOptions,
			}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{RandomSeed: 1},
		}, simsignals.CreateSignals())
		sim.Reset()
		warrior := sim.Raid.Parties[0].Players[0].(*DpsWarrior)
		for _, shout := range []*core.Spell{warrior.BattleShout, warrior.DemoralizingShout} {
			if got := shout.Cost.GetCurrentCost(); got != want {
				t.Errorf("%s: %s costs %v Rage, want %v", talents, shout.ActionID, got, want)
			}
		}
	}
}
