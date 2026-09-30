package warrior

import (
	"github.com/wowsims/forever/sim/core"
)

func (warrior *Warrior) registerPummel() {
	pummelRank := spellData.Pummel.ByID(6554)
	pummelBaseDamage := pummelRank.DamageEffect().Average(core.CharacterLevel)

	warrior.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: pummelRank.ID},
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		ClassSpellMask: SpellMaskPummel,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		SpellSchool:    pummelRank.SpellSchool(),
		DefenseType:    pummelRank.DefenseTypeCore(),
		MaxRange:       core.MaxMeleeRange,

		RageCost: core.RageCostOptions{
			Cost:   int32(pummelRank.Cost()),
			Refund: pummelRank.MissRefund(),
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: pummelRank.GCD(),
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: cooldownOf(pummelRank),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.StanceMatches(BerserkerStance)
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealDamage(sim, target, pummelBaseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
