package hunter

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// Expose Prey only procs on "targets with Hunter's Mark" (Wowhead Forever), off melee and ranged
// attacks (client 1310532 ProcTypeMask 340).
func TestExposePreyNeedsHuntersMark(t *testing.T) {
	opened := func(mark bool) bool {
		sim := core.NewSim(&proto.RaidSimRequest{
			SimOptions: &proto.SimOptions{RandomSeed: 1},
			Raid: &proto.Raid{Debuffs: &proto.Debuffs{HuntersMark: mark}, Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
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
		if hunter.Talents.ExposePrey == 0 {
			t.Fatal("the Survival talents take no Expose Prey")
		}
		// The proc lands one spell batch window after the shot.
		for range 500 {
			hunter.OnSpellHitDealt(sim, hunter.AutoAttacks.RangedAuto(), &core.SpellResult{Target: hunter.CurrentTarget, Outcome: core.OutcomeHit})
		}
		for sim.CurrentTime <= core.SpellBatchWindow && !hunter.DefensiveState.IsActive() {
			sim.Step()
		}
		return hunter.DefensiveState.IsActive()
	}

	if opened(false) {
		t.Error("Expose Prey opened Mongoose Bite on a target without Hunter's Mark")
	}
	if !opened(true) {
		t.Error("500 shots at a marked target never opened Mongoose Bite")
	}
}
