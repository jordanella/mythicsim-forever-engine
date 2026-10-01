package mage

import (
	"github.com/wowsims/forever/sim/core"
)

// Damage bonus against a target the mage counts as frozen, i.e. one held by Fingers of Frost.
const IceLanceFrozenMultiplier = 4.0

func (mage *Mage) registerIceLanceSpell() {
	if !mage.Talents.IceLance {
		return
	}

	iceLanceRank := spellData.IceLance.Highest()

	// The client's damage effect carries no spell power coefficient (the row reads 0), as Season of
	// Discovery's reworked Ice Lance (400640) does, whose 2024-12-04 hotfix raised it from .143 to .572.
	// Two level 20 frost mages in the beta logs side with .143: rank 1 (base 25.7-30.3) hit unfrozen
	// for 33-34 with 30-34 spell power (report 2668) and 29-34 with 19 (Toma, report 2687), and frozen
	// for about 4x that. .429 would put every hit at 36 or more. Not pinned closer: frost talents unknown.
	iceLanceCoefficient := 0.143

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: iceLanceRank.ID},
		SpellSchool:    iceLanceRank.SpellSchool(),
		DefenseType:    iceLanceRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL | core.SpellFlagBinary,
		ClassSpellMask: MageSpellIceLance,
		MissileSpeed:   float64(iceLanceRank.Speed),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(iceLanceRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: iceLanceRank.GCD(),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: iceLanceCoefficient,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, iceLanceRank.DamageEffect().Roll(sim, core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
			// A bonus on the whole hit rather than the base roll, so spell power is multiplied too.
			if mage.IsTargetFrozen() {
				result.Damage *= IceLanceFrozenMultiplier
			}
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
