package core

import (
	"strings"
	"testing"
	"time"
)

// expectPanic runs f and fails unless it panics with a message containing want.
func expectPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic containing %q, got none", want)
		}
		if msg, _ := r.(string); !strings.Contains(msg, want) {
			t.Fatalf("expected a panic containing %q, got %v", want, r)
		}
	}()
	f()
}

func fakeCaster(sim *Simulation) *FakeAgent {
	return sim.Raid.Parties[0].Players[0].(*FakeAgent)
}

// A spell that can crit but never declared a DefenseType has to fail on its first roll, even at 0%
// crit chance, rather than on the first crit that happens to land.
func TestCritRollWithoutDefenseTypePanics(t *testing.T) {
	sim := SetupFakeSim()
	fa := fakeCaster(sim)
	spell := fa.RegisterSpell(SpellConfig{
		ActionID:         ActionID{SpellID: 9001},
		SpellSchool:      SpellSchoolHoly,
		ProcMask:         ProcMaskSpellDamage,
		DamageMultiplier: 1,
		BonusCritPercent: -100,
		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			spell.CalcAndDealDamage(sim, target, 100, spell.OutcomeMagicHitAndCrit)
		},
	})

	expectPanic(t, "has no DefenseType", func() { spell.Cast(sim, fa.CurrentTarget) })
}

func TestHealingCritRollWithoutDefenseTypePanics(t *testing.T) {
	sim := SetupFakeSim()
	fa := fakeCaster(sim)
	spell := fa.RegisterSpell(SpellConfig{
		ActionID:         ActionID{SpellID: 9002},
		SpellSchool:      SpellSchoolHoly,
		ProcMask:         ProcMaskSpellHealing,
		Flags:            SpellFlagHelpful,
		DamageMultiplier: 1,
		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			spell.CalcAndDealHealing(sim, target, 100, spell.OutcomeHealingCrit)
		},
	})

	expectPanic(t, "has no DefenseType", func() { spell.Cast(sim, &fa.Unit) })
}

// The same spell with a DefenseType casts normally, so the guard only rejects the missing field.
func TestCritRollWithDefenseTypeCasts(t *testing.T) {
	sim := SetupFakeSim()
	fa := fakeCaster(sim)
	spell := fa.RegisterSpell(SpellConfig{
		ActionID:         ActionID{SpellID: 9003},
		SpellSchool:      SpellSchoolHoly,
		DefenseType:      DefenseTypeMagic,
		ProcMask:         ProcMaskSpellDamage,
		DamageMultiplier: 1,
		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			spell.CalcAndDealDamage(sim, target, 100, spell.OutcomeMagicHitAndCrit)
		},
	})

	spell.Cast(sim, fa.CurrentTarget)
	if spell.SpellMetrics[fa.CurrentTarget.UnitIndex].Casts != 1 {
		t.Fatal("expected the spell to cast once")
	}
}

// An OnSnapshot that sets the base amount but not the attacker multiplier leaves the multiplier at
// the 0 an expired dot resets to, and every tick would deal nothing. Dots cannot be registered once
// the environment is finalized, so these swap the fake shaman's own dot callback.
func TestSnapshotWithoutAttackerMultiplierPanics(t *testing.T) {
	sim := SetupFakeSim()
	fa := fakeCaster(sim)
	fa.Dot.onSnapshot = func(_ *Simulation, _ *Unit, dot *Dot) {
		dot.SnapshotBaseDamage = 50
	}

	expectPanic(t, "not SnapshotAttackerMultiplier", func() { fa.Dot.Apply(sim) })
}

// An OnSnapshot that computes a multiplier of 0 is left alone: only a multiplier that was never
// written is an error.
func TestSnapshotWithComputedZeroMultiplierIsAllowed(t *testing.T) {
	sim := SetupFakeSim()
	fa := fakeCaster(sim)
	fa.Dot.onSnapshot = func(_ *Simulation, _ *Unit, dot *Dot) {
		dot.SnapshotBaseDamage = 50
		dot.SnapshotAttackerMultiplier = 0
	}

	fa.Dot.Apply(sim)
	if !fa.Dot.IsActive() || fa.Dot.SnapshotAttackerMultiplier != 0 {
		t.Fatalf("expected an active dot with a multiplier of 0, got %v", fa.Dot.SnapshotAttackerMultiplier)
	}
}

func TestCooldownDurationWithoutTimerPanics(t *testing.T) {
	sim := SetupFakeSim()
	fa := fakeCaster(sim)
	expectPanic(t, "Cast.CD Duration w/o Timer", func() {
		fa.RegisterSpell(SpellConfig{
			ActionID: ActionID{SpellID: 9006},
			Cast:     CastConfig{CD: Cooldown{Duration: time.Minute}},
		})
	})
	expectPanic(t, "Cast.SharedCD Duration w/o Timer", func() {
		fa.RegisterSpell(SpellConfig{
			ActionID: ActionID{SpellID: 9007},
			Cast:     CastConfig{SharedCD: Cooldown{Duration: time.Minute}},
		})
	})
}
