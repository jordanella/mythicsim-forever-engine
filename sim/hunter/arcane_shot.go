package hunter

import (
	"github.com/wowsims/forever/sim/core"
)

// The beta client carries no spell power coefficient on Arcane Shot at all, so Classic's stand.
var arcaneShotCoefficients = [9]float64{0, .204, .3, .429, .429, .429, .429, .429, .429}

// Nor any attack power share, but the beta's combat logs show one: four level 20 hunters with known
// agility and gear (foreverlogs.gg, 2026-09-26) hit for base + 0.10-0.12 of ranged attack power.
// An aspect swap settles that it scales: one hunter (report 2673) lost 0.10-0.12 a point of the 35
// ranged attack power Aspect of the Hawk rank 2 gives while Cheetah was up instead.
// ponytail: fitted at level 20 from four hunters; refit when a level 60 log shows otherwise.
const arcaneShotRAPCoefficient = 0.11

func (hunter *Hunter) registerArcaneShotSpell(timer *core.Timer) {
	rank := spellData.ArcaneShot.Highest()
	baseDamage := rank.DamageEffect().Average(core.CharacterLevel)

	hunter.ArcaneShot = hunter.RegisterRangedSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ClassSpellMask: HunterSpellArcaneShot,
		ProcMask:       core.ProcMaskRangedSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		MissileSpeed:   float64(rank.Speed),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rank.Cost()),
		},
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    timer,
				Duration: max(rank.Cooldown(), rank.CategoryCooldown()),
			},
		},

		BonusCoefficient: arcaneShotCoefficients[rank.RankNumber()],

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, baseDamage+arcaneShotRAPCoefficient*spell.RangedAttackPower(target), spell.OutcomeRangedHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
