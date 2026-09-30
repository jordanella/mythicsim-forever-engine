package rogue

import (
	"github.com/wowsims/forever/sim/core"
)

// Was the TBC rank-6 id 26867, which the level squish removed. Derived from the table so
// it follows the data instead of naming a rank that may not exist.
var RuptureSpellID = spellData.Rupture.Highest().ID

var ruptureRank = spellData.Rupture.ByID(RuptureSpellID)

func (rogue *Rogue) registerRupture() {
	tick := ruptureRank.PeriodicEffect()
	tickLength := tick.Period()
	tickDamage := tick.Average(core.CharacterLevel)
	baseTickCount := int32(ruptureRank.Duration() / tickLength)

	// The beta client cut the per combo point step with the tick (rank 6: 60 + 8 -> 35 + 4.73).
	damagePerComboPoint := float64(tick.PointsPerResource)

	rogue.Rupture = rogue.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: ruptureRank.ID},
		SpellSchool:    ruptureRank.SpellSchool(),
		DefenseType:    ruptureRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | SpellFlagFinisher | core.SpellFlagAPL,
		MetricSplits:   6,
		ClassSpellMask: RogueSpellRupture,
		MaxRange:       core.MaxMeleeRange,

		EnergyCost: core.EnergyCostOptions{
			Cost:          int32(ruptureRank.Cost()),
			Refund:        ruptureRank.MissRefund(),
			RefundMetrics: rogue.EnergyRefundMetrics,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: ruptureRank.GCD(),
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

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Rupture",
				Tag:   RogueBleedTag,
			},
			NumberOfTicks: 0, // Set dynamically
			TickLength:    tickLength,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				// Hemorrhage scales the flat damage and the attack power share alike.
				multiplier := 1.0
				if rogue.isHemorrhaging(target) {
					multiplier = HemorrhageRuptureMultiplier
				}
				share := ruptureAttackPowerShare(rogue.ComboPoints()) * multiplier
				flat := (tickDamage + damagePerComboPoint*float64(rogue.ComboPoints())) * multiplier
				dot.SnapshotPhysical(target, flat+share*rogue.Rupture.MeleeAttackPower(target))
				dot.SnapshotAttackPowerShare(target, share, false)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, ruptureRank.TickOutcome(dot))
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHit)
			if result.Landed() {
				dot := spell.Dot(target)
				dot.BaseTickCount = baseTickCount + rogue.ComboPoints()
				dot.Apply(sim)
				rogue.ApplyFinisher(sim, spell)
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},
	})
}

// The share of attack power a tick adds, read at the tick. TBC's share, not in the client; beta logs
// agree: 1% of AP a tick per combo point, capped at 3.
func ruptureAttackPowerShare(comboPoints int32) float64 {
	return []float64{0, 0.01, 0.02, 0.03, 0.03, 0.03}[comboPoints]
}
