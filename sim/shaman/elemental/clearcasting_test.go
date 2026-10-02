package elemental

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/shaman"
)

// Elemental Focus's Clearcasting (16246) names its spells by class mask, and client 1.60.1.70170 changed
// the mask: Fire Nova moved from word 1 bit 18 to word 0 bit 27 (where Fire Nova Totem already sat) and
// Molten Blast (425339, a Season of Discovery rune the sim has no talent for) left it. Neither changes
// which of the sim's spells it names, which this holds the sim's list to: every registered shaman spell
// spends a Clearcasting charge exactly when the row's mask names it.
func TestClearcastingNamesTheSpellsTheClientRowDoes(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Race: proto.Race_RaceTroll, Class: proto.Class_ClassShaman, Equipment: &proto.EquipmentSpec{},
		Consumables: &proto.ConsumesSpec{}, TalentsString: DefaultTalents + "",
		Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{Options: &proto.ElementalShaman_Options{ClassOptions: &proto.ShamanOptions{}}}})
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 100},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	character := sim.Raid.Parties[0].Players[0].GetCharacter()

	clearcasting := character.GetAura("Clearcasting")
	if clearcasting == nil {
		t.Fatal("no Clearcasting aura on an Elemental Focus shaman")
	}
	mask := spelldata.MustFind(16246).EffectN(1)
	if mask.ClassFlags.Family != 11 {
		t.Fatalf("Clearcasting's class mask is in family %d, want 11", mask.ClassFlags.Family)
	}

	named := map[string]bool{}
	compared := 0
	for _, spell := range character.Spellbook {
		client := spelldata.Find(spell.ActionID.SpellID)
		if client == nil || client.ClassFlags.IsZero() || spell.ClassSpellMask == 0 || spell.Flags.Matches(core.SpellFlagNoOnCastComplete) {
			continue
		}
		// An overload is a copy of the cast on the same id, and Flame Shock's dot half is cast with its hit:
		// neither is a cast of its own, so neither spends a charge.
		if spell.ClassSpellMask&(shaman.SpellMaskOverload|shaman.SpellMaskFlameShockDot) != 0 {
			continue
		}
		compared++
		clearcasting.Activate(sim)
		clearcasting.SetStacks(sim, clearcasting.MaxStacks)
		before := clearcasting.GetStacks()
		clearcasting.OnCastComplete(clearcasting, sim, spell)
		spends := clearcasting.GetStacks() < before
		clearcasting.Deactivate(sim)

		if want := mask.Covers(client); spends != want {
			t.Errorf("%s %d: spends a charge = %v, the client's mask names it = %v", client.Name, client.ID, spends, want)
		}
		if spends {
			named[client.Name] = true
		}
	}
	if compared < 4 {
		t.Fatalf("only %d shaman spells were comparable with the client row", compared)
	}
	for _, name := range []string{"Lightning Bolt", "Chain Lightning", "Lava Burst", "Earth Shock"} {
		if !named[name] {
			t.Errorf("%s does not spend a Clearcasting charge", name)
		}
	}
}
