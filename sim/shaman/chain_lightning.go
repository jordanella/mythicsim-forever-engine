package shaman

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var ChainLightningRankMap = spellData.ChainLightning

func (shaman *Shaman) registerChainLightningSpell() {
	maxHits := min(3, shaman.Env.TotalTargetCount())
	sharedCDTimer := shaman.NewTimer()
	shaman.ChainLightningOverloads = make(map[int32][]*core.Spell, ChainLightningRankMap.Len())
	ChainLightningRankMap.Each(func(rank int32, config *spelldata.Spell) {
		shaman.newChainLightningSpell(config, rank, false, sharedCDTimer)
		for range maxHits {
			shaman.ChainLightningOverloads[rank] = append(shaman.ChainLightningOverloads[rank], shaman.newChainLightningSpell(config, rank, true, nil))
		}
	})
}

func (shaman *Shaman) newChainLightningSpell(config *spelldata.Spell, rank int32, isElementalOverload bool, sharedCDTimer *core.Timer) *core.Spell {
	shamConfig := ShamSpellConfig{
		ActionID:            core.ActionID{SpellID: config.ID},
		Rank:                rank,
		IsElementalOverload: isElementalOverload,
		BaseFlatCost:        int32(config.Cost()),
		BonusCoefficient:    config.DamageEffect().Coeff(),
		SpellSchool:         core.SpellSchoolNature,
		Overloads:           shaman.ChainLightningOverloads,
		BounceReduction:     0.7,
		ClassSpellMask:      core.TernaryInt64(isElementalOverload, SpellMaskChainLightningOverload, SpellMaskChainLightning),
		BaseCastTime:        config.CastTime(),
	}
	spellConfig := shaman.newElectricSpellConfig(shamConfig)
	if !isElementalOverload {
		spellConfig.Cast.CD = core.Cooldown{
			Timer:    sharedCDTimer,
			Duration: max(config.Cooldown(), config.CategoryCooldown()),
		}
	}
	maxHits := int32(3)
	maxHits = min(maxHits, shaman.Env.TotalTargetCount())

	spellConfig.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		curTarget := target
		bounce := shamConfig.BounceReduction + shaman.ChainLightningBounceBonus

		// Damage calculation and DealDamage are in separate loops so that e.g. a spell power proc
		// can't proc on the first target and apply to the second
		numHits := min(maxHits, shaman.Env.ActiveTargetCount())
		results := make([]*core.SpellResult, numHits)
		for hitIndex := range numHits {
			baseDamage := config.DamageEffect().Roll(sim, core.CharacterLevel)
			results[hitIndex] = spell.CalcDamage(sim, curTarget, baseDamage, spell.OutcomeMagicHitAndCrit)

			curTarget = sim.Environment.NextActiveTargetUnit(curTarget)
			spell.DamageMultiplier *= bounce
		}

		for hitIndex := range numHits {
			if !isElementalOverload && results[hitIndex].Landed() && sim.Proc(shaman.GetOverloadChance()/3, "Chain Lightning Elemental Overload") {
				shamConfig.Overloads[rank][hitIndex].Cast(sim, results[hitIndex].Target)
			}
			spell.DealDamage(sim, results[hitIndex])
			spell.DamageMultiplier /= bounce
		}
	}

	return shaman.RegisterSpell(spellConfig)
}
