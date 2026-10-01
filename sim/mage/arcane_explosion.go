package mage

import (
	"github.com/wowsims/forever/sim/core"
)

func (mage *Mage) registerArcaneExplosionSpell() {
	arcaneExplosionRank := spellData.ArcaneExplosion.Highest()

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: arcaneExplosionRank.ID},
		SpellSchool:    arcaneExplosionRank.SpellSchool(),
		DefenseType:    arcaneExplosionRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: MageSpellArcaneExplosion,

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(arcaneExplosionRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: arcaneExplosionRank.GCD(),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: arcaneExplosionRank.DamageEffect().Coeff(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealAoeDamageWithVariance(sim, spell.OutcomeMagicHitAndCrit, func(sim *core.Simulation, _ *core.Spell) float64 {
				return arcaneExplosionRank.DamageEffect().Roll(sim, core.CharacterLevel)
			})
		},
	})
}
