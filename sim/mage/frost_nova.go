package mage

import (
	"github.com/wowsims/forever/sim/core"
)

func (mage *Mage) registerFrostNovaSpell() {
	frostNovaRank := spellData.FrostNova.Highest()

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: frostNovaRank.ID},
		SpellSchool:    frostNovaRank.SpellSchool(),
		DefenseType:    frostNovaRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL | core.SpellFlagBinary,
		ClassSpellMask: MageSpellFrostNova,

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(frostNovaRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: frostNovaRank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    mage.NewTimer(),
				Duration: max(frostNovaRank.Cooldown(), frostNovaRank.CategoryCooldown()),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: frostNovaRank.DamageEffect().Coeff(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			spell.CalcAndDealAoeDamageWithVariance(sim, spell.OutcomeMagicHitAndCrit, func(sim *core.Simulation, _ *core.Spell) float64 {
				return frostNovaRank.DamageEffect().Roll(sim, core.CharacterLevel)
			})
		},
	})
}
