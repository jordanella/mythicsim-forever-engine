package warlock

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

func (warlock *Warlock) registerHellfire() {
	rank := spellData.Hellfire.Highest()
	// The self-burn tick: effect 1 is the periodic trigger that fires Hellfire Effect.
	tick := rank.Effect(dbcenums.A_PERIODIC_DAMAGE, 0)

	warlock.Hellfire = warlock.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagChanneled | core.SpellFlagAPL,
		ClassSpellMask: WarlockSpellHellfire,

		ManaCost: core.ManaCostOptions{FlatCost: int32(rank.Cost())},
		Cast:     core.CastConfig{DefaultCast: core.Cast{GCD: rank.GCD()}},

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Hellfire",
			},

			IsAOE:                true,
			TickLength:           tick.Period(),
			NumberOfTicks:        int32(rank.Duration() / tick.Period()),
			HasteReducesDuration: true,
			AffectedByCastSpeed:  true,
			BonusCoefficient:     tick.Coeff(),

			OnTick: func(sim *core.Simulation, _ *core.Unit, dot *core.Dot) {
				// Rolled once: the warlock burns exactly what it deals, before any crit.
				tickDamage := tick.Average(core.CharacterLevel)

				// Hellfire Effect (5857, 11681, 11682) lost Cannot Crit in client 1.60.1.70170, so each target's
				// hit rolls for a crit.
				resultSlice := dot.Spell.CalcPeriodicAoeDamage(sim, tickDamage, dot.Spell.OutcomeTickMagicHitAndCrit)
				if resultSlice[0].Damage > warlock.CurrentHealth() {
					dot.Deactivate(sim)
				}

				dot.Spell.DealBatchedPeriodicDamage(sim)
				warlock.RemoveHealth(sim, tickDamage)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.AOEDot().Apply(sim)
		},
	})
}
