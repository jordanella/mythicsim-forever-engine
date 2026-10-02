package rogue

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Eureka! 1259812's cost and damage effects as client 1.60.1.70170 states them (family 8, the same mask on
// both). The engine's spell store does not carry the racial rows, so the mask is copied here; the 70124
// masks were {68290334, 2099464, 0, 0} for the cost and {68290334, 2318, 0, 0} for the damage. The third
// effect, the periodic bonus Garrote and Rupture had, is a dummy with no mask now.
var eurekaAbilities = core.ClassFlags{Family: 8, Mask: [4]uint32{100794910, 2097408, 0, 0}}

func TestEurekaListsMatchTheClientRow(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Race: proto.Race_RaceGnome, Class: proto.Class_ClassRogue, Equipment: daggersOnly(), Consumables: &proto.ConsumesSpec{},
		TalentsString: AssassinationTalents, Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, DefaultOptions)
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	rogue := sim.Raid.Parties[0].Players[0].(RogueAgent).GetRogue()
	lists := rogue.EurekaSpells()
	if lists.Tick != 0 {
		t.Errorf("the periodic list is %d, want empty: the client's third effect is a dummy", lists.Tick)
	}

	checked := 0
	for _, spell := range rogue.Spellbook {
		client := spelldata.Find(spell.ActionID.SpellID)
		if client == nil || client.ClassFlags.IsZero() || spell.ClassSpellMask == 0 {
			continue
		}
		checked++
		want := eurekaAbilities.Matches(client.ClassFlags)
		if got := spell.ClassSpellMask&lists.Cost != 0; got != want {
			t.Errorf("%s %d: on the sim's cost list = %v, client row says %v", client.Name, client.ID, got, want)
		}
		if got := spell.ClassSpellMask&lists.Damage != 0; got != want {
			t.Errorf("%s %d: on the sim's damage list = %v, client row says %v", client.Name, client.ID, got, want)
		}
	}
	if checked < 6 {
		t.Fatalf("only %d rogue spells were comparable with the client rows", checked)
	}
}
