package rogue

import (
	"github.com/wowsims/forever/sim/core"
)

var garroteRank = spellData.Garrote.Highest()

// A tick adds 3% of attack power (beta logs, report 2683), read at the tick.
const garroteAttackPowerShare = 0.03

func (rogue *Rogue) registerGarrote() {
	tick := garroteRank.PeriodicEffect()
	tickLength := tick.Period()
	tickDamage := tick.Average(core.CharacterLevel)

	rogue.Garrote = rogue.GetOrRegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: garroteRank.ID},
		SpellSchool:    garroteRank.SpellSchool(),
		DefenseType:    garroteRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | SpellFlagBuilder | core.SpellFlagAPL,
		ClassSpellMask: RogueSpellGarrote,
		MaxRange:       core.MaxMeleeRange,

		EnergyCost: core.EnergyCostOptions{
			Cost:   int32(garroteRank.Cost()),
			Refund: garroteRank.MissRefund(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: garroteRank.GCD(),
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			if !rogue.IsStealthed() {
				return false
			}
			// Dirty Deeds drops the positional requirement.
			return rogue.Talents.DirtyDeeds > 0 || !rogue.PseudoStats.InFrontOfTarget
		},

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Garrote",
				Tag:   RogueBleedTag,
			},
			NumberOfTicks: int32(garroteRank.Duration() / tickLength),
			TickLength:    tickLength,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.SnapshotPhysical(target, tickDamage+dot.Spell.MeleeAttackPower(target)*garroteAttackPowerShare)
				dot.SnapshotAttackPowerShare(target, garroteAttackPowerShare, false)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, garroteRank.TickOutcome(dot))
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialNoBlockDodgeParryNoCrit)
			if result.Landed() {
				rogue.AddComboPoints(sim, 1, spell.ComboPointMetrics())
				spell.Dot(target).Apply(sim)
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},
	})
}
