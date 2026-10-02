package druid

import (
	"github.com/wowsims/forever/sim/core"
)

var swipeRank = spellData.Swipe.Highest()

// Client 1.60.1.70170: "Fixed a bug causing Swipe to not scale with Attack Power. It will now correctly gain
// 3% of the Druid's attack power" and "tooltip will not update". The client rows of every Swipe rank (779,
// 780, 769, 9754, 9908) still carry no BonusCoefficientFromAP, so the share is the patch note's and the
// engine states it. It is added to each target's base hit before Feral Instinct's and the other damage mods.
const swipeAttackPowerCoefficient = 0.03

func (druid *Druid) registerSwipeBearSpell() {
	druid.Swipe = druid.RegisterSpell(Bear, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: swipeRank.ID},
		SpellSchool:    swipeRank.SpellSchool(),
		DefenseType:    swipeRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		ClassSpellMask: DruidSpellSwipe,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

		RageCost: core.RageCostOptions{
			Cost:   int32(swipeRank.Cost()),
			Refund: swipeRank.MissRefund(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: swipeRank.GCD(),
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		// Season of Discovery's "Modifies Threat +101%", which the client does not carry.
		ThreatMultiplier: 2,
		MaxRange:         core.MaxMeleeRange,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			numHits := min(3, len(druid.Env.Encounter.AllTargetUnits))
			for i := 0; i < numHits; i++ {
				aoeTarget := druid.Env.Encounter.AllTargetUnits[i]
				baseDamage := swipeRank.DamageEffect().Average(core.CharacterLevel) +
					swipeAttackPowerCoefficient*spell.MeleeAttackPower(aoeTarget)
				spell.CalcAndDealDamage(sim, aoeTarget, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			}
		},
	})
}
