package mage

import (
	"github.com/wowsims/forever/sim/core"
)

func (mage *Mage) registerConeOfColdSpell() {
	coneOfColdRank := spellData.ConeOfCold.Highest()

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: coneOfColdRank.ID},
		SpellSchool:    coneOfColdRank.SpellSchool(),
		DefenseType:    coneOfColdRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL | core.SpellFlagBinary,
		ClassSpellMask: MageSpellConeOfCold,

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(coneOfColdRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: coneOfColdRank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    mage.NewTimer(),
				Duration: max(coneOfColdRank.Cooldown(), coneOfColdRank.CategoryCooldown()),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: coneOfColdRank.DamageEffect().Coeff(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			spell.CalcAndDealAoeDamageWithVariance(sim, spell.OutcomeMagicHitAndCrit, func(sim *core.Simulation, _ *core.Spell) float64 {
				return coneOfColdRank.DamageEffect().Roll(sim, core.CharacterLevel)
			})
		},
	})
}
