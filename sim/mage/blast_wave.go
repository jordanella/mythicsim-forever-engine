package mage

import (
	"github.com/wowsims/forever/sim/core"
)

func (mage *Mage) registerBlastWaveSpell() {
	if !mage.Talents.BlastWave {
		return
	}

	blastWaveRank := spellData.BlastWave.Highest()

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: blastWaveRank.ID},
		Flags:          core.SpellFlagAPL | core.SpellFlagBinary,
		SpellSchool:    blastWaveRank.SpellSchool(),
		DefenseType:    blastWaveRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		ClassSpellMask: MageSpellBlastWave,

		BonusCoefficient: blastWaveRank.DamageEffect().Coeff(),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(blastWaveRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: blastWaveRank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    mage.NewTimer(),
				Duration: max(blastWaveRank.Cooldown(), blastWaveRank.CategoryCooldown()),
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealAoeDamageWithVariance(sim, spell.OutcomeMagicHitAndCrit, func(sim *core.Simulation, _ *core.Spell) float64 {
				return blastWaveRank.DamageEffect().Roll(sim, core.CharacterLevel)
			})
		},
	})
}
