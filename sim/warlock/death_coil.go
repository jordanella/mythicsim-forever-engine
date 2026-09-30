package warlock

import (
	"github.com/wowsims/forever/sim/core"
)

// The generator files Death Coil's damage under E_HEALTH_LEECH, which it has no Direct role for, so
// the row's Direct is nil and the damage and its 0.214 coefficient are read off the effect.
func (warlock *Warlock) registerDeathCoil() {
	rank := spellData.DeathCoil.Highest()
	baseDamage := rank.EffectN(1).Average(core.CharacterLevel)

	healingSpell := warlock.GetOrRegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.ID}.WithTag(1),
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       core.SpellFlagPassiveSpell | core.SpellFlagHelpful,

		DamageMultiplier: 1,
		ThreatMultiplier: 0,
	})

	warlock.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL | core.SpellFlagBinary,
		ClassSpellMask: WarlockSpellDeathCoil,
		MissileSpeed:   float64(rank.Speed),
		MaxRange:       float64(rank.MaxRange),

		ManaCost: core.ManaCostOptions{FlatCost: int32(rank.Cost())},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: rank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    warlock.NewTimer(),
				Duration: max(rank.Cooldown(), rank.CategoryCooldown()),
			},
		},

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,
		BonusCoefficient:         rank.EffectN(1).Coeff(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
				if result.Landed() {
					healingSpell.CalcAndDealHealing(sim, healingSpell.Unit, result.Damage, healingSpell.OutcomeHealing)
				}
			})
		},
	})
}
