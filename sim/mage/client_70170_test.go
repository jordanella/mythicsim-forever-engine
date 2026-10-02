package mage

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// Client 1.60.1.70170 returned Combustion to 3 charges (ProcCharges 4 to 3 on 11129): the aura ends with the
// third non-periodic fire crit, which is what the sim reads from the row.
func TestCombustionEndsAfterThreeCrits(t *testing.T) {
	if got := spellData.Combustion.Highest().ProcCharges; got != 3 {
		t.Fatalf("Combustion's row states %d charges, want 3", got)
	}

	sim, mage := newGnomeMage(t)
	target := mage.CurrentTarget
	combustion := mage.GetSpell(core.ActionID{SpellID: spellData.Combustion.Highest().ID})
	aura := mage.GetAura("Combustion")
	scorch := mage.GetSpell(core.ActionID{SpellID: spellData.Scorch.Highest().ID})
	if combustion == nil || aura == nil || scorch == nil {
		t.Fatalf("Combustion %v, its aura %v, Scorch %v: want all registered", combustion != nil, aura != nil, scorch != nil)
	}
	scorch.BonusCritPercent = 200 // every landed hit crits

	combustion.SkipCastAndApplyEffects(sim, target)
	if !aura.IsActive() {
		t.Fatal("Combustion did not start")
	}
	crits := int32(0)
	for i := 0; i < 200 && aura.IsActive(); i++ {
		before := scorch.SpellMetrics[target.UnitIndex].Crits
		scorch.SkipCastAndApplyEffects(sim, target)
		crits += scorch.SpellMetrics[target.UnitIndex].Crits - before
	}
	if aura.IsActive() {
		t.Fatal("Combustion never ended")
	}
	if crits != 3 {
		t.Errorf("Combustion ended after %d crits, want 3", crits)
	}
}

// Client 1.60.1.70170 gave Fire Vulnerability (22959) and Winter's Chill (12579) Always Hit: the debuff
// "will not roll a second time to see if it resists". The sim has no second roll: a landed Scorch adds a
// stack at the talent's own chance, and so does a landed Frost hit.
func TestImprovedScorchRollsOnlyTheTalentChance(t *testing.T) {
	if got := spellData.ImprovedScorch.FractionAt(3); got != 1 {
		t.Fatalf("Improved Scorch 3/3 procs at %v, this test needs 1", got)
	}

	sim, mage := newGnomeMage(t)
	target := mage.CurrentTarget
	scorch := mage.GetSpell(core.ActionID{SpellID: spellData.Scorch.Highest().ID})
	stacks := int32(0)
	for i := 0; i < 30; i++ {
		before := scorch.SpellMetrics[target.UnitIndex]
		scorch.SkipCastAndApplyEffects(sim, target)
		after := scorch.SpellMetrics[target.UnitIndex]
		if after.Hits+after.Crits > before.Hits+before.Crits {
			stacks = min(stacks+1, 5)
		}
		if got := mage.ImprovedScorchAura.GetStacks(); got != stacks {
			t.Fatalf("Fire Vulnerability at %d stacks after a landed Scorch, want %d", got, stacks)
		}
	}
	if stacks != 5 {
		t.Fatalf("Fire Vulnerability reached %d stacks in 30 Scorches, want 5", stacks)
	}
}

func TestWintersChillStacksOnEveryLandedFrostHit(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome, TalentsString: FrostTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
			Spec:     &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
	target := mage.CurrentTarget
	if mage.WintersChillAura == nil {
		t.Fatal("no Winter's Chill aura")
	}
	if got := spellData.WintersChill.EffectAt(2).FractionAt(5); got != 1 {
		t.Fatalf("Winter's Chill 5/5 procs at %v, this test needs 1", got)
	}

	frostbolt := mage.GetSpell(core.ActionID{SpellID: spellData.Frostbolt.Highest().ID})
	maxStacks := mage.WintersChillAura.MaxStacks
	stacks := int32(0)
	for i := 0; i < 30; i++ {
		before := frostbolt.SpellMetrics[target.UnitIndex]
		frostbolt.SkipCastAndApplyEffects(sim, target)
		sim.Step() // Frostbolt's damage lands when its missile arrives
		after := frostbolt.SpellMetrics[target.UnitIndex]
		if after.Hits+after.Crits > before.Hits+before.Crits {
			stacks = min(stacks+1, maxStacks)
		}
		if got := mage.WintersChillAura.GetStacks(); got != stacks {
			t.Fatalf("Winter's Chill at %d stacks after %d landed Frostbolts, want %d", got, i+1, stacks)
		}
	}
	if stacks != maxStacks {
		t.Fatalf("Winter's Chill reached %d stacks in 30 Frostbolts, want %d", stacks, maxStacks)
	}
}
