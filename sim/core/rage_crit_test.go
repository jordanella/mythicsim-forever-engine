package core

import (
	"testing"

	"github.com/wowsims/forever/sim/core/stats"
)

// Client 1.60.1.70170: a critical strike with an auto attack gives 75% more Rage, 1.75 times what the
// same swing gives as a plain hit, and the damage it dealt does not matter: a crit that deals double a
// hit's damage still pays 1.75 times, not 2.
func TestCritAutoAttackRageIsOneAndThreeQuartersOfAHit(t *testing.T) {
	const hitDamage = 500.0

	for _, tc := range []struct {
		name     string
		prepare  func(fw *FakeRageWarrior)
		spell    func(fw *FakeRageWarrior) *Spell
		wantHit  float64
		wantCrit float64
	}{
		{
			name:    "one-hand main hand",
			prepare: func(fw *FakeRageWarrior) {},
			spell:   func(fw *FakeRageWarrior) *Spell { return fw.AutoAttacks.MHAuto() },
			// 2.6 * 3.46, then times 1.75
			wantHit: 8.996, wantCrit: 15.743,
		},
		{
			name: "two-hand main hand",
			prepare: func(fw *FakeRageWarrior) {
				fw.AutoAttacks.MH().NormalizedSwingSpeed = TwoHandNormalizedSwingSpeed
			},
			spell: func(fw *FakeRageWarrior) *Spell { return fw.AutoAttacks.MHAuto() },
			// 2.6 * 4.5
			wantHit: 11.7, wantCrit: 20.475,
		},
		{
			name:    "off hand",
			prepare: func(fw *FakeRageWarrior) {},
			spell:   func(fw *FakeRageWarrior) *Spell { return fw.AutoAttacks.OHAuto() },
			// 1.8 * 3.46 * 0.5
			wantHit: 3.114, wantCrit: 5.4495,
		},
		{
			name:    "multiplied off hand",
			prepare: func(fw *FakeRageWarrior) { fw.SetOffHandRageMultiplier(1.5) },
			spell:   func(fw *FakeRageWarrior) *Spell { return fw.AutoAttacks.OHAuto() },
			// 1.8 * 3.46 * 0.5 * 1.5
			wantHit: 4.671, wantCrit: 8.17425,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim := SetupFakeRageSim()
			fw := sim.Raid.Parties[0].Players[0].(*FakeRageWarrior)
			tc.prepare(fw)
			spell := tc.spell(fw)

			hit := rageFromAutoAttack(sim, fw, spell, OutcomeHit, hitDamage)
			fw.ResetRageBar(sim, 0)
			crit := rageFromAutoAttack(sim, fw, spell, OutcomeCrit, 2*hitDamage)

			if !WithinToleranceFloat64(tc.wantHit, hit, 0.0005) {
				t.Fatalf("hit paid %0.4f Rage, want %0.4f", hit, tc.wantHit)
			}
			if !WithinToleranceFloat64(tc.wantCrit, crit, 0.0005) {
				t.Fatalf("crit paid %0.4f Rage, want %0.4f", crit, tc.wantCrit)
			}
			if !WithinToleranceFloat64(1.75, crit/hit, 0.0000001) {
				t.Fatalf("crit paid %0.6f times a hit, want 1.75", crit/hit)
			}
		})
	}
}

// A bar built without the bonus (Cat Form's) pays a crit exactly what a hit pays.
func TestCritAutoAttackRageBonusIsOptional(t *testing.T) {
	previous := fakeCritRageBonus
	fakeCritRageBonus = 0
	t.Cleanup(func() { fakeCritRageBonus = previous })

	sim := SetupFakeRageSim()
	fw := sim.Raid.Parties[0].Players[0].(*FakeRageWarrior)

	hit := rageFromAutoAttack(sim, fw, fw.AutoAttacks.MHAuto(), OutcomeHit, 500)
	fw.ResetRageBar(sim, 0)
	crit := rageFromAutoAttack(sim, fw, fw.AutoAttacks.MHAuto(), OutcomeCrit, 1000)
	if !WithinToleranceFloat64(8.996, hit, 0.0005) || !WithinToleranceFloat64(8.996, crit, 0.0005) {
		t.Fatalf("without the bonus a hit paid %0.4f and a crit %0.4f Rage, want 8.996 for both", hit, crit)
	}
}

// Rage from a hit taken is a separate rule, and a critical hit taken pays by the damage it dealt like
// any other: 1000 damage on 10000 maximum health is 1000 * 10 / 10000 = 1 Rage.
func TestCritBonusLeavesDamageTakenRageAlone(t *testing.T) {
	sim := SetupFakeRageSim()
	fw := sim.Raid.Parties[0].Players[0].(*FakeRageWarrior)
	fw.Unit.stats[stats.Health] = 10000
	spell := sim.Encounter.ActiveTargetUnits[0].AutoAttacks.MHAuto()

	taken := func(outcome HitOutcome) float64 {
		fw.ResetRageBar(sim, 0)
		result := &SpellResult{
			Target:                           &fw.Unit,
			Outcome:                          outcome,
			PostArmorAndResistanceMultiplier: 1000,
			ArmorAndResistanceMultiplier:     1,
			Damage:                           1000,
		}
		before := fw.CurrentRage()
		fw.Unit.OnSpellHitTaken(sim, spell, result)
		return fw.CurrentRage() - before
	}

	hit := taken(OutcomeHit)
	crit := taken(OutcomeCrit)
	if !WithinToleranceFloat64(1.0, hit, 0.0000001) {
		t.Fatalf("a hit taken paid %0.6f Rage, want 1", hit)
	}
	if hit != crit {
		t.Fatalf("a crit taken paid %0.6f Rage and a hit %0.6f, want the same for the same damage", crit, hit)
	}
}
