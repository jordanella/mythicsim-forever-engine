package rogue

import (
	"github.com/wowsims/forever/sim/core"
)

var eviscerateRank = spellData.Eviscerate.Highest()

func (rogue *Rogue) registerEviscerate() {
	// Rank 9 rolls 54-162 (108, Variance 1) plus 170 a combo point (EffectPointsPerResource), as
	// Wowhead Forever's tooltip prints it; the spread is on the base only.
	damage := eviscerateRank.DamageEffect()
	comboDamageBonus := float64(damage.PointsPerResource) + rogue.DeathmantleBonus

	rogue.Eviscerate = rogue.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: eviscerateRank.ID},
		SpellSchool:    eviscerateRank.SpellSchool(),
		DefenseType:    eviscerateRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | SpellFlagFinisher | core.SpellFlagAPL,
		MetricSplits:   6,
		ClassSpellMask: RogueSpellEviscerate,
		MaxRange:       core.MaxMeleeRange,

		EnergyCost: core.EnergyCostOptions{
			Cost:          int32(eviscerateRank.Cost()),
			Refund:        eviscerateRank.MissRefund(),
			RefundMetrics: rogue.EnergyRefundMetrics,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: eviscerateRank.GCD(),
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(rogue.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.ComboPoints() > 0
		},

		DamageMultiplier:         1,
		DamageMultiplierAdditive: 1,
		ThreatMultiplier:         1,

		BonusCoefficient: eviscerateRank.DamageEffect().Coeff(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)

			comboPoints := float64(rogue.ComboPoints())
			baseDamage := damage.Roll(sim, core.CharacterLevel) + comboDamageBonus*comboPoints +
				0.03*comboPoints*spell.MeleeAttackPower(target)

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				rogue.ApplyFinisher(sim, spell)
			} else {
				spell.IssueRefund(sim)
			}

			spell.DealDamage(sim, result)
		},
	})
}
