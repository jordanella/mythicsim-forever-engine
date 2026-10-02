package warrior

import (
	"github.com/wowsims/forever/sim/core"
)

func (warrior *Warrior) registerWhirlwind() {
	whirlwindRank := spellData.Whirlwind.Highest()

	actionID := core.ActionID{SpellID: whirlwindRank.ID}

	// Hotfix 112347 (build 70170) rewrote 1680's text to "weapon damage from both melee weapons":
	// the off-hand swing Raging Blows used to add is now Whirlwind's own, for anyone dual wielding.
	var whirlwindOH *core.Spell
	if warrior.HasOHWeapon() {
		whirlwindOH = warrior.RegisterSpell(core.SpellConfig{
			ActionID:       actionID.WithTag(2),
			SpellSchool:    core.SpellSchoolPhysical,
			DefenseType:    core.DefenseTypeMelee,
			ProcMask:       core.ProcMaskMeleeOHSpecial,
			ClassSpellMask: SpellMaskWhirlwindOh,
			Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell | core.SpellFlagNoOnCastComplete,

			DamageMultiplier: 1,
			// Not in the client table; our Classic value until measured in game.
			ThreatMultiplier: 1.25,

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				baseDamage := warrior.OHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
				spell.CalcCleaveDamage(sim, target, int32(whirlwindRank.MaxTargets), baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
				spell.DealBatchedAoeDamage(sim)
			},
		})
	}

	warrior.RegisterSpell(core.SpellConfig{
		ActionID:       actionID.WithTag(1),
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		ClassSpellMask: SpellMaskWhirlwind,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

		RageCost: core.RageCostOptions{
			Cost: int32(whirlwindRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: whirlwindRank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: cooldownOf(whirlwindRank),
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		// Not in the client table; our Classic value until measured in game.
		ThreatMultiplier: 1.25,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.StanceMatches(BerserkerStance)
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := warrior.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
			results := spell.CalcCleaveDamage(sim, target, int32(whirlwindRank.MaxTargets), baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			warrior.CastNormalizedSweepingStrikesAttack(results, sim)
			spell.DealBatchedAoeDamage(sim)

			if whirlwindOH != nil {
				whirlwindOH.Cast(sim, target)
			}
		},
	})
}
