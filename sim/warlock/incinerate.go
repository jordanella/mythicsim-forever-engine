package warlock

import (
	"github.com/wowsims/forever/sim/core"
)

// Forever's Incinerate is a Destruction talent (412758 and up), 2.5 sec, and hits 25% harder on a
// target carrying Immolate - the second effect on every rank of the client's row.
func (warlock *Warlock) registerIncinerate() {
	if !warlock.Talents.Incinerate {
		return
	}

	rank := spellData.Incinerate.Highest()
	immolateBonus := 1 + rank.EffectN(2).Percent()

	warlock.Incinerate = warlock.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: WarlockSpellIncinerate,
		MissileSpeed:   float64(rank.Speed),

		ManaCost: core.ManaCostOptions{FlatCost: int32(rank.Cost())},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      rank.GCD(),
				CastTime: rank.CastTime(),
			},
		},

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,
		BonusCoefficient:         rank.DamageEffect().Coeff(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := rank.DamageEffect().Roll(sim, core.CharacterLevel)
			if warlock.Immolate.Dot(target).IsActive() {
				baseDamage *= immolateBonus
			}

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
