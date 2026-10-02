package core

import (
	"strings"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core/dbcenums"
)

var (
	wrathRequirement    = CastRequirement{Forms: 0x40000000, ExcludedForms: 0x2, CasterForm: true, NotShapeshifted: true}
	regrowthRequirement = CastRequirement{Forms: 0x2, ExcludedForms: 0x40000000, CasterForm: true, NotShapeshifted: true}
	chargeRequirement   = CastRequirement{Forms: 0x10000}
)

func TestShapeshiftFormRule(t *testing.T) {

	tests := []struct {
		name    string
		req     CastRequirement
		allowed []dbcenums.ShapeshiftForm
		refused []dbcenums.ShapeshiftForm
	}{
		{
			name:    "Wrath",
			req:     wrathRequirement,
			allowed: []dbcenums.ShapeshiftForm{0, dbcenums.FORM_MOONKIN_FORM},
			refused: []dbcenums.ShapeshiftForm{dbcenums.FORM_CAT_FORM, dbcenums.FORM_DIRE_BEAR_FORM, dbcenums.FORM_TREE_FORM},
		},
		{
			name:    "Regrowth",
			req:     regrowthRequirement,
			allowed: []dbcenums.ShapeshiftForm{0, dbcenums.FORM_TREE_FORM},
			refused: []dbcenums.ShapeshiftForm{dbcenums.FORM_MOONKIN_FORM, dbcenums.FORM_CAT_FORM},
		},
		{
			name:    "Charge",
			req:     chargeRequirement,
			allowed: []dbcenums.ShapeshiftForm{dbcenums.FORM_BATTLE_STANCE},
			refused: []dbcenums.ShapeshiftForm{dbcenums.FORM_DEFENSIVE_STANCE, dbcenums.FORM_BERSERKER_STANCE, 0},
		},
		{
			name:    "Charge by constructor",
			req:     InForms(dbcenums.FORM_BATTLE_STANCE),
			allowed: []dbcenums.ShapeshiftForm{dbcenums.FORM_BATTLE_STANCE},
			refused: []dbcenums.ShapeshiftForm{dbcenums.FORM_DEFENSIVE_STANCE, dbcenums.FORM_BERSERKER_STANCE, 0},
		},
		{
			name: "Heroic Strike",
			allowed: []dbcenums.ShapeshiftForm{0, dbcenums.FORM_CAT_FORM, dbcenums.FORM_BEAR_FORM, dbcenums.FORM_DIRE_BEAR_FORM,
				dbcenums.FORM_TREE_FORM, dbcenums.FORM_BATTLE_STANCE, dbcenums.FORM_DEFENSIVE_STANCE, dbcenums.FORM_BERSERKER_STANCE,
				dbcenums.FORM_MOONKIN_FORM},
		},
	}

	for _, test := range tests {
		for _, form := range test.allowed {
			if !test.req.allowsForm(form) {
				t.Errorf("%s should be castable in form %d", test.name, form)
			}
		}
		for _, form := range test.refused {
			if test.req.allowsForm(form) {
				t.Errorf("%s should not be castable in form %d", test.name, form)
			}
		}
	}
}

func TestCastRequirementConstructors(t *testing.T) {
	if InForms(dbcenums.FORM_BATTLE_STANCE) != chargeRequirement {
		t.Errorf("InForms(Battle Stance) = %+v, want %+v", InForms(dbcenums.FORM_BATTLE_STANCE), chargeRequirement)
	}
	wrath := InForms(dbcenums.FORM_MOONKIN_FORM).Excluding(dbcenums.FORM_TREE_FORM).OrCasterForm()
	wrath.NotShapeshifted = true
	if wrath != wrathRequirement {
		t.Errorf("Wrath by constructors = %+v, want %+v", wrath, wrathRequirement)
	}
}

func TestCastRequirementAutoUnshift(t *testing.T) {
	unit := &Unit{ShapeshiftForm: dbcenums.FORM_CAT_FORM}
	spell := &Spell{Unit: unit, CastRequirement: wrathRequirement}

	if reason, _ := spell.castRequirementFailure(); reason != "wrong form" {
		t.Errorf("Wrath in Cat Form without AutoUnshift: %q, want \"wrong form\"", reason)
	}

	unit.AutoUnshift = func(*Simulation) {}
	if reason, unshift := spell.castRequirementFailure(); reason != "" || !unshift {
		t.Errorf("Wrath in Cat Form with AutoUnshift: (%q, %v), want (\"\", true)", reason, unshift)
	}

	unit.ShapeshiftForm = dbcenums.FORM_TREE_FORM
	if reason, unshift := spell.castRequirementFailure(); reason != "" || !unshift {
		t.Errorf("Wrath in Tree of Life with AutoUnshift: (%q, %v), want (\"\", true)", reason, unshift)
	}

	unit.ShapeshiftForm = dbcenums.FORM_MOONKIN_FORM
	spell.CastRequirement = regrowthRequirement
	if reason, unshift := spell.castRequirementFailure(); reason != "" || !unshift {
		t.Errorf("Regrowth in Moonkin Form with AutoUnshift: (%q, %v), want (\"\", true)", reason, unshift)
	}

	unit.ShapeshiftForm = dbcenums.FORM_DEFENSIVE_STANCE
	spell.CastRequirement = chargeRequirement
	if reason, _ := spell.castRequirementFailure(); reason != "wrong form" {
		t.Errorf("Charge in Defensive Stance: %q, want \"wrong form\"", reason)
	}
}

func TestCastRequirementCasterAura(t *testing.T) {
	unit := &Unit{auraTracker: newAuraTracker()}
	catForm := unit.RegisterAura(Aura{Label: "Cat Form", ActionID: ActionID{SpellID: 768}, Duration: NeverExpires})
	shiftingPower := &Spell{Unit: unit, CastRequirement: CastRequirement{CasterAura: 768}}
	shiftingPower.resolveCasterAuras()

	if reason, _ := shiftingPower.castRequirementFailure(); reason != "missing caster aura" {
		t.Errorf("Shifting Power without Cat Form: %q, want \"missing caster aura\"", reason)
	}

	catForm.active = true
	if reason, _ := shiftingPower.castRequirementFailure(); reason != "" {
		t.Errorf("Shifting Power in Cat Form: %q, want castable", reason)
	}

	notInCat := &Spell{Unit: unit, CastRequirement: CastRequirement{ExcludeCasterAura: 768}}
	notInCat.resolveCasterAuras()
	if reason, _ := notInCat.castRequirementFailure(); reason != "excluded caster aura" {
		t.Errorf("a spell excluding Cat Form while in it: %q, want \"excluded caster aura\"", reason)
	}
}

func TestCastRequirementCastPath(t *testing.T) {
	sim := SetupFakeRageSim()
	fw := sim.Raid.Parties[0].Players[0].(*FakeRageWarrior)
	target := sim.Encounter.ActiveTargetUnits[0]
	sim.CurrentTime = 0

	var logs []string
	sim.Log = func(message string, vals ...interface{}) {
		logs = append(logs, message)
	}

	charge := fw.RegisterSpell(SpellConfig{
		ActionID:        ActionID{SpellID: 100},
		Cast:            CastConfig{DefaultCast: Cast{GCD: time.Second}},
		CastRequirement: chargeRequirement,
	})
	fw.ShapeshiftForm = dbcenums.FORM_DEFENSIVE_STANCE
	if charge.CanCast(sim, target) {
		t.Error("Charge should not be castable in Defensive Stance")
	}
	if charge.Cast(sim, target) {
		t.Error("Charge should not cast in Defensive Stance")
	}
	if len(logs) == 0 || !strings.Contains(logs[len(logs)-1], "wrong form") {
		t.Errorf("Charge's failure should log \"wrong form\", logs: %v", logs)
	}

	wrath := fw.RegisterSpell(SpellConfig{
		ActionID:        ActionID{SpellID: 5176},
		Cast:            CastConfig{DefaultCast: Cast{GCD: time.Second}},
		CastRequirement: wrathRequirement,
	})
	unshifts := 0
	fw.AutoUnshift = func(*Simulation) {
		unshifts++
		fw.ShapeshiftForm = 0
	}
	fw.ShapeshiftForm = dbcenums.FORM_CAT_FORM
	if !wrath.CanCast(sim, target) {
		t.Error("Wrath should be castable in Cat Form when the cast leaves it")
	}
	if unshifts != 0 {
		t.Errorf("CanCast left the form %d times, want 0", unshifts)
	}
	if !wrath.Cast(sim, target) {
		t.Error("Wrath should cast in Cat Form when the cast leaves it")
	}
	if unshifts != 1 || fw.ShapeshiftForm != 0 {
		t.Errorf("Wrath's cast left the form %d times, now in form %d; want once, no form", unshifts, fw.ShapeshiftForm)
	}
}
