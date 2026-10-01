package druid

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Every rank is registered: a rotation can downrank when mana runs short, and the sim applies no
// downranking penalty (the beta client has none at 20; nothing says otherwise for 60).
var StarfireRankMap = spellData.Starfire

func (druid *Druid) registerStarfireSpell(rankConfig *spelldata.Spell) {
	spell := druid.RegisterSpell(Humanoid|Moonkin, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rankConfig.ID},
		SpellSchool:    rankConfig.SpellSchool(),
		DefenseType:    rankConfig.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		ClassSpellMask: DruidSpellStarfire,
		Flags:          core.SpellFlagAPL,
		Rank:           rankConfig.RankNumber(),
		MaxRange:       float64(rankConfig.MaxRange),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rankConfig.Cost()),
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      rankConfig.GCD(),
				CastTime: rankConfig.CastTime(),
			},
		},

		BonusCoefficient: rankConfig.DamageEffect().Coeff(),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, rankConfig.DamageEffect().Roll(sim, core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
		},
	})

	druid.Starfire = append(druid.Starfire, spell)
}
