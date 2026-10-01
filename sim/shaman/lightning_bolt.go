package shaman

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var LightningBoltRankMap = spellData.LightningBolt

func (shaman *Shaman) registerLightningBoltSpell() {
	shaman.LightningBoltOverloads = make(map[int32]*core.Spell, LightningBoltRankMap.Len())
	LightningBoltRankMap.Each(func(rank int32, config *spelldata.Spell) {
		shaman.RegisterSpell(shaman.newLightningBoltSpellConfig(config, rank, false))
		shaman.LightningBoltOverloads[rank] = shaman.RegisterSpell(shaman.newLightningBoltSpellConfig(config, rank, true))
	})
}

func (shaman *Shaman) newLightningBoltSpellConfig(config *spelldata.Spell, rank int32, isElementalOverload bool) core.SpellConfig {
	shamConfig := ShamSpellConfig{
		ActionID:            core.ActionID{SpellID: config.ID},
		Rank:                rank,
		IsElementalOverload: isElementalOverload,
		BaseFlatCost:        int32(config.Cost()),
		BonusCoefficient:    config.DamageEffect().Coeff(),
		BaseCastTime:        config.CastTime(),
	}
	spellConfig := shaman.newElectricSpellConfig(shamConfig)

	spellConfig.ClassSpellMask = core.TernaryInt64(isElementalOverload, SpellMaskLightningBoltOverload, SpellMaskLightningBolt)
	spellConfig.MissileSpeed = 20

	spellConfig.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseDamage := config.DamageEffect().Roll(sim, core.CharacterLevel)
		result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

		spell.WaitTravelTime(sim, func(sim *core.Simulation) {
			if !isElementalOverload && result.Landed() && sim.Proc(shaman.GetOverloadChance(), "Lightning Bolt Elemental Overload") {
				shaman.LightningBoltOverloads[rank].Cast(sim, target)
			}

			spell.DealDamage(sim, result)
		})
	}

	return spellConfig
}
