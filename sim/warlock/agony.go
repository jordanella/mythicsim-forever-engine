package warlock

import (
	"github.com/wowsims/forever/sim/core"
)

// Forever renamed Curse of Agony to Bane of Agony and moved it onto the bane slot, so it holds the
// target beside a curse. It ramps: the snapshot pays half the tick, and the other half is added
// back every four ticks, leaving the last four at 150%. Amplify Curse adds half again and is spent
// on the snapshot.
func (warlock *Warlock) registerCurseOfAgony() {
	rank := spellData.BaneOfAgony.Highest()
	tick := rank.PeriodicEffect()
	amplify := 1 + spellData.AmplifyCurse.EffectAt(1).FractionAt(1)

	// Each target's Agony ramps by its own step: Amplify Curse raises one application's step and not
	// another's, so one shared step let a later cast on one target change the ramp on every other.
	var rampSteps []float64
	rampStep := func(target *core.Unit) *float64 {
		for int(target.UnitIndex) >= len(rampSteps) {
			rampSteps = append(rampSteps, 0)
		}
		return &rampSteps[target.UnitIndex]
	}

	warlock.CurseOfAgony = warlock.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: WarlockSpellCurseOfAgony,

		ManaCost: core.ManaCostOptions{FlatCost: int32(rank.Cost())},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: rank.GCD(),
			},
		},

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,
		BonusCoefficient:         tick.Coeff(),

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Bane of Agony",
				Tag:   "Affliction",
			},
			NumberOfTicks:    int32(rank.Duration() / tick.Period()),
			TickLength:       tick.Period(),
			BonusCoefficient: tick.Coeff(),

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				base := tick.Average(core.CharacterLevel)
				if warlock.AmplifyCurseAura.IsActive() {
					base *= amplify
					warlock.AmplifyCurseAura.Deactivate(sim)
				}

				step := rampStep(target)
				*step = base * 0.5
				dot.Snapshot(target, *step)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, periodicTickOutcome(rank, dot))
				if dot.TickCount()%4 == 0 {
					dot.SnapshotBaseDamage += *rampStep(target)
				}
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {
				dot := spell.Dot(target)
				warlock.takeBaneSlot(sim, target, dot.Aura)
				dot.Apply(sim)
			}
			spell.DealOutcome(sim, result)
		},

		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
			// The ramp averages out to the plain tick over the full duration.
			dot := spell.Dot(target)
			if useSnapshot {
				return dot.CalcSnapshotDamage(sim, target, spell.OutcomeExpectedMagicAlwaysHit)
			}
			return spell.CalcPeriodicDamage(sim, target, tick.Average(core.CharacterLevel), spell.OutcomeExpectedMagicAlwaysHit)
		},
	})
}
