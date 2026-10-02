package dps

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Eureka! 1259813's cost and damage effects as client 1.60.1.70170 states them (family 4, the same mask on
// both). The engine's spell store does not carry the racial rows, so the mask is copied here; the 70124
// mask was {710934758, 4, 0, 0}. The third effect, the periodic bonus Rend had, is a dummy with no mask now.
var eurekaAbilities = core.ClassFlags{Family: 4, Mask: [4]uint32{1784679630, 5, 1, 0}}

func TestEurekaListsMatchTheClientRow(t *testing.T) {
	player := &proto.Player{
		Name: "Gnome", Race: proto.Race_RaceGnome, Class: proto.Class_ClassWarrior,
		Equipment: &proto.EquipmentSpec{}, TalentsString: ArmsTalents, Spec: DefaultOptions,
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.Reset()
	war := sim.Raid.Parties[0].Players[0].(*DpsWarrior)
	lists := war.EurekaSpells()
	if lists.Tick != 0 {
		t.Errorf("the periodic list is %d, want empty: the client's third effect is a dummy", lists.Tick)
	}

	checked := 0
	for _, spell := range war.Spellbook {
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
	if checked < 8 {
		t.Fatalf("only %d warrior spells were comparable with the client rows", checked)
	}
}
