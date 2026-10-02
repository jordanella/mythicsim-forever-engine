package warlock

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

func (warlock *Warlock) registerHellfire() {
	rank := spellData.Hellfire.Highest()
	// The self-burn tick: effect 1 is the periodic trigger that fires Hellfire Effect.
	tick := rank.Effect(dbcenums.A_PERIODIC_DAMAGE, 0)
	// Each tick fires Hellfire Effect at the enemies around the warlock. Its row marked it Cannot Crit
	// up to client 70124; 70170 dropped the flag, so the area hits crit wherever the row allows it.
	burnCanCrit := !rank.Effect(dbcenums.A_PERIODIC_TRIGGER_SPELL, 0).Trigger().CannotCrit()

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
				// Rolled once: the warlock burns exactly what it deals.
				tickDamage := tick.Average(core.CharacterLevel)

				outcome := dot.Spell.OutcomeTickMagicHitNoHitCounter
				if burnCanCrit {
					outcome = dot.Spell.OutcomeTickMagicHitAndCrit
				}
				dot.Spell.CalcPeriodicAoeDamage(sim, tickDamage, outcome)
				if tickDamage > warlock.CurrentHealth() {
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
