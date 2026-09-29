package druid

import (
	"github.com/wowsims/forever/sim/core"
)

var ripRank = spellData.Rip.Highest()
var ripTick = ripRank.PeriodicEffect()

// The per-combo-point damage and the attack power share are not in the generated table - the client
// states the tick base only - so they stay the sim's client-read values: 25.5 a point a tick at rank
// 6, and 1% of attack power a point, which stops growing at four points.
const ripTickPerComboPoint = 25.5

func (druid *Druid) registerRipSpell() {
	druid.Rip = druid.RegisterSpell(Cat, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: ripRank.ID},
		SpellSchool:    ripRank.SpellSchool(),
		DefenseType:    ripRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		ClassSpellMask: DruidSpellRip,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		Rank:           ripRank.RankNumber(),

		EnergyCost: core.EnergyCostOptions{
			Cost:   int32(ripRank.Cost()),
			Refund: ripRank.MissRefund(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: ripRank.GCD(),
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return druid.ComboPoints() > 0
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		MaxRange:         core.MaxMeleeRange,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Rip",
			},
			NumberOfTicks: int32(ripRank.Duration() / ripTick.Period()),
			TickLength:    ripTick.Period(),

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				comboPoints := float64(druid.ComboPoints())
				share := ripAttackPowerShare(comboPoints)
				dot.SnapshotPhysical(target, ripFlatDamage(comboPoints)+share*dot.Spell.MeleeAttackPower(target))
				dot.SnapshotAttackPowerShare(target, share, false)
				druid.UpdateBleedPower(druid.Rip, sim, target, true, true)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, periodicTickOutcome(ripRank, dot))
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHitNoHitCounter)
			if result.Landed() {
				spell.Dot(target).Apply(sim)
				druid.SpendComboPoints(sim, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},

		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
			if useSnapshot {
				dot := spell.Dot(target)
				return dot.CalcSnapshotDamage(sim, target, dot.OutcomeTick)
			}
			// Assume 5 CP for projections.
			tick := ripTickDamage(5, spell.MeleeAttackPower(target))
			result := spell.CalcPeriodicDamage(sim, target, tick, spell.OutcomeExpectedMagicAlwaysHit)
			attackTable := spell.Unit.AttackTables[target.UnitIndex]
			critChance := spell.PhysicalCritChance(attackTable)
			result.Damage *= 1 + critChance*(spell.CritDamageMultiplier(attackTable)-1)
			return result
		},
	})

	druid.Rip.ShortName = "Rip"
}

func ripFlatDamage(comboPoints float64) float64 {
	return ripTick.Average(core.CharacterLevel) + ripTickPerComboPoint*comboPoints
}

// The share of attack power a tick adds, read at the tick.
func ripAttackPowerShare(comboPoints float64) float64 {
	return 0.01 * min(comboPoints, 4)
}

func ripTickDamage(comboPoints float64, attackPower float64) float64 {
	return ripFlatDamage(comboPoints) + ripAttackPowerShare(comboPoints)*attackPower
}

func (druid *Druid) CurrentRipCost() float64 {
	return druid.Rip.Cost.GetCurrentCost()
}
