package warrior

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
)

func (warrior *Warrior) registerThunderClap() {
	thunderClapRank := spellData.ThunderClap.Highest()
	thunderClapBaseDamage := thunderClapRank.DamageEffect().Average(core.CharacterLevel)
	// The client row carries no attack power share, but Forever adds one, as it does for Rend: the level
	// 20 beta claps (rank 2, 23) land 29-31 before Defensive Stance's -10% at 230-300 attack power
	// (foreverlogs 2677/2680/2683), and 61 rank 1 hits from 20 warriors at 77-393 attack power fit
	// 0.0255 of attack power (#583). Fitted below level 20; the same share at 60 is assumed.
	const apShare = 0.0255
	// The slow is -20 and adds to the time between attacks (1.2x); Conqueror's 5 piece (26110, all
	// effects +50%) makes it 30%. The bid is how far from 1 the target's speed factor is.
	thunderClapSlow := thunderClapRank.Effects[1].BaseValue()
	thunderClapBid := func() float64 {
		return 1 - 1/core.SlowedTimeMultiplier(thunderClapSlow*(1+warrior.thunderClapEffectBonus))
	}

	auras := warrior.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		// The clap bids its whole slow in the attack-speed category, so the strongest single slow on
		// the target is the only one applied; the effect applies what the bid is worth.
		aura := buffs.ThunderClapAura(target, true, 0)
		speed := 1.0
		bid := aura.ExclusiveEffects[0]
		bid.OnGain = func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			speed = 1 - ee.Priority
			ee.Aura.Unit.MultiplyMeleeSpeed(sim, speed)
		}
		bid.OnExpire = func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			ee.Aura.Unit.MultiplyMeleeSpeed(sim, 1/speed)
		}
		return aura
	})

	warrior.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: thunderClapRank.ID},
		SpellSchool: thunderClapRank.SpellSchool(),
		// Thunder Clap is Physical but Magic in SpellCategories: it rolls on the spell hit table
		// (Classic logs show full resists) and crits on spell crit chance for 1.5x. Warriors have
		// no base spell crit, so logs without Totem of Wrath show none (0 of 799 landed hits from
		// 6 prot warriors on fresh.warcraftlogs.com, 2026-09-14). In Forever armor doesn't touch it:
		// in the level 20 beta logs (foreverlogs 2677/2683) Chud and Skeefzy's white hits vary
		// 2.3-3.6x across mobs while their claps vary 1.1x.
		DefenseType:    thunderClapRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskRangedSpecial,
		Flags:          core.SpellFlagAPL | core.SpellFlagBinary | core.SpellFlagIgnoreResists,
		ClassSpellMask: SpellMaskThunderClap,

		RageCost: core.RageCostOptions{
			Cost: int32(thunderClapRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: thunderClapRank.GCD(),
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: cooldownOf(thunderClapRank),
			},
		},

		DamageMultiplier: 1,
		// Measured in the level 20 beta (2026-09-28): threat equals damage before the stance modifier,
		// 17 damage for 14 threat in Battle Stance (x0.8) and ~21 in Defensive (x1.3). Was 2.5.
		ThreatMultiplier: 1,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.StanceMatches(BattleStance | DefensiveStance)
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			results := spell.CalcCleaveDamage(sim, target, int32(thunderClapRank.MaxTargets), thunderClapBaseDamage+apShare*spell.MeleeAttackPower(target), spell.OutcomeMagicHitAndCrit)
			warrior.CastNormalizedSweepingStrikesAttack(results, sim)

			for _, result := range results {
				if result.Landed() {
					aura := auras.Get(result.Target)
					if bid := aura.ExclusiveEffects[0]; bid.Priority != thunderClapBid() {
						bid.SetPriority(sim, thunderClapBid())
					}
					aura.Activate(sim)
				}
				spell.DealDamage(sim, result)
			}
		},

		RelatedAuraArrays: auras.ToMap(),
	})
}
