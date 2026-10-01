package mage

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// TODO: was Ranks(7, 6); Forever's Flamestrike tops out at rank 6, so only that remains.
var FlameStrikeRankMap = spelldata.Ranked(spellData.Flamestrike.Rank(6).ID)

// Flamestrike
// https://www.wowhead.com/forever/spell=10216
//
// Calls down a pillar of fire, burning all enemies within the area for X Fire damage and an additional
// Y Fire damage over 8 sec. Only one Flamestrike can be active per Mage at a time.
func (mage *Mage) registerFlamestrike(rankConfig *spelldata.Spell) {
	actionID := core.ActionID{SpellID: rankConfig.ID}
	// Flamestrike's periodic damage is the spell FlamestrikeTriggered casts each tick, at the same
	// rank; the tick length is Flamestrike's own periodic dummy.
	tick := spellData.FlamestrikeTriggered.Rank(rankConfig.RankNumber()).DamageEffect()
	tickLength := rankConfig.Effect(dbcenums.A_PERIODIC_DUMMY, 0).Period()

	spell := mage.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    rankConfig.SpellSchool(),
		DefenseType:    rankConfig.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: MageSpellFlamestrike,
		Rank:           rankConfig.RankNumber(),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rankConfig.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      rankConfig.GCD(),
				CastTime: rankConfig.CastTime(),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: rankConfig.DamageEffect().Coeff(),
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			IsAOE: true,
			Aura: core.Aura{
				ActionID: actionID,
				Label:    "Flamestrike" + mage.Label + " " + rankConfig.Rank,
			},
			NumberOfTicks:    int32(rankConfig.Duration() / tickLength),
			TickLength:       tickLength,
			BonusCoefficient: tick.Coeff(),
			OnTick: func(sim *core.Simulation, _ *core.Unit, dot *core.Dot) {
				dot.Spell.CalcAndDealPeriodicAoeDamage(sim, tick.Average(core.CharacterLevel), dot.OutcomeTickMagicHit)
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			spell.CalcAndDealAoeDamageWithVariance(sim, spell.OutcomeMagicHitAndCrit, func(sim *core.Simulation, _ *core.Spell) float64 {
				return rankConfig.DamageEffect().Roll(sim, core.CharacterLevel)
			})
			spell.AOEDot().Apply(sim)
		},
	})

	mage.Flamestrike = append(mage.Flamestrike, spell)
}
