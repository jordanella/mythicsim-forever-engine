package hunter

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// Mongoose Bite "can only be performed after you dodge" (client CasterAuraState 1 on every rank,
// Wowhead Forever tooltip): a dodge the hunter makes opens the window, a hit taken does not.
func TestMongooseBiteOpensOnADodge(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "sv", Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc, TalentsString: SurvivalTalents,
			Equipment: WeaponsOnly, Buffs: &proto.IndividualBuffs{},
			Spec: &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{ClassOptions: &proto.HunterOptions{
				Ammo: proto.HunterOptions_Doomshot, PetType: proto.HunterOptions_PetNone}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()

	hunter := sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()
	swing := sim.Encounter.AllTargetUnits[0].AutoAttacks.MHAuto()

	hunter.OnSpellHitTaken(sim, swing, &core.SpellResult{Target: &hunter.Unit, Outcome: core.OutcomeHit})
	if hunter.DefensiveState.IsActive() {
		t.Fatal("a hit taken opened Mongoose Bite")
	}
	hunter.OnSpellHitTaken(sim, swing, &core.SpellResult{Target: &hunter.Unit, Outcome: core.OutcomeDodge})
	if !hunter.DefensiveState.IsActive() {
		t.Fatal("a dodge did not open Mongoose Bite")
	}
	if !hunter.MongooseBite.Cast(sim, hunter.CurrentTarget) || hunter.DefensiveState.IsActive() {
		t.Fatal("Mongoose Bite did not cast off the dodge, or left the window open")
	}
}
