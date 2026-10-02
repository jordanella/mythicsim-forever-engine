package feralcat

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/druid"
)

// Only Shifting Power: the Feral tab's eleventh talent, with no Furor to add Energy to a shift.
const shiftingPowerOnly = "-00000000001"

// Shifting Power with Improved Shifting Power at 1 and 2 points. Shredding Attacks stays at 3 points, as
// the tree needs it before Shifting Power.
func improvedShiftingPower(ranks int) string {
	return "-0000003" + "000" + "1" + "000" + string(rune('0'+ranks))
}

// A Cat Form druid at 1 second of the fight with no Energy, so a gain is the whole Energy bar.
func shiftingPowerCat(t *testing.T, talents string) (*core.Simulation, *druid.Druid) {
	t.Helper()
	player := &proto.Player{Name: "Cat", Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf,
		TalentsString: talents, Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
		Spec: DefaultSpecOptions, Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL}}
	sim := core.NewSim(&proto.RaidSimRequest{SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid:      core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: core.MakeSingleTargetEncounter(0)}, simsignals.CreateSignals())
	sim.Reset()
	cat := sim.Raid.Parties[0].Players[0].(druid.DruidAgent).GetDruid()
	sim.CurrentTime = time.Second
	cat.SpendEnergy(sim, cat.CurrentEnergy(), cat.NewEnergyMetrics(core.ActionID{SpellID: 1}))
	return sim, cat
}

// Client 1322605: 55% of base mana for 40 Energy, in Cat Form, on a 1 second global cooldown.
func TestShiftingPowerConvertsManaToEnergy(t *testing.T) {
	sim, cat := shiftingPowerCat(t, shiftingPowerOnly)

	manaBefore := cat.CurrentMana()
	wantCost := float64(int32(55*cat.BaseMana) / 100)
	if wantCost <= 0 {
		t.Fatalf("55%% of base mana %v is %v", cat.BaseMana, wantCost)
	}
	// "It costs the same Mana as Cat Form."
	if catFormCost := cat.CatForm.Cost.GetCurrentCost(); cat.ShiftingPower.Cost.GetCurrentCost() != catFormCost {
		t.Errorf("Shifting Power costs %v Mana, Cat Form %v", cat.ShiftingPower.Cost.GetCurrentCost(), catFormCost)
	}

	if !cat.ShiftingPower.Cast(sim, cat.CurrentTarget) {
		t.Fatal("Shifting Power did not cast in Cat Form")
	}
	if !cat.InForm(druid.Cat) {
		t.Error("Shifting Power left Cat Form")
	}
	if got := manaBefore - cat.CurrentMana(); got != wantCost {
		t.Errorf("Shifting Power spent %v Mana, want 55%% of base mana (%v)", got, wantCost)
	}
	if got := cat.CurrentEnergy(); got != 40 {
		t.Errorf("Shifting Power gave %v Energy, want 40", got)
	}
	if got := cat.GCD.TimeToReady(sim); got != time.Second {
		t.Errorf("Shifting Power's global cooldown is %v, want 1s", got)
	}
}

// The cooldown is 16 seconds, and Improved Shifting Power takes 4 a rank off it.
func TestShiftingPowerCooldownAndImproved(t *testing.T) {
	for _, tc := range []struct {
		name     string
		talents  string
		cooldown time.Duration
	}{
		{"no ranks", shiftingPowerOnly, 16 * time.Second},
		{"1 rank", improvedShiftingPower(1), 12 * time.Second},
		{"2 ranks", improvedShiftingPower(2), 8 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, cat := shiftingPowerCat(t, tc.talents)
			if !cat.ShiftingPower.Cast(sim, cat.CurrentTarget) {
				t.Fatal("Shifting Power did not cast")
			}
			if got := cat.ShiftingPower.TimeToReady(sim); got != tc.cooldown {
				t.Errorf("Shifting Power is ready in %v, want %v", got, tc.cooldown)
			}
		})
	}
}

// "Shifting Power's cost is reduced by effects that reduce the cost of Shapeshifting": Natural Shapeshifter
// takes 10% a rank off Cat Form and Shifting Power alike.
func TestShiftingPowerCostFollowsNaturalShapeshifter(t *testing.T) {
	var uncut float64
	for _, ranks := range []int{0, 1, 2, 3} {
		talents := shiftingPowerOnly + "-0000" + string(rune('0'+ranks))
		sim, cat := shiftingPowerCat(t, talents)
		cost := cat.ShiftingPower.Cost.GetCurrentCost()
		if ranks == 0 {
			uncut = cost
		}
		want := uncut * (1 - 0.1*float64(ranks))
		if diff := cost - want; diff < -1 || diff > 1 {
			t.Errorf("Natural Shapeshifter %d: Shifting Power costs %v Mana, want about %v", ranks, cost, want)
		}
		if catForm := cat.CatForm.Cost.GetCurrentCost(); catForm != cost {
			t.Errorf("Natural Shapeshifter %d: Cat Form costs %v, Shifting Power %v", ranks, catForm, cost)
		}

		manaBefore := cat.CurrentMana()
		if !cat.ShiftingPower.Cast(sim, cat.CurrentTarget) {
			t.Fatalf("Natural Shapeshifter %d: Shifting Power did not cast", ranks)
		}
		if spent := manaBefore - cat.CurrentMana(); spent < cost-1e-6 || spent > cost+1e-6 {
			t.Errorf("Natural Shapeshifter %d: spent %v Mana, cost read %v", ranks, spent, cost)
		}
	}
}

// Shifting Power needs Cat Form and the talent.
func TestShiftingPowerNeedsCatFormAndTheTalent(t *testing.T) {
	sim, cat := shiftingPowerCat(t, shiftingPowerOnly)
	cat.ClearForm(sim)
	if cat.ShiftingPower.CanCast(sim, cat.CurrentTarget) {
		t.Error("Shifting Power can be cast out of Cat Form")
	}

	_, untalented := shiftingPowerCat(t, "")
	if untalented.ShiftingPower != nil {
		t.Error("a Cat without the talent knows Shifting Power")
	}
}

// Client 70170 took Tiger's Fury out of the spell book; it is no longer cast.
func TestTigersFuryIsGone(t *testing.T) {
	_, cat := shiftingPowerCat(t, DefaultTalents)
	if spell := cat.GetSpell(core.ActionID{SpellID: 5217}); spell != nil {
		t.Errorf("the Cat still registers Tiger's Fury (%s)", spell.ActionID)
	}
	if aura := cat.GetAura("Tiger's Fury"); aura != nil {
		t.Error("the Cat still registers the Tiger's Fury aura")
	}
}
