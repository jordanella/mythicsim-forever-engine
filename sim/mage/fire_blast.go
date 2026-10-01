package mage

import (
	"github.com/wowsims/forever/sim/core"
)

// spellData.FireBlast holds the seven trainer ranks, 2136 to 10199. The client also carries 400616
// to 400623, the copies the Season of Discovery rune passive Overheat (400615) swaps onto the action
// bar. Overheat is an Engrave grant with no place in Forever, and the generator drops its stand-ins.
func (mage *Mage) registerFireBlastSpell() {
	fireBlastRank := spellData.FireBlast.Highest()

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: fireBlastRank.ID},
		SpellSchool:    fireBlastRank.SpellSchool(),
		DefenseType:    fireBlastRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: MageSpellFireBlast,

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(fireBlastRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: fireBlastRank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    mage.NewTimer(),
				Duration: max(fireBlastRank.Cooldown(), fireBlastRank.CategoryCooldown()),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: fireBlastRank.DamageEffect().Coeff(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, fireBlastRank.DamageEffect().Roll(sim, core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
		},
	})
}
